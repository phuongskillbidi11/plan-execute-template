# Tasks: V2 Harness Phase 4 (Context Engineering)

Each task must be completed and its test (see `tests.md`) must pass before moving to the
next. Mark `[x]` when done. Read `spec.md` in full — especially "Design decisions" — before
starting Task 1.

**Prerequisite:** Go 1.22+, and Phase 1/2/3's completed `cli/`/`harness/` trees (already
committed as of this plan).

---

## Task 1 — Context budget config (`internal/contextcfg`)

- [x] **1.1** Create `cli/internal/contextcfg/contextcfg.go`:

  ```go
  package contextcfg

  import (
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"
  )

  // Config is the effective, fully-resolved context budget — never has nil
  // fields once returned from Load or Default.
  type Config struct {
  	Strategy              string // full | selective
  	MaxSkills             int
  	MaxDocs               int
  	MaxLogLines           int
  	IncludeCompletedTasks bool
  	SummarizeToolOutput   bool
  }

  // override mirrors Config but with pointer fields, so YAML unmarshal can
  // distinguish "this key was absent" (nil) from "this key was explicitly
  // false/zero" — plain bool/int fields can't make that distinction, which
  // would otherwise silently reset an unspecified field to its zero value
  // every time any override file is loaded.
  type override struct {
  	Strategy              *string `yaml:"strategy"`
  	MaxSkills             *int    `yaml:"max_skills"`
  	MaxDocs               *int    `yaml:"max_docs"`
  	MaxLogLines           *int    `yaml:"max_log_lines"`
  	IncludeCompletedTasks *bool   `yaml:"include_completed_tasks"`
  	SummarizeToolOutput   *bool   `yaml:"summarize_tool_output"`
  }

  func Default() Config {
  	return Config{
  		Strategy:              "selective",
  		MaxSkills:             5,
  		MaxDocs:               8,
  		MaxLogLines:           300,
  		IncludeCompletedTasks: false,
  		SummarizeToolOutput:   true,
  	}
  }

  // Load reads .agent/context.yaml if present, else globalDefaultPath, else
  // returns Default() unchanged — a project with no context config at all
  // (the common case, per Requirement 15) works with zero extra files.
  func Load(projectDir, globalDefaultPath string) (Config, error) {
  	cfg := Default()

  	path := globalDefaultPath
  	local := filepath.Join(projectDir, ".agent", "context.yaml")
  	if _, err := os.Stat(local); err == nil {
  		path = local
  	}

  	data, err := os.ReadFile(path)
  	if err != nil {
  		if os.IsNotExist(err) {
  			return cfg, nil
  		}
  		return cfg, err
  	}

  	var o override
  	if err := yaml.Unmarshal(data, &o); err != nil {
  		return cfg, err
  	}
  	if o.Strategy != nil {
  		cfg.Strategy = *o.Strategy
  	}
  	if o.MaxSkills != nil {
  		cfg.MaxSkills = *o.MaxSkills
  	}
  	if o.MaxDocs != nil {
  		cfg.MaxDocs = *o.MaxDocs
  	}
  	if o.MaxLogLines != nil {
  		cfg.MaxLogLines = *o.MaxLogLines
  	}
  	if o.IncludeCompletedTasks != nil {
  		cfg.IncludeCompletedTasks = *o.IncludeCompletedTasks
  	}
  	if o.SummarizeToolOutput != nil {
  		cfg.SummarizeToolOutput = *o.SummarizeToolOutput
  	}
  	return cfg, nil
  }
  ```

