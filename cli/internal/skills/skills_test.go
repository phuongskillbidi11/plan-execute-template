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
