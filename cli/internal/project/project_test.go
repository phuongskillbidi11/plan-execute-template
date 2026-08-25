package project

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"eng/internal/executil"
	"eng/internal/toolpolicy"
)

func TestDetectModeLegacy(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# x"), 0o644)
	os.Mkdir(filepath.Join(dir, ".plans"), 0o755)
	if got := DetectMode(dir); got != "legacy" {
		t.Fatalf("expected legacy, got %q", got)
	}
}

func TestDetectModeNone(t *testing.T) {
	dir := t.TempDir()
	if got := DetectMode(dir); got != "none" {
		t.Fatalf("expected none, got %q", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ProjectName: "x", Mode: "modern", Stack: Stack{Type: "go", Test: executil.Command{Shell: "go test ./..."}}}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "modern" || got.Stack.Type != "go" || got.Stack.Test.Shell != "go test ./..." {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestPlainStringStackCommandStillParses(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
	content := "project_name: x\nmode: modern\nstack:\n  type: go\n  build_cmd: \"go build ./...\"\n"
	os.WriteFile(filepath.Join(dir, ConfigPath), []byte(content), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Stack.Build.Shell != "go build ./..." {
		t.Fatalf("expected plain string to parse as Shell, got %+v", cfg.Stack.Build)
	}
}

func TestDetectModeResultBroken(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
	os.WriteFile(filepath.Join(dir, ConfigPath), []byte(": not valid yaml :: ["), 0o644)
	r := DetectModeResult(dir)
	if r.Mode != "broken" || r.ParseErr == nil {
		t.Fatalf("expected broken with a parse error, got %+v", r)
	}
}

func TestEffectiveWorkflowDefaultsAllTrue(t *testing.T) {
	cfg := &Config{}
	w := cfg.EffectiveWorkflow()
	if !w.Triage || !w.PlanReview || !w.Verifier {
		t.Fatalf("expected all-true default, got %+v", w)
	}
}

func TestEffectiveRetryBudgetDefault(t *testing.T) {
	cfg := &Config{}
	b := cfg.EffectiveRetryBudget()
	if b.Build != 2 || b.UnitTest != 2 || b.IntegrationTest != 1 {
		t.Fatalf("expected default budget, got %+v", b)
	}
}

func TestConfigVersionDefaultsToOneOnLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ProjectName: "x", Mode: "modern"}
	data, _ := yaml.Marshal(cfg) // simulates a Phase-1 file with no config_version field
	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
	os.WriteFile(filepath.Join(dir, ConfigPath), data, 0o644)

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigVersion != 1 {
		t.Fatalf("expected ConfigVersion=1 for a pre-Phase-2 file, got %d", got.ConfigVersion)
	}
}

func TestPlanningModeDefaultsToAutoPlan(t *testing.T) {
	w := Workflow{}
	if got := w.PlanningModeOrDefault(); got != "auto_plan" {
		t.Fatalf("expected auto_plan, got %q", got)
	}
}

func TestPlanningModeExplicitSpecFirst(t *testing.T) {
	w := Workflow{PlanningMode: "spec_first"}
	if got := w.PlanningModeOrDefault(); got != "spec_first" {
		t.Fatalf("expected spec_first, got %q", got)
	}
}

func TestRequireSpecApprovalDefaultsTrue(t *testing.T) {
	w := Workflow{}
	if !w.RequireSpecApprovalOrDefault() {
		t.Fatal("expected default true")
	}
}

func TestRequireSpecApprovalExplicitFalse(t *testing.T) {
	f := false
	w := Workflow{RequireSpecApproval: &f}
	if w.RequireSpecApprovalOrDefault() {
		t.Fatal("expected explicit false to be respected")
	}
}

func TestLegacyProjectYAMLWithoutPlanningModeStillLoads(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
	// Simulates a Phase 1-4 project.yaml: no planning_mode/require_spec_approval keys at all.
	content := "project_name: x\nmode: modern\nworkflow:\n  triage: true\n  plan_review: true\n  verifier: true\n"
	os.WriteFile(filepath.Join(dir, ConfigPath), []byte(content), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Workflow.PlanningModeOrDefault() != "auto_plan" {
		t.Fatalf("expected a pre-Phase-5 project.yaml to resolve to auto_plan, got %q", cfg.Workflow.PlanningModeOrDefault())
	}
}

func TestDomainsAndPrivateSkillsPathDefaultToEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ProjectName: "x", Mode: "modern"}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Domains) != 0 || got.PrivateSkillsPath != "" {
		t.Fatalf("expected both to default to empty, got %+v", got)
	}
}

func TestDomainsAndPrivateSkillsPathRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ProjectName: "x", Mode: "modern", Domains: []string{"embedded", "automation"}, PrivateSkillsPath: "../company-skills"}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Domains) != 2 || got.Domains[0] != "embedded" || got.Domains[1] != "automation" || got.PrivateSkillsPath != "../company-skills" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestToolsPolicyDefaultsToEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ProjectName: "x", Mode: "modern"}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tools.Allow) != 0 || len(got.Tools.Deny) != 0 || len(got.Tools.RequireApproval) != 0 {
		t.Fatalf("expected empty tool policy by default, got %+v", got.Tools)
	}
}

func TestToolsPolicyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ProjectName: "x", Mode: "modern", Tools: toolpolicy.Policy{
		Allow:           []string{"git.status"},
		RequireApproval: []string{"github.issue.comment"},
		Deny:            []string{"git.force_push"},
	}}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tools.Allow) != 1 || got.Tools.Allow[0] != "git.status" ||
		len(got.Tools.RequireApproval) != 1 || len(got.Tools.Deny) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got.Tools)
	}
}
