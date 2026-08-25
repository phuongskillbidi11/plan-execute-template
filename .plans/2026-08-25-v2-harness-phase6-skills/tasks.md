# Phase 6 Tasks — Multi-Domain Skill Ecosystem + Skill Router Evolution

Execute in order. Run each task's own build/test check before moving to the next. Every
"Old" snippet below is verified against the actual current file content before writing this
document — if a real mismatch is found during execution, stop and treat it as a
contradiction per the standing instruction, not something to paper over.

---

## Task 1 — Hook plan-directory fix (early corrective task, Requirement 27)

- [x] **1.1** Replace the full contents of `cli/hooks_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"

  	"eng/internal/executil"
  	"eng/internal/hooks"
  	"eng/internal/project"
  )

  func cmdHooks(args []string) {
  	if len(args) < 2 || args[0] != "run" {
  		fmt.Println("Usage: eng hooks run <stage> [plan-dir]")
  		os.Exit(1)
  	}
  	stage := args[1]
  	planDir := "."
  	if len(args) > 2 {
  		planDir = args[2]
  	}

  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	globalDefault := filepath.Join(harnessDir(), "hooks", "default.yaml")
  	cfg, err := hooks.Load(dir, globalDefault)
  	if err != nil {
  		fmt.Println("error loading hooks:", err)
  		os.Exit(1)
  	}

  	names := cfg.Stage(stage)
  	if len(names) == 0 {
  		fmt.Printf("No hooks configured for stage %q\n", stage)
  		return
  	}

  	testCmd := ""
  	if pcfg, err := project.Load(dir); err == nil {
  		testCmd = pcfg.Stack.Test.String()
  	}

  	for _, name := range names {
  		cmd := cfg.Commands[name]
  		if cmd.Empty() {
  			fmt.Printf("[%s] %-16s manual step — no shell command; perform via the documented role\n", stage, name)
  			continue
  		}
  		if cmd.Shell != "" {
  			cmd.Shell = strings.ReplaceAll(cmd.Shell, "${test_cmd}", testCmd)
  			cmd.Shell = strings.ReplaceAll(cmd.Shell, "${plan_dir}", planDir)
  		}
  		fmt.Printf("[%s] %-16s -> %s\n", stage, name, cmd.String())
  		out, err := executil.Run(cmd, dir)
  		fmt.Print(out)
  		if err != nil {
  			fmt.Printf("HOOK FAILED: %s (%v)\n", name, err)
  			os.Exit(1)
  		}
  	}
  }
  ```

  (Only change from the current file: the new optional third positional argument, defaulting
  to `"."` — today's exact hard-coded value — and one new `${plan_dir}` substitution alongside
  the existing `${test_cmd}` one.)

- [x] **1.2** In `harness/hooks/default.yaml`, change the `drift_check` and `verify` command
  strings from a literal `.` to the new template token:

  Old:
  ```yaml
    drift_check: "eng plan drift ."
  ```
  New:
  ```yaml
    drift_check: "eng plan drift ${plan_dir}"
  ```

  Old:
  ```yaml
    verify: "eng verify ."
  ```
  New:
  ```yaml
    verify: "eng verify ${plan_dir}"
  ```

  With no third argument to `eng hooks run`, `planDir` defaults to `"."`, so
  `${plan_dir}` substitutes to `.` — byte-identical to today's literal command. This is a
  pure additive capability: `eng hooks run before_execute <plan-dir>` now also works.

**Verify:** `cd cli && go build ./... && echo BUILD_OK` (T1 in tests.md covers the full
before/after behavioral proof).

---

## Task 2 — Skill metadata schema evolution + qualified-name resolution

- [x] **2.1** In `cli/internal/skills/skills.go`, extend the `Skill` struct:

  Old:
  ```go
  type Skill struct {
  	Name         string   `yaml:"name"`
  	Domain       string   `yaml:"domain"`
  	Description  string   `yaml:"description"`
  	Tags         []string `yaml:"tags"`
  	Triggers     []string `yaml:"triggers"`
  	Version      string   `yaml:"version"`
  	Dependencies []string `yaml:"dependencies"`
  	Conflicts    []string `yaml:"conflicts"`
  	WhenToUse    string   `yaml:"when_to_use"`
  	WhenNotToUse string   `yaml:"when_not_to_use"`
  	Source       string   `yaml:"-"` // "global" or "local", set by Resolve
  	Path         string   `yaml:"-"`
  }
  ```

  New:
  ```go
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
  ```

  (`Dependencies` is renamed to `Requires`/`requires:` — see Phase 6 spec.md Decision 4: it
  is defined but read by no code and used by no shipped `SKILL.md` today, so this is not a
  behavior change for any real consumer.)

- [x] **2.2** In the same file, replace `Resolve` with a three-tier version, keeping
  `Resolve`'s own signature and behavior identical via delegation:

  Old:
  ```go
  // Resolve merges global and project-local skills by name; local overrides
  // global on a name collision.
  func Resolve(globalRoot, localRoot string) ([]Skill, error) {
  	global, err := Walk(globalRoot)
  	if err != nil {
  		return nil, err
  	}
  	for i := range global {
  		global[i].Source = "global"
  	}

  	local, err := Walk(localRoot)
  	if err != nil {
  		return nil, err
  	}
  	for i := range local {
  		local[i].Source = "local"
  	}

  	merged := map[string]Skill{}
  	for _, s := range global {
  		merged[s.Name] = s
  	}
  	for _, s := range local {
  		merged[s.Name] = s
  	}

  	out := make([]Skill, 0, len(merged))
  	for _, s := range merged {
  		out = append(out, s)
  	}
  	return out, nil
  }
  ```

  New:
  ```go
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

  	out := make([]Skill, 0, len(merged))
  	for _, s := range merged {
  		out = append(out, s)
  	}
  	return out, nil
  }
  ```

- [x] **2.3** Append to `cli/internal/skills/skills_test.go`:

  ```go
  func TestQualifiedNameLegacySkillStaysBareName(t *testing.T) {
  	s := Skill{Name: "example", Domain: "unknown"}
  	if s.QualifiedName() != "example" {
  		t.Fatalf("expected legacy skill to keep bare name, got %q", s.QualifiedName())
  	}
  }

  func TestQualifiedNameSelfNamespacedNameUnchanged(t *testing.T) {
  	s := Skill{Name: "company/internal-api", Domain: "company"}
  	if s.QualifiedName() != "company/internal-api" {
  		t.Fatalf("expected self-namespaced name unchanged, got %q", s.QualifiedName())
  	}
  }

  func TestResolveQualifiesByDomainToAvoidCollisions(t *testing.T) {
  	root := t.TempDir()
  	writeSkill(t, filepath.Join(root, "automation"), "modbus", "---\nname: modbus\ndomain: automation\ndescription: automation modbus\n---\n")
  	writeSkill(t, filepath.Join(root, "networking"), "modbus", "---\nname: modbus\ndomain: networking\ndescription: networking modbus\n---\n")
  	merged, err := Resolve(root, t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(merged) != 2 {
  		t.Fatalf("expected 2 distinct skills (domain-qualified), got %d: %+v", len(merged), merged)
  	}
  }

  func TestResolveWithPrivateEmptyRootSkipsTier(t *testing.T) {
  	g, l := t.TempDir(), t.TempDir()
  	writeSkill(t, g, "only-global", "---\nname: only-global\ndescription: g\n---\n")
  	merged, err := ResolveWithPrivate(g, "", l)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(merged) != 1 {
  		t.Fatalf("expected 1 skill, got %d", len(merged))
  	}
  }

  func TestResolveWithPrivatePrecedenceGlobalLtPrivateLtLocal(t *testing.T) {
  	g, p, l := t.TempDir(), t.TempDir(), t.TempDir()
  	writeSkill(t, g, "shared", "---\nname: shared\ndescription: global\n---\n")
  	writeSkill(t, p, "shared", "---\nname: shared\ndescription: private\n---\n")

  	merged, err := ResolveWithPrivate(g, p, l)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(merged) != 1 || merged[0].Description != "private" {
  		t.Fatalf("expected private to override global, got %+v", merged)
  	}

  	writeSkill(t, l, "shared", "---\nname: shared\ndescription: local\n---\n")
  	merged, err = ResolveWithPrivate(g, p, l)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(merged) != 1 || merged[0].Description != "local" {
  		t.Fatalf("expected local to override private, got %+v", merged)
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/skills/... -v`. All existing
`skills` tests (`TestParseFrontmatter`, `TestParseLegacyHeading`,
`TestResolveLocalOverridesGlobal`, `TestResolveMissingRoots`) must still pass unmodified —
they use no `domain:` in the "shared"-name fixture, so `QualifiedName()` returns the bare
name for them and behavior is unchanged.

---

## Task 3 — Project config: `Domains` and `PrivateSkillsPath`

