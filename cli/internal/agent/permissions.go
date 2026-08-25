package agent

// RolePermissions is a static, reporting-only table — nothing yet enforces
// this against a real tool invocation (see Phase 5 spec.md Decision 11).
var RolePermissions = map[Role][]string{
	RolePlanner:  {"git"},
	RoleReviewer: {"git"},
	RoleExecutor: {"git", "claude", "codex", "docker"},
	RoleVerifier: {"git", "docker"},
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
