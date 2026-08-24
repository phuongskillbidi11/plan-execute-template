package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name         string   `yaml:"name"`
	Domain       string   `yaml:"domain"`
	Description  string   `yaml:"description"`
	Tags         []string `yaml:"tags"`
	Triggers     []string `yaml:"triggers"`
	Version      string   `yaml:"version"`
	Dependencies []string `yaml:"dependencies"`
	Conflicts    []string `yaml:"conflicts"`
	WhenToUse    string   `yaml:"when_to_use"`
	WhenNotToUse string   `yaml:"when_not_to_use"`
	Source       string   `yaml:"-"` // "global" or "local", set by Resolve
	Path         string   `yaml:"-"`
}

// ParseSkillFile reads one SKILL.md. It prefers YAML frontmatter; if none is
// present it falls back to the legacy "# Skill: name" + "## Purpose" convention
// used by scripts/update-manifest.sh, so pre-V2 project skills keep resolving.
func ParseSkillFile(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	content := string(data)

	if strings.HasPrefix(content, "---\n") {
		if end := strings.Index(content[4:], "\n---"); end >= 0 {
			var s Skill
			if err := yaml.Unmarshal([]byte(content[4:4+end]), &s); err == nil && s.Name != "" {
				s.Path = path
				return s, nil
			}
		}
	}

	return parseLegacy(content, path), nil
}

func parseLegacy(content, path string) Skill {
	var name, desc string
	inPurpose := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# Skill:"):
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "# Skill:"))
		case trimmed == "## Purpose":
			inPurpose = true
		case inPurpose && strings.HasPrefix(trimmed, "## "):
			inPurpose = false
		case inPurpose && trimmed != "" && desc == "":
			desc = trimmed
		}
	}
	return Skill{Name: name, Description: desc, Domain: "unknown", Path: path}
}

// Walk finds every SKILL.md under root and parses it. A missing root is not
// an error — it returns an empty slice.
func Walk(root string) ([]Skill, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}
	var out []Skill
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), "SKILL.md") {
			if s, parseErr := ParseSkillFile(path); parseErr == nil && s.Name != "" {
				out = append(out, s)
			}
		}
		return nil
	})
	return out, err
}

// Resolve merges global and project-local skills by name; local overrides
// global on a name collision.
func Resolve(globalRoot, localRoot string) ([]Skill, error) {
	global, err := Walk(globalRoot)
	if err != nil {
		return nil, err
	}
	for i := range global {
		global[i].Source = "global"
	}

	local, err := Walk(localRoot)
	if err != nil {
		return nil, err
	}
	for i := range local {
		local[i].Source = "local"
	}

	merged := map[string]Skill{}
	for _, s := range global {
		merged[s.Name] = s
	}
	for _, s := range local {
		merged[s.Name] = s
	}

	out := make([]Skill, 0, len(merged))
	for _, s := range merged {
		out = append(out, s)
	}
	return out, nil
}
