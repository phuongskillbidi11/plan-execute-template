package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeLegacySkill writes a frontmatter-less SKILL.md using the legacy
// "# Skill: name" / "## Purpose" convention — parses with Domain "unknown".
func writeLegacySkill(t *testing.T, dir, name string) {
	t.Helper()
	writeSkill(t, dir, name, "# Skill: "+name+"\n\n## Purpose\n\nLegacy skill body.\n")
}

func TestParseFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "modbus", "---\nname: modbus\ndomain: automation\ndescription: Modbus knowledge\n---\n\nbody\n")
	skills, err := Walk(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "modbus" || skills[0].Domain != "automation" {
		t.Fatalf("got %+v", skills)
	}
}

func TestParseLegacyHeading(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "example", "# Skill: example\n\n## Purpose\n\nLegacy skill description.\n")
	skills, err := Walk(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "example" || skills[0].Domain != "unknown" {
		t.Fatalf("got %+v", skills)
	}
}

func TestResolveLocalOverridesGlobal(t *testing.T) {
	g, l := t.TempDir(), t.TempDir()
	writeSkill(t, g, "shared", "---\nname: shared\ndescription: global version\n---\n")
	writeSkill(t, l, "shared", "---\nname: shared\ndescription: local override\n---\n")
	merged, err := Resolve(g, l)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0].Description != "local override" {
		t.Fatalf("got %+v", merged)
	}
}

func TestResolveMissingRoots(t *testing.T) {
	merged, err := Resolve(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "nope2"))
	if err != nil || len(merged) != 0 {
		t.Fatalf("expected empty, no error; got %+v, %v", merged, err)
	}
}

func TestQualifiedNameLegacySkillStaysBareName(t *testing.T) {
	s := Skill{Name: "example", Domain: "unknown"}
	if s.QualifiedName() != "example" {
		t.Fatalf("expected legacy skill to keep bare name, got %q", s.QualifiedName())
	}
}

func TestQualifiedNameSelfNamespacedNameUnchanged(t *testing.T) {
	s := Skill{Name: "company/internal-api", Domain: "company"}
	if s.QualifiedName() != "company/internal-api" {
		t.Fatalf("expected self-namespaced name unchanged, got %q", s.QualifiedName())
	}
}

func TestResolveQualifiesByDomainToAvoidCollisions(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "automation"), "modbus", "---\nname: modbus\ndomain: automation\ndescription: automation modbus\n---\n")
	writeSkill(t, filepath.Join(root, "networking"), "modbus", "---\nname: modbus\ndomain: networking\ndescription: networking modbus\n---\n")
	merged, err := Resolve(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 distinct skills (domain-qualified), got %d: %+v", len(merged), merged)
	}
}

func TestResolveWithPrivateEmptyRootSkipsTier(t *testing.T) {
	g, l := t.TempDir(), t.TempDir()
	writeSkill(t, g, "only-global", "---\nname: only-global\ndescription: g\n---\n")
	merged, err := ResolveWithPrivate(g, "", l)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(merged))
	}
}

func TestResolveWithPrivatePrecedenceGlobalLtPrivateLtLocal(t *testing.T) {
	g, p, l := t.TempDir(), t.TempDir(), t.TempDir()
	writeSkill(t, g, "shared", "---\nname: shared\ndescription: global\n---\n")
	writeSkill(t, p, "shared", "---\nname: shared\ndescription: private\n---\n")

	merged, err := ResolveWithPrivate(g, p, l)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0].Description != "private" {
		t.Fatalf("expected private to override global, got %+v", merged)
	}

	writeSkill(t, l, "shared", "---\nname: shared\ndescription: local\n---\n")
	merged, err = ResolveWithPrivate(g, p, l)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0].Description != "local" {
		t.Fatalf("expected local to override private, got %+v", merged)
	}
}

// TestResolveCollapsesLegacyDuplicateOfGlobalSkill is the direct regression
// test for the real Phase 9 P2-3 defect: a global engineering/
// karpathy-guidelines and a project-local legacy karpathy-guidelines
// (identical conceptual skill, no frontmatter) used to both resolve as
// distinct entries, since QualifiedName() only collapses skills with an
// exactly matching key and a legacy skill's key is its bare Name, not
// "engineering/karpathy-guidelines".
func TestResolveCollapsesLegacyDuplicateOfGlobalSkill(t *testing.T) {
	g, l := t.TempDir(), t.TempDir()
	writeSkill(t, filepath.Join(g, "engineering"), "karpathy-guidelines",
		"---\nname: karpathy-guidelines\ndomain: engineering\ndescription: global karpathy\n---\n")
	writeLegacySkill(t, l, "karpathy-guidelines")

	merged, err := Resolve(g, l)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 {
		t.Fatalf("expected the legacy duplicate to collapse into one entry, got %d: %+v", len(merged), merged)
	}
	if merged[0].Source != "local" {
		t.Fatalf("expected the higher-precedence (local) tier to win, got source %q", merged[0].Source)
	}
}

// TestResolveDoesNotCollapseDistinctQualifiedSkillsSharingABareName is the
// explicit guard for Decision 6: two real, distinct-domain skills that
// happen to share a bare name (both have real frontmatter, neither is
// legacy) must never be collapsed — automation/modbus vs. a future
// networking/modbus, per docs/skills.md's own documented precedent.
// (TestResolveQualifiesByDomainToAvoidCollisions above already covers this
// exact shape and still passes after the P2-3 fix; this test asserts it
// explicitly under the collapse pass's own name for traceability.)
func TestResolveDoesNotCollapseDistinctQualifiedSkillsSharingABareName(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "automation"), "modbus", "---\nname: modbus\ndomain: automation\ndescription: automation modbus\n---\n")
	writeSkill(t, filepath.Join(root, "networking"), "modbus", "---\nname: modbus\ndomain: networking\ndescription: networking modbus\n---\n")
	merged, err := Resolve(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 {
		t.Fatalf("expected both distinct qualified skills to survive, got %d: %+v", len(merged), merged)
	}
}

// TestResolveWithPrivateCollapsePrecedenceLocalWinsOverPrivateLegacy
// confirms tier precedence still applies correctly to the collapsed
// group: a legacy skill in the private tier loses to a qualified skill of
// the same bare name in the local tier (local > private, as always).
func TestResolveWithPrivateCollapsePrecedenceLocalWinsOverPrivateLegacy(t *testing.T) {
	g, p, l := t.TempDir(), t.TempDir(), t.TempDir()
	writeLegacySkill(t, p, "shared")
	writeSkill(t, l, "shared", "---\nname: shared\ndomain: software\ndescription: local qualified\n---\n")

	merged, err := ResolveWithPrivate(g, p, l)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 {
		t.Fatalf("expected the legacy/qualified pair to collapse into one entry, got %d: %+v", len(merged), merged)
	}
	if merged[0].Source != "local" || merged[0].Description != "local qualified" {
		t.Fatalf("expected local (qualified) to win over private (legacy), got %+v", merged[0])
	}
}
