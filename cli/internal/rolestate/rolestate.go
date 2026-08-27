// Package rolestate is the Phase 10 role runtime model — a minimal,
// explicit answer to "what role is active right now, was its prompt
// actually composed, and is the current workflow state compatible with
// that role." See .plans/2026-08-27-v2-harness-phase10-role-enforcement/
// spec.md for the full design and DECISION_LOG.md for why this is a
// separate file from plan.yaml.
package rolestate

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"eng/internal/workflow"
)

const FileName = "role-state.yaml"

// RoleState is one plan's current role-activation snapshot — not a
// history (events.jsonl already carries that, via role_activated/
// role_activation_denied events).
type RoleState struct {
	CurrentRole       string `yaml:"current_role"`
	ActivatedAt       string `yaml:"activated_at,omitempty"`
	ActivatedForState string `yaml:"activated_for_state,omitempty"`
	PromptGeneratedAt string `yaml:"prompt_generated_at,omitempty"`
	ContextManifest   string `yaml:"context_manifest,omitempty"`
}

// Load returns the plan's role state, or a usable zero value (CurrentRole
// "") if role-state.yaml doesn't exist yet — a pre-Phase-10 plan, or a
// Phase-10 plan whose role has never been activated, is not an error
// case: it behaves exactly like "no role is active," the correct, safe
// default (see spec.md's Backward compatibility strategy).
func Load(planDir string) (*RoleState, error) {
	data, err := os.ReadFile(filepath.Join(planDir, FileName))
	if err != nil {
		if os.IsNotExist(err) {
			return &RoleState{}, nil
		}
		return nil, err
	}
	var s RoleState
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func Save(planDir string, s *RoleState) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(planDir, FileName), data, 0o644)
}

// Reset clears any activation — called whenever a plan re-enters
// NEEDS_REPLAN (drift or review rejection), so a stale activation from
// before a replan cycle can never satisfy a later gate.
func Reset(planDir string) error {
	return Save(planDir, &RoleState{})
}

// roleForState is the one authoritative state-to-role table (spec.md's
// "State-to-role mapping" section) — AllowedForState and NextRole are
// both thin views over it, so there is exactly one place this mapping is
// defined.
var roleForState = map[string][]string{
	workflow.StateTriaged:           {"planner"}, // quick-fix's own executor case is handled separately below
	workflow.StateNeedsSpecApproval: {"planner"},
	workflow.StateSpecApproved:      {"planner"},
	workflow.StateNeedsReplan:       {"planner"},
	workflow.StatePlanned:           {"plan-reviewer"},
	workflow.StateReviewed:          {"plan-reviewer"},
	workflow.StateApproved:          {"executor"},
	workflow.StateExecuting:         {"executor"},
	workflow.StateNeedsFix:          {"executor"},
	workflow.StateVerifying:         {"verifier"},
	workflow.StateCompleted:         {"verifier"},
}

// AllowedForState reports whether role may be activated (or may act)
// while the plan is in state — pure and testable, no I/O. isQuickFix
// covers the one state that maps to two different roles depending on
// risk level: TRIAGED normally means "planner is next," but a quick-fix
// plan's TRIAGED->EXECUTING fast path means "executor is next" instead.
func AllowedForState(role, state string, isQuickFix bool) (bool, string) {
	if isQuickFix && state == workflow.StateTriaged && role == "executor" {
		return true, ""
	}
	roles, ok := roleForState[state]
	if !ok {
		return false, "state " + state + " has no compatible role"
	}
	for _, r := range roles {
		if r == role {
			return true, ""
		}
	}
	return false, "role " + role + " is not compatible with state " + state
}

// NextRole is the single deterministic "who should act next" answer, for
// display (eng doctor/eng workflow status) — not a gate. NEEDS_APPROVAL
// is a display-only special case: it isn't in roleForState (a role may
// not be *activated* while waiting on a human approval — that's what
// AllowedForState enforces), but "executor" is still the informative
// answer to "who's up next once approval lands." Returns "" for a
// terminal or otherwise genuinely role-less state.
func NextRole(state string, isQuickFix bool) string {
	if isQuickFix && state == workflow.StateTriaged {
		return "executor"
	}
	if state == workflow.StateNeedsApproval {
		return "executor"
	}
	roles, ok := roleForState[state]
	if !ok || len(roles) == 0 {
		return ""
	}
	return roles[0]
}
