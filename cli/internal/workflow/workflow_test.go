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
	ok := Decide(Facts{State: StateApproved, DriftDetected: false})
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
