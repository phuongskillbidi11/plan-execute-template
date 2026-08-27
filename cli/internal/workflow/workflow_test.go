package workflow

import "testing"

func TestTriagedWaitsForPlanFiles(t *testing.T) {
	d := Decide(Facts{State: StateTriaged, PlanFilesReady: false})
	if d.NextState != StateTriaged {
		t.Fatalf("expected to stay TRIAGED, got %+v", d)
	}
	d = Decide(Facts{State: StateTriaged, PlanFilesReady: true})
	if d.NextState != StatePlanned {
		t.Fatalf("expected PLANNED, got %+v", d)
	}
}

func TestPlannedReviewPassAndReject(t *testing.T) {
	pass := Decide(Facts{State: StatePlanned, ReviewRequired: true, ReviewVerdict: "PASS"})
	if pass.NextState != StateReviewed {
		t.Fatalf("expected REVIEWED, got %+v", pass)
	}
	reject := Decide(Facts{State: StatePlanned, ReviewRequired: true, ReviewVerdict: "REJECT"})
	if reject.NextState != StateNeedsReplan {
		t.Fatalf("expected NEEDS_REPLAN, got %+v", reject)
	}
	skip := Decide(Facts{State: StatePlanned, ReviewRequired: false})
	if skip.NextState != StateReviewed {
		t.Fatalf("expected REVIEWED when review not required, got %+v", skip)
	}
}

func TestApprovalGate(t *testing.T) {
	blocked := Decide(Facts{State: StateReviewed, RequiresApproval: true, Approved: false})
	if blocked.NextState != StateNeedsApproval {
		t.Fatalf("expected NEEDS_APPROVAL, got %+v", blocked)
	}
	approved := Decide(Facts{State: StateReviewed, RequiresApproval: true, Approved: true})
	if approved.NextState != StateApproved {
		t.Fatalf("expected APPROVED, got %+v", approved)
	}
	notNeeded := Decide(Facts{State: StateReviewed, RequiresApproval: false})
	if notNeeded.NextState != StateApproved {
		t.Fatalf("expected APPROVED when not required, got %+v", notNeeded)
	}
}

func TestDriftBeforeExecutionForcesReplan(t *testing.T) {
	d := Decide(Facts{State: StateApproved, DriftDetected: true})
	if d.NextState != StateNeedsReplan {
		t.Fatalf("expected NEEDS_REPLAN, got %+v", d)
	}
	// Phase 10: APPROVED -> EXECUTING now also requires executor activation.
	ok := Decide(Facts{State: StateApproved, DriftDetected: false, ExecutorActivated: true})
	if ok.NextState != StateExecuting {
		t.Fatalf("expected EXECUTING, got %+v", ok)
	}
}

func TestExecutingToVerifyingTriggersVerify(t *testing.T) {
	d := Decide(Facts{State: StateExecuting, TasksComplete: true})
	if d.NextState != StateVerifying || d.Action != "run_verify" {
		t.Fatalf("expected VERIFYING with run_verify action, got %+v", d)
	}
	notDone := Decide(Facts{State: StateExecuting, TasksComplete: false})
	if notDone.NextState != StateExecuting {
		t.Fatalf("expected to stay EXECUTING, got %+v", notDone)
	}
}

func TestVerifyingRoutesOnVerdict(t *testing.T) {
	pass := Decide(Facts{State: StateVerifying, VerificationVerdict: "PASS"})
	if pass.NextState != StateCompleted {
		t.Fatalf("expected COMPLETED, got %+v", pass)
	}
	failRetry := Decide(Facts{State: StateVerifying, VerificationVerdict: "FAIL", RetryExhausted: false})
	if failRetry.NextState != StateNeedsFix {
		t.Fatalf("expected NEEDS_FIX, got %+v", failRetry)
	}
	failExhausted := Decide(Facts{State: StateVerifying, VerificationVerdict: "FAIL", RetryExhausted: true})
	if failExhausted.NextState != StateFailed {
		t.Fatalf("expected FAILED, got %+v", failExhausted)
	}
}

