package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRolePromptIncludesMethodAndPlanPath(t *testing.T) {
	harness := t.TempDir()
	methodDir := filepath.Join(harness, "core", "executor")
	os.MkdirAll(methodDir, 0o755)
	os.WriteFile(filepath.Join(methodDir, "METHOD.md"), []byte("# Core Method: Executor\nDo the thing."), 0o644)

	a := ClaudeCodeAdapter{HarnessDir: harness}
	prompt, err := a.RolePrompt(RoleExecutor, "/some/plan/dir")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Do the thing.") {
		t.Fatalf("prompt missing method content: %s", prompt)
	}
	if !strings.Contains(prompt, "executor") {
		t.Fatalf("prompt missing role name: %s", prompt)
	}
}

func TestRolePromptErrorsOnMissingMethod(t *testing.T) {
	a := ClaudeCodeAdapter{HarnessDir: t.TempDir()}
	if _, err := a.RolePrompt(RolePlanner, "."); err == nil {
		t.Fatal("expected an error for a missing METHOD.md")
	}
}

func TestAvailableReturnsWithoutPanicking(t *testing.T) {
	a := ClaudeCodeAdapter{}
	_ = a.Available()
}
