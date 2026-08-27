package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"eng/internal/planmeta"
	"eng/internal/project"
	"eng/internal/workflow"
)

// planSummary is the one fact eng start's bootstrap needs about a plan that
// eng doctor/eng workflow status don't already report project-wide: is it
// still in flight. See spec.md's "gap analysis" section 3.
type planSummary struct {
	Dir   string // relative to the project root, e.g. ".plans/2026-08-27-add-x"
	State string
}

// scanPlans walks <projectRoot>/.plans/* one level deep and returns every
// plan whose state is not terminal (workflow.Terminal), sorted by Dir for
// deterministic output. A project with no .plans/ directory, or where every
// plan is terminal, returns an empty (non-nil-safe) slice and no error —
// absence is not a failure, matching every other harness command's
// convention for a directory that simply doesn't exist yet.
func scanPlans(projectRoot string) []planSummary {
	entries, err := os.ReadDir(filepath.Join(projectRoot, ".plans"))
	if err != nil {
		return nil
	}

	var out []planSummary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rel := filepath.Join(".plans", e.Name())
		meta, err := planmeta.Load(filepath.Join(projectRoot, rel))
		if err != nil {
			continue // no plan.yaml here — not a plan directory, skip silently
		}
		if workflow.Terminal(meta.State) {
			continue
		}
		out = append(out, planSummary{Dir: rel, State: meta.State})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out
}

// bootstrapStatus is the small, bounded set of facts eng start hands a
// freshly launched agent so it knows, from session start, that it is
// running under the Global Engineering Harness — without having to be told
// in chat. Every field is read from the same sources eng doctor itself
// reads (harnessDir/harnessVersion/project.Load/registeredAdapters), so the
// bootstrap prompt and eng doctor's own next output cannot disagree.
type bootstrapStatus struct {
	HarnessInstalled bool
	HarnessHome      string
	HarnessVersion   string
	ProjectRoot      string
	ProjectMode      string // "" | legacy | none | broken | modern | hybrid
	PlanningMode     string // e.g. "spec_first" — "" if the project isn't initialized

	TriageEnabled     bool
	PlanReviewEnabled bool
	VerifierEnabled   bool

	UnfinishedPlans []planSummary

	CodexInstalled bool
	CodexWired     bool
	CodexInvokable bool
}

// gatherBootstrapStatus assembles bootstrapStatus for dir. Filesystem reads
// only — no process spawn, no LLM call — so it's fully unit-testable
// against a temp directory.
func gatherBootstrapStatus(dir string) bootstrapStatus {
	s := bootstrapStatus{ProjectRoot: dir}

	hDir := harnessDir()
	if info, err := os.Stat(hDir); err == nil && info.IsDir() {
		s.HarnessInstalled = true
		s.HarnessHome = hDir
		s.HarnessVersion = harnessVersion()
	}

	modeResult := project.DetectModeResult(dir)
	s.ProjectMode = modeResult.Mode

	if cfg, err := project.Load(dir); err == nil {
		s.PlanningMode = cfg.Workflow.PlanningModeOrDefault()
		s.TriageEnabled = cfg.Workflow.TriageEnabled()
		s.PlanReviewEnabled = cfg.Workflow.PlanReviewEnabled()
		s.VerifierEnabled = cfg.Workflow.VerifierEnabled()
	}

	s.UnfinishedPlans = scanPlans(dir)

	for _, a := range registeredAdapters(dir) {
		if a.Name() != "codex" {
			continue
		}
		s.CodexInstalled = a.Available()
		if s.CodexInstalled {
			if _, err := a.Doctor(); err == nil {
				s.CodexInvokable = true
			}
		}
		s.CodexWired = true // reaching this loop at all means it's registered
	}

	return s
}

const maxListedUnfinishedPlans = 5

// renderBootstrapPrompt turns a bootstrapStatus into the exact text handed
// to `claude --append-system-prompt`. Pure string formatting — no I/O — and
// deterministic: the same status always renders identically. See spec.md's
// "Bootstrap prompt shape" section for the field-by-field rationale.
func renderBootstrapPrompt(s bootstrapStatus) string {
	var b strings.Builder

	b.WriteString("You are running under the Global Engineering Harness.\n\n")

	installed := "NOT installed"
	if s.HarnessInstalled {
		installed = s.HarnessHome
	}
	fmt.Fprintf(&b, "Harness home:    %s\n", installed)
	fmt.Fprintf(&b, "Harness version: %s\n", orNone(s.HarnessVersion))
	fmt.Fprintf(&b, "Project root:    %s\n", s.ProjectRoot)
	fmt.Fprintf(&b, "Project mode:    %s\n", orNone(s.ProjectMode))
	fmt.Fprintf(&b, "Workflow:        planning=%s triage=%s plan_review=%s verifier=%s\n",
		orNone(s.PlanningMode), onOff(s.TriageEnabled), onOff(s.PlanReviewEnabled), onOff(s.VerifierEnabled))
	fmt.Fprintf(&b, "Codex:           installed=%v wired=%v invokable=%v\n",
		s.CodexInstalled, s.CodexWired, s.CodexInvokable)
	fmt.Fprintf(&b, "Plans:           %s\n\n", renderPlansLine(s.UnfinishedPlans))

	b.WriteString("Before any workflow-sensitive action, verify current state through `eng` " +
		"(e.g. `eng doctor`, `eng workflow status <plan-dir>`) rather than trusting this " +
		"summary alone — it is a snapshot from session start. Role activation, state-role " +
		"compatibility, the executor/verifier gates, and tool policy (`eng tools invoke`) all " +
		"remain enforced exactly as documented in core/runtime/METHOD.md — this prompt does " +
		"not bypass any of them. Do not conclude the harness is absent from the lack of a " +
		"project-local CLAUDE.md/.claude/ — the harness install above is the source of truth. " +
		"Do not auto-resume a COMPLETED plan; treat it as history unless the human explicitly " +
		"asks to resume it.")

	return b.String()
}

func renderPlansLine(plans []planSummary) string {
	switch {
	case len(plans) == 0:
		return "no unfinished plans — start clean"
	case len(plans) == 1:
		return fmt.Sprintf("1 unfinished: %s (%s)", plans[0].Dir, plans[0].State)
	default:
		shown := plans
		suffix := ""
		if len(shown) > maxListedUnfinishedPlans {
			suffix = fmt.Sprintf(", ...and %d more", len(shown)-maxListedUnfinishedPlans)
			shown = shown[:maxListedUnfinishedPlans]
		}
		parts := make([]string, len(shown))
		for i, p := range shown {
			parts[i] = fmt.Sprintf("%s (%s)", p.Dir, p.State)
		}
		return fmt.Sprintf("%d unfinished — ask the human which to resume, do not guess: %s%s",
			len(plans), strings.Join(parts, ", "), suffix)
	}
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
