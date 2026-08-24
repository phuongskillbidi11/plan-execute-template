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

func cmdTriage(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: eng triage "<request text>"`)
		os.Exit(1)
	}
	text := strings.ToLower(strings.Join(args, " "))

	for _, k := range triageKeywords {
		for _, w := range k.words {
			if strings.Contains(text, w) {
				printTriage(k.level, k.workflow)
				return
			}
		}
	}
	printTriage("feature", "full spec + tasks + tests")
}

func printTriage(level, workflow string) {
	fmt.Printf("Suggested level: %s\n", level)
	fmt.Printf("Suggested workflow: %s\n", workflow)
	fmt.Println("\n(heuristic hint only — the Planner makes the final call)")
}
