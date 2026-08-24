package hooks

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BeforePlan    []string          `yaml:"before_plan"`
	AfterPlan     []string          `yaml:"after_plan"`
	BeforeExecute []string          `yaml:"before_execute"`
	AfterTask     []string          `yaml:"after_task"`
	AfterExecute  []string          `yaml:"after_execute"`
	OnFailure     []string          `yaml:"on_failure"`
	Commands      map[string]string `yaml:"commands"`
}

// Load reads .agent/hooks.yaml if present, else globalDefaultPath. A
// project-local file fully replaces the global default — no partial merge.
func Load(projectDir, globalDefaultPath string) (*Config, error) {
	path := globalDefaultPath
	local := filepath.Join(projectDir, ".agent", "hooks.yaml")
	if _, err := os.Stat(local); err == nil {
		path = local
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Stage returns the ordered hook names for a lifecycle stage.
func (c *Config) Stage(name string) []string {
	switch name {
	case "before_plan":
		return c.BeforePlan
	case "after_plan":
		return c.AfterPlan
	case "before_execute":
		return c.BeforeExecute
	case "after_task":
		return c.AfterTask
	case "after_execute":
		return c.AfterExecute
	case "on_failure":
		return c.OnFailure
	default:
		return nil
	}
}
