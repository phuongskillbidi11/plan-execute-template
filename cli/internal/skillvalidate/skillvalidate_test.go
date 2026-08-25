package skillvalidate

import (
	"os"
	"path/filepath"
	"strings"
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

func TestValidateLegacySkillWarnsNotErrors(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "legacy", "# Skill: legacy\n\n## Purpose\n\nAn old-style skill.\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Errors()) != 0 {
		t.Fatalf("expected no errors for a legacy skill, got %+v", report.Errors())
	}
	if len(report.Warnings()) == 0 {
		t.Fatal("expected a warning for a legacy skill")
	}
}

func TestValidateMissingDescriptionWarns(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "nodesc", "---\nname: nodesc\ndomain: x\n---\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, i := range report.Warnings() {
		if i.Skill == "nodesc" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a missing-description warning, got %+v", report.Issues)
	}
}

func TestValidateUnknownRequiresErrors(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nrequires: [x/nonexistent]\n---\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Errors()) == 0 {
		t.Fatal("expected an error for an unknown required skill")
	}
}

func TestValidateUnknownRecommendsWarnsOnly(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nrecommends: [x/nonexistent]\n---\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Errors()) != 0 {
		t.Fatalf("expected an unknown recommend to be a warning, not an error: %+v", report.Errors())
	}
	if len(report.Warnings()) == 0 {
		t.Fatal("expected a warning for an unknown recommended skill")
	}
}

func TestValidateDuplicateQualifiedNameWithinRootWarns(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, filepath.Join(dir, "automation", "path-one"), "dup", "---\nname: dup\ndomain: automation\ndescription: d1\n---\n")
	writeSkill(t, filepath.Join(dir, "automation", "path-two"), "dup", "---\nname: dup\ndomain: automation\ndescription: d2\n---\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, i := range report.Warnings() {
		if i.Skill == "automation/dup" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a duplicate-qualified-name warning, got %+v", report.Issues)
	}
}

func TestValidateSameBareNameDifferentDomainsDoesNotWarnAsDuplicate(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, filepath.Join(dir, "automation"), "modbus", "---\nname: modbus\ndomain: automation\ndescription: d1\n---\n")
	writeSkill(t, filepath.Join(dir, "networking"), "modbus", "---\nname: modbus\ndomain: networking\ndescription: d2\n---\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range report.Issues {
		if strings.Contains(i.Message, "duplicate") {
			t.Fatalf("did not expect a duplicate warning for legitimately domain-qualified same-name skills: %+v", report.Issues)
		}
	}
}

func TestValidateCycleErrors(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nrequires: [x/b]\n---\n")
	writeSkill(t, dir, "b", "---\nname: b\ndomain: x\ndescription: d\nrequires: [x/a]\n---\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Errors()) == 0 {
		t.Fatal("expected a cycle to be reported as an error")
	}
}

func TestValidateBadVersionWarns(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nversion: not-a-version\n---\n")
	report, err := Validate(dir, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, i := range report.Warnings() {
		if i.Skill == "a" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a bad-version warning, got %+v", report.Issues)
	}
}

func TestReportErrorsExcludesWarnings(t *testing.T) {
	r := Report{Issues: []Issue{{Skill: "a", Severity: SeverityWarning, Message: "w"}, {Skill: "b", Severity: SeverityError, Message: "e"}}}
	if len(r.Errors()) != 1 || len(r.Warnings()) != 1 {
		t.Fatalf("got errors=%v warnings=%v", r.Errors(), r.Warnings())
	}
}
