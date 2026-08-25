package toolpolicy

import (
	"eng/internal/agent"
	"eng/internal/toolcap"
)

// Policy is the project-level tool policy — .agent/project.yaml's
// `tools:` block (Phase 7 spec.md Decision 2: deliberately new/nested,
// not a reuse of the pre-existing, undocumented, unread
// Config.RequireApproval field).
type Policy struct {
	Allow           []string `yaml:"allow,omitempty"`
	RequireApproval []string `yaml:"require_approval,omitempty"`
	Deny            []string `yaml:"deny,omitempty"`
}

// HardDeny is the built-in, project-config-immune deny list (Requirement
// 21) — no override mechanism exists in Phase 7, on purpose.
var HardDeny = map[string]bool{
	"git.force_push": true,
}

type Verdict string

const (
	Allowed       Verdict = "ALLOWED"
	NeedsApproval Verdict = "NEEDS_APPROVAL"
	Denied        Verdict = "DENIED"
)

type Decision struct {
	Verdict Verdict
	Reason  string
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Decide applies the fixed precedence in Phase 7 spec.md Decision 7:
// hard deny -> project deny -> role toolbox -> role risk ceiling ->
// project require_approval -> project allow -> default (READ open,
// WRITE+ needs approval). adapterName is the owning adapter's Name()
// (e.g. "git"), used for the coarse role-toolbox check; approved is
// whether this plan's execution approval (plan.yaml's approved_at) has
// already been granted.
func Decide(capability string, risk toolcap.Risk, adapterName, role string, policy Policy, approved bool) Decision {
	if HardDeny[capability] {
		return Decision{Denied, "hard deny — never invocable regardless of policy"}
	}
	if contains(policy.Deny, capability) {
		return Decision{Denied, "denied by project tools.deny"}
	}
	if !agent.RoleMayUse(role, adapterName) {
		return Decision{Denied, "role's toolbox does not include adapter " + adapterName}
	}
	if !agent.RoleMayInvokeRisk(role, risk) {
		return Decision{Denied, "role may not invoke " + string(risk) + "-risk capabilities"}
	}
	if contains(policy.RequireApproval, capability) {
		if approved {
			return Decision{Allowed, "requires approval — plan is approved"}
		}
		return Decision{NeedsApproval, "listed in project tools.require_approval — plan not yet approved"}
	}
	if contains(policy.Allow, capability) {
		return Decision{Allowed, "allowed by project tools.allow"}
	}
	if risk == toolcap.RiskRead {
		return Decision{Allowed, "read capability, no explicit policy — default-open for READ"}
	}
	if approved {
		return Decision{Allowed, string(risk) + "-risk capability, plan is approved"}
	}
	return Decision{NeedsApproval, string(risk) + "-risk capability not explicitly listed — requires plan approval before invocation"}
}
