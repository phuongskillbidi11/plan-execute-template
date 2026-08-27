package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"eng/internal/planmeta"
	"eng/internal/project"
)

func writePlan(t *testing.T, projectRoot, name, state string) {
	t.Helper()
	planDir := filepath.Join(projectRoot, ".plans", name)
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := &planmeta.Meta{Plan: name, RiskLevel: "feature", State: state}
	if err := planmeta.Save(planDir, meta); err != nil {
		t.Fatal(err)
	}
}

func TestScanPlansNoPlansDir(t *testing.T) {
	dir := t.TempDir()
	if got := scanPlans(dir); got != nil {
		t.Fatalf("expected nil/empty for no .plans dir, got %v", got)
	}
}

func TestScanPlansAllTerminal(t *testing.T) {
	dir := t.TempDir()
	writePlan(t, dir, "2026-08-27-done-one", "COMPLETED")
	writePlan(t, dir, "2026-08-27-done-two", "CANCELLED")

	if got := scanPlans(dir); len(got) != 0 {
		t.Fatalf("expected zero unfinished plans, got %v", got)
	}
}

func TestScanPlansMixedTerminalAndUnfinished(t *testing.T) {
	dir := t.TempDir()
	writePlan(t, dir, "2026-08-27-done", "COMPLETED")
	writePlan(t, dir, "2026-08-27-in-progress", "EXECUTING")
	writePlan(t, dir, "2026-08-27-triaged", "TRIAGED")

	got := scanPlans(dir)
	if len(got) != 2 {
		t.Fatalf("expected 2 unfinished plans, got %d: %v", len(got), got)
	}
	if got[0].Dir != filepath.Join(".plans", "2026-08-27-in-progress") || got[0].State != "EXECUTING" {
		t.Fatalf("unexpected first entry: %+v", got[0])
	}
	if got[1].Dir != filepath.Join(".plans", "2026-08-27-triaged") || got[1].State != "TRIAGED" {
		t.Fatalf("unexpected second entry: %+v", got[1])
	}
}

func TestScanPlansSkipsDirWithoutPlanYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".plans", "not-a-plan"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := scanPlans(dir); len(got) != 0 {
		t.Fatalf("expected non-plan directory to be skipped silently, got %v", got)
	}
}

func TestGatherBootstrapStatusUninitializedProject(t *testing.T) {
	dir := t.TempDir()
	s := gatherBootstrapStatus(dir)
	if s.ProjectMode != "none" {
		t.Fatalf("expected ProjectMode 'none', got %q", s.ProjectMode)
	}
	if s.PlanningMode != "" {
		t.Fatalf("expected empty PlanningMode for uninitialized project, got %q", s.PlanningMode)
	}
	if len(s.UnfinishedPlans) != 0 {
		t.Fatalf("expected no unfinished plans, got %v", s.UnfinishedPlans)
	}
}

func TestGatherBootstrapStatusInitializedProject(t *testing.T) {
	dir := t.TempDir()
	verifierOff := false
	cfg := &project.Config{
		ProjectName: "x",
		Mode:        "modern",
		Stack:       project.Stack{Type: "go"},
		Workflow: project.Workflow{
			PlanningMode: "spec_first",
			Verifier:     &verifierOff,
		},
	}
	if err := project.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}

	s := gatherBootstrapStatus(dir)
	if s.ProjectMode != "modern" {
		t.Fatalf("expected ProjectMode 'modern', got %q", s.ProjectMode)
	}
	if s.PlanningMode != "spec_first" {
		t.Fatalf("expected PlanningMode 'spec_first', got %q", s.PlanningMode)
	}
	if !s.TriageEnabled || !s.PlanReviewEnabled {
		t.Fatalf("expected triage/plan_review to default enabled, got %+v", s)
	}
	if s.VerifierEnabled {
		t.Fatalf("expected verifier disabled per explicit config, got %+v", s)
	}
}

