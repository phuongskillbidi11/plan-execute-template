package agent

import "testing"

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
