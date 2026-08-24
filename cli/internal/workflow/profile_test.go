package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfile(t *testing.T) {
	harness := t.TempDir()
	os.MkdirAll(filepath.Join(harness, "workflows"), 0o755)
	os.WriteFile(filepath.Join(harness, "workflows", "feature.yaml"),
		[]byte("name: feature\nstages: [triage, plan, review, execute, verify]\n"), 0o644)

	p, err := LoadProfile(harness, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "feature" || len(p.Stages) != 5 {
		t.Fatalf("got %+v", p)
	}
}

func TestProfileForRiskLevel(t *testing.T) {
	cases := map[string]string{
		"quick-fix":    "quick-fix",
		"bug":          "bug-fix",
		"architecture": "architecture",
		"high-risk":    "high-risk",
		"feature":      "feature",
		"":             "feature",
	}
	for risk, want := range cases {
		if got := ProfileForRiskLevel(risk); got != want {
			t.Errorf("ProfileForRiskLevel(%q) = %q, want %q", risk, got, want)
		}
	}
}
