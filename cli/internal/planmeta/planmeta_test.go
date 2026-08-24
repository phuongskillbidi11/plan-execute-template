package planmeta

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Meta{
		Plan:       "2026-08-24-example",
		RiskLevel:  "feature",
		PlannedAt:  PlannedAt{GitSHA: "abc123"},
		State:      "PLANNED",
		WriteScope: []string{"src/api/**"},
	}
	if err := Save(dir, m); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlannedAt.GitSHA != "abc123" || got.State != "PLANNED" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestDefaultBudget(t *testing.T) {
	b := DefaultBudget()
	if b.Build != 2 || b.UnitTest != 2 || b.IntegrationTest != 1 {
		t.Fatalf("unexpected default budget: %+v", b)
	}
}

func TestLegacyStatusMigratesToState(t *testing.T) {
	dir := t.TempDir()
	// Simulates a Phase-2-created plan.yaml: has `status`, no `state`.
	os.WriteFile(filepath.Join(dir, FileName), []byte("plan: x\nstatus: executing\n"), 0o644)

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "EXECUTING" {
		t.Fatalf("expected EXECUTING, got %q", got.State)
	}
}

func TestNoStatusOrStateDefaultsToNew(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte("plan: x\n"), 0o644)

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "NEW" {
		t.Fatalf("expected NEW, got %q", got.State)
	}
}

func TestAppendEventWritesJSONLine(t *testing.T) {
	dir := t.TempDir()
	if err := AppendEvent(dir, "triaged", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := AppendEvent(dir, "approved", "alice"); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join(dir, EventsFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 event lines, got %d: %v", len(lines), lines)
	}
}