func TestTerminalStates(t *testing.T) {
	for _, s := range []string{StateCompleted, StateFailed, StateCancelled, StateBlocked} {
		if !Terminal(s) {
			t.Fatalf("%s should be terminal", s)
		}
	}
	for _, s := range []string{StateNew, StateTriaged, StateExecuting} {
		if Terminal(s) {
			t.Fatalf("%s should not be terminal", s)
		}
	}
}

func TestQuickFixSkipsStraightToExecuting(t *testing.T) {
	waiting := Decide(Facts{State: StateTriaged, IsQuickFix: true, PlanFilesReady: false})
	if waiting.NextState != StateTriaged {
		t.Fatalf("expected to stay TRIAGED until minimal plan exists, got %+v", waiting)
	}
	// Phase 10: Quick Fix's TRIAGED -> EXECUTING now also requires executor activation.
	ready := Decide(Facts{State: StateTriaged, IsQuickFix: true, PlanFilesReady: true, ExecutorActivated: true})
	if ready.NextState != StateExecuting {
		t.Fatalf("expected EXECUTING directly, got %+v", ready)
	}
}

func TestSpecFirstRequiresSpecApprovalBeforeTasks(t *testing.T) {
	waitingSpec := Decide(Facts{State: StateTriaged, PlanningMode: "spec_first", SpecReady: false})
	if waitingSpec.NextState != StateTriaged {
		t.Fatalf("expected to stay TRIAGED, got %+v", waitingSpec)
	}
	needsApproval := Decide(Facts{State: StateTriaged, PlanningMode: "spec_first", SpecReady: true, RequireSpecApproval: true})
	if needsApproval.NextState != StateNeedsSpecApproval {
		t.Fatalf("expected NEEDS_SPEC_APPROVAL, got %+v", needsApproval)
	}
	skipApproval := Decide(Facts{State: StateTriaged, PlanningMode: "spec_first", SpecReady: true, RequireSpecApproval: false})
	if skipApproval.NextState != StateSpecApproved {
		t.Fatalf("expected SPEC_APPROVED when approval not required, got %+v", skipApproval)
	}
}

func TestNeedsSpecApprovalGate(t *testing.T) {
	blocked := Decide(Facts{State: StateNeedsSpecApproval, SpecApproved: false})
	if blocked.NextState != StateNeedsSpecApproval {
		t.Fatalf("expected to stay blocked, got %+v", blocked)
	}
	approved := Decide(Facts{State: StateNeedsSpecApproval, SpecApproved: true})
	if approved.NextState != StateSpecApproved {
		t.Fatalf("expected SPEC_APPROVED, got %+v", approved)
	}
}

func TestSpecApprovedWaitsForTasksAndTests(t *testing.T) {
	waiting := Decide(Facts{State: StateSpecApproved, TasksAndTestsReady: false})
	if waiting.NextState != StateSpecApproved {
		t.Fatalf("expected to stay SPEC_APPROVED, got %+v", waiting)
	}
	ready := Decide(Facts{State: StateSpecApproved, TasksAndTestsReady: true})
	if ready.NextState != StatePlanned {
		t.Fatalf("expected PLANNED, got %+v", ready)
	}
}

// --- Phase 10: executor-activation gate + temporal invariant + role verification ---

func TestApprovedStaysApprovedUntilExecutorActivated(t *testing.T) {
	d := Decide(Facts{State: StateApproved, DriftDetected: false, ExecutorActivated: false})
	if d.NextState != StateApproved {
		t.Fatalf("expected to stay APPROVED without executor activation, got %+v", d)
	}
	if d.Action == "invariant_violation" {
		t.Fatalf("expected a plain activation-wait reason, not an invariant violation, got %+v", d)
	}
}

// TestApprovedInvariantViolationBlocksRetroactiveCompletion is the direct
// regression test for the reproduced CredoID-shaped bypass
// (benchmarks/fixtures/investigation-bypass/): tasks.md already showing
// complete at the moment APPROVED -> EXECUTING would fire must refuse the
// transition, regardless of executor-activation state.
func TestApprovedInvariantViolationBlocksRetroactiveCompletion(t *testing.T) {
	cases := []Facts{
		{State: StateApproved, TasksComplete: true, ExecutorActivated: false},
		{State: StateApproved, TasksComplete: true, ExecutorActivated: true},
	}
	for _, f := range cases {
		d := Decide(f)
		if d.NextState != StateApproved {
			t.Fatalf("expected to stay APPROVED, got %+v (facts: %+v)", d, f)
		}
		if d.Action != "invariant_violation" {
			t.Fatalf("expected Action=invariant_violation, got %+v (facts: %+v)", d, f)
		}
	}
}

