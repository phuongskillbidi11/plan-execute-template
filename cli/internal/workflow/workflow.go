package workflow

const (
	StateNew           = "NEW"
	StateTriaged       = "TRIAGED"
	StatePlanned       = "PLANNED"
	StateReviewed      = "REVIEWED"
	StateApproved      = "APPROVED"
	StateExecuting     = "EXECUTING"
	StateVerifying     = "VERIFYING"
	StateCompleted     = "COMPLETED"
	StateBlocked       = "BLOCKED"
	StateFailed        = "FAILED"
	StateNeedsReplan   = "NEEDS_REPLAN"
	StateNeedsApproval = "NEEDS_APPROVAL"
	StateNeedsFix      = "NEEDS_FIX"
	StateCancelled     = "CANCELLED"
)

// Terminal reports whether a state has no further automatic transitions —
// eng workflow advance refuses to do anything once a plan reaches one of
// these, requiring an explicit human action instead.
func Terminal(state string) bool {
	switch state {
	case StateCompleted, StateFailed, StateCancelled, StateBlocked:
		return true
	default:
		return false
	}
}

// Facts is everything Decide needs, gathered by the caller from plan.yaml,
// tasks.md, and .agent/project.yaml. Decide itself does no I/O, which makes
// every transition rule independently testable.
type Facts struct {
	State               string
	PlanFilesReady      bool // spec.md/tasks.md/tests.md all exist and are non-empty
	ReviewRequired      bool
	ReviewVerdict       string // "" | PASS | REJECT
	RequiresApproval    bool
	Approved            bool
	DriftDetected       bool
	TasksComplete       bool   // zero remaining "- [ ]" lines in tasks.md
	VerificationVerdict string // "" | PASS | FAIL
	RetryExhausted      bool
}

// Decision is the one transition Decide recommends for the current Facts,
// plus a side effect hint the caller may need to perform (e.g. running
// `eng verify`) before the next Decide call would see updated Facts.
type Decision struct {
	NextState string
	Reason    string
	Action    string // "" | "run_verify"
}

// Decide implements the Phase 3 spec.md "Design decisions / Decision 6"
// transition table exactly. It applies at most one transition per call —
// the caller (eng workflow advance) never chains multiple automatic
// transitions silently past a state a human should see.
func Decide(f Facts) Decision {
	switch f.State {
	case StateTriaged:
		if f.PlanFilesReady {
			return Decision{NextState: StatePlanned, Reason: "spec.md/tasks.md/tests.md are present"}
		}
		return Decision{NextState: StateTriaged, Reason: "waiting on Planner to write spec.md/tasks.md/tests.md"}

	case StatePlanned:
		if !f.ReviewRequired {
			return Decision{NextState: StateReviewed, Reason: "plan review not required for this project/risk level"}
		}
		switch f.ReviewVerdict {
		case "PASS":
			return Decision{NextState: StateReviewed, Reason: "review verdict PASS"}
		case "REJECT":
			return Decision{NextState: StateNeedsReplan, Reason: "review verdict REJECT"}
		default:
			return Decision{NextState: StatePlanned, Reason: "waiting on `eng plan review`"}
		}

	case StateReviewed:
		if !f.RequiresApproval || f.Approved {
			return Decision{NextState: StateApproved, Reason: "approval not required or already granted"}
		}
		return Decision{NextState: StateNeedsApproval, Reason: "run `eng plan approve` before execution can begin"}

	case StateNeedsApproval:
		if f.Approved {
			return Decision{NextState: StateApproved, Reason: "approval granted"}
		}
		return Decision{NextState: StateNeedsApproval, Reason: "still waiting on `eng plan approve`"}

	case StateApproved:
		if f.DriftDetected {
			return Decision{NextState: StateNeedsReplan, Reason: "PLAN_DRIFT_DETECTED before execution started"}
		}
		return Decision{NextState: StateExecuting, Reason: "no drift detected — Executor may begin"}

	case StateExecuting, StateNeedsFix:
		if f.TasksComplete {
			return Decision{NextState: StateVerifying, Reason: "all tasks.md items checked off", Action: "run_verify"}
		}
		return Decision{NextState: f.State, Reason: "tasks.md still has unchecked items"}

	case StateVerifying:
		switch f.VerificationVerdict {
		case "PASS":
			return Decision{NextState: StateCompleted, Reason: "eng verify reported PASS"}
		case "FAIL":
			if f.RetryExhausted {
				return Decision{NextState: StateFailed, Reason: "eng verify FAILed and the retry budget is exhausted"}
			}
			return Decision{NextState: StateNeedsFix, Reason: "eng verify FAILed — retry budget remains"}
		default:
			return Decision{NextState: StateVerifying, Reason: "waiting on `eng verify`"}
		}

	case StateNeedsReplan:
		return Decision{NextState: StatePlanned, Reason: "replanning acknowledged — re-entering review"}

	default:
		return Decision{NextState: f.State, Reason: "no automatic transition from this state"}
	}
}
