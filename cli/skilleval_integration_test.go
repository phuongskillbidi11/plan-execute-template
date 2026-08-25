package main

import (
	"path/filepath"
	"testing"

	"eng/internal/skilleval"
	"eng/internal/skillrouter"
	"eng/internal/skills"
)

// TestRouterEvalScenarios runs every harness/evals/**/*.yaml scenario
// against the real, committed harness/skills tree — Phase 6 Requirement
// 17. It intentionally does not use a synthetic fixture: the point is to
// prove the router resolves the actual shipped skill set the way the
// instruction's own worked example describes.
func TestRouterEvalScenarios(t *testing.T) {
	skillsRoot := filepath.Join("..", "harness", "skills")
	evalsRoot := filepath.Join("..", "harness", "evals")

	all, err := skills.Resolve(skillsRoot, filepath.Join(t.TempDir(), "no-local-skills"))
	if err != nil {
		t.Fatal(err)
	}
	scenarios, err := skilleval.LoadScenarios(evalsRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenarios) == 0 {
		t.Fatal("expected at least one eval scenario under harness/evals/")
	}

	for _, sc := range scenarios {
		sc := sc
		t.Run(sc.Name, func(t *testing.T) {
			sel, err := skillrouter.Route(all, sc.Request, nil, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]bool{}
			for _, s := range sel.Skills {
				got[s.Name] = true
			}
			for _, want := range sc.ExpectedSkills {
				if !got[want] {
					t.Errorf("scenario %q: expected %q to be selected, got %v", sc.Name, want, skillNames(sel.Skills))
				}
			}
		})
	}
}

func skillNames(sel []skills.Skill) []string {
	out := make([]string, len(sel))
	for i, s := range sel {
		out[i] = s.Name
	}
	return out
}