- [x] **3.1** In `cli/internal/project/project.go`, extend `Config`:

  Old:
  ```go
  type Config struct {
  	ProjectName     string      `yaml:"project_name"`
  	Mode            string      `yaml:"mode"` // legacy | hybrid | modern
  	HarnessProfile  string      `yaml:"harness_profile"`
  	ConfigVersion   int         `yaml:"config_version"`
  	Stack           Stack       `yaml:"stack"`
  	EnabledSkills   []string    `yaml:"enabled_skills"`
  	Workflow        Workflow    `yaml:"workflow,omitempty"`
  	RetryBudget     RetryBudget `yaml:"retry_budget,omitempty"`
  	RequireApproval []string    `yaml:"require_approval,omitempty"`
  }
  ```

  New:
  ```go
  type Config struct {
  	ProjectName     string      `yaml:"project_name"`
  	Mode            string      `yaml:"mode"` // legacy | hybrid | modern
  	HarnessProfile  string      `yaml:"harness_profile"`
  	ConfigVersion   int         `yaml:"config_version"`
  	Stack           Stack       `yaml:"stack"`
  	EnabledSkills   []string    `yaml:"enabled_skills"`
  	Workflow        Workflow    `yaml:"workflow,omitempty"`
  	RetryBudget     RetryBudget `yaml:"retry_budget,omitempty"`
  	RequireApproval []string    `yaml:"require_approval,omitempty"`

  	// Domains is the Phase 6 domain-profile list (e.g. ["embedded",
  	// "automation"]) used by the skill router's domain-profile fill tier.
  	// Deliberately separate from HarnessProfile (a singular, unused-since-
  	// eng-init field from V1) — see Phase 6 spec.md Decision 7.
  	Domains []string `yaml:"domains,omitempty"`

  	// PrivateSkillsPath, if set, is resolved relative to the project root
  	// (or used as-is if absolute) as an extra skill root between global and
  	// local precedence. Empty (the default for every existing
  	// project.yaml) skips the private tier entirely — see Phase 6 spec.md
  	// Decision 8.
  	PrivateSkillsPath string `yaml:"private_skills_path,omitempty"`
  }
  ```

