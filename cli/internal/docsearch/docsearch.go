package docsearch

import (
	"os"
	"sort"
	"strings"
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

// Match returns sections whose title or body contains any word (len > 2)
// from request, ranked by match count, capped at maxDocs. maxDocs <= 0
// means "no cap" (used by strategy: full).
func Match(sections []Section, request string, maxDocs int) []Section {
	words := strings.Fields(strings.ToLower(request))
	type scored struct {
		section Section
		score   int
	}
	var list []scored
	for _, s := range sections {
		text := strings.ToLower(s.Title + " " + s.Body)
		score := 0
		for _, w := range words {
			if len(w) > 2 && strings.Contains(text, w) {
				score++
			}
		}
		if score > 0 {
			list = append(list, scored{s, score})
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
