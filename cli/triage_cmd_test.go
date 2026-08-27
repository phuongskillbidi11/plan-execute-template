package main

import "testing"

// TestKeywordLevelQuickFixParameterTweak guards against a real defect found
// during Phase 8 dogfooding and confirmed in benchmarks/results/
// quick-fix-timeout-harness-v2.yaml: this exact phrase — the Phase 8
// instruction's own canonical Quick Fix example — triaged to "feature", not
// "quick-fix", because the old quick-fix word list only covered
// typo/rename/comment/formatting phrasing.
func TestKeywordLevelQuickFixParameterTweak(t *testing.T) {
	level, _ := keywordLevel("Increase the reconnect timeout from 1000 ms to 1500 ms.")
	if level != "quick-fix" {
		t.Fatalf("expected quick-fix, got %q", level)
	}
}

// TestKeywordLevelRiskyParameterTweakStaysElevated guards the other half of
// the P1-1 fix: a one-line change must not automatically become Quick Fix
// just because it has size-hinting wording, if it also touches something
// risky (here: authentication).
func TestKeywordLevelRiskyParameterTweakStaysElevated(t *testing.T) {
	level, _ := keywordLevel("Increase the session timeout for authenticated users from 30 to 60 minutes.")
	if level == "quick-fix" {
		t.Fatalf("expected an elevated (non-quick-fix) level for an auth-touching change, got %q", level)
	}
}

func TestKeywordLevelHardwareControlStaysElevated(t *testing.T) {
	level, _ := keywordLevel("Increase the relay output hold time from 200 to 500 ms.")
	if level == "quick-fix" {
		t.Fatalf("expected an elevated (non-quick-fix) level for a hardware-control change, got %q", level)
	}
}

func TestKeywordLevelBreakingPublicAPIChangeStaysArchitecture(t *testing.T) {
	level, _ := keywordLevel("This is a breaking change to the public API response shape.")
	if level != "architecture" {
		t.Fatalf("expected architecture, got %q", level)
	}
}

// A data/schema migration was already covered by the pre-existing bare
// "migration" high-risk keyword before this phase — confirming it stays at
// least that conservative, not merely architecture, is the correct bar.
func TestKeywordLevelDataMigrationStaysHighRisk(t *testing.T) {
	level, _ := keywordLevel("Perform a database migration to add the new schema column.")
	if level != "high-risk" {
		t.Fatalf("expected high-risk, got %q", level)
	}
}

// Existing keyword categories must keep classifying exactly as before.
func TestKeywordLevelExistingCategoriesUnchanged(t *testing.T) {
	cases := map[string]string{
		"rename the internal helper variable for clarity": "quick-fix",
		"the checkout page is broken":                     "bug",
		"redesign the plugin system":                      "architecture",
		"deploy the new build to production":              "high-risk",
		"add a CSV export feature":                        "feature",
	}
	for text, want := range cases {
		if got, _ := keywordLevel(text); got != want {
			t.Errorf("keywordLevel(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestLooksLikeParameterTweak(t *testing.T) {
	cases := map[string]bool{
		"increase the reconnect timeout from 1000 ms to 1500 ms": true,
		"bump the retry count to 5":                              true,
		"adjust the cache size to 256mb":                         true,
		"rewrite the entire module":                              false, // no digit
		"increase test coverage":                                 false, // no digit
	}
	for text, want := range cases {
		if got := looksLikeParameterTweak(text); got != want {
			t.Errorf("looksLikeParameterTweak(%q) = %v, want %v", text, got, want)
		}
	}
}
