package planmeta

import "testing"

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Meta{
		Plan:       "2026-08-24-example",
		RiskLevel:  "feature",
		PlannedAt:  PlannedAt{GitSHA: "abc123"},
		Status:     "planned",
		WriteScope: []string{"src/api/**"},
	}
	if err := Save(dir, m); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlannedAt.GitSHA != "abc123" || got.RiskLevel != "feature" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestDefaultBudget(t *testing.T) {
	b := DefaultBudget()
	if b.Build != 2 || b.UnitTest != 2 || b.IntegrationTest != 1 {
		t.Fatalf("unexpected default budget: %+v", b)
	}
}
