package project

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Stack struct {
	Type  string `yaml:"type"`
	Build string `yaml:"build_cmd"`
	Test  string `yaml:"test_cmd"`
	Run   string `yaml:"run_cmd"`
	Lint  string `yaml:"lint_cmd"`
}

type Config struct {
	ProjectName    string   `yaml:"project_name"`
	Mode           string   `yaml:"mode"` // legacy | hybrid | modern
	HarnessProfile string   `yaml:"harness_profile"`
	Stack          Stack    `yaml:"stack"`
	EnabledSkills  []string `yaml:"enabled_skills"`
}

const ConfigPath = ".agent/project.yaml"

func Load(dir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, ConfigPath))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(dir string, cfg *Config) error {
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

// DetectMode reports the project's mode without requiring .agent/ to exist:
//   - .agent/project.yaml present -> its own "mode" field (default "hybrid" if unset)
//   - CLAUDE.md or .plans/ present, no .agent/ -> "legacy"
//   - neither -> "none" (not yet initialized)
func DetectMode(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, ConfigPath)); err == nil {
		if cfg, loadErr := Load(dir); loadErr == nil && cfg.Mode != "" {
			return cfg.Mode
		}
		return "hybrid"
	}
	_, claudeErr := os.Stat(filepath.Join(dir, "CLAUDE.md"))
	_, plansErr := os.Stat(filepath.Join(dir, ".plans"))
	if claudeErr == nil || plansErr == nil {
		return "legacy"
	}
	return "none"
}
