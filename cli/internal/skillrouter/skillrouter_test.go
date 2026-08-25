package skillrouter

import (
	"testing"

	"eng/internal/skills"
)

func mk(name, domain string, requires, recommends []string) skills.Skill {
	return skills.Skill{Name: name, Domain: domain, Description: "d " + name, Requires: requires, Recommends: recommends}
}

func names(sel Selection) []string {
	out := make([]string, len(sel.Skills))
	for i, s := range sel.Skills {
		out[i] = s.Name
	}
	return out
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func TestExplicitNeverDroppedEvenWithZeroScoreAndTinyBudget(t *testing.T) {
	all := []skills.Skill{mk("a", "x", nil, nil), mk("b", "x", nil, nil)}
	sel, err := Route(all, "unrelated text", []string{"a"}, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(names(sel), "a") {
		t.Fatalf("expected explicit skill 'a' to survive, got %v", names(sel))
	}
}

func TestRequiredDependencyIgnoresBudget(t *testing.T) {
	all := []skills.Skill{
		mk("child", "automation", nil, nil),
		mk("parent", "automation", []string{"automation/child"}, nil),
	}
	sel, err := Route(all, "", []string{"parent"}, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Skills) != 2 {
		t.Fatalf("expected both parent and its required child despite budget 1, got %v", names(sel))
	}
}

func TestUnknownRequiredSkillReturnsError(t *testing.T) {
	all := []skills.Skill{mk("a", "x", []string{"x/nonexistent"}, nil)}
	if _, err := Route(all, "", []string{"a"}, nil, 0); err == nil {
		t.Fatal("expected an error for an unknown required skill")
	}
}

func TestRecommendsDroppedWhenBudgetExhausted(t *testing.T) {
	all := []skills.Skill{
		mk("main", "x", nil, []string{"x/extra"}),
		mk("extra", "x", nil, nil),
	}
	sel, err := Route(all, "main", nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if contains(names(sel), "extra") {
		t.Fatalf("expected the recommend to be dropped at budget 1, got %v", names(sel))
	}
}

func TestRecommendsIncludedWhenBudgetAllows(t *testing.T) {
	all := []skills.Skill{
		mk("main", "x", nil, []string{"x/extra"}),
		mk("extra", "x", nil, nil),
	}
	sel, err := Route(all, "main", nil, nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(names(sel), "extra") {
		t.Fatalf("expected the recommend to be included when budget allows, got %v", names(sel))
	}
	for _, e := range sel.Explanations {
		if e.Skill == "extra" && e.Reason != "recommended by x/main" {
			t.Fatalf("unexpected reason for extra: %q", e.Reason)
		}
	}
}

func TestHigherScoringMatchWinsBudget(t *testing.T) {
	strong := skills.Skill{Name: "strong", Domain: "x", Description: "d", Tags: []string{"alpha", "beta"}, Triggers: []string{"gamma"}}
	weak := skills.Skill{Name: "weak", Domain: "x", Description: "d", Tags: []string{"alpha"}}
	all := []skills.Skill{strong, weak}
	sel, err := Route(all, "alpha beta gamma", nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(names(sel), "strong") || contains(names(sel), "weak") {
		t.Fatalf("expected only the higher-scoring skill to survive budget 1, got %v", names(sel))
	}
}

func TestDomainProfileFillAfterStrongMatches(t *testing.T) {
	all := []skills.Skill{
		mk("matched", "automation", nil, nil),
		mk("profileonly", "automation", nil, nil),
	}
	sel, err := Route(all, "matched", nil, []string{"automation"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(names(sel), "matched") || !contains(names(sel), "profileonly") {
		t.Fatalf("expected both a strong match and a domain-profile fill, got %v", names(sel))
	}
}

func TestDeterministicOrderingAcrossRuns(t *testing.T) {
	all := []skills.Skill{
		{Name: "b", Domain: "x", Description: "d", Tags: []string{"shared"}},
		{Name: "a", Domain: "x", Description: "d", Tags: []string{"shared"}},
		{Name: "c", Domain: "x", Description: "d", Tags: []string{"shared"}},
	}
	sel1, err := Route(all, "shared", nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	sel2, err := Route(all, "shared", nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(sel1.Skills) != 3 {
		t.Fatalf("expected all 3 equally-scored skills selected, got %v", names(sel1))
	}
	for i := range sel1.Skills {
		if sel1.Skills[i].Name != sel2.Skills[i].Name {
			t.Fatalf("order differs across identical runs: %v vs %v", names(sel1), names(sel2))
		}
	}
	if names(sel1)[0] != "a" || names(sel1)[1] != "b" || names(sel1)[2] != "c" {
		t.Fatalf("expected alphabetical tie-break order, got %v", names(sel1))
	}
}
