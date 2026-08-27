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
		[]string{
			"production", "deploy", "migration", "flash", "firmware", "plc write", "delete data", "drop table",
			// Security/auth: a one-line change must not become Quick Fix just because it's
			// small — see Phase 9 spec.md P1-1. Stems ("authenticat", "authoriz") catch
			// authenticate/authenticated/authentication and authorize/authorization; kept
			// deliberately more specific than bare "auth" to avoid colliding with unrelated
			// words like "author".
			"authenticat", "authoriz", "credential", "password", "login", "permission", "access control", "auth token",
			// Hardware/control: a PLC/actuator output write is risky regardless of size.
			"actuator", "motor control", "relay output", "move axis", "gpio write",
		}},
	{"architecture", "research + ADR + full spec/tasks/tests",
		[]string{
			"architecture", "redesign", "migrate to", "replace", "rewrite",
			// Public API contract changes: bigger blast radius than their diff size suggests.
			// (Schema/database migration wording is already covered by high-risk's existing
			// "migration" keyword above, checked first — no separate entry needed here.)
			"breaking change", "public api",
		}},
	{"bug", "bug workflow — reproduce, fix, regression test",
		[]string{"bug", "fix", "broken", "error", "crash", "fails"}},
	{"quick-fix", "quick workflow — skip full spec, single-file plan",
		[]string{"typo", "rename", "comment", "formatting", "small change"}},
}

// parameterTweakVerbs are change-verb words that, combined with a digit
// anywhere in the request, suggest a localized numeric/parameter-value
// tweak (Phase 9 spec.md P1-1's canonical example: "Increase the reconnect
// timeout from 1000 ms to 1500 ms."). This check runs only after every
// category in triageKeywords above has already had a chance to match, so a
// risk-elevating or bug keyword always wins — this can only ever suggest
// quick-fix, never override something riskier.
var parameterTweakVerbs = []string{
	"increase", "decrease", "adjust", "tune", "bump", "raise", "lower",
	"change the", "set the", "update the",
}

// looksLikeParameterTweak is a narrow, deterministic heuristic — not a
// general "is this small" classifier. It intentionally has false negatives
// (it will miss plenty of genuinely small changes) in exchange for staying
// explainable: a human can read the verb list and know exactly why a
// request did or didn't match.
func looksLikeParameterTweak(lower string) bool {
	hasChangeVerb := false
	for _, v := range parameterTweakVerbs {
		if strings.Contains(lower, v) {
			hasChangeVerb = true
			break
		}
	}
	if !hasChangeVerb {
		return false
	}
	return strings.ContainsAny(lower, "0123456789")
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
	if looksLikeParameterTweak(lower) {
		return "quick-fix", "quick workflow — localized parameter/config value change, skip full spec"
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
