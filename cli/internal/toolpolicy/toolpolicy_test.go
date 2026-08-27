package toolpolicy

import (
	"testing"

	"eng/internal/toolcap"
)

func TestDecideHardDenyWinsOverProjectAllow(t *testing.T) {
	p := Policy{Allow: []string{"git.force_push"}}
	d := Decide("git.force_push", toolcap.RiskDestructive, "git", "executor", p, true)
	if d.Verdict != Denied {
		t.Fatalf("expected hard deny to win even when explicitly allowed, got %+v", d)
	}
}

func TestDecideProjectDeny(t *testing.T) {
	p := Policy{Deny: []string{"git.push"}}
	d := Decide("git.push", toolcap.RiskWrite, "git", "executor", p, true)
	if d.Verdict != Denied {
		t.Fatalf("expected DENIED, got %+v", d)
	}
}

func TestDecideRoleToolboxDenied(t *testing.T) {
	d := Decide("docker.inspect", toolcap.RiskRead, "docker", "planner", Policy{}, false)
	if d.Verdict != Denied {
		t.Fatalf("expected planner's toolbox to exclude docker, got %+v", d)
	}
}

func TestDecideRoleRiskCeilingDenied(t *testing.T) {
	d := Decide("git.push", toolcap.RiskWrite, "git", "planner", Policy{}, true)
	if d.Verdict != Denied {
		t.Fatalf("expected planner's READ ceiling to deny WRITE, got %+v", d)
	}
}

// TestDecideRoleRiskCeilingDeniedForPlanReviewer is the direct regression
// test for Phase 10 spec.md's acceptance criterion "A Plan Reviewer
// cannot perform implementation mutation" — plan-reviewer's RoleMaxRisk
// is READ, same as planner, unaffected/unweakened by Phase 10 (Phase 10
// adds a role-vs-state check ahead of this one; it doesn't loosen this
// existing Phase 7 guarantee).
func TestDecideRoleRiskCeilingDeniedForPlanReviewer(t *testing.T) {
	d := Decide("git.push", toolcap.RiskWrite, "git", "plan-reviewer", Policy{}, true)
	if d.Verdict != Denied {
		t.Fatalf("expected plan-reviewer's READ ceiling to deny WRITE, got %+v", d)
	}
}

// TestDecideRoleRiskCeilingDeniedForVerifier is the direct regression test
// for "Verifier cannot mutate implementation" — verifier's RoleMaxRisk is
// READ.
func TestDecideRoleRiskCeilingDeniedForVerifier(t *testing.T) {
	d := Decide("git.push", toolcap.RiskWrite, "git", "verifier", Policy{}, true)
	if d.Verdict != Denied {
		t.Fatalf("expected verifier's READ ceiling to deny WRITE, got %+v", d)
	}
}

func TestDecideRequireApprovalNotYetApproved(t *testing.T) {
	p := Policy{RequireApproval: []string{"github.issue.comment"}}
	d := Decide("github.issue.comment", toolcap.RiskWrite, "github", "executor", p, false)
	if d.Verdict != NeedsApproval {
		t.Fatalf("expected NEEDS_APPROVAL, got %+v", d)
	}
}

func TestDecideRequireApprovalApproved(t *testing.T) {
	p := Policy{RequireApproval: []string{"github.issue.comment"}}
	d := Decide("github.issue.comment", toolcap.RiskWrite, "github", "executor", p, true)
	if d.Verdict != Allowed {
		t.Fatalf("expected ALLOWED once approved, got %+v", d)
	}
}

func TestDecideProjectAllowList(t *testing.T) {
	p := Policy{Allow: []string{"git.push"}}
	d := Decide("git.push", toolcap.RiskWrite, "git", "executor", p, false)
	if d.Verdict != Allowed {
		t.Fatalf("expected ALLOWED via tools.allow even without plan approval, got %+v", d)
	}
}

func TestDecideDefaultReadOpen(t *testing.T) {
	d := Decide("git.status", toolcap.RiskRead, "git", "executor", Policy{}, false)
	if d.Verdict != Allowed {
		t.Fatalf("expected READ to default-allow with no policy, got %+v", d)
	}
}

func TestDecideDefaultWriteNeedsApproval(t *testing.T) {
	d := Decide("git.push", toolcap.RiskWrite, "git", "executor", Policy{}, false)
	if d.Verdict != NeedsApproval {
		t.Fatalf("expected unlisted WRITE to require approval by default, got %+v", d)
	}
}

// No role's RoleMaxRisk ceiling reaches DESTRUCTIVE today (Executor's is
// WRITE, every other role's is READ) — so a DESTRUCTIVE capability is
// denied by the role-ceiling check before it can ever reach the
// default-needs-approval step. This is intentional: Requirement 4 asks
// to "establish the safety model before such adapters exist," and this
// is the concrete consequence — nothing can invoke a DESTRUCTIVE
// capability today without a role/policy change a human makes on purpose.
func TestDecideDestructiveDeniedByRoleCeilingWithNoElevatedRole(t *testing.T) {
	d := Decide("some.destructive_op", toolcap.RiskDestructive, "git", "executor", Policy{}, false)
	if d.Verdict != Denied {
		t.Fatalf("expected DESTRUCTIVE to be denied by every current role's risk ceiling, got %+v", d)
	}
}