- [x] **1.2** Create `cli/internal/contextcfg/contextcfg_test.go`:

  ```go
  package contextcfg

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func TestLoadMissingEverythingReturnsDefault(t *testing.T) {
  	cfg, err := Load(t.TempDir(), filepath.Join(t.TempDir(), "nope.yaml"))
  	if err != nil {
  		t.Fatal(err)
  	}
  	if cfg != Default() {
  		t.Fatalf("expected Default(), got %+v", cfg)
  	}
  }

  func TestProjectOverrideReplacesGlobal(t *testing.T) {
  	global := filepath.Join(t.TempDir(), "default.yaml")
  	os.WriteFile(global, []byte("max_skills: 3\n"), 0o644)

  	project := t.TempDir()
  	os.MkdirAll(filepath.Join(project, ".agent"), 0o755)
  	os.WriteFile(filepath.Join(project, ".agent", "context.yaml"), []byte("max_skills: 9\n"), 0o644)

  	cfg, err := Load(project, global)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if cfg.MaxSkills != 9 {
  		t.Fatalf("expected project override (9), got %d", cfg.MaxSkills)
  	}
  }

  func TestUnsetBoolDoesNotResetToFalse(t *testing.T) {
  	global := filepath.Join(t.TempDir(), "default.yaml")
  	// summarize_tool_output is NOT mentioned — must stay at Default()'s true.
  	os.WriteFile(global, []byte("max_skills: 2\n"), 0o644)

  	cfg, err := Load(t.TempDir(), global)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !cfg.SummarizeToolOutput {
  		t.Fatal("expected SummarizeToolOutput to remain true (Default), got false")
  	}
  }

  func TestExplicitFalseIsRespected(t *testing.T) {
  	global := filepath.Join(t.TempDir(), "default.yaml")
  	os.WriteFile(global, []byte("summarize_tool_output: false\n"), 0o644)

  	cfg, err := Load(t.TempDir(), global)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if cfg.SummarizeToolOutput {
  		t.Fatal("expected SummarizeToolOutput to be explicitly false")
  	}
  }
  ```

---

## Task 2 — Skill matching (`internal/skillmatch`)

- [x] **2.1** Create `cli/internal/skillmatch/skillmatch.go`:

  ```go
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
  	must := map[string]bool{}
  	for _, name := range mustInclude {
  		must[name] = true
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
  ```

- [x] **2.2** Create `cli/internal/skillmatch/skillmatch_test.go`:

  ```go
  package skillmatch

  import (
  	"testing"

  	"eng/internal/skills"
  )

  func TestScoreCountsTagMatches(t *testing.T) {
  	s := skills.Skill{Tags: []string{"planning", "methodology"}, Description: "x"}
  	if got := Score(s, "I need help with planning"); got < 1 {
  		t.Fatalf("expected at least 1, got %d", got)
  	}
  }

  func TestScoreZeroForNoMatch(t *testing.T) {
  	s := skills.Skill{Tags: []string{"modbus"}, Description: "industrial protocol"}
  	if got := Score(s, "add a login page"); got != 0 {
  		t.Fatalf("expected 0, got %d", got)
  	}
  }

  func TestSelectRespectsCapAndRanking(t *testing.T) {
  	all := []skills.Skill{
  		{Name: "a", Tags: []string{"web"}},
  		{Name: "b", Tags: []string{"web", "api"}},
  		{Name: "c", Tags: []string{"database"}},
  	}
  	selected := Select(all, "build a web api", nil, 1)
  	if len(selected) != 1 || selected[0].Name != "b" {
  		t.Fatalf("expected only the highest-scoring skill 'b', got %+v", selected)
  	}
  }

  func TestSelectAlwaysIncludesRequiredEvenBeyondCap(t *testing.T) {
  	all := []skills.Skill{
  		{Name: "required-skill", Tags: []string{"unrelated"}},
  		{Name: "matched", Tags: []string{"web"}},
  	}
  	selected := Select(all, "build a web thing", []string{"required-skill"}, 1)
  	names := map[string]bool{}
  	for _, s := range selected {
  		names[s.Name] = true
  	}
  	if !names["required-skill"] {
  		t.Fatalf("required-skill must always be included, got %+v", selected)
  	}
  }

  func TestSelectNoCapReturnsAllMatches(t *testing.T) {
  	all := []skills.Skill{
  		{Name: "a", Tags: []string{"web"}},
  		{Name: "b", Tags: []string{"web"}},
  	}
  	selected := Select(all, "web", nil, 0)
  	if len(selected) != 2 {
  		t.Fatalf("expected both matches with no cap, got %d", len(selected))
  	}
  }
  ```

---

## Task 3 — Project-doc retrieval (`internal/docsearch`)

- [x] **3.1** Create `cli/internal/docsearch/docsearch.go`:

  ```go
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
  ```

