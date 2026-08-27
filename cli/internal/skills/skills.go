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
	Level        string   `yaml:"level"` // "" | engineering | domain | technology
	Requires     []string `yaml:"requires"`
	Recommends   []string `yaml:"recommends"`
	Capabilities []string `yaml:"capabilities"`
	Conflicts    []string `yaml:"conflicts"`
	WhenToUse    string   `yaml:"when_to_use"`
	WhenNotToUse string   `yaml:"when_not_to_use"`
	Source       string   `yaml:"-"` // "global", "private", or "local" — set by Resolve
	Path         string   `yaml:"-"`
}

// QualifiedName is a skill's identity for merge/collision purposes:
// domain-qualified when Domain is set and Name doesn't already contain a
// "/", unchanged otherwise. This covers both a self-namespaced
// "company/internal-api"-style name and every legacy skill (Domain is the
// literal "unknown" from parseLegacy) — legacy skills keep
// merging/overriding by bare Name exactly as before Phase 6. See Phase 6
// spec.md Decision 3.
func (s Skill) QualifiedName() string {
	if s.Domain == "" || s.Domain == "unknown" || strings.Contains(s.Name, "/") {
		return s.Name
	}
	return s.Domain + "/" + s.Name
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

// Resolve merges global and project-local skills by QualifiedName; local
// overrides global on a collision. Equivalent to
// ResolveWithPrivate(globalRoot, "", localRoot) — kept as its own function
// since it's the exact two-tier shape every pre-Phase-6 caller and test
// expects.
func Resolve(globalRoot, localRoot string) ([]Skill, error) {
	return ResolveWithPrivate(globalRoot, "", localRoot)
}

// ResolveWithPrivate merges up to three tiers by QualifiedName, in
// increasing precedence: global < private < local. An empty privateRoot
// skips that tier entirely (see Phase 6 spec.md Decision 3 for why there
// are three tiers, not the four the instruction first proposed).
func ResolveWithPrivate(globalRoot, privateRoot, localRoot string) ([]Skill, error) {
	merged := map[string]Skill{}

	tiers := []struct {
		root   string
		source string
	}{
		{globalRoot, "global"},
		{privateRoot, "private"},
		{localRoot, "local"},
	}
	for _, t := range tiers {
		if t.root == "" {
			continue
		}
		found, err := Walk(t.root)
		if err != nil {
			return nil, err
		}
		for _, s := range found {
			s.Source = t.source
			merged[s.QualifiedName()] = s
		}
	}

	collapseLegacyDuplicates(merged)

	out := make([]Skill, 0, len(merged))
	for _, s := range merged {
		out = append(out, s)
	}
	return out, nil
}

// isLegacy reports whether s came from a frontmatter-less SKILL.md (the
// "# Skill: name" / "## Purpose" fallback convention) — its Domain is
// always "" or "unknown" in that case, which is also what makes its
// QualifiedName() collapse to its bare Name.
func isLegacy(s Skill) bool {
	return s.Domain == "" || s.Domain == "unknown"
}

// bareName strips a QualifiedName's domain prefix, if any.
func bareName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// collapseLegacyDuplicates removes a legacy (frontmatter-less) skill
// whenever it shares a bare name with an already-merged qualified skill —
// the same conceptual skill re-declared without frontmatter (e.g. a
// project-local leftover from before it adopted the harness), not a
// deliberate cross-domain reuse of a short name. A group made entirely of
// qualified skills (real, distinct-domain skills that happen to share a
// bare name — e.g. a future networking/modbus alongside automation/modbus,
// per docs/skills.md) is left untouched: collapsing that would be a real
// regression, not a fix. Within a collapsed group, the highest-precedence
// tier survives (local > private > global — the same precedence the
// per-key merge above already applies via map overwrite order). See Phase
// 9 spec.md P2-3 and DECISION_LOG.md Decision 6.
func collapseLegacyDuplicates(merged map[string]Skill) {
	tierRank := map[string]int{"global": 0, "private": 1, "local": 2}

	byBare := map[string][]string{}
	for qn, s := range merged {
		byBare[bareName(s.Name)] = append(byBare[bareName(s.Name)], qn)
	}

	for _, qns := range byBare {
		if len(qns) < 2 {
			continue
		}
		hasLegacy, hasQualified := false, false
		for _, qn := range qns {
			if isLegacy(merged[qn]) {
				hasLegacy = true
			} else {
				hasQualified = true
			}
		}
		if !hasLegacy || !hasQualified {
			continue // all-qualified: genuinely distinct cross-domain skills — never collapse
		}

		best := qns[0]
		for _, qn := range qns[1:] {
			if tierRank[merged[qn].Source] > tierRank[merged[best].Source] {
				best = qn
			}
		}
		for _, qn := range qns {
			if qn != best {
				delete(merged, qn)
			}
		}
	}
}
