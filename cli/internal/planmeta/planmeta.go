package planmeta

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type RetryCounters struct {
	Build           int `yaml:"build"`
	UnitTest        int `yaml:"unit_test"`
	IntegrationTest int `yaml:"integration_test"`
}

type RetryBudget struct {
	Build           int `yaml:"build"`
	UnitTest        int `yaml:"unit_test"`
	IntegrationTest int `yaml:"integration_test"`
}

type PlannedAt struct {
	GitSHA string `yaml:"git_sha"`
}

type Meta struct {
	Plan        string        `yaml:"plan"`
	RiskLevel   string        `yaml:"risk_level"` // quick-fix | bug | feature | architecture | high-risk
	PlannedAt   PlannedAt     `yaml:"planned_at"`
	Status      string        `yaml:"status"` // planned | reviewed | executing | verified | failed
	WriteScope  []string      `yaml:"write_scope"`
	Retry       RetryCounters `yaml:"retry"`
	RetryBudget RetryBudget   `yaml:"retry_budget"`
}

const FileName = "plan.yaml"

func Load(planDir string) (*Meta, error) {
	data, err := os.ReadFile(filepath.Join(planDir, FileName))
	if err != nil {
		return nil, err
	}
	var m Meta
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func Save(planDir string, m *Meta) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(planDir, FileName), data, 0o644)
}

// DefaultBudget is used when neither plan.yaml nor .agent/project.yaml
// declares a retry_budget.
func DefaultBudget() RetryBudget {
	return RetryBudget{Build: 2, UnitTest: 2, IntegrationTest: 1}
}
