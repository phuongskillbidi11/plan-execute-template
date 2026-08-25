package skillgraph

import (
	"fmt"
	"sort"

	"eng/internal/skills"
)

// Expand returns the transitive closure of every seed skill's Requires
// edges over all (the full resolved skill set), plus the seed skills
// themselves, deduplicated, in deterministic order (alphabetical by
// QualifiedName). An unknown required skill name is a hard error — never
// silently dropped. A cycle is also a hard error, reporting the path that
// found it.
func Expand(all []skills.Skill, seed []skills.Skill) ([]skills.Skill, error) {
	byName := map[string]skills.Skill{}
	for _, s := range all {
		byName[s.QualifiedName()] = s
		byName[s.Name] = s // a requires: entry may use either form
	}

	included := map[string]skills.Skill{}
	var order []string
	visiting := map[string]bool{}

	var visit func(name string, path []string) error
	visit = func(name string, path []string) error {
		s, ok := byName[name]
		if !ok {
			return fmt.Errorf("unknown required skill %q (required by %s)", name, lastOr(path, "<seed>"))
		}
		qn := s.QualifiedName()
		if _, done := included[qn]; done {
			return nil
		}
		if visiting[qn] {
			return fmt.Errorf("dependency cycle detected: %s -> %s", joinPath(path), qn)
		}
		visiting[qn] = true
		reqs := append([]string{}, s.Requires...)
		sort.Strings(reqs)
		for _, r := range reqs {
			if err := visit(r, append(path, qn)); err != nil {
				return err
			}
		}
		visiting[qn] = false
		included[qn] = s
		order = append(order, qn)
		return nil
	}

	var seedNames []string
	for _, s := range seed {
		seedNames = append(seedNames, s.QualifiedName())
	}
	sort.Strings(seedNames)
	for _, n := range seedNames {
		if err := visit(n, nil); err != nil {
			return nil, err
		}
	}

	sort.Strings(order)
	out := make([]skills.Skill, 0, len(order))
	for _, n := range order {
		out = append(out, included[n])
	}
	return out, nil
}

func joinPath(path []string) string {
	if len(path) == 0 {
		return "<seed>"
	}
	out := path[0]
	for _, p := range path[1:] {
		out += " -> " + p
	}
	return out
}

func lastOr(path []string, fallback string) string {
	if len(path) == 0 {
		return fallback
	}
	return path[len(path)-1]
}
