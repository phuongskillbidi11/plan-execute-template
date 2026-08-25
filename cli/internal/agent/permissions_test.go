package agent

import (
	"testing"

	"eng/internal/toolcap"
)

func TestPlannerMayUseGitOnly(t *testing.T) {
	if !RoleMayUse("planner", "git") {
		t.Fatal("expected planner to be permitted git")
	}
	if RoleMayUse("planner", "docker") {
		t.Fatal("expected planner NOT to be permitted docker")
	}
}

func TestExecutorMayUseDocker(t *testing.T) {
	if !RoleMayUse("executor", "docker") {
		t.Fatal("expected executor to be permitted docker")
	}
}

func TestUnknownRoleMayUseNothing(t *testing.T) {
	if RoleMayUse("not-a-real-role", "git") {
		t.Fatal("expected an unknown role to be permitted nothing")
	}
}

func TestRoleMayUseGitHubInEveryRolesToolbox(t *testing.T) {
	for _, role := range []string{"planner", "plan-reviewer", "executor", "verifier"} {
		if !RoleMayUse(role, "github") {
			t.Fatalf("expected %s to have github in its toolbox", role)
		}
	}
}

// Regression test for a real Phase 7 defect: the ReferenceMCPAdapter
// (Name() == "mcp-docs") was implemented and wired into
// registeredAdapters/toolrouter.Route, but never added to
// RolePermissions — meaning every role's toolbox check silently denied
// it forever, regardless of availability or policy. Caught via a live
// `eng capabilities explain` walkthrough, not by a unit test — this test
// exists so the same class of "new adapter, forgotten toolbox entry"
// mistake fails loudly next time.
func TestRoleMayUseMCPDocsInEveryRolesToolbox(t *testing.T) {
	for _, role := range []string{"planner", "plan-reviewer", "executor", "verifier"} {
		if !RoleMayUse(role, "mcp-docs") {
			t.Fatalf("expected %s to have mcp-docs in its toolbox", role)
		}
	}
}

// Regression test for a real Phase 7 defect found during final
// verification: `eng capabilities list --role <role> --verbose` filters
// internal/capabilities.Known entries (binary names, e.g. "gh") by
// RoleMayUse, but RolePermissions only listed the adapter's conceptual
// name "github" — so "gh" silently never appeared for any role, even
// though the actual gh binary and GitHub adapter were both available.
func TestRoleMayUseGhBinaryNameInEveryRolesToolbox(t *testing.T) {
	for _, role := range []string{"planner", "plan-reviewer", "executor", "verifier"} {
		if !RoleMayUse(role, "gh") {
			t.Fatalf("expected %s to have gh in its toolbox", role)
		}
	}
}

func TestRoleMaxRiskPlannerReadOnly(t *testing.T) {
	if !RoleMayInvokeRisk("planner", toolcap.RiskRead) {
		t.Fatal("expected planner to invoke READ")
	}
	if RoleMayInvokeRisk("planner", toolcap.RiskWrite) {
		t.Fatal("expected planner NOT to invoke WRITE")
	}
}

func TestRoleMaxRiskExecutorReadWrite(t *testing.T) {
	if !RoleMayInvokeRisk("executor", toolcap.RiskWrite) {
		t.Fatal("expected executor to invoke WRITE")
	}
	if RoleMayInvokeRisk("executor", toolcap.RiskDestructive) {
		t.Fatal("expected executor NOT to invoke DESTRUCTIVE")
	}
}

func TestRoleMaxRiskUnknownRoleDeniedEverything(t *testing.T) {
	if RoleMayInvokeRisk("not-a-real-role", toolcap.RiskRead) {
		t.Fatal("expected an unknown role to be denied even READ")
	}
}
