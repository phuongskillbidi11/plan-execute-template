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

	StateNeedsSpecApproval = "NEEDS_SPEC_APPROVAL"
	StateSpecApproved      = "SPEC_APPROVED"
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

	IsQuickFix          bool   // risk_level == "quick-fix"
	PlanningMode        string // "auto_plan" (default/legacy) | "spec_first"
	SpecReady           bool   // spec.md exists, non-empty, placeholder-free
	SpecApproved        bool   // plan.yaml's spec_approved_at is set
	RequireSpecApproval bool
	TasksAndTestsReady  bool // tasks.md AND tests.md exist, non-empty, placeholder-free

	// ExecutorActivated is Phase 10's role-runtime fact: has `eng adapter
	// prompt executor <plan-dir>` actually succeeded for this plan since
	// its last replan cycle (rolestate.RoleState.CurrentRole == "executor")?
	// Gates every transition into EXECUTING — see spec.md's "Two new hard
	// transition gates."
	ExecutorActivated bool

	// RoleVerificationRequired/RoleVerificationVerdict are Phase 10's
	// mechanical-vs-role-verification facts. RoleVerificationRequired is
	// project.Workflow.VerifierEnabled() — Phase 9 built the accessor,
	// Phase 10 is its first real consumer. RoleVerificationVerdict is
	// plan.yaml's role_verification.verdict ("" | PASS | FAIL).
	RoleVerificationRequired bool
	RoleVerificationVerdict  string
}

// Decision is the one transition Decide recommends for the current Facts,
// plus a side effect hint the caller may need to perform (e.g. running
// `eng verify`) before the next Decide call would see updated Facts.
type Decision struct {
	NextState string
	Reason    string
	Action    string // "" | "run_verify" | "invariant_violation"
}

// invariantViolation is the Phase 10 temporal-invariant reason: tasks.md
// already shows every Completion checklist item checked at the exact
// moment a transition into EXECUTING would otherwise fire. Under every
// normal flow this is false (the template's checklist starts unchecked);
// it being true here means something checked every box before execution
// legitimately began — the reproduced CredoID-shaped bypass. Decide
// refuses the transition rather than let the state machine retroactively
// legitimize already-completed work. See spec.md's temporal invariant
// section and DECISION_LOG.md Decision 3 (no override flag, by design).
const invariantViolationReason = "tasks.md already shows complete before execution began — " +
	"this looks like retroactive completion; uncheck the Completion checklist and let the " +
	"Executor genuinely complete it under EXECUTING"

// Decide implements the Phase 3 spec.md "Design decisions / Decision 6"
// transition table exactly. It applies at most one transition per call —
// the caller (eng workflow advance) never chains multiple automatic
// transitions silently past a state a human should see.
func Decide(f Facts) Decision {
	switch f.State {
	case StateTriaged:
		if f.IsQuickFix {
			if !f.PlanFilesReady {
				return Decision{NextState: StateTriaged, Reason: "waiting on the minimal quick-fix plan (spec.md + tasks.md + tests.md)"}
			}
			if f.TasksComplete {
				return Decision{NextState: StateTriaged, Reason: invariantViolationReason, Action: "invariant_violation"}
			}
			if !f.ExecutorActivated {
				return Decision{NextState: StateTriaged, Reason: "waiting on executor role activation (eng adapter prompt executor <plan-dir>)"}
			}
			return Decision{NextState: StateExecuting, Reason: "quick-fix: minimal plan present, skipping review/approval"}
		}
		if f.PlanningMode == "spec_first" {
			if !f.SpecReady {
				return Decision{NextState: StateTriaged, Reason: "waiting on Planner to write spec.md"}
			}
			if f.RequireSpecApproval {
				return Decision{NextState: StateNeedsSpecApproval, Reason: "spec.md written — waiting on `eng plan approve-spec`"}
			}
			return Decision{NextState: StateSpecApproved, Reason: "spec.md written, spec approval not required"}
		}
		// auto_plan (default when PlanningMode is unset) — Phase 3 behavior, unchanged.
		if f.PlanFilesReady {
			return Decision{NextState: StatePlanned, Reason: "spec.md/tasks.md/tests.md are present"}
		}
		return Decision{NextState: StateTriaged, Reason: "waiting on Planner to write spec.md/tasks.md/tests.md"}

	case StateNeedsSpecApproval:
		if f.SpecApproved {
			return Decision{NextState: StateSpecApproved, Reason: "spec approved"}
		}
		return Decision{NextState: StateNeedsSpecApproval, Reason: "still waiting on `eng plan approve-spec`"}

	case StateSpecApproved:
		if f.TasksAndTestsReady {
			return Decision{NextState: StatePlanned, Reason: "tasks.md/tests.md are present"}
		}
		return Decision{NextState: StateSpecApproved, Reason: "waiting on Planner to write tasks.md/tests.md"}

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
		if f.TasksComplete {
			return Decision{NextState: StateApproved, Reason: invariantViolationReason, Action: "invariant_violation"}
		}
		if !f.ExecutorActivated {
			return Decision{NextState: StateApproved, Reason: "waiting on executor role activation (eng adapter prompt executor <plan-dir>)"}
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
			if f.RoleVerificationRequired {
				switch f.RoleVerificationVerdict {
				case "":
					return Decision{NextState: StateVerifying, Reason: "mechanical verify PASSed — waiting on Verifier role verdict (eng plan verify-review)"}
				case "FAIL":
					if f.RetryExhausted {
						return Decision{NextState: StateFailed, Reason: "Verifier role rejected the result and the retry budget is exhausted"}
					}
					return Decision{NextState: StateNeedsFix, Reason: "Verifier role rejected the result — retry budget remains"}
				}
			}
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
