package agent

import "eng/internal/toolcap"

// RolePermissions is a static, reporting-only table — nothing yet
// enforces this against a real tool invocation (see Phase 5 spec.md
// Decision 11); Phase 7's toolpolicy.Decide is the first real consumer.
// Both "github" (tooladapter.GitHubAdapter's conceptual Name(), what
// toolpolicy.Decide checks) and "gh" (the literal binary name
// internal/capabilities.Known must use for exec.LookPath, what `eng
// capabilities list --role` checks) are listed — two names for the same
// tool at two different layers; omitting either would silently exclude
// "gh" from the older binary-detection command's --role filter.
// Every role's toolbox includes "codex" — Phase 10 spec.md Decision 8: all
// three shipped Codex capabilities are READ-risk, so every role's
// existing RoleMaxRisk already permits them; which specific capability a
// role reaches for (codex.inspect/codex.review/codex.verify) is a
// METHOD.md/documentation concern, not a toolbox restriction.
var RolePermissions = map[Role][]string{
	RolePlanner:  {"git", "github", "gh", "mcp-docs", "codex"},
	RoleReviewer: {"git", "github", "gh", "mcp-docs", "codex"},
	RoleExecutor: {"git", "github", "gh", "mcp-docs", "claude", "codex", "docker"},
	RoleVerifier: {"git", "github", "gh", "mcp-docs", "docker", "codex"},
}

// RoleMayUse reports whether role is permitted to consider capability.
func RoleMayUse(role, capability string) bool {
	for _, c := range RolePermissions[Role(role)] {
		if c == capability {
			return true
		}
	}
	return false
}

// RoleMaxRisk is the highest capability risk tier a role may invoke
// without an explicit approval escalation (Phase 7 Requirement 5) — an
// axis independent from RolePermissions above: an adapter being in a
// role's toolbox does not by itself grant every risk tier that adapter
// exposes (Phase 7 spec.md Decision 4).
var RoleMaxRisk = map[Role]toolcap.Risk{
	RolePlanner:  toolcap.RiskRead,
	RoleReviewer: toolcap.RiskRead,
	RoleExecutor: toolcap.RiskWrite,
	RoleVerifier: toolcap.RiskRead,
}

// RoleMayInvokeRisk reports whether role's risk ceiling covers risk. An
// unknown role has no ceiling — nothing is permitted.
func RoleMayInvokeRisk(role string, risk toolcap.Risk) bool {
	max, ok := RoleMaxRisk[Role(role)]
	if !ok {
		return false
	}
	return toolcap.RiskRank(risk) <= toolcap.RiskRank(max)
}
