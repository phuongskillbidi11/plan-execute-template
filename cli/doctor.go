package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/capabilities"
	"eng/internal/project"
	"eng/internal/skillvalidate"
)

func cmdDoctor(args []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Println("eng doctor")
	fmt.Println()

	hDir := harnessDir()
	if info, err := os.Stat(hDir); err == nil && info.IsDir() {
		fmt.Printf("Harness install:   found at %s (version %s)\n", hDir, harnessVersion())
	} else {
		fmt.Println("Harness install:   NOT FOUND — run `eng install --from <path>`")
	}

	modeResult := project.DetectModeResult(dir)
	switch modeResult.Mode {
	case "legacy":
		fmt.Println("Project mode:      legacy (CLAUDE.md/.plans found, no .agent/) — fully compatible, no action required")
	case "none":
		fmt.Println("Project mode:      none — not yet initialized (`eng init` to enable)")
	case "broken":
		fmt.Printf("Project mode:      BROKEN — %s exists but failed to parse: %v\n", project.ConfigPath, modeResult.ParseErr)
	default:
		fmt.Printf("Project mode:      %s (.agent/project.yaml present)\n", modeResult.Mode)
	}

	if cfg, err := project.Load(dir); err == nil {
		fmt.Printf("Detected stack:    %s\n", cfg.Stack.Type)
		printWorkflowStatus(cfg.Workflow)
	}

	report, err := skillvalidate.Validate(filepath.Join(hDir, "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
	if err == nil {
		broken := 0
		for _, issue := range report.Errors() {
			if strings.Contains(issue.Message, "cycle") || strings.Contains(issue.Message, "requires unknown") {
				broken++
			}
		}
		fmt.Println("\nSkills:")
		fmt.Printf("  %d discovered\n", report.Discovered)
		fmt.Printf("  %d valid\n", report.Discovered-len(issuedSkillNames(report.Issues)))
		fmt.Printf("  %d warning(s)\n", len(report.Warnings()))
		fmt.Printf("  %d broken dependency issue(s)\n", broken)
		fmt.Println("  (run `eng skills list` or `eng skills validate` for detail)")
	}

	adapters := registeredAdapters(dir)
	fmt.Println("\nTools:")
	for _, a := range adapters {
		// Phase 10: installed (binary on PATH) / wired (adapter registered
		// — always yes for anything reaching this loop) / invokable
		// (Doctor() actually succeeds, e.g. authenticated) are kept
		// distinct — a generic "available" flag conflates "the binary
		// exists" with "the harness can actually invoke it for something
		// useful" (see spec.md's Codex gap analysis).
		installed := a.Available()
		invokable := false
		if installed {
			if _, err := a.Doctor(); err == nil {
				invokable = true
			}
		}
		fmt.Printf("  %-10s installed=%-5v wired=yes invokable=%-5v [%d capabilities]\n",
			a.Name(), installed, invokable, len(a.Capabilities()))
	}

	fmt.Println("\nCapabilities:")
	for _, name := range capabilities.Known {
		status := "unavailable"
		if capabilities.Detect(name) {
			status = "available"
		}
		fmt.Printf("  %-10s %s\n", name, status)
	}
}

// printWorkflowStatus makes eng doctor's resolved workflow behavior
// explicit enough that a literal config can never silently mean the
// opposite (Phase 9 spec.md "Project Config Validation" / P1-4). Each line
// shows the resolved value and whether it came from an explicit setting or
// the default.
func printWorkflowStatus(w project.Workflow) {
	fmt.Println("\nWorkflow:")
	printWorkflowField("triage", w.Triage, w.TriageEnabled())
	printWorkflowField("plan_review", w.PlanReview, w.PlanReviewEnabled())
	printWorkflowField("verifier", w.Verifier, w.VerifierEnabled())
	fmt.Printf("  planning     %s\n", w.PlanningModeOrDefault())
}

func printWorkflowField(name string, explicit *bool, resolved bool) {
	state := "enabled"
	if !resolved {
		state = "disabled"
	}
	source := "default"
	if explicit != nil {
		source = "explicit"
	}
	fmt.Printf("  %-12s %s (%s)\n", name, state, source)
}

// issuedSkillNames counts distinct skills with at least one issue, so
// doctor's "valid" count doesn't double-subtract a skill with two
// warnings.
func issuedSkillNames(issues []skillvalidate.Issue) map[string]bool {
	out := map[string]bool{}
	for _, i := range issues {
		if i.Skill != "(graph)" {
			out[i.Skill] = true
		}
	}
	return out
}
