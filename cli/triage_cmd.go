package main

import (
	"fmt"
	"os"
	"strings"
)

var triageKeywords = []struct {
	level    string
	workflow string
	words    []string
}{
	{"high-risk", "high-risk workflow — requires human approval before executing",
		[]string{"production", "deploy", "migration", "flash", "firmware", "plc write", "delete data", "drop table"}},
	{"architecture", "research + ADR + full spec/tasks/tests",
		[]string{"architecture", "redesign", "migrate to", "replace", "rewrite"}},
	{"bug", "bug workflow — reproduce, fix, regression test",
		[]string{"bug", "fix", "broken", "error", "crash", "fails"}},
	{"quick-fix", "quick workflow — skip full spec, single-file plan",
		[]string{"typo", "rename", "comment", "formatting", "small change"}},
}

// triageLevel is the pure heuristic, factored out so `eng workflow start`
// can call it directly instead of going through cmdTriage's print/exit path.
func triageLevel(text string) (level, workflowDesc string) {
	lower := strings.ToLower(text)
	for _, k := range triageKeywords {
		for _, w := range k.words {
			if strings.Contains(lower, w) {
				return k.level, k.workflow
			}
		}
	}
	return "feature", "full spec + tasks + tests"
}

func cmdTriage(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: eng triage "<request text>"`)
		os.Exit(1)
	}
	level, wf := triageLevel(strings.Join(args, " "))
	fmt.Printf("Suggested level: %s\n", level)
	fmt.Printf("Suggested workflow: %s\n", wf)
	fmt.Println("\n(heuristic hint only — the Planner makes the final call)")
}
