package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/docsearch"
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

// levelRank orders risk levels so a gotcha match can "hold or raise" but
// never lower the keyword-based suggestion (Requirement 21: a heuristic
// must stay observable and must never claim more authority than it has —
// this only ever nudges upward, it does not reclassify).
var levelRank = map[string]int{
	"quick-fix":    0,
	"bug":          1,
	"feature":      2,
	"architecture": 3,
	"high-risk":    4,
}

// triageLevel is the pure heuristic, factored out so `eng workflow start`
// can call it directly instead of going through cmdTriage's print/exit path.
func triageLevel(text string) (level, workflowDesc string) {
	level, workflowDesc = keywordLevel(text)

	dir, err := os.Getwd()
	if err != nil {
		return level, workflowDesc
	}
	sections, err := docsearch.ParseSections(filepath.Join(dir, "docs", "gotchas.md"))
	if err != nil {
		return level, workflowDesc
	}
	if matched := docsearch.Match(sections, text, 1); len(matched) > 0 {
		if levelRank["architecture"] > levelRank[level] {
			return "architecture", "matched a documented gotcha (" + matched[0].Title + ") — research + ADR + full spec/tasks/tests, elevated from the keyword-only suggestion"
		}
	}
	return level, workflowDesc
}

func keywordLevel(text string) (level, workflowDesc string) {
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
