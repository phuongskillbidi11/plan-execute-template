package skilleval

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scenario is one router evaluation case — a small, deterministic
// foundation (Phase 6 Requirement 16), not an LLM benchmark: it only
// asserts which skills the router selects for a given request, never
// anything about model output.
type Scenario struct {
	Name           string   `yaml:"name"`
	Request        string   `yaml:"request"`
	ExpectedSkills []string `yaml:"expected_skills"`
	Notes          string   `yaml:"notes,omitempty"`
}

// LoadScenarios walks root for *.yaml files and parses each as one
// Scenario. A missing root is not an error — it returns an empty slice.
func LoadScenarios(root string) ([]Scenario, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}
	var out []Scenario
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var s Scenario
		if err := yaml.Unmarshal(data, &s); err != nil {
			return err
		}
		if s.Name == "" {
			s.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
		}
		out = append(out, s)
		return nil
	})
	return out, err
}
