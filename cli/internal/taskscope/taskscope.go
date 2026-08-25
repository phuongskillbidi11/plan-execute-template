package taskscope

import (
	"os"
	"regexp"
	"strings"
)

var taskHeaderRe = regexp.MustCompile(`(?m)^## Task \d+`)

// CurrentTask returns the first task block (delimited by "## Task N —"
// headers, the convention every plan's tasks.md already uses) that still
// contains an unchecked "- [ ]" subtask — the same signal
// scripts/plan-executor.sh (V1) and workflow_cmd.go's tasksComplete
// (Phase 3) already trust. Returns "" (no error) if every task is checked.
func CurrentTask(tasksPath string) (string, error) {
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return "", err
	}
	content := string(data)
	idx := taskHeaderRe.FindAllStringIndex(content, -1)
	for i, loc := range idx {
		end := len(content)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		block := content[loc[0]:end]
		if strings.Contains(block, "- [ ]") {
			return strings.TrimSpace(block), nil
		}
	}
	return "", nil
}

// GoalSummary returns spec.md's "## Goal" section — the one-paragraph
// context an Executor needs, without the rest of spec.md's design
// discussion, design decisions, and file tables.
func GoalSummary(specPath string) (string, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return "", err
	}
	content := string(data)
	start := strings.Index(content, "## Goal")
	if start < 0 {
		return "", nil
	}
	rest := content[start+len("## Goal"):]
	end := strings.Index(rest, "\n## ")
	if end < 0 {
		end = len(rest)
	}
	return strings.TrimSpace(rest[:end]), nil
}