func TestApprovedTransitionsOnceActivatedAndNotPreComplete(t *testing.T) {
	d := Decide(Facts{State: StateApproved, DriftDetected: false, ExecutorActivated: true, TasksComplete: false})
	if d.NextState != StateExecuting {
		t.Fatalf("expected EXECUTING, got %+v", d)
	}
}

func TestQuickFixStaysTriagedUntilExecutorActivated(t *testing.T) {
	d := Decide(Facts{State: StateTriaged, IsQuickFix: true, PlanFilesReady: true, ExecutorActivated: false})
	if d.NextState != StateTriaged {
		t.Fatalf("expected to stay TRIAGED without executor activation, got %+v", d)
	}
}

func TestQuickFixInvariantViolationBlocksRetroactiveCompletion(t *testing.T) {
	d := Decide(Facts{State: StateTriaged, IsQuickFix: true, PlanFilesReady: true, TasksComplete: true, ExecutorActivated: true})
	if d.NextState != StateTriaged || d.Action != "invariant_violation" {
		t.Fatalf("expected to stay TRIAGED with an invariant violation, got %+v", d)
	}
}

func TestVerifyingWaitsForRoleVerificationVerdict(t *testing.T) {
	d := Decide(Facts{State: StateVerifying, VerificationVerdict: "PASS", RoleVerificationRequired: true, RoleVerificationVerdict: ""})
	if d.NextState != StateVerifying {
		t.Fatalf("expected to stay VERIFYING pending the Verifier role verdict, got %+v", d)
	}
}

func TestVerifyingRoleVerificationFailGoesToNeedsFix(t *testing.T) {
	d := Decide(Facts{State: StateVerifying, VerificationVerdict: "PASS", RoleVerificationRequired: true, RoleVerificationVerdict: "FAIL", RetryExhausted: false})
	if d.NextState != StateNeedsFix {
		t.Fatalf("expected NEEDS_FIX, got %+v", d)
	}
	failed := Decide(Facts{State: StateVerifying, VerificationVerdict: "PASS", RoleVerificationRequired: true, RoleVerificationVerdict: "FAIL", RetryExhausted: true})
	if failed.NextState != StateFailed {
		t.Fatalf("expected FAILED once retry budget is exhausted, got %+v", failed)
	}
}

func TestVerifyingCompletesWhenRoleVerificationPasses(t *testing.T) {
	d := Decide(Facts{State: StateVerifying, VerificationVerdict: "PASS", RoleVerificationRequired: true, RoleVerificationVerdict: "PASS"})
	if d.NextState != StateCompleted {
		t.Fatalf("expected COMPLETED, got %+v", d)
	}
}

// TestVerifyingUnchangedWhenRoleVerificationNotRequired is the direct
// backward-compatibility regression guard: a project with
// workflow.verifier: false (RoleVerificationRequired false) must reach
// COMPLETED on mechanical PASS alone, byte-identical to Phase 9.
func TestVerifyingUnchangedWhenRoleVerificationNotRequired(t *testing.T) {
	d := Decide(Facts{State: StateVerifying, VerificationVerdict: "PASS", RoleVerificationRequired: false})
	if d.NextState != StateCompleted {
		t.Fatalf("expected COMPLETED (role verification not required), got %+v", d)
	}
}

func TestAutoPlanPathUnaffectedByNewFields(t *testing.T) {
	// Zero-value PlanningMode/IsQuickFix must reproduce Phase 3's exact behavior.
	d := Decide(Facts{State: StateTriaged, PlanFilesReady: true})
	if d.NextState != StatePlanned {
		t.Fatalf("expected PLANNED (auto_plan, unchanged), got %+v", d)
	}
}
