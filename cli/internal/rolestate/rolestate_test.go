package rolestate

import (
	"testing"

	"eng/internal/workflow"
)

func TestLoadMissingFileReturnsUsableZeroValue(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if s.CurrentRole != "" {
		t.Fatalf("expected empty CurrentRole for a missing file, got %+v", s)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := &RoleState{
		CurrentRole:       "executor",
		ActivatedAt:       "2026-08-27T10:00:00Z",
		ActivatedForState: workflow.StateApproved,
		PromptGeneratedAt: "2026-08-27T10:00:00Z",
		ContextManifest:   "context-manifest-executor.yaml",
	}
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if *got != *want {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestReset(t *testing.T) {
	dir := t.TempDir()
	Save(dir, &RoleState{CurrentRole: "executor"})
	if err := Reset(dir); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentRole != "" {
		t.Fatalf("expected Reset to clear CurrentRole, got %+v", got)
	}
}

func TestAllowedForStatePositiveCases(t *testing.T) {
	cases := []struct {
		role, state string
		isQuickFix  bool
	}{
		{"planner", workflow.StateTriaged, false},
		{"planner", workflow.StateNeedsSpecApproval, false},
		{"planner", workflow.StateSpecApproved, false},
		{"planner", workflow.StateNeedsReplan, false},
		{"plan-reviewer", workflow.StatePlanned, false},
		{"plan-reviewer", workflow.StateReviewed, false},
		{"executor", workflow.StateApproved, false},
		{"executor", workflow.StateExecuting, false},
		{"executor", workflow.StateNeedsFix, false},
		{"executor", workflow.StateTriaged, true}, // quick-fix fast path
		{"verifier", workflow.StateVerifying, false},
		{"verifier", workflow.StateCompleted, false},
	}
	for _, c := range cases {
		ok, reason := AllowedForState(c.role, c.state, c.isQuickFix)
		if !ok {
			t.Errorf("AllowedForState(%q, %q, quickFix=%v) = false (%q), want true", c.role, c.state, c.isQuickFix, reason)
		}
	}
}

func TestAllowedForStateNegativeCases(t *testing.T) {
	cases := []struct {
		role, state string
		isQuickFix  bool
	}{
		{"executor", workflow.StateTriaged, false},   // not quick-fix — executor not yet relevant
		{"planner", workflow.StateApproved, false},   // planner has no business in APPROVED
		{"plan-reviewer", workflow.StateExecuting, false},
		{"verifier", workflow.StateApproved, false},
		{"executor", workflow.StateVerifying, false},
		{"planner", workflow.StateCompleted, false},
	}
	for _, c := range cases {
		ok, reason := AllowedForState(c.role, c.state, c.isQuickFix)
		if ok {
			t.Errorf("AllowedForState(%q, %q, quickFix=%v) = true, want false", c.role, c.state, c.isQuickFix)
		}
		if reason == "" {
			t.Errorf("AllowedForState(%q, %q) denied with no reason", c.role, c.state)
		}
	}
}

func TestNextRole(t *testing.T) {
	cases := []struct {
		state      string
		isQuickFix bool
		want       string
	}{
		{workflow.StateTriaged, false, "planner"},
		{workflow.StateTriaged, true, "executor"},
		{workflow.StateNeedsSpecApproval, false, "planner"},
		{workflow.StatePlanned, false, "plan-reviewer"},
		{workflow.StateReviewed, false, "plan-reviewer"},
		{workflow.StateNeedsApproval, false, "executor"},
		{workflow.StateApproved, false, "executor"},
		{workflow.StateExecuting, false, "executor"},
		{workflow.StateVerifying, false, "verifier"},
		{workflow.StateCompleted, false, "verifier"},
		{workflow.StateFailed, false, ""},
		{workflow.StateBlocked, false, ""},
		{workflow.StateCancelled, false, ""},
	}
	for _, c := range cases {
		if got := NextRole(c.state, c.isQuickFix); got != c.want {
			t.Errorf("NextRole(%q, quickFix=%v) = %q, want %q", c.state, c.isQuickFix, got, c.want)
		}
	}
}
