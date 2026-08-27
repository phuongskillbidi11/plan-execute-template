package skillrouter

import (
	"sort"
	"strings"

	"eng/internal/skillgraph"
	"eng/internal/skillmatch"
	"eng/internal/skills"
)

// Explanation is one line of "why was this skill selected" — the router's
// entire contribution to observability (Phase 6 Requirement 8).
type Explanation struct {
	Skill  string
	Reason string
}

// Selection is skillrouter.Route's full result: the skills to load, in a
// stable order, and a parallel explanation for each.
type Selection struct {
	Skills       []skills.Skill
	Explanations []Explanation
}

type entry struct {
	skill  skills.Skill
	reason string
}

// Route implements the Phase 6 spec.md Decision 5 precedence: explicit ->
// required (transitive) -> strong request matches -> domain-profile fills
// -> recommends -> budget cutoff, then one final forced pass that adds any
// still-missing required dependency of the FINAL selection regardless of
// budget. maxSkills <= 0 means no cap (used by strategy: full).
func Route(all []skills.Skill, request string, explicit []string, domains []string, maxSkills int) (Selection, error) {
	must := normalizeMustInclude(explicit)
	byQualified := map[string]skills.Skill{}
	requiredBy := map[string][]string{} // a required name -> requesters' qualified names
	for _, s := range all {
		byQualified[s.QualifiedName()] = s
	}
	for _, s := range all {
		for _, r := range s.Requires {
			requiredBy[r] = append(requiredBy[r], s.QualifiedName())
		}
	}

	selected := map[string]entry{}
	var order []string
	add := func(s skills.Skill, reason string) {
		qn := s.QualifiedName()
		if _, ok := selected[qn]; ok {
			return
		}
		selected[qn] = entry{s, reason}
		order = append(order, qn)
	}
	roomLeft := func() bool { return maxSkills <= 0 || len(selected) < maxSkills }

	// Tier A: explicit, deterministic order.
	var explicitSkills []skills.Skill
	for _, s := range all {
		if must[s.Name] || must[s.QualifiedName()] {
			explicitSkills = append(explicitSkills, s)
		}
	}
	sort.Slice(explicitSkills, func(i, j int) bool { return explicitSkills[i].QualifiedName() < explicitSkills[j].QualifiedName() })
	for _, s := range explicitSkills {
		add(s, "explicitly enabled")
	}

	// Tier A continued: transitive requires of every explicit skill — never budget-limited.
	closure, err := skillgraph.Expand(all, explicitSkills)
	if err != nil {
		return Selection{}, err
	}
	applyRequiredReasons(closure, requiredBy, add)

	// Tier B: strong request matches, best score first, alphabetical tie-break.
	type scored struct {
		skill skills.Skill
		score int
	}
	var candidates []scored
	for _, s := range all {
		if _, ok := selected[s.QualifiedName()]; ok {
			continue
		}
		if sc := skillmatch.Score(s, request); sc >= skillmatch.MinMatchScore {
			candidates = append(candidates, scored{s, sc})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].skill.QualifiedName() < candidates[j].skill.QualifiedName()
	})
	for _, c := range candidates {
		if !roomLeft() {
			break
		}
		add(c.skill, "matched request text")
	}

	// Tier C: domain/profile fills.
	if len(domains) > 0 {
		want := map[string]bool{}
		for _, d := range domains {
			want[d] = true
		}
		var domainSkills []skills.Skill
		for _, s := range all {
			if _, ok := selected[s.QualifiedName()]; ok {
				continue
			}
			if want[s.Domain] {
				domainSkills = append(domainSkills, s)
			}
		}
		sort.Slice(domainSkills, func(i, j int) bool { return domainSkills[i].QualifiedName() < domainSkills[j].QualifiedName() })
		for _, s := range domainSkills {
			if !roomLeft() {
				break
			}
			add(s, "project domain profile (\""+s.Domain+"\")")
		}
	}

	// Tier D: recommends, collected only from what's selected so far
	// (Tiers A/B/C — see Phase 6 spec.md Decision Log entry 2 for why this
	// doesn't cascade through the final forced-dependency pass below).
	recBy := map[string]string{}
	for _, qn := range order {
		for _, r := range selected[qn].skill.Recommends {
			if _, ok := recBy[r]; !ok {
				recBy[r] = qn
			}
		}
	}
	var recKeys []string
	for k := range recBy {
		recKeys = append(recKeys, k)
	}
	sort.Strings(recKeys)
	for _, k := range recKeys {
		if _, ok := selected[k]; ok {
			continue
		}
		s, ok := byQualified[k]
		if !ok {
			continue // an unresolved recommend is a validation warning, not a router error
		}
		if !roomLeft() {
			break
		}
		add(s, "recommended by "+recBy[k])
	}

	// Final pass: force in any still-missing required dependency of the
	// FINAL selection, ignoring the budget (Requirement 4/18).
	var finalSeed []skills.Skill
	for _, qn := range order {
		finalSeed = append(finalSeed, selected[qn].skill)
	}
	closure, err = skillgraph.Expand(all, finalSeed)
	if err != nil {
		return Selection{}, err
	}
	applyRequiredReasons(closure, requiredBy, add)

	out := Selection{}
	for _, qn := range order {
		e := selected[qn]
		out.Skills = append(out.Skills, e.skill)
		out.Explanations = append(out.Explanations, Explanation{Skill: e.skill.Name, Reason: e.reason})
	}
	return out, nil
}

// applyRequiredReasons adds every skill in closure via add, attributing a
// "required by X" reason when a direct requester is present within the
// same closure, or a generic fallback otherwise. add is a no-op for a
// skill that's already selected, so this never overwrites an existing
// reason (e.g. "explicitly enabled").
func applyRequiredReasons(closure []skills.Skill, requiredBy map[string][]string, add func(skills.Skill, string)) {
	inClosure := map[string]bool{}
	for _, s := range closure {
		inClosure[s.QualifiedName()] = true
	}
	for _, s := range closure {
		reason := "required dependency"
		for _, key := range []string{s.QualifiedName(), s.Name} {
			for _, requester := range requiredBy[key] {
				if inClosure[requester] {
					reason = "required by " + requester
					break
				}
			}
			if reason != "required dependency" {
				break
			}
		}
		add(s, reason)
	}
}

// normalizeMustInclude mirrors the Phase 4 enabled_skills gotcha fix
// (skillmatch.Select): an entry may be domain-qualified
// ("engineering/karpathy-guidelines") or bare ("karpathy-guidelines") —
// register both forms so either matches.
func normalizeMustInclude(names []string) map[string]bool {
	must := map[string]bool{}
	for _, name := range names {
		must[name] = true
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			must[name[idx+1:]] = true
		}
	}
	return must
}