- [x] **3.2** Create `cli/internal/docsearch/docsearch_test.go`:

  ```go
  package docsearch

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func writeFile(t *testing.T, path, content string) {
  	t.Helper()
  	os.MkdirAll(filepath.Dir(path), 0o755)
  	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
  		t.Fatal(err)
  	}
  }

  func TestParseSections(t *testing.T) {
  	path := filepath.Join(t.TempDir(), "src-map.md")
  	writeFile(t, path, "# Title\n\n### cli/ — the eng CLI\n\nBody one.\n\n### harness/ — payload\n\nBody two.\n")

  	sections, err := ParseSections(path)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(sections) != 2 || sections[0].Title != "cli/ — the eng CLI" {
  		t.Fatalf("got %+v", sections)
  	}
  }

  func TestMatchRanksByScoreAndCaps(t *testing.T) {
  	sections := []Section{
  		{Title: "Modbus adapter", Body: "industrial protocol details"},
  		{Title: "Login page", Body: "web authentication flow"},
  		{Title: "Web API", Body: "http handlers and routes"},
  	}
  	matched := Match(sections, "add a web login", 1)
  	if len(matched) != 1 {
  		t.Fatalf("expected 1 result capped, got %d", len(matched))
  	}
  	if matched[0].Title != "Login page" {
  		t.Fatalf("expected 'Login page' to rank highest, got %q", matched[0].Title)
  	}
  }

  func TestMatchNoCapReturnsAll(t *testing.T) {
  	sections := []Section{{Title: "web x", Body: "y"}, {Title: "web z", Body: "y"}}
  	matched := Match(sections, "web", 0)
  	if len(matched) != 2 {
  		t.Fatalf("expected 2 with no cap, got %d", len(matched))
  	}
  }
  ```

---

## Task 4 — Task-scoped extraction (`internal/taskscope`)

