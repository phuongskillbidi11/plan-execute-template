package taskscope

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentTaskReturnsFirstUnchecked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	content := "# Tasks\n\n## Task 1 — Done thing\n\n- [x] **1.1** already done\n\n## Task 2 — Pending thing\n\n- [ ] **2.1** not done yet\n"
	os.WriteFile(path, []byte(content), 0o644)

	task, err := CurrentTask(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(task, "Task 2") || strings.Contains(task, "Task 1") {
		t.Fatalf("expected only Task 2's block, got: %s", task)
	}
}

func TestCurrentTaskEmptyWhenAllChecked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	os.WriteFile(path, []byte("## Task 1 — Done\n\n- [x] **1.1** done\n"), 0o644)

	task, err := CurrentTask(path)
	if err != nil {
		t.Fatal(err)
	}
	if task != "" {
		t.Fatalf("expected empty string, got: %s", task)
	}
}

func TestGoalSummaryExtractsOnlyGoalSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.md")
	content := "# Spec\n\n## Goal\n\nDo the thing.\n\n## Design\n\nLots of detail here.\n"
	os.WriteFile(path, []byte(content), 0o644)

	goal, err := GoalSummary(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(goal, "Do the thing.") || strings.Contains(goal, "Lots of detail") {
		t.Fatalf("got: %q", goal)
	}
}
