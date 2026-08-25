package skillvalidate

import (
	"fmt"
	"regexp"
	"sort"

	"eng/internal/skillgraph"
	"eng/internal/skills"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Skill    string
	Severity Severity
	Message  string
}

type Report struct {
	Discovered int
	Issues     []Issue
}

func (r Report) Errors() []Issue {
	var out []Issue
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			out = append(out, i)
		}
	}
	return out
}

func (r Report) Warnings() []Issue {
	var out []Issue
	for _, i := range r.Issues {
		if i.Severity == SeverityWarning {
			out = append(out, i)
		}
	}
	return out
}

var versionRe = regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)

// Validate walks each non-empty root separately (to catch a genuine
// duplicate — two files under the SAME root resolving to the SAME
// QualifiedName — distinct from the supported cross-domain bare-name
// reuse pattern, and distinct from the expected global/private/local
// override) and also validates the fully merged set for cross-skill
// issues (unknown requires/recommends/conflicts, dependency cycles).
func Validate(globalRoot, privateRoot, localRoot string) (Report, error) {
	merged, err := skills.ResolveWithPrivate(globalRoot, privateRoot, localRoot)
	if err != nil {
		return Report{}, err
	}
	report := Report{Discovered: len(merged)}

	byQualified := map[string]skills.Skill{}
	for _, s := range merged {
		byQualified[s.QualifiedName()] = s
	}

	for _, root := range []string{globalRoot, privateRoot, localRoot} {
		if root == "" {
			continue
		}
		found, err := skills.Walk(root)
		if err != nil {
			return Report{}, err
		}
		counts := map[string]int{}
		for _, s := range found {
			counts[s.QualifiedName()]++
		}
		var dup []string
		for qn, count := range counts {
			if count > 1 {
				dup = append(dup, qn)
			}
		}
		sort.Strings(dup)
		for _, qn := range dup {
			report.Issues = append(report.Issues, Issue{qn, SeverityWarning, fmt.Sprintf("duplicate skill %q authored more than once under %s", qn, root)})
		}
	}

	for _, s := range merged {
		isLegacy := s.Domain == "" || s.Domain == "unknown"
		if isLegacy {
			report.Issues = append(report.Issues, Issue{s.Name, SeverityWarning, "legacy skill — no frontmatter metadata"})
		} else if s.Description == "" {
			report.Issues = append(report.Issues, Issue{s.Name, SeverityWarning, "missing description"})
		}

		for _, r := range s.Requires {
			if !nameExists(byQualified, merged, r) {
				report.Issues = append(report.Issues, Issue{s.Name, SeverityError, fmt.Sprintf("requires unknown skill %q", r)})
			}
		}
		for _, r := range s.Recommends {
			if !nameExists(byQualified, merged, r) {
				report.Issues = append(report.Issues, Issue{s.Name, SeverityWarning, fmt.Sprintf("recommends unknown skill %q", r)})
			}
		}
		for _, c := range s.Conflicts {
			if !nameExists(byQualified, merged, c) {
				report.Issues = append(report.Issues, Issue{s.Name, SeverityWarning, fmt.Sprintf("conflicts with unknown skill %q", c)})
			}
		}

		if s.Version != "" && !versionRe.MatchString(s.Version) {
			report.Issues = append(report.Issues, Issue{s.Name, SeverityWarning, fmt.Sprintf("version %q doesn't look like a version number", s.Version)})
		}
	}

	if _, err := skillgraph.Expand(merged, merged); err != nil {
		report.Issues = append(report.Issues, Issue{"(graph)", SeverityError, err.Error()})
	}

	sort.SliceStable(report.Issues, func(i, j int) bool { return report.Issues[i].Skill < report.Issues[j].Skill })
	return report, nil
}

func nameExists(byQualified map[string]skills.Skill, all []skills.Skill, name string) bool {
	if _, ok := byQualified[name]; ok {
		return true
	}
	for _, s := range all {
		if s.Name == name {
			return true
		}
	}
	return false
}
