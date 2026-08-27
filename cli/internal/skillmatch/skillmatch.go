package skillmatch

import (
	"sort"
	"strings"
	"unicode"

	"eng/internal/skills"
)

// TagTriggerWeight is how much a single tag or trigger match contributes to
// a skill's score — curated, deliberately-chosen signals, so one hit alone
// is enough to clear MinMatchScore.
const TagTriggerWeight = 3

// DescriptionWordWeight is how much a single (deduplicated) description-word
// match contributes — prose, not a curated signal, so one hit alone must
// NOT be enough to clear MinMatchScore on its own (Phase 9 spec.md P2-2: a
// generic word shared between a request and an unrelated skill's prose
// description was causing real false positives).
const DescriptionWordWeight = 1

// MinMatchScore is the minimum total score a skill needs before a Tier-B
// "matched request text" inclusion is allowed. Calibrated so a single
// tag/trigger hit clears it on its own, but a single description-word hit
// does not.
const MinMatchScore = 2

// minDescriptionWordLen is the same length cutoff skillmatch always used
// (keep words longer than 3 characters). It is deliberately NOT raised
// further: word-boundary tokenizing (below) is what stops a word like
// "form" from matching inside an unrelated request word like "WinForms" —
// that was never actually a length problem. What length filtering alone
// could never fix is a genuine whole-word collision on generic vocabulary
// (e.g. "protocol", "framing") — that's what DescriptionWordWeight/
// MinMatchScore below are for.
const minDescriptionWordLen = 3

// symbolLanguageNames normalizes symbol-only language names to a stable
// word BEFORE tokenizing — without this, "C++" and "C#" both degenerate to
// the same bare "c" token once punctuation is stripped (their only letter),
// falsely colliding two unrelated languages. Found during Phase 9
// dogfooding reproduction (a C# WinForms request matching the C++ skill) —
// see spec.md P2-2.
var symbolLanguageNames = strings.NewReplacer(
	"c++", " cpp ",
	"c#", " csharp ",
)

// tokenize splits s into lowercase letter/digit runs — the same boundary
// rule used for both a skill's signals and the request text, so matching is
// always a whole-token (or whole-phrase) comparison, never a raw substring
// check. This is what stops a description word like "form" from matching
// inside an unrelated request word like "WinForms".
func tokenize(s string) []string {
	normalized := symbolLanguageNames.Replace(strings.ToLower(s))
	return strings.FieldsFunc(normalized, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// containsPhrase reports whether needle's tokens appear as a contiguous
// subsequence of textTokens — correct for both single-word needles ("tcp")
// and multi-word/hyphenated ones ("modbus tcp", "s7-1200").
func containsPhrase(textTokens []string, needle string) bool {
	needleTokens := tokenize(needle)
	if len(needleTokens) == 0 {
		return false
	}
	for i := 0; i+len(needleTokens) <= len(textTokens); i++ {
		match := true
		for j, t := range needleTokens {
			if textTokens[i+j] != t {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// uniqueWords returns the distinct, length-filtered words of s — used for
// description scoring so a word repeated several times in one skill's prose
// (e.g. "framing" appearing twice in automation/modbus's description) is
// never counted more than once.
func uniqueWords(s string, minLen int) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range tokenize(s) {
		if len(w) <= minLen || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}

// Score is a weighted sum, not a raw match count: a tag/trigger match is
// worth TagTriggerWeight, a deduplicated description-word match is worth
// DescriptionWordWeight. Callers decide inclusion by comparing against
// MinMatchScore, not merely score > 0 — see Phase 9 spec.md P1-2/P2-2 for
// the two real false-positive classes this fixes (a substring match with no
// word boundary, and a single generic-vocabulary word treated as a strong
// signal).
func Score(s skills.Skill, request string) int {
	textTokens := tokenize(request)
	score := 0
	for _, tag := range s.Tags {
		if tag != "" && containsPhrase(textTokens, tag) {
			score += TagTriggerWeight
		}
	}
	for _, trig := range s.Triggers {
		if trig != "" && containsPhrase(textTokens, trig) {
			score += TagTriggerWeight
		}
	}
	for _, word := range uniqueWords(s.Description, minDescriptionWordLen) {
		if containsPhrase(textTokens, word) {
			score += DescriptionWordWeight
		}
	}
	return score
}

// Select ranks resolved skills by Score against request, always keeps any
// skill named in mustInclude (a project's own enabled_skills — never
// silently dropped by this filtering layer) regardless of maxSkills, and
// fills any remaining budget with matches whose Score clears MinMatchScore,
// highest first. maxSkills <= 0 means "no cap" (used by strategy: full).
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
		if sc := Score(s, request); sc >= MinMatchScore {
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