- [x] **4.1** Create `cli/internal/taskscope/taskscope.go`:

  ```go
  package taskscope

  import (
  	"os"
  	"regexp"
  	"strings"
  )

  var taskHeaderRe = regexp.MustCompile(`(?m)^## Task \d+`)

  // CurrentTask returns the first task block (delimited by "## Task N —"
  // headers, the convention every plan's tasks.md already uses) that still
  // contains an unchecked "- [ ]" subtask — the same signal
  // scripts/plan-executor.sh (V1) and workflow_cmd.go's tasksComplete
  // (Phase 3) already trust. Returns "" (no error) if every task is checked.
  func CurrentTask(tasksPath string) (string, error) {
  	data, err := os.ReadFile(tasksPath)
  	if err != nil {
  		return "", err
  	}
  	content := string(data)
  	idx := taskHeaderRe.FindAllStringIndex(content, -1)
  	for i, loc := range idx {
  		end := len(content)
  		if i+1 < len(idx) {
  			end = idx[i+1][0]
  		}
  		block := content[loc[0]:end]
  		if strings.Contains(block, "- [ ]") {
  			return strings.TrimSpace(block), nil
  		}
  	}
  	return "", nil
  }

  // GoalSummary returns spec.md's "## Goal" section — the one-paragraph
  // context an Executor needs, without the rest of spec.md's design
  // discussion, design decisions, and file tables.
  func GoalSummary(specPath string) (string, error) {
  	data, err := os.ReadFile(specPath)
  	if err != nil {
  		return "", err
  	}
  	content := string(data)
  	start := strings.Index(content, "## Goal")
  	if start < 0 {
  		return "", nil
  	}
  	rest := content[start+len("## Goal"):]
  	end := strings.Index(rest, "\n## ")
  	if end < 0 {
  		end = len(rest)
  	}
  	return strings.TrimSpace(rest[:end]), nil
  }
  ```

- [x] **4.2** Create `cli/internal/taskscope/taskscope_test.go`:

  ```go
  package taskscope

  import (
  	"os"
  	"path/filepath"
  	"strings"
  	"testing"
  )

  func TestCurrentTaskReturnsFirstUnchecked(t *testing.T) {
  	path := filepath.Join(t.TempDir(), "tasks.md")
  	content := "# Tasks\n\n## Task 1 — Done thing\n\n- [x] **1.1** already done\n\n## Task 2 — Pending thing\n\n- [ ] **2.1** not done yet\n"
  	os.WriteFile(path, []byte(content), 0o644)

  	task, err := CurrentTask(path)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !strings.Contains(task, "Task 2") || strings.Contains(task, "Task 1") {
  		t.Fatalf("expected only Task 2's block, got: %s", task)
  	}
  }

  func TestCurrentTaskEmptyWhenAllChecked(t *testing.T) {
  	path := filepath.Join(t.TempDir(), "tasks.md")
  	os.WriteFile(path, []byte("## Task 1 — Done\n\n- [x] **1.1** done\n"), 0o644)

  	task, err := CurrentTask(path)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if task != "" {
  		t.Fatalf("expected empty string, got: %s", task)
  	}
  }

  func TestGoalSummaryExtractsOnlyGoalSection(t *testing.T) {
  	path := filepath.Join(t.TempDir(), "spec.md")
  	content := "# Spec\n\n## Goal\n\nDo the thing.\n\n## Design\n\nLots of detail here.\n"
  	os.WriteFile(path, []byte(content), 0o644)

  	goal, err := GoalSummary(path)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !strings.Contains(goal, "Do the thing.") || strings.Contains(goal, "Lots of detail") {
  		t.Fatalf("got: %q", goal)
  	}
  }
  ```

---

## Task 5 — Global default config and the Context Manager methodology

- [x] **5.1** Create `harness/context/default.yaml`:

  ```yaml
  strategy: selective
  max_skills: 5
  max_docs: 8
  max_log_lines: 300
  include_completed_tasks: false
  summarize_tool_output: true
  ```

- [x] **5.2** Create `harness/core/context-manager/METHOD.md`:

  ```markdown
  # Core Method: Context Manager

  Not an AI role — a mechanical selection step any role's session runs before starting real
  work, so the harness maximizes *relevant* context instead of total context.

  ## What it does

  `eng context bundle <role> <plan-dir> ["<request text>"]` composes:

  - **Planner** — matching `docs/src-map.md`/`docs/gotchas.md` sections + matching skills
  - **Plan Reviewer** — plan facts (`risk_level`, `requires_approval`) + matching project context
  - **Executor** — the current unchecked task block + matching skills (not the whole plan)
  - **Verifier** — the plan's `write_scope`/verification rules

  ## Fail-safe rule

  If the request text is empty, too short, or scores zero matches against everything
  available, **say so and ask for more specific input or fall back to `strategy: full` for this
  one call** — do not invent a plausible-looking selection. A context bundle that silently
  picked the wrong things is worse than one that visibly returned nothing.

  ## Observability

  Every `eng context bundle` call writes `<plan-dir>/context-manifest.yaml` recording what was
  selected and how much was omitted. Read it when a role behaves as if it's missing context
  that should have been included — it answers "why wasn't X selected" without re-running
  anything.

  ## Constraint

  Read-only with respect to project source and skills. Its only write is
  `context-manifest.yaml`.
  ```

---

## Task 6 — `eng context skills/project/task/bundle`

- [x] **6.1** Create `cli/context_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"
  	"time"

  	"eng/internal/contextcfg"
  	"eng/internal/docsearch"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/skillmatch"
  	"eng/internal/skills"
  	"eng/internal/taskscope"
  )

  func cmdContext(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng context <skills|project|task|bundle> ...")
  		os.Exit(1)
  	}
  	switch args[0] {
  	case "skills":
  		contextSkills(args[1:])
  	case "project":
  		contextProject(args[1:])
  	case "task":
  		contextTask(args[1:])
  	case "bundle":
  		contextBundle(args[1:])
  	default:
  		fmt.Println("Usage: eng context <skills|project|task|bundle> ...")
  		os.Exit(1)
  	}
  }

  func loadContextConfig(dir string) contextcfg.Config {
  	global := filepath.Join(harnessDir(), "context", "default.yaml")
  	cfg, err := contextcfg.Load(dir, global)
  	if err != nil {
  		return contextcfg.Default()
  	}
  	return cfg
  }

  // selectSkills is the pure core behind `eng context skills`, reused by
  // `eng context bundle` so the manifest can record exactly what was chosen.
  func selectSkills(dir, request string, cfg contextcfg.Config) (selected []skills.Skill, total int, err error) {
  	all, err := skills.Resolve(filepath.Join(harnessDir(), "skills"), filepath.Join(dir, "skills"))
  	if err != nil {
  		return nil, 0, err
  	}
  	var mustInclude []string
  	if pcfg, err := project.Load(dir); err == nil {
  		mustInclude = pcfg.EnabledSkills
  	}
  	maxSkills := cfg.MaxSkills
  	if cfg.Strategy == "full" {
  		maxSkills = 0
  	}
  	return skillmatch.Select(all, request, mustInclude, maxSkills), len(all), nil
  }

  func printSkillSelection(selected []skills.Skill, total int, cfg contextcfg.Config) {
  	fmt.Printf("Selected %d/%d skills (strategy: %s, max_skills: %d)\n\n", len(selected), total, cfg.Strategy, cfg.MaxSkills)
  	for _, s := range selected {
  		fmt.Printf("- %-30s [%s] %s\n", s.Name, s.Domain, s.Description)
  	}
  	if len(selected) < total {
  		fmt.Printf("\n%d skill(s) omitted as not relevant to this request.\n", total-len(selected))
  	}
  }

  func contextSkills(args []string) {
  	if len(args) == 0 {
  		fmt.Println(`Usage: eng context skills "<request text>"`)
  		os.Exit(1)
  	}
  	request := strings.Join(args, " ")
  	dir, _ := os.Getwd()
  	cfg := loadContextConfig(dir)
  	selected, total, err := selectSkills(dir, request, cfg)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	printSkillSelection(selected, total, cfg)
  }

  // selectProjectContext is the pure core behind `eng context project`.
  func selectProjectContext(dir, request string, cfg contextcfg.Config) map[string][]docsearch.Section {
  	maxDocs := cfg.MaxDocs
  	if cfg.Strategy == "full" {
  		maxDocs = 0
  	}
  	result := map[string][]docsearch.Section{}
  	for _, name := range []string{"docs/src-map.md", "docs/gotchas.md"} {
  		sections, err := docsearch.ParseSections(filepath.Join(dir, name))
  		if err != nil {
  			continue
  		}
  		result[name] = docsearch.Match(sections, request, maxDocs)
  	}
  	return result
  }

  func contextProject(args []string) {
  	if len(args) == 0 {
  		fmt.Println(`Usage: eng context project "<request text>"`)
  		os.Exit(1)
  	}
  	request := strings.Join(args, " ")
  	dir, _ := os.Getwd()
  	cfg := loadContextConfig(dir)

  	byFile := selectProjectContext(dir, request, cfg)
  	for _, name := range []string{"docs/src-map.md", "docs/gotchas.md"} {
  		matched, ok := byFile[name]
  		if !ok {
  			fmt.Printf("(%s not found or unreadable — skipping)\n", name)
  			continue
  		}
  		fmt.Printf("## From %s (%d matched)\n\n", name, len(matched))
  		for _, s := range matched {
  			fmt.Printf("### %s\n%s\n", s.Title, s.Body)
  		}
  	}
  }

  func contextTask(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng context task <plan-dir>")
  		os.Exit(1)
  	}
  	planDir, _ := filepath.Abs(args[0])

  	goal, _ := taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
  	task, err := taskscope.CurrentTask(filepath.Join(planDir, "tasks.md"))
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Println("## Goal summary")
  	fmt.Println(goal)
  	fmt.Println("\n## Current task")
  	if task == "" {
  		fmt.Println("(no unchecked task found — all tasks may be complete)")
  	} else {
  		fmt.Println(task)
  	}
  }

  func contextBundle(args []string) {
  	if len(args) < 2 {
  		fmt.Println(`Usage: eng context bundle <planner|plan-reviewer|executor|verifier> <plan-dir> ["<request text>"]`)
  		os.Exit(1)
  	}
  	role := args[0]
  	planDir, _ := filepath.Abs(args[1])
  	request := ""
  	if len(args) > 2 {
  		request = strings.Join(args[2:], " ")
  	}

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}
  	if request == "" {
  		request, _ = taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
  	}

  	repoRoot, _ := os.Getwd()
  	cfg := loadContextConfig(repoRoot)

  	var manifest strings.Builder
  	fmt.Fprintf(&manifest, "role: %s\nplan: %s\ngenerated_at: %s\nrequest: %q\n", role, meta.Plan, time.Now().UTC().Format(time.RFC3339), request)

  	fmt.Printf("# Context bundle for role: %s\n\n", role)

  	switch role {
  	case "planner":
  		byFile := selectProjectContext(repoRoot, request, cfg)
  		fmt.Fprintf(&manifest, "project_sections:\n")
  		for name, sections := range byFile {
  			fmt.Printf("## From %s\n\n", name)
  			for _, s := range sections {
  				fmt.Printf("### %s\n%s\n", s.Title, s.Body)
  				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
  			}
  		}
  		selected, total, _ := selectSkills(repoRoot, request, cfg)
  		fmt.Println("## Skills")
  		printSkillSelection(selected, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for _, s := range selected {
  			fmt.Fprintf(&manifest, "  - %s\n", s.Name)
  		}

  	case "plan-reviewer":
  		fmt.Printf("## Plan\nrisk_level: %s\nrequires_approval: %v\n\n", meta.RiskLevel, meta.RequiresApproval)
  		fmt.Fprintf(&manifest, "risk_level: %s\nrequires_approval: %v\nproject_sections:\n", meta.RiskLevel, meta.RequiresApproval)
  		byFile := selectProjectContext(repoRoot, request, cfg)
  		for name, sections := range byFile {
  			fmt.Printf("## From %s\n\n", name)
  			for _, s := range sections {
  				fmt.Printf("### %s\n%s\n", s.Title, s.Body)
  				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
  			}
  		}

  	case "executor":
  		fmt.Println("## Task scope")
  		contextTask([]string{planDir})
  		task, _ := taskscope.CurrentTask(filepath.Join(planDir, "tasks.md"))
  		fmt.Fprintf(&manifest, "current_task_present: %v\n", task != "")
  		selected, total, _ := selectSkills(repoRoot, request, cfg)
  		fmt.Println("\n## Skills")
  		printSkillSelection(selected, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for _, s := range selected {
  			fmt.Fprintf(&manifest, "  - %s\n", s.Name)
  		}

  	case "verifier":
  		fmt.Printf("## Verification rules\nwrite_scope: %v\n", meta.WriteScope)
  		fmt.Fprintf(&manifest, "write_scope: %v\n", meta.WriteScope)

  	default:
  		fmt.Println("Unknown role:", role)
  		os.Exit(1)
  	}

  	manifestPath := filepath.Join(planDir, "context-manifest.yaml")
  	if err := os.WriteFile(manifestPath, []byte(manifest.String()), 0o644); err != nil {
  		fmt.Println("warning: could not write context-manifest.yaml:", err)
  		return
  	}
  	fmt.Printf("\n(context selection recorded in %s)\n", manifestPath)
  }
  ```

---

## Task 7 — Wire up dispatch in `main.go`

- [x] **7.1** In `cli/main.go`, update the `switch` in `main()`:

  Old:
  ```go
  	case "start":
  		cmdStart(os.Args[2:])
  	default:
  ```

  New:
  ```go
  	case "start":
  		cmdStart(os.Args[2:])
  	case "context":
  		cmdContext(os.Args[2:])
  	default:
  ```

- [x] **7.2** In `cli/main.go`'s `usage()` function, append after the `start` line:

  ```
    context skills "<text>"            Show the skills selected for a request
    context project "<text>"           Show matching docs/src-map.md and docs/gotchas.md sections
    context task <plan-dir>            Show the current task and goal summary
    context bundle <role> <plan-dir>   Compose role-specific context and write a manifest`)
  ```

- [x] **7.3** Run `cd cli && go vet ./... && go build -o eng . 2>&1` — fix any compile errors
  before proceeding.

---

## Task 8 — `eng verify` log compaction

- [x] **8.1** In `cli/verify_cmd.go`, add the new imports and replace the test-run block:

  Old imports:
  ```go
  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"
  	"time"

  	"eng/internal/executil"
  	"eng/internal/gitutil"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  )
  ```

  New imports:
  ```go
  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"
  	"time"

  	"eng/internal/contextcfg"
  	"eng/internal/executil"
  	"eng/internal/gitutil"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  )
  ```

  Old test-run block:
  ```go
  	if cfg, err := project.Load(repoRoot); err == nil && !cfg.Stack.Test.Empty() {
  		fmt.Fprintf(&report, "\n## Test run\n\nCommand: `%s`\n\n", cfg.Stack.Test.String())
  		out, testErr := executil.Run(cfg.Stack.Test, repoRoot)
  		fmt.Fprintf(&report, "```\n%s\n```\n\n", out)
  		if testErr != nil {
  			fmt.Fprintf(&report, "Test command exited with error: %v\n\n", testErr)
  			pass = false
  		}
  	}
  ```

  New test-run block:
  ```go
  	if cfg, err := project.Load(repoRoot); err == nil && !cfg.Stack.Test.Empty() {
  		fmt.Fprintf(&report, "\n## Test run\n\nCommand: `%s`\n\n", cfg.Stack.Test.String())
  		out, testErr := executil.Run(cfg.Stack.Test, repoRoot)

  		ctxCfg := loadContextConfig(repoRoot)
  		logPath, logErr := writeFullLog(repoRoot, "verify", out)
  		display := out
  		if ctxCfg.SummarizeToolOutput {
  			display = summarizeOutput(out, ctxCfg.MaxLogLines)
  		}
  		fmt.Fprintf(&report, "```\n%s\n```\n\n", display)
  		if logErr == nil && display != out {
  			fmt.Fprintf(&report, "Full output: `%s`\n\n", logPath)
  		}
  		if testErr != nil {
  			fmt.Fprintf(&report, "Test command exited with error: %v\n\n", testErr)
  			pass = false
  		}
  	}
  ```

- [x] **8.2** Append two new functions to `cli/verify_cmd.go`:

  ```go
  // writeFullLog persists the complete tool output to .agent/logs/, keeping
  // the report/stdout bounded regardless of test-suite size (Requirement 8).
  func writeFullLog(repoRoot, kind, content string) (string, error) {
  	dir := filepath.Join(repoRoot, ".agent", "logs")
  	if err := os.MkdirAll(dir, 0o755); err != nil {
  		return "", err
  	}
  	name := fmt.Sprintf("%s-%s.log", kind, time.Now().UTC().Format("20060102-150405"))
  	path := filepath.Join(dir, name)
  	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
  		return "", err
  	}
  	return path, nil
  }

  // summarizeOutput bounds out to maxLines by keeping the head and tail —
  // deliberately not head-only, since failures are conventionally reported
  // at the end of test output. Line-count based, not token-based, per
  // Requirement 7's explicit instruction not to hard-code token counts to a
  // single model.
  func summarizeOutput(out string, maxLines int) string {
  	if maxLines <= 0 {
  		return out
  	}
  	lines := strings.Split(out, "\n")
  	if len(lines) <= maxLines {
  		return out
  	}
  	half := maxLines / 2
  	head := lines[:half]
  	tail := lines[len(lines)-half:]
  	omitted := len(lines) - len(head) - len(tail)
  	return strings.Join(head, "\n") +
  		fmt.Sprintf("\n... [%d lines omitted, see full log] ...\n", omitted) +
  		strings.Join(tail, "\n")
  }
  ```

  Note: `loadContextConfig` is defined in `cli/context_cmd.go` (Task 6) — same package, no
  import needed, just ensure Task 6 lands before this task's code is expected to compile.

---

## Task 9 — `eng init` ensures `.agent/logs/` is gitignored

- [x] **9.1** In `cli/init_cmd.go`, after the `project.Save(dir, cfg)` block and before the
  final `fmt.Printf`/`if mode == "hybrid"` block, insert:

  ```go
  	ensureGitignoreEntry(dir, ".agent/logs/")
  ```

- [x] **9.2** Append a new function to `cli/init_cmd.go`:

  ```go
  // ensureGitignoreEntry appends entry to the project's .gitignore if it
  // isn't already present, creating the file if none exists. Never
  // overwrites or reorders existing content — purely additive, matching
  // this repo's own "never touch what you don't need to" convention.
  func ensureGitignoreEntry(dir, entry string) {
  	path := filepath.Join(dir, ".gitignore")
  	existing, _ := os.ReadFile(path)
  	if strings.Contains(string(existing), entry) {
  		return
  	}
  	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
  	if err != nil {
  		return
  	}
  	defer f.Close()
  	prefix := ""
  	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
  		prefix = "\n"
  	}
  	f.WriteString(prefix + entry + "\n")
  }
  ```

  Add `"strings"` to `cli/init_cmd.go`'s import block if not already present.

---

## Task 10 — Update role methodology docs to reference context bundling

- [x] **10.1** In `harness/core/planner/METHOD.md`, in the "Before writing spec.md" section,
  add a step after the Triage step:

  Old:
  ```markdown
  ## Before writing spec.md

  1. Run Triage (see `core/triage/METHOD.md`) to determine the risk level.
  2. Read the project's own context docs (`docs/src-map.md`, `docs/gotchas.md`, or
  ```

  New:
  ```markdown
  ## Before writing spec.md

  1. Run Triage (see `core/triage/METHOD.md`) to determine the risk level.
  2. Run `eng context bundle planner <plan-dir> "<request text>"` (see
     `core/context-manager/METHOD.md`) for a curated set of matching project context and
     skills — read the full `docs/src-map.md`/`docs/gotchas.md` only if the bundle's fail-safe
     rule triggers (nothing scored a match).
  3. Read the project's own context docs (`docs/src-map.md`, `docs/gotchas.md`, or
  ```

  (Renumber the remaining list items by one.)

- [x] **10.2** In `harness/core/plan-reviewer/METHOD.md`, after the "## Role" section, add:

  ```markdown
  ## Before reviewing

  Run `eng context bundle plan-reviewer <plan-dir>` for the plan's risk/approval facts plus
  matching project context, in addition to reading `spec.md`/`tasks.md`/`tests.md` directly.
  ```

- [x] **10.3** In `harness/core/executor/METHOD.md`, in the "Task loop" section, add a step
  before step 1:

  Old:
  ```markdown
  ## Task loop

  1. Read `spec.md` → `tasks.md` → `tests.md`, in that order.
  ```

  New:
  ```markdown
  ## Task loop

  0. Run `eng context bundle executor <plan-dir>` for the current unchecked task and a goal
     summary, instead of re-reading the entirety of `tasks.md` for every task.
  1. Read `spec.md` → `tasks.md` → `tests.md`, in that order.
  ```

- [x] **10.4** In `harness/core/verifier/METHOD.md`, after the "## Role" numbered list, add:

  ```markdown
  ## Full tool output

  `eng verify`'s report shows a bounded head+tail summary of the test command's output. If
  that isn't enough to diagnose a FAIL, the complete output is at
  `.agent/logs/verify-<timestamp>.log` (the report names the exact path when truncated).
  ```

---

## Task 11 — Version bump and docs integration (last task)

- [x] **11.1** Update `harness/VERSION`:

  ```
  0.4.0-phase4-context
  ```

- [x] **11.2** In `README.md`, immediately after the Phase 3 section added previously (before
  the following `---`), add:

  ```markdown

  Phase 4 adds context selection so the harness maximizes relevant context, not total context:

  ```bash
  cd cli && go build -o eng .
  ./eng context skills "add Modbus TCP monitoring"
  ./eng context project "add Modbus TCP monitoring"
  ./eng context task .plans/2026-08-24-my-feature
  ./eng context bundle executor .plans/2026-08-24-my-feature
  ```

  See `.plans/2026-08-24-v2-harness-phase4-context/spec.md` for the full design.
  ```

- [x] **11.3** In `ROADMAP.md`, extend the note to include the Phase 4 plan link, following the
  same pattern as the Phase 3 addition.

- [x] **11.4** In `docs/src-map.md`, add a final module section after the Phase 3 entries:

  ```markdown

  ### `cli/internal/contextcfg/`, `cli/internal/skillmatch/`, `cli/internal/docsearch/`, `cli/internal/taskscope/` — Phase 4 context selection

  What it does: `contextcfg` loads the optional `.agent/context.yaml` budget (falling back to
  `harness/context/default.yaml`, then hard-coded defaults); `skillmatch` finally makes
  Phase 1's `tags`/`triggers` skill-frontmatter fields load-bearing by scoring them against a
  request; `docsearch` parses `docs/src-map.md`/`docs/gotchas.md`'s existing `### ` sections
  and keyword-matches them; `taskscope` extracts just the current unchecked task block and
  `spec.md`'s Goal summary instead of whole files.

  Key files: `cli/context_cmd.go` (`eng context skills/project/task/bundle`)

  Notable: `enabled_skills` (Phase 1) is always included by `skillmatch.Select` regardless of
  `max_skills` — the cap only limits additional discovered-but-not-required skills. `eng
  verify`'s test output is now capped (`max_log_lines`, default 300, head+tail) with the full
  output written to `.agent/logs/` — `eng workflow advance`'s gating logic is unaffected since
  it only ever reads `plan.yaml`'s `verification.verdict`, not the report text.

  From: `.plans/2026-08-24-v2-harness-phase4-context/`
  ```
