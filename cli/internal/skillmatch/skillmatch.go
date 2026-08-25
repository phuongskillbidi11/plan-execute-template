package skillmatch

import (
	"sort"
	"strings"

	"eng/internal/skills"
)

// Score counts how many of a skill's tags/triggers/description words
// appear as substrings of the (lowercased) request text.
func Score(s skills.Skill, request string) int {
	text := strings.ToLower(request)
	score := 0
	for _, tag := range s.Tags {
		if tag != "" && strings.Contains(text, strings.ToLower(tag)) {
			score++
		}
	}
	for _, trig := range s.Triggers {
		if trig != "" && strings.Contains(text, strings.ToLower(trig)) {
			score++
		}
	}
	for _, word := range strings.Fields(strings.ToLower(s.Description)) {
		word = strings.Trim(word, ".,;:()")
		if len(word) > 3 && strings.Contains(text, word) {
			score++
		}
	}
	return score
}

// Select ranks resolved skills by Score against request, always keeps any
// skill named in mustInclude (a project's own enabled_skills — never
// silently dropped by this new filtering layer) regardless of maxSkills,
// and fills any remaining budget with the highest-scoring matches.
// maxSkills <= 0 means "no cap" (used by strategy: full).
func Select(all []skills.Skill, request string, mustInclude []string, maxSkills int) []skills.Skill {
	// enabled_skills entries may be domain-qualified (e.g.
	// "engineering/karpathy-guidelines", the exact form `eng init` writes)
	// while a resolved skill's Name is its bare frontmatter name
	// ("karpathy-guidelines"). Register both the full entry and its
	// basename so either form matches — otherwise the "always included"
	// guarantee would silently fail for every project.yaml eng init itself
	// creates.
	must := map[string]bool{}
	for _, name := range mustInclude {
		must[name] = true
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			must[name[idx+1:]] = true
		}
	}

	var required []skills.Skill
	type scored struct {
		skill skills.Skill
		score int
	}
	var candidates []scored
	for _, s := range all {
		if must[s.Name] {
			required = append(required, s)
			continue
		}
		if sc := Score(s, request); sc > 0 {
			candidates = append(candidates, scored{s, sc})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })

	out := append([]skills.Skill{}, required...)
	if maxSkills <= 0 {
		for _, c := range candidates {
			out = append(out, c.skill)
		}
		return out
	}
	budget := maxSkills - len(required)
	for _, c := range candidates {
		if budget <= 0 {
			break
		}
		out = append(out, c.skill)
		budget--
	}
	return out
}
