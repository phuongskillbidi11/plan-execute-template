package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"eng/internal/executil"
	"eng/internal/toolpolicy"
)

type Stack struct {
	Type  string           `yaml:"type"`
	Build executil.Command `yaml:"build_cmd"`
	Test  executil.Command `yaml:"test_cmd"`
	Run   executil.Command `yaml:"run_cmd"`
	Lint  executil.Command `yaml:"lint_cmd"`
}

type Workflow struct {
	Triage     bool `yaml:"triage"`
	PlanReview bool `yaml:"plan_review"`
	Verifier   bool `yaml:"verifier"`

	// PlanningMode: "" (unset) | "auto_plan" | "spec_first". Empty means
	// "this project predates Phase 5, or never set it" — PlanningModeOrDefault
	// resolves that to "auto_plan", the exact behavior every plan created
	// under Phases 1-4 already has. Only a fresh `eng init` writes
	// "spec_first" explicitly for a brand-new project.
	PlanningMode string `yaml:"planning_mode,omitempty"`

	// RequireSpecApproval is a pointer so YAML can distinguish "not set"
	// (nil, defaults to true) from "explicitly false" — yaml.v3 handles
	// pointer fields natively, no custom unmarshaling needed here.
	RequireSpecApproval *bool `yaml:"require_spec_approval,omitempty"`
}

// enabled reports whether this Workflow struct was ever explicitly set.
// An all-false zero value means "the workflow block was absent" — callers
// treat that as "everything enabled" via EffectiveWorkflow.
func (w Workflow) enabled() bool {
	return w.Triage || w.PlanReview || w.Verifier
}

// PlanningModeOrDefault returns "auto_plan" when unset — the behavior
// every project.yaml written before Phase 5 already has.
func (w Workflow) PlanningModeOrDefault() string {
	if w.PlanningMode == "" {
		return "auto_plan"
	}
	return w.PlanningMode
}

// RequireSpecApprovalOrDefault returns true when unset.
func (w Workflow) RequireSpecApprovalOrDefault() bool {
	if w.RequireSpecApproval == nil {
		return true
	}
	return *w.RequireSpecApproval
}

type RetryBudget struct {
	Build           int `yaml:"build"`
	UnitTest        int `yaml:"unit_test"`
	IntegrationTest int `yaml:"integration_test"`
}

func (b RetryBudget) isZero() bool {
	return b.Build == 0 && b.UnitTest == 0 && b.IntegrationTest == 0
}

type Config struct {
	ProjectName     string      `yaml:"project_name"`
	Mode            string      `yaml:"mode"` // legacy | hybrid | modern
	HarnessProfile  string      `yaml:"harness_profile"`
	ConfigVersion   int         `yaml:"config_version"`
	Stack           Stack       `yaml:"stack"`
	EnabledSkills   []string    `yaml:"enabled_skills"`
	Workflow        Workflow    `yaml:"workflow,omitempty"`
	RetryBudget     RetryBudget `yaml:"retry_budget,omitempty"`
	RequireApproval []string    `yaml:"require_approval,omitempty"`

	// Domains is the Phase 6 domain-profile list (e.g. ["embedded",
	// "automation"]) used by the skill router's domain-profile fill tier.
	// Deliberately separate from HarnessProfile (a singular, unused-since-
	// eng-init field from V1) — see Phase 6 spec.md Decision 7.
	Domains []string `yaml:"domains,omitempty"`

	// PrivateSkillsPath, if set, is resolved relative to the project root
	// (or used as-is if absolute) as an extra skill root between global and
	// local precedence. Empty (the default for every existing
	// project.yaml) skips the private tier entirely — see Phase 6 spec.md
	// Decision 8.
	PrivateSkillsPath string `yaml:"private_skills_path,omitempty"`

	// Tools is the Phase 7 project-level tool policy (allow/require_approval/
	// deny, by capability name). Deliberately not a reuse of the
	// pre-existing, unread RequireApproval field above — see Phase 7
	// spec.md Decision 2.
	Tools toolpolicy.Policy `yaml:"tools,omitempty"`
}

// EffectiveWorkflow returns the configured Workflow, or all-enabled if this
// project.yaml predates Phase 2 (no workflow block at all).
func (c *Config) EffectiveWorkflow() Workflow {
	if c.Workflow.enabled() {
		return c.Workflow
	}
	return Workflow{Triage: true, PlanReview: true, Verifier: true}
}

// EffectiveRetryBudget returns the configured budget, or Phase 2's default
// if the project.yaml doesn't declare one.
func (c *Config) EffectiveRetryBudget() RetryBudget {
	if !c.RetryBudget.isZero() {
		return c.RetryBudget
	}
	return RetryBudget{Build: 2, UnitTest: 2, IntegrationTest: 1}
}

const ConfigPath = ".agent/project.yaml"

func Load(dir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, ConfigPath))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ConfigPath, err)
	}
	if cfg.ConfigVersion == 0 {
		cfg.ConfigVersion = 1 // Phase 1 files predate this field
	}
	return &cfg, nil
}

func Save(dir string, cfg *Config) error {
	if cfg.ConfigVersion == 0 {
		cfg.ConfigVersion = 2
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	full := filepath.Join(dir, ConfigPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

// ModeResult distinguishes "no .agent/ at all" from "project.yaml exists but
// is broken" — the Phase 1 DetectMode collapsed these into "hybrid" silently.
type ModeResult struct {
	Mode     string // legacy | hybrid | modern | none | broken
	ParseErr error  // non-nil only when Mode == "broken"
}

func DetectModeResult(dir string) ModeResult {
	if _, err := os.Stat(filepath.Join(dir, ConfigPath)); err == nil {
		cfg, loadErr := Load(dir)
		if loadErr != nil {
			return ModeResult{Mode: "broken", ParseErr: loadErr}
		}
		if cfg.Mode != "" {
			return ModeResult{Mode: cfg.Mode}
		}
		return ModeResult{Mode: "hybrid"}
	}
	_, claudeErr := os.Stat(filepath.Join(dir, "CLAUDE.md"))
	_, plansErr := os.Stat(filepath.Join(dir, ".plans"))
	if claudeErr == nil || plansErr == nil {
		return ModeResult{Mode: "legacy"}
	}
	return ModeResult{Mode: "none"}
}

// DetectMode is the Phase 1 string-only API, kept for existing callers
// (cmdInit's flow never reaches the "broken" case — it stats ConfigPath
// itself first and returns early when the file already exists).
func DetectMode(dir string) string {
	r := DetectModeResult(dir)
	if r.Mode == "broken" {
		return "hybrid"
	}
	return r.Mode
}
