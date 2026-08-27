package skillmatch

import (
	"testing"

	"eng/internal/skills"
)

func TestScoreCountsTagMatches(t *testing.T) {
	s := skills.Skill{Tags: []string{"planning", "methodology"}, Description: "x"}
	if got := Score(s, "I need help with planning"); got < 1 {
		t.Fatalf("expected at least 1, got %d", got)
	}
}

func TestScoreZeroForNoMatch(t *testing.T) {
	s := skills.Skill{Tags: []string{"modbus"}, Description: "industrial protocol"}
	if got := Score(s, "add a login page"); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestSelectRespectsCapAndRanking(t *testing.T) {
	all := []skills.Skill{
		{Name: "a", Tags: []string{"web"}},
		{Name: "b", Tags: []string{"web", "api"}},
		{Name: "c", Tags: []string{"database"}},
	}
	selected := Select(all, "build a web api", nil, 1)
	if len(selected) != 1 || selected[0].Name != "b" {
		t.Fatalf("expected only the highest-scoring skill 'b', got %+v", selected)
	}
}

func TestSelectAlwaysIncludesRequiredEvenBeyondCap(t *testing.T) {
	all := []skills.Skill{
		{Name: "required-skill", Tags: []string{"unrelated"}},
		{Name: "matched", Tags: []string{"web"}},
	}
	selected := Select(all, "build a web thing", []string{"required-skill"}, 1)
	names := map[string]bool{}
	for _, s := range selected {
		names[s.Name] = true
	}
	if !names["required-skill"] {
		t.Fatalf("required-skill must always be included, got %+v", selected)
	}
}

// TestSelectMatchesDomainQualifiedEnabledSkills guards against a real
// defect found during Phase 4 integration testing: eng init writes
// enabled_skills entries as "domain/name" (e.g.
// "engineering/karpathy-guidelines"), but a resolved skill's Name is its
// bare frontmatter name ("karpathy-guidelines") — the "always included"
// guarantee silently failed for exactly the entry eng init itself creates.
func TestSelectMatchesDomainQualifiedEnabledSkills(t *testing.T) {
	all := []skills.Skill{
		{Name: "karpathy-guidelines", Tags: []string{"planning"}},
	}
	selected := Select(all, "totally unrelated request text", []string{"engineering/karpathy-guidelines"}, 5)
	if len(selected) != 1 || selected[0].Name != "karpathy-guidelines" {
		t.Fatalf("domain-qualified enabled_skills entry must still match bare skill Name, got %+v", selected)
	}
}

// TestScoreIgnoresSubstringInsideUnrelatedWord guards the real Phase 9
// P2-2 defect: "form" (a description word of an unrelated skill, e.g. "...
// form a hypothesis...") must not match merely because it's a substring of
// the request word "WinForms" — matching is now word-boundary/token based,
// not raw substring containment.
func TestScoreIgnoresSubstringInsideUnrelatedWord(t *testing.T) {
	s := skills.Skill{Description: "form a hypothesis about the cause"}
	if got := Score(s, "Maintain a C# WinForms configuration tool"); got != 0 {
		t.Fatalf("expected 0 (no word-boundary match), got %d", got)
	}
}

// TestSingleDescriptionWordAloneDoesNotClearThreshold guards the other half
// of P2-2: a single generic-vocabulary description word (weight
// DescriptionWordWeight=1) must not, on its own, reach MinMatchScore.
func TestSingleDescriptionWordAloneDoesNotClearThreshold(t *testing.T) {
	s := skills.Skill{Description: "covers protocol framing details"}
	got := Score(s, "a binary protocol framing issue")
	if got < MinMatchScore {
		t.Fatalf("sanity: expected the two whole-word desc hits to sum above threshold, got %d", got)
	}
	single := skills.Skill{Description: "covers protocol basics only"}
	if got := Score(single, "a totally unrelated firmware question"); got >= MinMatchScore {
		t.Fatalf("expected a lone description word not to clear MinMatchScore, got %d", got)
	}
}

// TestSingleTagOrTriggerAloneClearsThreshold guards that a single curated
// signal (a tag or trigger) is still enough on its own — only prose-word
// matches were weakened by the P2-2 fix.
func TestSingleTagOrTriggerAloneClearsThreshold(t *testing.T) {
	byTag := skills.Skill{Tags: []string{"rs485"}}
	if got := Score(byTag, "an RS485 configuration question"); got < MinMatchScore {
		t.Fatalf("expected a single tag match to clear MinMatchScore, got %d", got)
	}
	byTrigger := skills.Skill{Triggers: []string{"usb-hid"}}
	if got := Score(byTrigger, "reading from a USB-HID device"); got < MinMatchScore {
		t.Fatalf("expected a single trigger match to clear MinMatchScore, got %d", got)
	}
}

// TestScoreDoesNotDoubleCountRepeatedDescriptionWord guards against a word
// appearing more than once in a description inflating the score — modbus's
// real description contains "framing" twice ("framing pitfalls" / "framing
// differs"), which must still only count once.
func TestScoreDoesNotDoubleCountRepeatedDescriptionWord(t *testing.T) {
	s := skills.Skill{Description: "framing pitfalls are common; framing differs by mode"}
	if got := Score(s, "a question about framing"); got != DescriptionWordWeight {
		t.Fatalf("expected exactly one DescriptionWordWeight for a repeated word, got %d", got)
	}
}

// TestScoreDistinguishesCSharpFromCpp guards a real defect found while
// reproducing P2-2: "C++" and "C#" both degenerate to the same bare "c"
// token once punctuation is stripped, unless normalized first — causing a
// C# request to falsely match the unrelated C++ skill (found via
// software/cpp's own "c++" tag during real RS485/WinForms reproduction).
func TestScoreDistinguishesCSharpFromCpp(t *testing.T) {
	cpp := skills.Skill{Tags: []string{"cpp", "c++"}, Triggers: []string{"cpp", "c++"}}
	if got := Score(cpp, "Maintain a C# WinForms desktop application"); got != 0 {
		t.Fatalf("expected the C++ skill not to match a C# request, got %d", got)
	}
	csharp := skills.Skill{Tags: []string{"csharp", "c#"}}
	if got := Score(csharp, "Maintain a C# WinForms desktop application"); got < MinMatchScore {
		t.Fatalf("expected the C# skill to match a C# request, got %d", got)
	}
}

func TestSelectNoCapReturnsAllMatches(t *testing.T) {
	all := []skills.Skill{
		{Name: "a", Tags: []string{"web"}},
		{Name: "b", Tags: []string{"web"}},
	}
	selected := Select(all, "web", nil, 0)
	if len(selected) != 2 {
		t.Fatalf("expected both matches with no cap, got %d", len(selected))
	}
}