- [x] **3.2** Append to `cli/internal/project/project_test.go` (implemented using the file's own established Save/Load round-trip fixture style):

  ```go
  func TestDomainsDefaultsToEmpty(t *testing.T) {
  	dir := t.TempDir()
  	os.WriteFile(filepath.Join(dir, ConfigPath[:len(ConfigPath)-len("/project.yaml")]), nil, 0o644) // no-op guard, ignored if it errors
  	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
  	os.WriteFile(filepath.Join(dir, ConfigPath), []byte("project_name: x\nmode: modern\n"), 0o644)
  	cfg, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(cfg.Domains) != 0 {
  		t.Fatalf("expected no domains by default, got %v", cfg.Domains)
  	}
  }

  func TestDomainsAndPrivateSkillsPathRoundTrip(t *testing.T) {
  	dir := t.TempDir()
  	cfg := &Config{ProjectName: "x", Mode: "modern", Domains: []string{"embedded", "automation"}, PrivateSkillsPath: "../company-skills"}
  	if err := Save(dir, cfg); err != nil {
  		t.Fatal(err)
  	}
  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(got.Domains) != 2 || got.Domains[0] != "embedded" || got.PrivateSkillsPath != "../company-skills" {
  		t.Fatalf("round-trip mismatch: %+v", got)
  	}
  }
  ```

  (`TestDomainsDefaultsToEmpty`'s first `os.WriteFile` line is a harmless no-op guard against
  a directory-vs-file path edge case and can be simplified to just the `MkdirAll` + second
  `WriteFile` if a mismatch is found — verify against the file's actual existing test helper
  patterns before applying, since `project_test.go` already has its own established fixture
  style from Phases 1–5.)

**Verify:** `cd cli && go build ./... && go test ./internal/project/... -v`. All existing
`project` tests continue to pass — both new fields are `omitempty`, so a project.yaml
without them round-trips identically to before.

---

## Task 4 — `internal/skillgraph`: dependency resolution

- [x] **4.1** Create `cli/internal/skillgraph/skillgraph.go`:

  ```go
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
  ```

- [x] **4.2** Create `cli/internal/skillgraph/skillgraph_test.go`:

  ```go
  package skillgraph

  import "testing"

  import "eng/internal/skills"

  func mk(name, domain string, requires []string) skills.Skill {
  	return skills.Skill{Name: name, Domain: domain, Requires: requires}
  }

  func TestExpandTransitiveClosure(t *testing.T) {
  	all := []skills.Skill{
  		mk("c", "x", nil),
  		mk("b", "x", []string{"x/c"}),
  		mk("a", "x", []string{"x/b"}),
  	}
  	out, err := Expand(all, []skills.Skill{all[2]})
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(out) != 3 {
  		t.Fatalf("expected a, b, c all included, got %+v", out)
  	}
  }

  func TestExpandDeduplicatesDiamond(t *testing.T) {
  	all := []skills.Skill{
  		mk("d", "x", nil),
  		mk("b", "x", []string{"x/d"}),
  		mk("c", "x", []string{"x/d"}),
  		mk("a", "x", []string{"x/b", "x/c"}),
  	}
  	out, err := Expand(all, []skills.Skill{all[3]})
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(out) != 4 {
  		t.Fatalf("expected exactly 4 distinct skills (d not duplicated), got %d: %+v", len(out), out)
  	}
  }

  func TestExpandDetectsCycle(t *testing.T) {
  	all := []skills.Skill{
  		mk("a", "x", []string{"x/b"}),
  		mk("b", "x", []string{"x/a"}),
  	}
  	if _, err := Expand(all, []skills.Skill{all[0]}); err == nil {
  		t.Fatal("expected a cycle error")
  	}
  }

  func TestExpandUnknownRequiredSkillErrors(t *testing.T) {
  	all := []skills.Skill{mk("a", "x", []string{"x/nonexistent"})}
  	if _, err := Expand(all, []skills.Skill{all[0]}); err == nil {
  		t.Fatal("expected an unknown-required-skill error")
  	}
  }

  func TestExpandDeterministicOrder(t *testing.T) {
  	all := []skills.Skill{mk("b", "x", nil), mk("a", "x", nil), mk("c", "x", nil)}
  	out1, err := Expand(all, all)
  	if err != nil {
  		t.Fatal(err)
  	}
  	shuffled := []skills.Skill{all[2], all[0], all[1]}
  	out2, err := Expand(shuffled, shuffled)
  	if err != nil {
  		t.Fatal(err)
  	}
  	for i := range out1 {
  		if out1[i].Name != out2[i].Name {
  			t.Fatalf("order not deterministic: %v vs %v", out1, out2)
  		}
  	}
  }

  func TestExpandRequiresByBareNameAlsoWorks(t *testing.T) {
  	all := []skills.Skill{
  		mk("child", "x", nil),
  		mk("parent", "x", []string{"child"}), // bare name, not qualified
  	}
  	out, err := Expand(all, []skills.Skill{all[1]})
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(out) != 2 {
  		t.Fatalf("expected requires-by-bare-name to resolve too, got %+v", out)
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/skillgraph/... -v`.

---

## Task 5 — `internal/skillrouter`: the routing engine

- [x] **5.1** Create `cli/internal/skillrouter/skillrouter.go`:

  ```go
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
  		if sc := skillmatch.Score(s, request); sc > 0 {
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
  ```

  (`skillmatch.Score`/`skillmatch.Select` are unchanged — `Score` is reused directly;
  `Select` is simply no longer called by `context_cmd.go` after Task 6, see Phase 6 spec.md
  Decision 5.)

- [x] **5.2** Create `cli/internal/skillrouter/skillrouter_test.go`:

  ```go
  package skillrouter

  import (
  	"testing"

  	"eng/internal/skills"
  )

  func mk(name, domain string, requires, recommends []string) skills.Skill {
  	return skills.Skill{Name: name, Domain: domain, Description: "d " + name, Requires: requires, Recommends: recommends}
  }

  func names(sel Selection) []string {
  	out := make([]string, len(sel.Skills))
  	for i, s := range sel.Skills {
  		out[i] = s.Name
  	}
  	return out
  }

  func contains(list []string, want string) bool {
  	for _, v := range list {
  		if v == want {
  			return true
  		}
  	}
  	return false
  }

  func TestExplicitNeverDroppedEvenWithZeroScoreAndTinyBudget(t *testing.T) {
  	all := []skills.Skill{mk("a", "x", nil, nil), mk("b", "x", nil, nil)}
  	sel, err := Route(all, "unrelated text", []string{"a"}, nil, 1)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !contains(names(sel), "a") {
  		t.Fatalf("expected explicit skill 'a' to survive, got %v", names(sel))
  	}
  }

  func TestRequiredDependencyIgnoresBudget(t *testing.T) {
  	all := []skills.Skill{
  		mk("child", "automation", nil, nil),
  		mk("parent", "automation", []string{"automation/child"}, nil),
  	}
  	sel, err := Route(all, "", []string{"parent"}, nil, 1)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(sel.Skills) != 2 {
  		t.Fatalf("expected both parent and its required child despite budget 1, got %v", names(sel))
  	}
  }

  func TestUnknownRequiredSkillReturnsError(t *testing.T) {
  	all := []skills.Skill{mk("a", "x", []string{"x/nonexistent"}, nil)}
  	if _, err := Route(all, "", []string{"a"}, nil, 0); err == nil {
  		t.Fatal("expected an error for an unknown required skill")
  	}
  }

  func TestRecommendsDroppedWhenBudgetExhausted(t *testing.T) {
  	all := []skills.Skill{
  		mk("main", "x", nil, []string{"x/extra"}),
  		mk("extra", "x", nil, nil),
  	}
  	sel, err := Route(all, "main", nil, nil, 1)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if contains(names(sel), "extra") {
  		t.Fatalf("expected the recommend to be dropped at budget 1, got %v", names(sel))
  	}
  }

  func TestRecommendsIncludedWhenBudgetAllows(t *testing.T) {
  	all := []skills.Skill{
  		mk("main", "x", nil, []string{"x/extra"}),
  		mk("extra", "x", nil, nil),
  	}
  	sel, err := Route(all, "main", nil, nil, 5)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !contains(names(sel), "extra") {
  		t.Fatalf("expected the recommend to be included when budget allows, got %v", names(sel))
  	}
  	for _, e := range sel.Explanations {
  		if e.Skill == "extra" && e.Reason != "recommended by x/main" {
  			t.Fatalf("unexpected reason for extra: %q", e.Reason)
  		}
  	}
  }

  func TestHigherScoringMatchWinsBudget(t *testing.T) {
  	strong := skills.Skill{Name: "strong", Domain: "x", Description: "d", Tags: []string{"alpha", "beta"}, Triggers: []string{"gamma"}}
  	weak := skills.Skill{Name: "weak", Domain: "x", Description: "d", Tags: []string{"alpha"}}
  	all := []skills.Skill{strong, weak}
  	sel, err := Route(all, "alpha beta gamma", nil, nil, 1)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !contains(names(sel), "strong") || contains(names(sel), "weak") {
  		t.Fatalf("expected only the higher-scoring skill to survive budget 1, got %v", names(sel))
  	}
  }

  func TestDomainProfileFillAfterStrongMatches(t *testing.T) {
  	all := []skills.Skill{
  		mk("matched", "automation", nil, nil),
  		mk("profileonly", "automation", nil, nil),
  	}
  	sel, err := Route(all, "matched", nil, []string{"automation"}, 5)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !contains(names(sel), "matched") || !contains(names(sel), "profileonly") {
  		t.Fatalf("expected both a strong match and a domain-profile fill, got %v", names(sel))
  	}
  }

  func TestDeterministicOrderingAcrossRuns(t *testing.T) {
  	all := []skills.Skill{
  		{Name: "b", Domain: "x", Description: "d", Tags: []string{"shared"}},
  		{Name: "a", Domain: "x", Description: "d", Tags: []string{"shared"}},
  		{Name: "c", Domain: "x", Description: "d", Tags: []string{"shared"}},
  	}
  	sel1, err := Route(all, "shared", nil, nil, 0)
  	if err != nil {
  		t.Fatal(err)
  	}
  	sel2, err := Route(all, "shared", nil, nil, 0)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(sel1.Skills) != 3 {
  		t.Fatalf("expected all 3 equally-scored skills selected, got %v", names(sel1))
  	}
  	for i := range sel1.Skills {
  		if sel1.Skills[i].Name != sel2.Skills[i].Name {
  			t.Fatalf("order differs across identical runs: %v vs %v", names(sel1), names(sel2))
  		}
  	}
  	if names(sel1)[0] != "a" || names(sel1)[1] != "b" || names(sel1)[2] != "c" {
  		t.Fatalf("expected alphabetical tie-break order, got %v", names(sel1))
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/skillrouter/... -v`.

---

## Task 6 — `context_cmd.go`: the router becomes the one authoritative path

- [x] **6.1** Replace the full contents of `cli/context_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"io"
  	"os"
  	"path/filepath"
  	"strings"
  	"time"

  	"eng/internal/contextcfg"
  	"eng/internal/docsearch"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/skillrouter"
  	"eng/internal/skills"
  	"eng/internal/taskscope"
  )

  func cmdContext(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng context <skills|project|task|bundle|manifest> ...")
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
  	case "manifest":
  		contextManifest(args[1:])
  	default:
  		fmt.Println("Usage: eng context <skills|project|task|bundle|manifest> ...")
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

  // privateSkillsRoot resolves .agent/project.yaml's optional
  // private_skills_path relative to dir, or "" if unset/unreadable — "" means
  // skip the private tier entirely (Phase 6 spec.md Decision 8).
  func privateSkillsRoot(dir string) string {
  	cfg, err := project.Load(dir)
  	if err != nil || cfg.PrivateSkillsPath == "" {
  		return ""
  	}
  	if filepath.IsAbs(cfg.PrivateSkillsPath) {
  		return cfg.PrivateSkillsPath
  	}
  	return filepath.Join(dir, cfg.PrivateSkillsPath)
  }

  // selectSkills is the pure core behind `eng context skills`, reused by
  // buildContextBundle so the manifest can record exactly what was chosen.
  // It is the one authoritative skill-selection path (Phase 6 Requirement
  // 19) — all routing (dependency expansion, domain-profile fills,
  // recommends, budget) happens inside skillrouter.Route.
  func selectSkills(dir, request string, cfg contextcfg.Config) (skillrouter.Selection, int, error) {
  	all, err := skills.ResolveWithPrivate(filepath.Join(harnessDir(), "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
  	if err != nil {
  		return skillrouter.Selection{}, 0, err
  	}
  	var mustInclude, domains []string
  	if pcfg, err := project.Load(dir); err == nil {
  		mustInclude = pcfg.EnabledSkills
  		domains = pcfg.Domains
  	}
  	maxSkills := cfg.MaxSkills
  	if cfg.Strategy == "full" {
  		maxSkills = 0
  	}
  	sel, err := skillrouter.Route(all, request, mustInclude, domains, maxSkills)
  	if err != nil {
  		return skillrouter.Selection{}, 0, err
  	}
  	return sel, len(all), nil
  }

  func writeSkillSelection(w io.Writer, sel skillrouter.Selection, total int, cfg contextcfg.Config) {
  	fmt.Fprintf(w, "Selected %d/%d skills (strategy: %s, max_skills: %d)\n\n", len(sel.Skills), total, cfg.Strategy, cfg.MaxSkills)
  	for i, s := range sel.Skills {
  		reason := ""
  		if i < len(sel.Explanations) {
  			reason = sel.Explanations[i].Reason
  		}
  		fmt.Fprintf(w, "- %-30s [%s] %s\n    selected because %s\n", s.Name, s.Domain, s.Description, reason)
  	}
  	if len(sel.Skills) < total {
  		fmt.Fprintf(w, "\n%d skill(s) omitted as not relevant to this request.\n", total-len(sel.Skills))
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
  	sel, total, err := selectSkills(dir, request, cfg)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	writeSkillSelection(os.Stdout, sel, total, cfg)
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

  func allSectionsEmpty(byFile map[string][]docsearch.Section) bool {
  	for _, sections := range byFile {
  		if len(sections) > 0 {
  			return false
  		}
  	}
  	return true
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

  // buildContextBundle is the pure core behind both `eng context bundle` and
  // `eng adapter prompt` (Phase 5 Decision 2) — it returns the composed
  // context text and writes context-manifest.yaml as a side effect.
  func buildContextBundle(role, planDir, request string) (string, error) {
  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		return "", fmt.Errorf("no %s found in %s", planmeta.FileName, planDir)
  	}
  	if request == "" {
  		request, _ = taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
  	}

  	repoRoot, _ := os.Getwd()
  	cfg := loadContextConfig(repoRoot)

  	var manifest strings.Builder
  	var out strings.Builder
  	fmt.Fprintf(&manifest, "role: %s\nplan: %s\ngenerated_at: %s\nrequest: %q\n", role, meta.Plan, time.Now().UTC().Format(time.RFC3339), request)
  	fmt.Fprintf(&out, "# Context bundle for role: %s\n\n", role)

  	switch role {
  	case "planner":
  		byFile := selectProjectContext(repoRoot, request, cfg)
  		sel, total, _ := selectSkills(repoRoot, request, cfg)
  		if allSectionsEmpty(byFile) && len(sel.Skills) == 0 && cfg.Strategy != "full" {
  			fbCfg := cfg
  			fbCfg.Strategy = "full"
  			byFile = selectProjectContext(repoRoot, request, fbCfg)
  			sel, total, _ = selectSkills(repoRoot, request, fbCfg)
  			fmt.Fprintf(&manifest, "fallback_to_full: true\n")
  			fmt.Fprintf(&out, "(no matches under 'selective' strategy — fell back to 'full' for this call)\n\n")
  		}
  		fmt.Fprintf(&manifest, "project_sections:\n")
  		for name, sections := range byFile {
  			fmt.Fprintf(&out, "## From %s\n\n", name)
  			for _, s := range sections {
  				fmt.Fprintf(&out, "### %s\n%s\n", s.Title, s.Body)
  				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
  			}
  		}
  		fmt.Fprintf(&out, "## Skills\n")
  		writeSkillSelection(&out, sel, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for i, s := range sel.Skills {
  			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, sel.Explanations[i].Reason)
  		}

  	case "plan-reviewer":
  		fmt.Fprintf(&out, "## Plan\nrisk_level: %s\nrequires_approval: %v\n\n", meta.RiskLevel, meta.RequiresApproval)
  		fmt.Fprintf(&manifest, "risk_level: %s\nrequires_approval: %v\nproject_sections:\n", meta.RiskLevel, meta.RequiresApproval)
  		byFile := selectProjectContext(repoRoot, request, cfg)
  		for name, sections := range byFile {
  			fmt.Fprintf(&out, "## From %s\n\n", name)
  			for _, s := range sections {
  				fmt.Fprintf(&out, "### %s\n%s\n", s.Title, s.Body)
  				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
  			}
  		}

  	case "executor":
  		goal, _ := taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
  		task, _ := taskscope.CurrentTask(filepath.Join(planDir, "tasks.md"))
  		fmt.Fprintf(&out, "## Goal summary\n%s\n\n## Current task\n", goal)
  		if task == "" {
  			fmt.Fprintf(&out, "(no unchecked task found — all tasks may be complete)\n")
  		} else {
  			fmt.Fprintf(&out, "%s\n", task)
  		}
  		fmt.Fprintf(&manifest, "current_task_present: %v\n", task != "")
  		sel, total, _ := selectSkills(repoRoot, request, cfg)
  		fmt.Fprintf(&out, "\n## Skills\n")
  		writeSkillSelection(&out, sel, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for i, s := range sel.Skills {
  			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, sel.Explanations[i].Reason)
  		}

  	case "verifier":
  		fmt.Fprintf(&out, "## Verification rules\nwrite_scope: %v\n", meta.WriteScope)
  		fmt.Fprintf(&manifest, "write_scope: %v\n", meta.WriteScope)

  	default:
  		return "", fmt.Errorf("unknown role: %s", role)
  	}

  	manifestPath := filepath.Join(planDir, "context-manifest.yaml")
  	if err := os.WriteFile(manifestPath, []byte(manifest.String()), 0o644); err != nil {
  		return out.String(), fmt.Errorf("context bundle built but failed to write manifest: %w", err)
  	}
  	fmt.Fprintf(&out, "\n(context selection recorded in %s)\n", manifestPath)
  	return out.String(), nil
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
  	out, err := buildContextBundle(role, planDir, request)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	fmt.Println(out)
  }

  func contextManifest(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng context manifest <plan-dir>")
  		os.Exit(1)
  	}
  	planDir, _ := filepath.Abs(args[0])
  	data, err := os.ReadFile(filepath.Join(planDir, "context-manifest.yaml"))
  	if err != nil {
  		fmt.Println("no context-manifest.yaml found — run `eng context bundle <role> <plan-dir>` first")
  		os.Exit(1)
  	}
  	fmt.Print(string(data))
  }
  ```

**Verify:** `cd cli && go build ./... && echo BUILD_OK` (T6 in tests.md covers a live
`eng context skills` run showing reasons).

---

## Task 7 — `internal/skillvalidate`

- [x] **7.1** Create `cli/internal/skillvalidate/skillvalidate.go`:

  ```go
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
  ```

- [x] **7.2** Create `cli/internal/skillvalidate/skillvalidate_test.go`:

  ```go
  package skillvalidate

  import (
  	"os"
  	"path/filepath"
  	"strings"
  	"testing"
  )

  func writeSkill(t *testing.T, dir, name, content string) {
  	t.Helper()
  	d := filepath.Join(dir, name)
  	if err := os.MkdirAll(d, 0o755); err != nil {
  		t.Fatal(err)
  	}
  	if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte(content), 0o644); err != nil {
  		t.Fatal(err)
  	}
  }

  func TestValidateLegacySkillWarnsNotErrors(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "legacy", "# Skill: legacy\n\n## Purpose\n\nAn old-style skill.\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(report.Errors()) != 0 {
  		t.Fatalf("expected no errors for a legacy skill, got %+v", report.Errors())
  	}
  	if len(report.Warnings()) == 0 {
  		t.Fatal("expected a warning for a legacy skill")
  	}
  }

  func TestValidateMissingDescriptionWarns(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "nodesc", "---\nname: nodesc\ndomain: x\n---\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	found := false
  	for _, i := range report.Warnings() {
  		if i.Skill == "nodesc" {
  			found = true
  		}
  	}
  	if !found {
  		t.Fatalf("expected a missing-description warning, got %+v", report.Issues)
  	}
  }

  func TestValidateUnknownRequiresErrors(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nrequires: [x/nonexistent]\n---\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(report.Errors()) == 0 {
  		t.Fatal("expected an error for an unknown required skill")
  	}
  }

  func TestValidateUnknownRecommendsWarnsOnly(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nrecommends: [x/nonexistent]\n---\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(report.Errors()) != 0 {
  		t.Fatalf("expected an unknown recommend to be a warning, not an error: %+v", report.Errors())
  	}
  	if len(report.Warnings()) == 0 {
  		t.Fatal("expected a warning for an unknown recommended skill")
  	}
  }

  func TestValidateDuplicateQualifiedNameWithinRootWarns(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, filepath.Join(dir, "automation", "path-one"), "dup", "---\nname: dup\ndomain: automation\ndescription: d1\n---\n")
  	writeSkill(t, filepath.Join(dir, "automation", "path-two"), "dup", "---\nname: dup\ndomain: automation\ndescription: d2\n---\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	found := false
  	for _, i := range report.Warnings() {
  		if i.Skill == "automation/dup" {
  			found = true
  		}
  	}
  	if !found {
  		t.Fatalf("expected a duplicate-qualified-name warning, got %+v", report.Issues)
  	}
  }

  func TestValidateSameBareNameDifferentDomainsDoesNotWarnAsDuplicate(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, filepath.Join(dir, "automation"), "modbus", "---\nname: modbus\ndomain: automation\ndescription: d1\n---\n")
  	writeSkill(t, filepath.Join(dir, "networking"), "modbus", "---\nname: modbus\ndomain: networking\ndescription: d2\n---\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	for _, i := range report.Issues {
  		if strings.Contains(i.Message, "duplicate") {
  			t.Fatalf("did not expect a duplicate warning for legitimately domain-qualified same-name skills: %+v", report.Issues)
  		}
  	}
  }

  func TestValidateCycleErrors(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nrequires: [x/b]\n---\n")
  	writeSkill(t, dir, "b", "---\nname: b\ndomain: x\ndescription: d\nrequires: [x/a]\n---\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(report.Errors()) == 0 {
  		t.Fatal("expected a cycle to be reported as an error")
  	}
  }

  func TestValidateBadVersionWarns(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "a", "---\nname: a\ndomain: x\ndescription: d\nversion: not-a-version\n---\n")
  	report, err := Validate(dir, "", t.TempDir())
  	if err != nil {
  		t.Fatal(err)
  	}
  	found := false
  	for _, i := range report.Warnings() {
  		if i.Skill == "a" {
  			found = true
  		}
  	}
  	if !found {
  		t.Fatalf("expected a bad-version warning, got %+v", report.Issues)
  	}
  }

  func TestReportErrorsExcludesWarnings(t *testing.T) {
  	r := Report{Issues: []Issue{{Skill: "a", Severity: SeverityWarning, Message: "w"}, {Skill: "b", Severity: SeverityError, Message: "e"}}}
  	if len(r.Errors()) != 1 || len(r.Warnings()) != 1 {
  		t.Fatalf("got errors=%v warnings=%v", r.Errors(), r.Warnings())
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/skillvalidate/... -v`.

---

## Task 8 — `eng skills validate` + private-tier wiring

- [x] **8.1** Replace the full contents of `cli/skills_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/skills"
  	"eng/internal/skillvalidate"
  )

  func cmdSkills(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng skills <list|validate>")
  		os.Exit(1)
  	}
  	switch args[0] {
  	case "list":
  		skillsList(args[1:])
  	case "validate":
  		skillsValidate(args[1:])
  	default:
  		fmt.Println("Usage: eng skills <list|validate>")
  		os.Exit(1)
  	}
  }

  func skillsList(args []string) {
  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	resolved, err := skills.ResolveWithPrivate(filepath.Join(harnessDir(), "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	for _, s := range resolved {
  		level := s.Level
  		if level == "" {
  			level = "-"
  		}
  		fmt.Printf("%-30s [%-7s] domain=%-12s level=%-11s %s\n", s.Name, s.Source, s.Domain, level, s.Description)
  	}
  }

  func skillsValidate(args []string) {
  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	report, err := skillvalidate.Validate(filepath.Join(harnessDir(), "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Printf("%d skill(s) discovered, %d error(s), %d warning(s)\n\n", report.Discovered, len(report.Errors()), len(report.Warnings()))
  	for _, issue := range report.Issues {
  		fmt.Printf("[%s] %-30s %s\n", issue.Severity, issue.Skill, issue.Message)
  	}

  	if len(report.Errors()) > 0 {
  		os.Exit(1)
  	}
  }
  ```

  (`privateSkillsRoot` was defined in `cli/context_cmd.go` in Task 6 — same `main` package,
  no new import needed here.)

**Verify:** `cd cli && go build ./... && echo BUILD_OK`.

---

## Task 9 — `eng doctor`: compact skill summary

- [x] **9.1** Replace the full contents of `cli/doctor.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"

  	"eng/internal/capabilities"
  	"eng/internal/project"
  	"eng/internal/skillvalidate"
  )

  func cmdDoctor(args []string) {
  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Println("eng doctor")
  	fmt.Println()

  	hDir := harnessDir()
  	if info, err := os.Stat(hDir); err == nil && info.IsDir() {
  		versionData, _ := os.ReadFile(filepath.Join(hDir, "VERSION"))
  		fmt.Printf("Harness install:   found at %s (version %s)\n", hDir, strings.TrimSpace(string(versionData)))
  	} else {
  		fmt.Println("Harness install:   NOT FOUND — run `eng install --from <path>`")
  	}

  	modeResult := project.DetectModeResult(dir)
  	switch modeResult.Mode {
  	case "legacy":
  		fmt.Println("Project mode:      legacy (CLAUDE.md/.plans found, no .agent/) — fully compatible, no action required")
  	case "none":
  		fmt.Println("Project mode:      none — not yet initialized (`eng init` to enable)")
  	case "broken":
  		fmt.Printf("Project mode:      BROKEN — %s exists but failed to parse: %v\n", project.ConfigPath, modeResult.ParseErr)
  	default:
  		fmt.Printf("Project mode:      %s (.agent/project.yaml present)\n", modeResult.Mode)
  	}

  	if cfg, err := project.Load(dir); err == nil {
  		fmt.Printf("Detected stack:    %s\n", cfg.Stack.Type)
  	}

  	report, err := skillvalidate.Validate(filepath.Join(hDir, "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
  	if err == nil {
  		broken := 0
  		for _, issue := range report.Errors() {
  			if strings.Contains(issue.Message, "cycle") || strings.Contains(issue.Message, "requires unknown") {
  				broken++
  			}
  		}
  		fmt.Println("\nSkills:")
  		fmt.Printf("  %d discovered\n", report.Discovered)
  		fmt.Printf("  %d valid\n", report.Discovered-len(issuedSkillNames(report.Issues)))
  		fmt.Printf("  %d warning(s)\n", len(report.Warnings()))
  		fmt.Printf("  %d broken dependency issue(s)\n", broken)
  		fmt.Println("  (run `eng skills list` or `eng skills validate` for detail)")
  	}

  	fmt.Println("\nCapabilities:")
  	for _, name := range capabilities.Known {
  		status := "unavailable"
  		if capabilities.Detect(name) {
  			status = "available"
  		}
  		fmt.Printf("  %-10s %s\n", name, status)
  	}
  }

  // issuedSkillNames counts distinct skills with at least one issue, so
  // doctor's "valid" count doesn't double-subtract a skill with two
  // warnings.
  func issuedSkillNames(issues []skillvalidate.Issue) map[string]bool {
  	out := map[string]bool{}
  	for _, i := range issues {
  		if i.Skill != "(graph)" {
  			out[i.Skill] = true
  		}
  	}
  	return out
  }
  ```

**Verify:** `cd cli && go build ./... && go vet ./... && echo OK`.

---

## Task 10 — Representative skill set (9 new `SKILL.md` files)

Create each file exactly as shown. All nine share the frontmatter shape from
`harness/skills/engineering/karpathy-guidelines/SKILL.md`, extended with the new `level`,
`requires`, `recommends`, `capabilities` fields (empty `capabilities: []` in every file below
— none of the nine needs it, it exists for future skills).

- [x] **10.1** Create `harness/skills/engineering/debugging/SKILL.md`:

  ```markdown
  ---
  name: debugging
  domain: engineering
  level: engineering
  description: Systematic root-cause debugging methodology — reproduce, isolate, form a hypothesis, verify — independent of language or platform.
  tags: [debugging, root-cause, troubleshooting]
  triggers: [debug, bug, crash, broken, "not working", error, fails, reproduce]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Any time something behaves unexpectedly and the cause isn't already known.
  when_not_to_use: A requirement that's still being designed — that's planning, not debugging.
  ---

  # Skill: debugging

  ## Purpose

  A domain-independent method for finding the actual cause of a defect instead of guessing at
  fixes.

  ## Method

  1. **Reproduce** — get a reliable, minimal repro before touching any code.
  2. **Isolate** — bisect the change/input space until the smallest failing case is known.
  3. **Form a hypothesis** — state what you believe is wrong and how you'd know if you're right.
  4. **Verify** — test the hypothesis directly (a log line, a debugger breakpoint, a targeted
     assertion) before writing the fix.
  5. **Fix at the cause**, not the symptom — and add a regression test that would have caught it.

  ## Anti-patterns

  - Changing code to see if it helps without a hypothesis.
  - Fixing the first suspicious-looking line instead of the line the reproduction actually
    implicates.
  - Declaring victory because the symptom disappeared, without knowing why.
  ```

- [x] **10.2** Create `harness/skills/engineering/testing/SKILL.md`:

  ```markdown
  ---
  name: testing
  domain: engineering
  level: engineering
  description: What makes a test worth writing — deterministic, isolated, and testing behavior rather than implementation.
  tags: [testing, unit-test, regression, verification]
  triggers: [test, testing, "unit test", "regression test", coverage]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Writing or reviewing any automated test, in any language or framework.
  when_not_to_use: Manual/exploratory QA sessions — this skill is about automated, repeatable tests.
  ---

  # Skill: testing

  ## Purpose

  Domain-independent rules for what makes a test worth keeping in a suite, as opposed to a
  test that merely exists.

  ## Rules

  1. **Deterministic** — no real network calls, no wall-clock time, no unseeded randomness. A
     flaky test is worse than no test: it teaches people to ignore red CI.
  2. **Isolated** — a test doesn't depend on another test's side effects or run order.
  3. **Test behavior, not implementation** — a test that breaks every time a private function
     is renamed is testing the wrong thing.
  4. **One clear failure message** — when it fails, the assertion should say what was expected
     and what happened, not just "false is not true."
  5. **A regression test accompanies every real bug fix** — the test should fail against the
     old code and pass against the fix.

  ## Anti-patterns

  - Asserting on incidental output (log text, formatting) instead of the actual contract.
  - A test suite that takes so long nobody runs it locally.
  - Mocking so much of the system that the test no longer exercises real integration points.
  ```

- [x] **10.3** Create `harness/skills/software/cpp/SKILL.md`:

  ```markdown
  ---
  name: cpp
  domain: software
  level: technology
  description: C++ pitfalls worth checking before trusting a change — ownership, lifetime, and undefined behavior over syntax preference.
  tags: [cpp, "c++", memory, raii]
  triggers: [cpp, "c++", raii, "undefined behavior", segfault, "memory leak"]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Reviewing or writing C++ code, especially anything touching raw pointers, manual memory management, or object lifetime.
  when_not_to_use: Pure build-system or packaging questions with no C++ source involved.
  ---

  # Skill: cpp

  ## Purpose

  The C++-specific checks that matter more than style: who owns this memory, how long does it
  live, and does anything here invoke undefined behavior.

  ## Checklist

  1. **Ownership** — for every `new`/raw pointer, who deletes it, and can that path be skipped
     by an early return or an exception? Prefer RAII (`unique_ptr`, `shared_ptr`, containers)
     over manual `delete`.
  2. **Lifetime** — does a reference or pointer outlive the object it points to (a common bug
     with lambda captures and container reallocation)?
  3. **Undefined behavior** — signed overflow, use-after-move, uninitialized reads, and
     out-of-bounds access don't reliably crash; they corrupt silently. Treat compiler
     warnings (`-Wall -Wextra`) about these as blocking, not cosmetic.
  4. **Const-correctness** communicates intent — a non-const reference parameter that's never
     mutated is a readability bug waiting to become a real one.

  ## Anti-patterns

  - Catching `...` and swallowing the exception without knowing what it was.
  - Storing a reference to a `std::vector` element across a push_back that could reallocate.
  - Using `reinterpret_cast` where `static_cast` (or no cast at all) would do.
  ```

- [x] **10.4** Create `harness/skills/embedded/esp32/SKILL.md`:

  ```markdown
  ---
  name: esp32
  domain: embedded
  level: technology
  description: ESP32-specific constraints — dual-core scheduling, watchdogs, flash wear, and peripheral pin conflicts — before assuming general embedded advice applies.
  tags: [esp32, embedded, microcontroller]
  triggers: [esp32, "esp-32", esp-idf, arduino]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Any firmware task targeting an ESP32 (Arduino framework or ESP-IDF).
  when_not_to_use: A different microcontroller family — this skill's specifics (dual-core, WiFi/BT coexistence, flash partitioning) don't transfer directly.
  ---

  # Skill: esp32

  ## Purpose

  The constraints specific to the ESP32 that generic "embedded systems" advice glosses over.

  ## Constraints to check

  1. **Dual-core** — WiFi/BT stack runs pinned tasks on core 0 by default; blocking user code
     on the wrong core can starve the radio stack. Know which core a task runs on before
     diagnosing a "random" disconnect.
  2. **Watchdog timers** — a long blocking loop (or a busy spin with no yield) trips the task
     watchdog and reboots the device; yield (`vTaskDelay`, `yield()`) inside any loop that
     might run long.
  3. **Flash wear** — frequent writes to the same NVS/SPIFFS/LittleFS region wear out flash;
     batch writes, or use wear-leveling storage, for anything written more than occasionally.
  4. **Pin conflicts** — several "general purpose" pins are strapping pins or shared with
     flash/PSRAM on many ESP32 variants; a pinout that looks free on paper can still be
     unusable on the actual board revision.

  ## Anti-patterns

  - Assuming `delay()` inside a FreeRTOS task is harmless — it blocks that task, not the CPU,
    but a task blocked too long still trips its own watchdog.
  - Writing to NVS every loop iteration instead of only on a real state change.
  ```

- [x] **10.5** Create `harness/skills/automation/plc/SKILL.md`:

  ```markdown
  ---
  name: plc
  domain: automation
  level: domain
  description: Vendor-agnostic PLC methodology — scan cycles, IO addressing conventions, and safety-relevant state — before any vendor-specific detail.
  tags: [plc, ladder-logic, automation, "scan cycle"]
  triggers: [plc, "ladder logic", "programmable logic controller", "scan cycle"]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Any task involving a PLC, regardless of vendor — load this before a vendor-specific skill like automation/siemens-s7.
  when_not_to_use: A microcontroller-only project with no PLC in the loop.
  ---

  # Skill: plc

  ## Purpose

  The methodology every PLC shares, independent of Siemens/Allen-Bradley/Delta/etc. — load
  this first, then a vendor skill for the specifics.

  ## Method

  1. **Scan cycle** — a PLC repeatedly reads inputs, executes the whole program, then writes
     outputs, in that order, every cycle. Code that assumes an output changes mid-cycle
     (instead of at the next scan) is a common source of confusing behavior.
  2. **Addressing conventions are not universal** — the same physical register can be
     addressed differently across vendors and even across protocol layers on the same
     vendor's own hardware (see automation/modbus's PDU-vs-reference-number gotcha). Never
     assume an address maps directly without checking the specific vendor/protocol
     convention in use.
  3. **Safety-relevant state** (E-stops, interlocks, watchdog outputs) is generally
     fail-safe-by-design at the hardware level — don't "fix" a safety circuit in software
     without understanding why it was wired the way it was.
  4. **IO simulation before hardware** — when available, test logic changes against simulated
     IO before writing to a live process; a PLC output can move real machinery.

  ## Anti-patterns

  - Assuming an output takes effect immediately rather than at the next scan boundary.
  - Copy-pasting a register address from one project to another without re-verifying the
    addressing convention for the new vendor/protocol.
  ```

- [x] **10.6** Create `harness/skills/automation/modbus/SKILL.md`:

  ```markdown
  ---
  name: modbus
  domain: automation
  level: technology
  description: Modbus addressing and framing pitfalls — PDU addresses are not the same as the conventional 4xxxx/3xxxx register numbers, and RTU vs TCP framing differs.
  tags: [modbus, "modbus tcp", "modbus rtu", register]
  triggers: [modbus, "modbus tcp", "modbus rtu", "holding register", "40001"]
  version: "1.0.0"
  requires: []
  recommends: [networking/tcp-ip]
  capabilities: []
  conflicts: []
  when_to_use: Reading or writing Modbus registers/coils over either TCP or serial (RTU/ASCII).
  when_not_to_use: A protocol that merely resembles Modbus in concept (e.g. a proprietary polling protocol) — don't assume Modbus-specific addressing applies.
  ---

  # Skill: modbus

  ## Purpose

  The addressing and framing details that cause almost every real Modbus bug.

  ## Method

  1. **PDU address ≠ conventional register number.** "Read holding register 40001" in vendor
     documentation almost always means PDU address `0x0000` (the conventional numbering is
     1-indexed and offset by a function-code range: 40001 → PDU 0, 40002 → PDU 1, ...). Do not
     assume the literal number in a spec sheet is the value to send on the wire — check which
     convention the specific device/library uses.
  2. **Function code determines the address space.** Coils, discrete inputs, input registers,
     and holding registers are four separate address spaces that can each start at 0 —
     "register 5" is ambiguous without knowing which function code (space) it's in.
  3. **RTU vs TCP framing differs.** RTU adds a slave/unit ID and CRC over serial; TCP wraps
     the same PDU in an MBAP header with a transaction ID instead of a CRC (TCP already
     guarantees byte integrity). A library written for one doesn't automatically speak the
     other.
  4. **Byte/word order (endianness) for multi-register values is not standardized** — a 32-bit
     value spanning two registers can be big-endian, little-endian, or word-swapped depending
     on the vendor; verify against the specific device's documentation rather than assuming.

  ## Anti-patterns

  - Blindly assuming a spec's "40001" is the literal address to request.
  - Assuming a Modbus TCP library will work unmodified against RTU hardware (or vice versa).
  ```

- [x] **10.7** Create `harness/skills/automation/siemens-s7/SKILL.md`:

  ```markdown
  ---
  name: siemens-s7
  domain: automation
  level: technology
  description: Siemens S7 (S7-1200/1500/300/400) specifics — DB/data-block addressing and the S7comm/S7comm-plus protocol split — layered on general PLC methodology.
  tags: [siemens, s7, s7-1200, s7-1500, tia-portal]
  triggers: [siemens, s7, "s7-1200", "s7-1500", "tia portal", s7comm]
  version: "1.0.0"
  requires: [automation/plc]
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Any task targeting a Siemens S7-family PLC specifically.
  when_not_to_use: A different PLC vendor — load automation/plc plus that vendor's own skill instead.
  ---

  # Skill: siemens-s7

  ## Purpose

  What's specific to Siemens S7 hardware, on top of the vendor-agnostic PLC methodology in
  `automation/plc` (a hard dependency of this skill — always loaded alongside it).

  ## Specifics

  1. **Data blocks (DBs) are the primary addressing unit** — a tag lives at `DBx.DBWy`/
     `DBx.DBXy.z` (block number, then byte/bit offset), not a flat global address space. Two
     different DB numbers with the same offset are unrelated memory.
  2. **S7-1200/1500 default to "optimized" block access**, which hides absolute byte offsets
     unless the DB's optimized-access setting is explicitly disabled in TIA Portal — a
     symbolic tag name in optimized blocks has no fixed byte offset to hand to an external
     tool.
  3. **Protocol split** — older S7-300/400 and S7-1200/1500 in compatible mode speak S7comm;
     newer S7-1500 (and S7-1200 by default since firmware V4) prefer S7comm-plus, which adds
     session-based authentication and isn't a drop-in wire-compatible replacement for tools
     built against S7comm.
  4. **This is a real S7-family PLC** — everything in `automation/plc` (scan cycle,
     addressing conventions are not universal, safety-relevant state) still applies
     underneath.

  ## Anti-patterns

  - Assuming a symbolic tag name maps to a fixed byte offset without checking the DB's
    optimized-access setting.
  - Reusing an S7comm client library against an S7-1500 configured for S7comm-plus-only
    access and assuming a connection failure means the network is broken.
  ```

- [x] **10.8** Create `harness/skills/networking/tcp-ip/SKILL.md`:

  ```markdown
  ---
  name: tcp-ip
  domain: networking
  level: technology
  description: TCP/IP fundamentals that explain most "intermittent" network bugs — connection state, timeouts, and the difference between a closed port and an unreachable host.
  tags: [tcp, ip, networking, socket, timeout]
  triggers: [tcp, "tcp/ip", socket, timeout, "connection refused", "connection reset"]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Debugging or designing anything that opens a TCP connection — a service, a device integration, an API client.
  when_not_to_use: A purely UDP-based or link-layer-only problem — TCP's specific guarantees (ordering, retransmission, connection state) don't apply.
  ---

  # Skill: tcp-ip

  ## Purpose

  The handful of TCP/IP facts that explain most bugs reported as "sometimes it just doesn't
  connect."

  ## Facts worth checking

  1. **"Connection refused" vs "timeout" mean different things.** Refused means a host
     responded and nothing is listening on that port (or a firewall actively rejected it,
     depending on config) — the network path works. A timeout with no response at all usually
     means the host is unreachable, or a firewall is silently dropping packets, not that the
     remote service is merely slow.
  2. **TCP is a stream, not a message protocol.** One `write()` is not guaranteed to arrive as
     one `read()` — application-level framing (a length prefix, a delimiter, a fixed record
     size) is the caller's responsibility, not TCP's.
  3. **Keep-alive is not the same as an application-level heartbeat.** OS-level TCP keep-alive
     can take minutes to detect a dead peer by default; if a protocol needs to detect a stale
     connection quickly, it needs its own heartbeat/timeout at the application layer.
  4. **A half-open connection can look alive** — if one side crashes without sending a FIN
     (e.g. power loss on an embedded device), the other side may not notice until it tries to
     write and gets a reset, or until a read timeout fires.

  ## Anti-patterns

  - Retrying a "connection refused" in a tight loop assuming it's transient, without checking
    whether the target service is actually running.
  - Assuming a single `write()` call corresponds to a single `read()` on the other end.
  ```

- [x] **10.9** Create `harness/skills/devops/docker/SKILL.md`:

  ```markdown
  ---
  name: docker
  domain: devops
  level: technology
  description: Docker layering, build-cache invalidation, and the difference between an image problem and a container-runtime problem.
  tags: [docker, container, dockerfile, "build cache"]
  triggers: [docker, dockerfile, container, "docker compose", "build cache"]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Writing/debugging a Dockerfile, a container that behaves differently than the same code run locally, or a slow/broken image build.
  when_not_to_use: A pure Kubernetes orchestration question with no Dockerfile/image concern.
  ---

  # Skill: docker

  ## Purpose

  The layering and caching model that explains most "works on my machine, fails in the
  container" and "the build is slow" complaints.

  ## Method

  1. **Layer order determines cache invalidation.** Every instruction is a layer; changing a
     line invalidates every layer after it. Put the least-frequently-changing steps
     (installing system/OS packages) before the most-frequently-changing ones (copying
     application source) so an unrelated code change doesn't force a full dependency
     reinstall.
  2. **"Works locally, fails in the container" is almost always an environment difference**,
     not a Docker bug — check base image OS/library versions, working directory, environment
     variables, and file permissions (a file copied with `COPY` can end up owned by a
     different user than the one the container runs as) before suspecting Docker itself.
  3. **A container's filesystem changes are ephemeral** unless a volume/bind-mount is used —
     data written inside a container without one disappears when the container is removed,
     not just when it stops.
  4. **`docker build` context** includes everything not excluded by `.dockerignore` — a
     surprisingly slow build is often caused by an unintentionally large build context (e.g.
     a `node_modules` or `.git` directory being sent to the daemon).

  ## Anti-patterns

  - Debugging a "container-only" failure by staring at the Dockerfile before checking whether
    it's actually an environment/version difference.
  - Storing state a service needs to survive a restart with no volume behind it.
  ```

- [x] **10.10** Create `harness/skills/it/linux/SKILL.md`:

  ```markdown
  ---
  name: linux
  domain: it
  level: technology
  description: Linux troubleshooting fundamentals — where to look for the actual error (logs, exit codes, permissions) before assuming a tool is broken.
  tags: [linux, systemd, permissions, logs]
  triggers: [linux, systemd, journalctl, permission, "exit code", syslog]
  version: "1.0.0"
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  when_to_use: Diagnosing a failure on a Linux host — a service that won't start, a permission error, an unexplained process exit.
  when_not_to_use: A pure application-logic bug with no OS/system interaction involved.
  ---

  # Skill: linux

  ## Purpose

  Where the actual cause of a Linux-level failure is almost always visible, before guessing.

  ## Method

  1. **Check the exit code and `journalctl`/`syslog` before anything else.** A service that
     "just doesn't start" almost always logged why; `systemctl status <unit>` and
     `journalctl -u <unit> -n 50` answer more questions than re-reading the config file does.
  2. **Permission errors name the actual problem in the error text** — "Permission denied" vs
     "No such file or directory" vs "Operation not permitted" are different failure classes
     (file ownership/mode, a wrong path, or a capability/SELinux/AppArmor restriction
     respectively); read which one it actually is before changing permissions blindly.
  3. **A process that dies with no log output** may have been OOM-killed — check `dmesg` /
     `journalctl -k` for an OOM-killer entry before assuming the application crashed on its
     own.
  4. **`$PATH` and shell environment differ between an interactive login shell and a
     service's environment** — a command that works when typed manually can fail
     identically-named but differently-resolved under systemd/cron with a minimal
     environment.

  ## Anti-patterns

  - Changing file permissions to `777` to "fix" a permission error without reading which
    specific permission was actually missing.
  - Assuming a silently-dead process crashed in application code before checking for an
    OOM-kill.
  ```

**Verify:** `cd cli && go build -o eng . && ./eng skills list` — expect 11 lines (the ten
above plus the existing `karpathy-guidelines`), and `./eng skills validate` — expect
`11 skill(s) discovered, 0 error(s), 0 warning(s)`.

---

## Task 11 — `internal/skilleval` + router evaluation scenarios

- [x] **11.1** Create `cli/internal/skilleval/skilleval.go`:

  ```go
  package skilleval

  import (
  	"io/fs"
  	"os"
  	"path/filepath"
  	"strings"

  	"gopkg.in/yaml.v3"
  )

  // Scenario is one router evaluation case — a small, deterministic
  // foundation (Phase 6 Requirement 16), not an LLM benchmark: it only
  // asserts which skills the router selects for a given request, never
  // anything about model output.
  type Scenario struct {
  	Name           string   `yaml:"name"`
  	Request        string   `yaml:"request"`
  	ExpectedSkills []string `yaml:"expected_skills"`
  	Notes          string   `yaml:"notes,omitempty"`
  }

  // LoadScenarios walks root for *.yaml files and parses each as one
  // Scenario. A missing root is not an error — it returns an empty slice.
  func LoadScenarios(root string) ([]Scenario, error) {
  	if _, err := os.Stat(root); os.IsNotExist(err) {
  		return nil, nil
  	}
  	var out []Scenario
  	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
  		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
  			return err
  		}
  		data, err := os.ReadFile(path)
  		if err != nil {
  			return err
  		}
  		var s Scenario
  		if err := yaml.Unmarshal(data, &s); err != nil {
  			return err
  		}
  		if s.Name == "" {
  			s.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
  		}
  		out = append(out, s)
  		return nil
  	})
  	return out, err
  }
  ```

- [x] **11.2** Create `cli/internal/skilleval/skilleval_test.go`:

  ```go
  package skilleval

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func TestLoadScenariosParsesFields(t *testing.T) {
  	dir := t.TempDir()
  	os.WriteFile(filepath.Join(dir, "example.yaml"), []byte("name: example\nrequest: \"debug a c++ build\"\nexpected_skills: [debugging, cpp]\n"), 0o644)
  	scenarios, err := LoadScenarios(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(scenarios) != 1 || scenarios[0].Request != "debug a c++ build" || len(scenarios[0].ExpectedSkills) != 2 {
  		t.Fatalf("got %+v", scenarios)
  	}
  }

  func TestLoadScenariosMissingRootIsNotError(t *testing.T) {
  	scenarios, err := LoadScenarios(filepath.Join(t.TempDir(), "nope"))
  	if err != nil || len(scenarios) != 0 {
  		t.Fatalf("expected empty, no error; got %+v, %v", scenarios, err)
  	}
  }

  func TestLoadScenariosDefaultsNameToFilename(t *testing.T) {
  	dir := t.TempDir()
  	os.WriteFile(filepath.Join(dir, "unnamed.yaml"), []byte("request: \"x\"\nexpected_skills: []\n"), 0o644)
  	scenarios, err := LoadScenarios(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(scenarios) != 1 || scenarios[0].Name != "unnamed" {
  		t.Fatalf("got %+v", scenarios)
  	}
  }
  ```

- [x] **11.3** Create `harness/evals/embedded/esp32-siemens-modbus.yaml`:

  ```yaml
  name: esp32-siemens-modbus
  request: "ESP32 reads Siemens S7-1200 over Modbus TCP"
  expected_skills:
    - esp32
    - siemens-s7
    - plc
    - modbus
    - tcp-ip
  notes: >
    The headline example from the Phase 6 instruction. plc is pulled in only via
    siemens-s7's requires:; tcp-ip is matched both directly (the request text contains
    "TCP") and via modbus's recommends: — either path is sufficient.
  ```

- [x] **11.4** Create `harness/evals/engineering/cpp-debug.yaml`:

  ```yaml
  name: cpp-debug
  request: "Debug a C++ build issue"
  expected_skills:
    - debugging
    - cpp
  ```

- [x] **11.5** Create `harness/evals/devops/docker-linux-ci.yaml`:

  ```yaml
  name: docker-linux-ci
  request: "Set up Docker on Linux for CI"
  expected_skills:
    - docker
    - linux
  ```

- [x] **11.6** Create `cli/skilleval_integration_test.go`:

  ```go
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
  ```

  (`filepath.Join("..", "harness", ...)` resolves correctly because `go test` runs with the
  package directory — `cli/` — as its working directory, so `../harness` is this repo's real
  `harness/` tree, the same relative-path convention already implicit throughout this
  project's `$REPO/cli/eng` bash-test invocations.)

**Verify:** `cd cli && go build ./... && go test ./internal/skilleval/... -v && go test ./... -run TestRouterEvalScenarios -v`.

---

## Task 12 — `main.go` usage string

- [x] **12.1** In `cli/main.go`'s `usage()`, update the skills line and add the new one:

  Old:
  ```
    skills list             List resolved skills (global + project-local)
  ```

  New:
  ```
    skills list             List resolved skills (global + private + project-local)
    skills validate         Check skill metadata/dependencies for issues (exit 1 on errors)
  ```

**Verify:** `cd cli && go build ./... && echo BUILD_OK`.

---

## Task 13 — `context-manager/METHOD.md`: one-sentence pointer to the router

- [x] **13.1** In `harness/core/context-manager/METHOD.md`, add one paragraph under
  "## What it does":

  Old:
  ```markdown
  ## What it does

  `eng context bundle <role> <plan-dir> ["<request text>"]` composes:
  ```

  New:
  ```markdown
  ## What it does

  Skill selection itself is delegated to `internal/skillrouter.Route` (Phase 6) — explicit
  project skills and their dependencies first, then request matches, domain-profile fills,
  and recommendations, all budget-aware and explained. This is still the *only* place skill
  selection happens; no role prompt or adapter re-implements it.

  `eng context bundle <role> <plan-dir> ["<request text>"]` composes:
  ```

**Verify:** manual read — no code to build.

---

## Task 14 — `docs/skills.md`: the skill authoring/routing guide

- [x] **14.1** Create `docs/skills.md`:

  ```markdown
  # Skills — levels, metadata, routing, and how to add one

  ## What a skill is

  One `SKILL.md` file under `harness/skills/<domain>/<skill-name>/` (or a project's own
  `skills/<domain>/<skill-name>/`), YAML frontmatter plus a markdown body. The frontmatter is
  what the harness reads mechanically; the body is what a role reads once the skill is
  selected.

  ## The three levels

  - **`level: engineering`** — reusable across every domain (`engineering/debugging`,
    `engineering/testing`). `domain: engineering`.
  - **`level: domain`** — methodology specific to one domain but not one vendor/technology
    (`automation/plc`).
  - **`level: technology`** — a specific technology, protocol, or vendor
    (`automation/modbus`, `automation/siemens-s7`, `embedded/esp32`).

  `level` is metadata for humans and `eng skills list`; nothing in the router currently
  branches on it directly (routing runs on tags/triggers/domain/requires/recommends).

  ## What a skill pack is

  A directory under `harness/skills/<domain>/` (or a project's `skills/<domain>/`, or a
  private root — see below) containing one or more related skills. There's no separate
  manifest file for a "pack" — the domain directory itself is the pack.

  ## Metadata schema (all optional except `name`)

  ```yaml
  name: modbus                        # required — the skill's identity within its domain
  domain: automation                  # groups the skill; also namespaces it (see below)
  level: technology                   # engineering | domain | technology
  description: One sentence.
  tags: [modbus, register]            # scored against a request's text
  triggers: [modbus, "40001"]         # scored against a request's text
  version: "1.0.0"
  requires: []                        # hard dependency — always included when this skill is
  recommends: [networking/tcp-ip]     # soft — included only if the context budget allows
  capabilities: []                    # free-form, reserved for future capability-based routing
  conflicts: []                       # surfaced by `eng skills validate`, not enforced by the router
  when_to_use: ...
  when_not_to_use: ...
  ```

  A skill with **no frontmatter at all** still resolves via the legacy `# Skill: name` /
  `## Purpose` heading convention (`domain: unknown`, every new field at its zero value) —
  nothing here requires migrating an old skill.

  ## Namespacing and collisions

  A skill's *qualified name* is `domain/name` (e.g. `automation/modbus`) unless `name:`
  already contains a `/` (a self-namespaced name like `company/internal-api`) or `domain` is
  empty/`unknown` (a legacy skill, which stays addressed by its bare name). Two skills in
  different domains may share a bare name on purpose — `automation/modbus` and a future
  `networking/modbus` would not collide. `eng skills validate` warns only when the exact same
  qualified name is authored twice under one source root — a real mistake.

  ## Dependencies

  `requires:` is transitive and never dropped by the context budget, even when the skill
  that needed it was — a `requires` edge means "this skill's advice doesn't make sense
  without the other one," not "this is loosely related." Use `recommends:` for that instead.
  Cycles and unknown targets are hard errors from `eng skills validate` and from the router
  itself (`eng context bundle`/`eng adapter prompt` will report the error rather than
  silently drop the broken skill).

  ## Routing precedence

  ```
  explicit project-enabled skills (.agent/project.yaml's enabled_skills)
        ↓  (never dropped)
  required dependencies (transitive)
        ↓  (never dropped)
  strong request matches (tag/trigger/description score > 0)
        ↓  (best score first)
  project domain-profile fills (.agent/project.yaml's domains:)
        ↓
  recommended related skills
        ↓  (dropped first if the budget is tight)
  budget cutoff
  ```

  See it explain itself for any request: `eng context skills "<request text>"`.

  ## Project profiles (`domains:`)

  ```yaml
  # .agent/project.yaml
  domains:
    - embedded
    - automation
  ```

  Optional. Fills the router's domain-profile tier with every skill in those domains that
  wasn't already pulled in by a stronger match. Absent (every project before this field
  existed) means that tier is simply empty — zero behavior change.

  ## Skill sources and precedence

  ```
  global (~/.engineering-harness/skills/, from `eng install`)
        <
  private (optional, .agent/project.yaml's private_skills_path)
        <
  project-local (./skills/)
  ```

  A private pack needs no registry — point `private_skills_path` at any local directory or
  git checkout laid out the same way as `harness/skills/`.

  ## Inspecting routing and validity

  - `eng skills list` — every resolved skill, one line each, with source/domain/level.
  - `eng skills validate` — metadata/dependency issues; exits non-zero only on an error
    (a legacy skill's missing metadata is always a warning, never an error).
  - `eng context skills "<request>"` — what would be selected for a request, and why.
  - `eng doctor` — a four-line skill summary; run the two commands above for detail.

  ## Adding a new skill

  1. `mkdir -p harness/skills/<domain>/<name>` (or `skills/<domain>/<name>` in a project for
     a project-local skill).
  2. Write `SKILL.md` with frontmatter (copy an existing skill under
     `harness/skills/` as a starting template) plus a body explaining the actual method —
     not what the skill does (the `description:` covers that), but how to apply it.
  3. `eng skills validate` — fix anything it flags as an error.
  4. If it composes with another skill, add `requires:`/`recommends:` rather than repeating
     that skill's content.
  5. If it's meant to be shipped globally, it needs to live under this repo's own
     `harness/skills/` and reach users via `eng install --from <path>`; a private/company
     skill never needs to touch this repo at all — see "Skill sources" above.
  ```

**Verify:** manual read — no code to build.

---

## Task 15 — `docs/src-map.md`, `README.md`, `ROADMAP.md`

- [x] **15.1** In `docs/src-map.md`, add a final module section after the Phase 5 entry
  (following the exact established pattern from every prior phase):

  ```markdown

  ### `cli/internal/skillgraph/`, `cli/internal/skillrouter/`, `cli/internal/skillvalidate/`, `cli/internal/skilleval/` — Phase 6 multi-domain skill ecosystem

  What it does: `skillgraph` expands a skill's `requires:` transitively (deterministic order,
  cycle detection, unknown-target errors). `skillrouter.Route` is the one authoritative
  selection path `context_cmd.go`'s `selectSkills` calls — explicit skills, then their
  required dependencies, then request matches, then a project's `domains:` profile fill,
  then `recommends:`, all budget-aware, each with an explanation. `skillvalidate` checks
  metadata/dependency issues (`eng skills validate`), warning-only for legacy skills.
  `skilleval` loads `harness/evals/**/*.yaml` scenarios exercised by a `cli/` integration
  test against the real `harness/skills` tree. `skills.Skill` gained `level`, `requires`
  (renamed from the unused `dependencies`), `recommends`, `capabilities`, and a
  `QualifiedName()` identity (`domain/name`) that `Resolve`/the new `ResolveWithPrivate`
  merge by instead of bare name, preventing cross-domain collisions. `project.Config` gained
  `domains` (plural, combinable domain profile) and `private_skills_path` (a third,
  optional resolution tier between global and project-local).

  Key files: `cli/internal/skillrouter/skillrouter.go`, `cli/internal/skills/skills.go`
  (`QualifiedName`, `ResolveWithPrivate`), `docs/skills.md` (the authoring/routing guide)

  Notable: `Resolve(global, local)` is now `ResolveWithPrivate(global, "", local)` — a pure
  delegation, byte-identical behavior, covered by a regression test — so no pre-Phase-6
  caller needed to change. `eng hooks run <stage> [plan-dir]` gained an optional third
  argument fixing the Phase 5 gotcha where `drift_check`/`verify` only worked when invoked
  from a directory that itself contained `plan.yaml`; omitting the argument defaults to `.`,
  identical to every existing invocation.

  From: `.plans/2026-08-25-v2-harness-phase6-skills/`
  ```

- [x] **15.2** In `README.md`, add a Phase 6 paragraph immediately after the Phase 5 section
  (before the following `---`):

  ```markdown

  Phase 6 turns the single flat skill list into a three-level, multi-domain, dependency-aware
  ecosystem:

  ```bash
  cd cli && go build -o eng .
  ./eng context skills "ESP32 reads Siemens S7-1200 over Modbus TCP"
  ./eng skills validate
  ```

  See `docs/skills.md` for the full skill model and `.plans/2026-08-25-v2-harness-phase6-skills/spec.md`
  for the full design.
  ```

- [x] **15.3** In `ROADMAP.md`, extend the note to include the Phase 6 plan link, following
  the same pattern as the Phase 5 addition:

  Old:
  ```
  > `.plans/2026-08-24-v2-harness-phase4-context/` (context selection,
  > skill/doc/task retrieval, bounded verification output), and
  > `.plans/2026-08-24-v2-harness-phase5-runtime/` (natural-language runtime routing, Quick
  > Fix/Spec-First workflows, log retention, Agent/Tool adapter separation) — see those plans
  > for the current architecture.
  ```

  New:
  ```
  > `.plans/2026-08-24-v2-harness-phase4-context/` (context selection,
  > skill/doc/task retrieval, bounded verification output),
  > `.plans/2026-08-24-v2-harness-phase5-runtime/` (natural-language runtime routing, Quick
  > Fix/Spec-First workflows, log retention, Agent/Tool adapter separation), and
  > `.plans/2026-08-25-v2-harness-phase6-skills/` (multi-domain skill ecosystem, dependency
  > graph, skill router) — see those plans for the current architecture.
  ```

**Verify:** manual read — no code to build.

---

## Task 16 — Version bump (last task)

- [x] **16.1** Update `harness/VERSION`:

  ```
  0.6.0-phase6-skills
  ```

**Verify:** `cat harness/VERSION`.
