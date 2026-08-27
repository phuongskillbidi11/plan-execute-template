package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTasksFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestUncheckedChecklistLinesNamesTheBlockingLines guards the real Phase 9
// P2-1 UX defect: "tasks.md still has unchecked items" gave no indication
// of which line was blocking, even when every per-task **Status:** marker
// was already [x] — only the bottom Completion checklist actually gates
// eng workflow advance.
func TestUncheckedChecklistLinesNamesTheBlockingLines(t *testing.T) {
	content := "### Task 1.1\n**Status:** `[x]`\n\n## Completion checklist\n\n" +
		"- [x] All tasks marked `[x]`\n" +
		"- [ ] No tasks marked `[!]`\n" +
		"- [ ] Build passes\n" +
		"- [x] All tests pass\n"
	dir := writeTasksFixture(t, content)

	got := uncheckedChecklistLines(dir)
	want := []string{"- [ ] No tasks marked `[!]`", "- [ ] Build passes"}
	if len(got) != len(want) {
		t.Fatalf("expected %d unchecked lines, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestUncheckedChecklistLinesEmptyWhenAllChecked(t *testing.T) {
	dir := writeTasksFixture(t, "## Completion checklist\n\n- [x] All tasks marked `[x]`\n- [x] Build passes\n")
	if got := uncheckedChecklistLines(dir); len(got) != 0 {
		t.Fatalf("expected no unchecked lines, got %v", got)
	}
}

// TestTasksCompleteIgnoresPerTaskStatusMarker guards the "zero behavior
// change" requirement from DECISION_LOG.md Decision 5: tasksComplete must
// keep gating only on the bottom checklist, exactly as it always has — a
// plan whose per-task Status markers are still [ ] but whose bottom
// checklist is fully [x] must still be considered complete.
func TestTasksCompleteIgnoresPerTaskStatusMarker(t *testing.T) {
	content := "### Task 1.1\n**Status:** `[ ]`\n\n## Completion checklist\n\n- [x] All tasks marked `[x]`\n"
	dir := writeTasksFixture(t, content)
	if !tasksComplete(dir) {
		t.Fatal("expected tasksComplete to ignore the per-task Status marker and gate only on the bottom checklist")
	}
}

func TestTasksCompleteFalseWhenChecklistItemUnchecked(t *testing.T) {
	dir := writeTasksFixture(t, "## Completion checklist\n\n- [ ] All tasks marked `[x]`\n")
	if tasksComplete(dir) {
		t.Fatal("expected tasksComplete to be false with an unchecked checklist item")
	}
}
