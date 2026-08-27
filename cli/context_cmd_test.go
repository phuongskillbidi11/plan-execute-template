package main

import (
	"os"
	"path/filepath"
	"testing"

	"eng/internal/planmeta"
)

// TestBuildContextBundleWritesPerRoleManifest is the direct regression
// test for Phase 10 spec.md's "role/context history exists for each
// activated role" acceptance criterion: activating two different roles
// for the same plan must not have one overwrite the other's evidence —
// each gets its own context-manifest-<role>.yaml, and the unqualified
// context-manifest.yaml still exists as the last-activation pointer.
func TestBuildContextBundleWritesPerRoleManifest(t *testing.T) {
	planDir := t.TempDir()
	meta := &planmeta.Meta{Plan: "test-plan", RiskLevel: "feature", State: "PLANNED"}
	if err := planmeta.Save(planDir, meta); err != nil {
		t.Fatal(err)
	}

	if _, err := buildContextBundle("planner", planDir, "a request"); err != nil {
		t.Fatalf("planner: %v", err)
	}
	if _, err := buildContextBundle("executor", planDir, "a request"); err != nil {
		t.Fatalf("executor: %v", err)
	}

	for _, name := range []string{"context-manifest-planner.yaml", "context-manifest-executor.yaml", "context-manifest.yaml"} {
		if _, err := os.Stat(filepath.Join(planDir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}

	// The unqualified file must reflect whichever role activated most
	// recently (executor), not be stuck on the first (planner).
	unqualified, err := os.ReadFile(filepath.Join(planDir, "context-manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	executorManifest, err := os.ReadFile(filepath.Join(planDir, "context-manifest-executor.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(unqualified) != string(executorManifest) {
		t.Fatalf("expected the unqualified manifest to match the most recent (executor) activation")
	}
}

func TestBuildContextBundleUnknownRoleErrors(t *testing.T) {
	planDir := t.TempDir()
	meta := &planmeta.Meta{Plan: "test-plan", RiskLevel: "feature", State: "PLANNED"}
	if err := planmeta.Save(planDir, meta); err != nil {
		t.Fatal(err)
	}
	if _, err := buildContextBundle("not-a-real-role", planDir, "x"); err == nil {
		t.Fatal("expected an error for an unknown role")
	}
}
