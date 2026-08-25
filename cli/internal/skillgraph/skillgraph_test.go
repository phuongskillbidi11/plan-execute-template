package skillgraph

import "testing"

import "eng/internal/skills"

func mk(name, domain string, requires []string) skills.Skill {
	return skills.Skill{Name: name, Domain: domain, Requires: requires}
}

func TestExpandTransitiveClosure(t *testing.T) {
	all := []skills.Skill{
		mk("c", "x", nil),
		mk("b", "x", []string{"x/c"}),
		mk("a", "x", []string{"x/b"}),
	}
	out, err := Expand(all, []skills.Skill{all[2]})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("expected a, b, c all included, got %+v", out)
	}
}

func TestExpandDeduplicatesDiamond(t *testing.T) {
	all := []skills.Skill{
		mk("d", "x", nil),
		mk("b", "x", []string{"x/d"}),
		mk("c", "x", []string{"x/d"}),
		mk("a", "x", []string{"x/b", "x/c"}),
	}
	out, err := Expand(all, []skills.Skill{all[3]})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 {
		t.Fatalf("expected exactly 4 distinct skills (d not duplicated), got %d: %+v", len(out), out)
	}
}

func TestExpandDetectsCycle(t *testing.T) {
	all := []skills.Skill{
		mk("a", "x", []string{"x/b"}),
		mk("b", "x", []string{"x/a"}),
	}
	if _, err := Expand(all, []skills.Skill{all[0]}); err == nil {
		t.Fatal("expected a cycle error")
	}
}

func TestExpandUnknownRequiredSkillErrors(t *testing.T) {
	all := []skills.Skill{mk("a", "x", []string{"x/nonexistent"})}
	if _, err := Expand(all, []skills.Skill{all[0]}); err == nil {
		t.Fatal("expected an unknown-required-skill error")
	}
}

func TestExpandDeterministicOrder(t *testing.T) {
	all := []skills.Skill{mk("b", "x", nil), mk("a", "x", nil), mk("c", "x", nil)}
	out1, err := Expand(all, all)
	if err != nil {
		t.Fatal(err)
	}
	shuffled := []skills.Skill{all[2], all[0], all[1]}
	out2, err := Expand(shuffled, shuffled)
	if err != nil {
		t.Fatal(err)
	}
	for i := range out1 {
		if out1[i].Name != out2[i].Name {
			t.Fatalf("order not deterministic: %v vs %v", out1, out2)
		}
	}
}

func TestExpandRequiresByBareNameAlsoWorks(t *testing.T) {
	all := []skills.Skill{
		mk("child", "x", nil),
		mk("parent", "x", []string{"child"}), // bare name, not qualified
	}
	out, err := Expand(all, []skills.Skill{all[1]})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected requires-by-bare-name to resolve too, got %+v", out)
	}
}