func TestRenderBootstrapPromptDeterministic(t *testing.T) {
	s := bootstrapStatus{
		HarnessInstalled: true,
		HarnessHome:      "/home/x/.engineering-harness",
		HarnessVersion:   "0.10.1-beta",
		ProjectRoot:      "/home/x/project",
		ProjectMode:      "modern",
		PlanningMode:     "spec_first",
		TriageEnabled:    true, PlanReviewEnabled: true, VerifierEnabled: true,
		CodexInstalled: true, CodexWired: true, CodexInvokable: true,
	}
	a := renderBootstrapPrompt(s)
	b := renderBootstrapPrompt(s)
	if a != b {
		t.Fatalf("expected deterministic output, got two different renders")
	}
	for _, want := range []string{
		"/home/x/.engineering-harness", "0.10.1-beta", "modern",
		"installed=true wired=true invokable=true",
	} {
		if !strings.Contains(a, want) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", want, a)
		}
	}
}

func TestRenderBootstrapPromptInstructions(t *testing.T) {
	a := renderBootstrapPrompt(bootstrapStatus{})
	for _, want := range []string{
		"verify current state through `eng`",
		"Do not conclude the harness is absent",
		"Do not auto-resume a COMPLETED plan",
	} {
		if !strings.Contains(a, want) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", want, a)
		}
	}
}

func TestRenderBootstrapPromptPlanCountsZero(t *testing.T) {
	a := renderBootstrapPrompt(bootstrapStatus{})
	if !strings.Contains(a, "no unfinished plans") {
		t.Fatalf("expected 'no unfinished plans', got:\n%s", a)
	}
}

func TestRenderBootstrapPromptPlanCountsOne(t *testing.T) {
	s := bootstrapStatus{UnfinishedPlans: []planSummary{{Dir: ".plans/2026-08-27-x", State: "EXECUTING"}}}
	a := renderBootstrapPrompt(s)
	if !strings.Contains(a, "1 unfinished: .plans/2026-08-27-x (EXECUTING)") {
		t.Fatalf("expected single-plan phrasing, got:\n%s", a)
	}
}

func TestRenderBootstrapPromptPlanCountsMany(t *testing.T) {
	var plans []planSummary
	for i := 0; i < 6; i++ {
		plans = append(plans, planSummary{Dir: ".plans/plan-" + string(rune('a'+i)), State: "TRIAGED"})
	}
	s := bootstrapStatus{UnfinishedPlans: plans}
	a := renderBootstrapPrompt(s)
	if !strings.Contains(a, "6 unfinished — ask the human which to resume, do not guess") {
		t.Fatalf("expected many-plan phrasing, got:\n%s", a)
	}
	if !strings.Contains(a, "...and 1 more") {
		t.Fatalf("expected capped listing with '...and 1 more', got:\n%s", a)
	}
}

func TestRenderBootstrapPromptBounded(t *testing.T) {
	var plans []planSummary
	for i := 0; i < 5; i++ {
		plans = append(plans, planSummary{Dir: ".plans/2026-08-27-some-longer-plan-name-" + string(rune('a'+i)), State: "EXECUTING"})
	}
	s := bootstrapStatus{
		HarnessInstalled: true,
		HarnessHome:      `C:\Users\Admin\.engineering-harness`,
		HarnessVersion:   "0.10.1-beta",
		ProjectRoot:      `C:\Users\Admin\source\repos\some-project`,
		ProjectMode:      "modern",
		PlanningMode:     "spec_first",
		TriageEnabled:    true, PlanReviewEnabled: true, VerifierEnabled: true,
		CodexInstalled: true, CodexWired: true, CodexInvokable: true,
		UnfinishedPlans: plans,
	}
	a := renderBootstrapPrompt(s)
	if len(a) > 1600 {
		t.Fatalf("expected bootstrap prompt under 1600 chars, got %d:\n%s", len(a), a)
	}
}

func TestStartCommandIncludesBootstrapPrompt(t *testing.T) {
	args := startClaudeArgs("hello harness")
	want := []string{"--append-system-prompt", "hello harness"}
	if len(args) != len(want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, args)
		}
	}
}
