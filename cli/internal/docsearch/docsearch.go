package docsearch

import (
	"os"
	"sort"
	"strings"
	"unicode"
)

type Section struct {
	Title string
	Body  string
}

// ParseSections splits a markdown file on "### " headers — the convention
// both docs/src-map.md and docs/gotchas.md already use for one module/
// gotcha per section.
func ParseSections(path string) ([]Section, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var sections []Section
	var cur *Section
	for _, line := range lines {
		if strings.HasPrefix(line, "### ") {
			if cur != nil {
				sections = append(sections, *cur)
			}
			cur = &Section{Title: strings.TrimPrefix(line, "### ")}
			continue
		}
		if cur != nil {
			cur.Body += line + "\n"
		}
	}
	if cur != nil {
		sections = append(sections, *cur)
	}
	return sections, nil
}

// TitleWordWeight is how much a single (deduplicated) title-word match
// contributes to a section's score — a section's own heading is a strong,
// curated relevance signal, so one hit alone is enough to clear
// MinMatchScore.
const TitleWordWeight = 2

// BodyWordWeight is how much a single (deduplicated) body-word match
// contributes — prose, so one hit alone must not be enough on its own.
const BodyWordWeight = 1

// MinMatchScore is the minimum total score a section needs before Match
// includes it. Calibrated so a single title-word hit clears it on its own,
// but a single body-word hit does not.
const MinMatchScore = 2

// minWordLen is the same length cutoff docsearch always used (keep words
// longer than 2 characters). Word-boundary tokenizing (below) already
// removes substring-only false positives; what a length filter alone could
// never fix is a genuine whole-word match on generic vocabulary shared by
// several sections — that's what TitleWordWeight/BodyWordWeight/
// MinMatchScore below are for. See Phase 9 spec.md P1-2.
const minWordLen = 2

// symbolLanguageNames mirrors skillmatch's normalization exactly — see its
// comment for why "C++"/"C#" need this before tokenizing.
var symbolLanguageNames = strings.NewReplacer(
	"c++", " cpp ",
	"c#", " csharp ",
)

func tokenize(s string) []string {
	normalized := symbolLanguageNames.Replace(strings.ToLower(s))
	return strings.FieldsFunc(normalized, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

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

// uniqueWords returns the distinct, length-filtered words of s — a word
// repeated several times in one section's body must not be counted more
// than once.
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

func score(s Section, request string) int {
	textTokens := tokenize(request)
	sc := 0
	for _, w := range uniqueWords(s.Title, minWordLen) {
		if containsPhrase(textTokens, w) {
			sc += TitleWordWeight
		}
	}
	for _, w := range uniqueWords(s.Body, minWordLen) {
		if containsPhrase(textTokens, w) {
			sc += BodyWordWeight
		}
	}
	return sc
}

// Match returns sections whose weighted score clears MinMatchScore, ranked
// by score, capped at maxDocs. maxDocs <= 0 means "no cap" (used by
// strategy: full). A title-word match is weighted higher than a body-word
// match — see Phase 9 spec.md P1-2: the old uniform, threshold-free scoring
// materially over-matched (4 of 6 sections selected where only 1 was
// relevant, in a real reproduced case).
func Match(sections []Section, request string, maxDocs int) []Section {
	type scored struct {
		section Section
		score   int
	}
	var list []scored
	for _, s := range sections {
		if sc := score(s, request); sc >= MinMatchScore {
			list = append(list, scored{s, sc})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	if maxDocs > 0 && len(list) > maxDocs {
		list = list[:maxDocs]
	}
	out := make([]Section, len(list))
	for i, s := range list {
		out[i] = s.section
	}
	return out
}
