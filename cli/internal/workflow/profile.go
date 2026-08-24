package workflow

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile is a named, ordered, descriptive list of stages a plan's risk
// level routes through. It is informational (used by `eng workflow status`
// to show the human the whole path) — the authoritative gating logic is
// Decide's transition table, not this file.
type Profile struct {
	Name   string   `yaml:"name"`
	Stages []string `yaml:"stages"`
}

func LoadProfile(harnessDir, name string) (*Profile, error) {
	path := filepath.Join(harnessDir, "workflows", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ProfileForRiskLevel maps a risk_level (per core/triage/METHOD.md) to its
// workflow profile's file name under harness/workflows/.
func ProfileForRiskLevel(riskLevel string) string {
	switch riskLevel {
	case "quick-fix":
		return "quick-fix"
	case "bug":
		return "bug-fix"
	case "architecture":
		return "architecture"
	case "high-risk":
		return "high-risk"
	default:
		return "feature"
	}
}
