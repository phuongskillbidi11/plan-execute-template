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
