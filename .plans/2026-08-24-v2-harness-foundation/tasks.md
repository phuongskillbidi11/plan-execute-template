# Tasks: V2 Harness Foundation

Each task must be completed and its test (see `tests.md`) must pass before moving to the
next. Mark `[x]` when done. Read `spec.md` in full before starting Task 1.

**Prerequisite (blocking, not a task):** confirm `go version` reports Go 1.22+ before starting.
See `tests.md` T0. If Go is not installed, install it first — this is environment setup, not
part of the plan's own tasks.

---

## Task 1 — Go module scaffold and CLI entrypoint

- [x] **1.1** Create `cli/go.mod`:

  ```
  module eng

  go 1.22

  require gopkg.in/yaml.v3 v3.0.1
  ```

- [x] **1.2** Create `cli/main.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  )

  func main() {
  	if len(os.Args) < 2 {
  		usage()
  		os.Exit(1)
  	}

  	switch os.Args[1] {
  	case "install":
  		cmdInstall(os.Args[2:])
  	case "init":
  		cmdInit(os.Args[2:])
  	case "doctor":
  		cmdDoctor(os.Args[2:])
  	case "scan":
  		cmdScan(os.Args[2:])
  	case "skills":
  		cmdSkills(os.Args[2:])
  	default:
  		usage()
  		os.Exit(1)
  	}
  }

  func usage() {
  	fmt.Println(`Usage: eng <command> [args]

  Commands:
    install --from <path>   Install the harness payload into ~/.engineering-harness
    init                    Initialize the current directory as a harness-aware project
    doctor                  Report harness install status, project mode, and resolved skills
    scan                    Print detected stack and a file summary
    skills list             List resolved skills (global + project-local)`)
  }
  ```

- [x] **1.3** Run `cd cli && go build ./...`. This will fail until Tasks 2–6 add the
  referenced functions and internal packages — that is expected at this point. Do not
  attempt to make it compile yet; proceed to Task 2.

---

## Task 2 — Stack detection (`internal/detect`)

- [x] **2.1** Create `cli/internal/detect/detect.go` — a Go port of
  `scripts/detect-project.sh`'s detection table, same priority order (ESP-IDF before generic
  CMake, C# solution before single `.csproj`):

  ```go
  package detect

  import (
  	"os"
  	"path/filepath"
  )

  type Result struct {
  	Type  string
  	Build string
  	Test  string
  	Run   string
  	Lint  string
  }

  func exists(path string) bool {
  	_, err := os.Stat(path)
  	return err == nil
  }

  func hasGlob(dir, pattern string) bool {
  	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
  	return len(matches) > 0
  }

  // Detect scans dir for known project markers, mirroring
  // scripts/detect-project.sh's detection order (first match wins).
  func Detect(dir string) Result {
  	has := func(name string) bool { return exists(filepath.Join(dir, name)) }

  	switch {
  	case has("sdkconfig") || has("idf_component.yml"):
  		return Result{Type: "esp-idf",
  			Build: ". ~/esp/esp-idf/export.sh && idf.py build",
  			Test:  ". ~/esp/esp-idf/export.sh && idf.py build 2>&1 | tail -5",
  			Run:   ". ~/esp/esp-idf/export.sh && idf.py flash monitor"}
  	case has("Cargo.toml"):
  		return Result{Type: "rust", Build: "cargo build", Test: "cargo test",
  			Run: "cargo run", Lint: "cargo clippy -- -D warnings"}
  	case has("package.json"):
  		pm := "npm"
  		if has("pnpm-lock.yaml") {
  			pm = "pnpm"
  		} else if has("yarn.lock") {
  			pm = "yarn"
  		}
  		return Result{Type: "nodejs", Build: pm + " run build", Test: pm + " test",
  			Run: pm + " start", Lint: pm + " run lint"}
  	case has("pyproject.toml") || has("setup.py") || has("requirements.txt"):
  		return Result{Type: "python", Build: "pip install -e .", Test: "pytest -x",
  			Run: "python main.py", Lint: "ruff check ."}
  	case has("go.mod"):
  		return Result{Type: "go", Build: "go build ./...", Test: "go test ./...",
  			Run: "go run .", Lint: "golangci-lint run"}
  	case hasGlob(dir, "*.sln"):
  		return Result{Type: "csharp", Build: "dotnet build", Test: "dotnet test",
  			Lint: "dotnet format --verify-no-changes"}
  	case hasGlob(dir, "*.csproj"):
  		return Result{Type: "csharp", Build: "dotnet build", Test: "dotnet test",
  			Lint: "dotnet format --verify-no-changes", Run: "dotnet run"}
  	case has("CMakeLists.txt"):
  		return Result{Type: "c-cpp", Build: "cmake -B build && cmake --build build",
  			Test: "ctest --test-dir build"}
  	case has("Makefile") || has("makefile"):
  		return Result{Type: "make", Build: "make", Test: "make test", Run: "make run"}
  	default:
  		return Result{Type: "unknown"}
  	}
  }
  ```

- [x] **2.2** Create `cli/internal/detect/detect_test.go`:

  ```go
  package detect

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func TestDetectGo(t *testing.T) {
  	dir := t.TempDir()
  	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
  		t.Fatal(err)
  	}
  	got := Detect(dir)
  	if got.Type != "go" {
  		t.Fatalf("expected type=go, got %q", got.Type)
  	}
  }

  func TestDetectUnknown(t *testing.T) {
  	dir := t.TempDir()
  	got := Detect(dir)
  	if got.Type != "unknown" {
  		t.Fatalf("expected type=unknown, got %q", got.Type)
  	}
  }
  ```

---

## Task 3 — Project config (`internal/project`)

- [x] **3.1** Create `cli/internal/project/project.go`:

  ```go
  package project

  import (
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"
  )

  type Stack struct {
  	Type  string `yaml:"type"`
  	Build string `yaml:"build_cmd"`
  	Test  string `yaml:"test_cmd"`
  	Run   string `yaml:"run_cmd"`
  	Lint  string `yaml:"lint_cmd"`
  }

  type Config struct {
  	ProjectName    string   `yaml:"project_name"`
  	Mode           string   `yaml:"mode"` // legacy | hybrid | modern
  	HarnessProfile string   `yaml:"harness_profile"`
  	Stack          Stack    `yaml:"stack"`
  	EnabledSkills  []string `yaml:"enabled_skills"`
  }

  const ConfigPath = ".agent/project.yaml"

  func Load(dir string) (*Config, error) {
  	data, err := os.ReadFile(filepath.Join(dir, ConfigPath))
  	if err != nil {
  		return nil, err
  	}
  	var cfg Config
  	if err := yaml.Unmarshal(data, &cfg); err != nil {
  		return nil, err
  	}
  	return &cfg, nil
  }

  func Save(dir string, cfg *Config) error {
  	data, err := yaml.Marshal(cfg)
  	if err != nil {
  		return err
  	}
  	full := filepath.Join(dir, ConfigPath)
  	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
  		return err
  	}
  	return os.WriteFile(full, data, 0o644)
  }

  // DetectMode reports the project's mode without requiring .agent/ to exist:
  //   - .agent/project.yaml present -> its own "mode" field (default "hybrid" if unset)
  //   - CLAUDE.md or .plans/ present, no .agent/ -> "legacy"
  //   - neither -> "none" (not yet initialized)
  func DetectMode(dir string) string {
  	if _, err := os.Stat(filepath.Join(dir, ConfigPath)); err == nil {
  		if cfg, loadErr := Load(dir); loadErr == nil && cfg.Mode != "" {
  			return cfg.Mode
  		}
  		return "hybrid"
  	}
  	_, claudeErr := os.Stat(filepath.Join(dir, "CLAUDE.md"))
  	_, plansErr := os.Stat(filepath.Join(dir, ".plans"))
  	if claudeErr == nil || plansErr == nil {
  		return "legacy"
  	}
  	return "none"
  }
  ```

- [x] **3.2** Create `cli/internal/project/project_test.go`:

  ```go
  package project

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func TestDetectModeLegacy(t *testing.T) {
  	dir := t.TempDir()
  	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# x"), 0o644)
  	os.Mkdir(filepath.Join(dir, ".plans"), 0o755)
  	if got := DetectMode(dir); got != "legacy" {
  		t.Fatalf("expected legacy, got %q", got)
  	}
  }

  func TestDetectModeNone(t *testing.T) {
  	dir := t.TempDir()
  	if got := DetectMode(dir); got != "none" {
  		t.Fatalf("expected none, got %q", got)
  	}
  }

  func TestSaveLoadRoundTrip(t *testing.T) {
  	dir := t.TempDir()
  	cfg := &Config{ProjectName: "x", Mode: "modern", Stack: Stack{Type: "go"}}
  	if err := Save(dir, cfg); err != nil {
  		t.Fatal(err)
  	}
  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got.Mode != "modern" || got.Stack.Type != "go" {
  		t.Fatalf("round-trip mismatch: %+v", got)
  	}
  }
  ```

---

## Task 4 — Skill resolution (`internal/skills`)

- [x] **4.1** Create `cli/internal/skills/skills.go`:

  ```go
  package skills

  import (
  	"io/fs"
  	"os"
  	"path/filepath"
  	"strings"

  	"gopkg.in/yaml.v3"
  )

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

  // ParseSkillFile reads one SKILL.md. It prefers YAML frontmatter; if none is
  // present it falls back to the legacy "# Skill: name" + "## Purpose" convention
  // used by scripts/update-manifest.sh, so pre-V2 project skills keep resolving.
  func ParseSkillFile(path string) (Skill, error) {
  	data, err := os.ReadFile(path)
  	if err != nil {
  		return Skill{}, err
  	}
  	content := string(data)

  	if strings.HasPrefix(content, "---\n") {
  		if end := strings.Index(content[4:], "\n---"); end >= 0 {
  			var s Skill
  			if err := yaml.Unmarshal([]byte(content[4:4+end]), &s); err == nil && s.Name != "" {
  				s.Path = path
  				return s, nil
  			}
  		}
  	}

  	return parseLegacy(content, path), nil
  }

  func parseLegacy(content, path string) Skill {
  	var name, desc string
  	inPurpose := false
  	for _, line := range strings.Split(content, "\n") {
  		trimmed := strings.TrimSpace(line)
  		switch {
  		case strings.HasPrefix(trimmed, "# Skill:"):
  			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "# Skill:"))
  		case trimmed == "## Purpose":
  			inPurpose = true
  		case inPurpose && strings.HasPrefix(trimmed, "## "):
  			inPurpose = false
  		case inPurpose && trimmed != "" && desc == "":
  			desc = trimmed
  		}
  	}
  	return Skill{Name: name, Description: desc, Domain: "unknown", Path: path}
  }

  // Walk finds every SKILL.md under root and parses it. A missing root is not
  // an error — it returns an empty slice.
  func Walk(root string) ([]Skill, error) {
  	if _, err := os.Stat(root); os.IsNotExist(err) {
  		return nil, nil
  	}
  	var out []Skill
  	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
  		if err != nil || d.IsDir() {
  			return nil
  		}
  		if strings.EqualFold(filepath.Base(path), "SKILL.md") {
  			if s, parseErr := ParseSkillFile(path); parseErr == nil && s.Name != "" {
  				out = append(out, s)
  			}
  		}
  		return nil
  	})
  	return out, err
  }

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

- [x] **4.2** Create `cli/internal/skills/skills_test.go`:

  ```go
  package skills

  import (
  	"os"
  	"path/filepath"
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

  func TestParseFrontmatter(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "modbus", "---\nname: modbus\ndomain: automation\ndescription: Modbus knowledge\n---\n\nbody\n")
  	skills, err := Walk(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(skills) != 1 || skills[0].Name != "modbus" || skills[0].Domain != "automation" {
  		t.Fatalf("got %+v", skills)
  	}
  }

  func TestParseLegacyHeading(t *testing.T) {
  	dir := t.TempDir()
  	writeSkill(t, dir, "example", "# Skill: example\n\n## Purpose\n\nLegacy skill description.\n")
  	skills, err := Walk(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(skills) != 1 || skills[0].Name != "example" || skills[0].Domain != "unknown" {
  		t.Fatalf("got %+v", skills)
  	}
  }

  func TestResolveLocalOverridesGlobal(t *testing.T) {
  	g, l := t.TempDir(), t.TempDir()
  	writeSkill(t, g, "shared", "---\nname: shared\ndescription: global version\n---\n")
  	writeSkill(t, l, "shared", "---\nname: shared\ndescription: local override\n---\n")
  	merged, err := Resolve(g, l)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(merged) != 1 || merged[0].Description != "local override" {
  		t.Fatalf("got %+v", merged)
  	}
  }

  func TestResolveMissingRoots(t *testing.T) {
  	merged, err := Resolve(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "nope2"))
  	if err != nil || len(merged) != 0 {
  		t.Fatalf("expected empty, no error; got %+v, %v", merged, err)
  	}
  }
  ```

---

## Task 5 — `harness/` source tree

- [x] **5.1** Create `harness/VERSION`:

  ```
  0.1.0-mvp
  ```

- [x] **5.2** Create `harness/core/planner/METHOD.md`:

  ```markdown
  # Core Method: Planner

  Domain-agnostic Planner methodology, installed globally at
  `~/.engineering-harness/core/planner/METHOD.md`. Adapters reference this file instead of
  restating the rules; it does not replace a project's own CLAUDE.md/AGENTS.md, which stay
  project-owned per `.agent/project.yaml`'s `mode`.

  ## Role

  The Planner thinks before anything is built and never edits source files. It reads project
  context, resolves relevant skills, writes a plan, and hands it to an Executor.

  ## Principles (non-negotiable)

  1. **Think Before Planning** — do not write `tasks.md` until `spec.md`'s design is settled.
  2. **Simplicity First** — the plan that changes the fewest files wins; `spec.md` lists 3+
     explicit out-of-scope items.
  3. **Surgical Changes** — every task names an exact file and symbol/anchor; line numbers are
     hints, not the primary reference.
  4. **Goal-Driven Execution** — `tests.md` defines done, not `tasks.md`.

  ## Plan folder

  ```
  .plans/YYYY-MM-DD-feature-name/
    spec.md    — goal, design decisions, scope, affected files, risks
    tasks.md   — ordered checklist, [ ] [~] [x] [!] status markers
    tests.md   — exact command + binary pass/fail per test
  ```

  ## Before writing spec.md

  1. Read the project's own context docs (`docs/src-map.md`, `docs/gotchas.md`, or
     `docs/context/*` if present) — do not re-invent what's already documented.
  2. Resolve enabled skills (`eng skills list`) and load only the ones relevant to the request.
  3. Read prior `.plans/*/DECISION_LOG.md` entries touching the same area.
  ```

- [x] **5.3** Create `harness/core/executor/METHOD.md`:

  ```markdown
  # Core Method: Executor

  Domain-agnostic Executor methodology, shared by every adapter (Claude Code, Copilot, Codex,
  future agents). Installed globally; a project's adapter-specific instruction file points
  here instead of restating the rules.

  ## Role

  The Executor implements a Planner-authored plan, one task at a time, and never designs or
  makes judgment calls beyond what the plan specifies.

  ## Task loop

  1. Read `spec.md` → `tasks.md` → `tests.md`, in that order.
  2. Find the first unchecked `[ ]` subtask.
  3. Make only the change that subtask describes.
  4. Run the subtask's verification command from `tests.md`.
  5. PASS → mark `[x]` → next task. FAIL → mark `[!]`, stop, report the exact output — do
     not guess a fix.

  ## Constraints

  No unplanned changes, no new files unless the task says to create one, no refactoring
  beyond the task, no premature abstraction, no skipping a verification command.
  ```

- [x] **5.4** Copy the three principle files unmodified (already domain-agnostic — no edits
  needed):

  ```bash
  mkdir -p harness/core/principles
  cp .claude/principles/karpathy.md harness/core/principles/karpathy.md
  cp .claude/principles/plan-quality.md harness/core/principles/plan-quality.md
  cp .claude/principles/thinking-checklist.md harness/core/principles/thinking-checklist.md
  ```

- [x] **5.5** Create `harness/skills/engineering/karpathy-guidelines/SKILL.md` — the existing
  `skills/karpathy-guidelines/SKILL.md` body, promoted to global with a new frontmatter block
  prepended:

  ```markdown
  ---
  name: karpathy-guidelines
  domain: engineering
  description: Core behavioral rules preventing the most common LLM coding mistakes. Load alongside any domain skill — applies universally.
  tags: [planning, methodology, universal]
  triggers: []
  version: "1.0.0"
  dependencies: []
  conflicts: []
  when_to_use: Always — load first, before any domain-specific skill.
  when_not_to_use: ""
  ---

  ```

  Then append the full existing body of `skills/karpathy-guidelines/SKILL.md` (everything from
  its `# Skill: karpathy-guidelines` line onward) unmodified after the frontmatter block.

- [x] **5.6** Create `harness/profiles/software.yaml`:

  ```yaml
  name: software
  description: Default profile for general software engineering projects.
  skills:
    - engineering/karpathy-guidelines
  ```

- [x] **5.7** Copy the plan templates unmodified:

  ```bash
  mkdir -p harness/templates/plan
  cp .plans/_template/spec.md harness/templates/plan/spec.md
  cp .plans/_template/tasks.md harness/templates/plan/tasks.md
  cp .plans/_template/tests.md harness/templates/plan/tests.md
  cp .plans/_template/DECISION_LOG.md harness/templates/plan/DECISION_LOG.md
  cp .plans/_template/sprint-summary.md harness/templates/plan/sprint-summary.md
  cp .plans/_template/errors.log harness/templates/plan/errors.log
  ```

- [x] **5.8** Create `harness/templates/.agentignore`:

  ```
  .git/
  node_modules/
  dist/
  build/
  vendor/
  third_party/
  target/
  __pycache__/
  .venv/
  *.pyc
  .next/
  out/
  ```

---

## Task 6 — `eng install`

- [x] **6.1** Create `cli/install.go`:

  ```go
  package main

  import (
  	"flag"
  	"fmt"
  	"io/fs"
  	"os"
  	"path/filepath"
  )

  func harnessDir() string {
  	home, err := os.UserHomeDir()
  	if err != nil {
  		fmt.Println("error: cannot resolve home directory:", err)
  		os.Exit(1)
  	}
  	return filepath.Join(home, ".engineering-harness")
  }

  func cmdInstall(args []string) {
  	flagset := flag.NewFlagSet("install", flag.ExitOnError)
  	from := flagset.String("from", ".", "path to a checkout containing a harness/ directory")
  	flagset.Parse(args)

  	src := filepath.Join(*from, "harness")
  	if info, err := os.Stat(src); err != nil || !info.IsDir() {
  		fmt.Printf("error: %s does not contain a harness/ directory\n", *from)
  		os.Exit(1)
  	}

  	dst := harnessDir()
  	if err := copyTree(src, dst); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Printf("Installed harness to %s\n", dst)
  }

  func copyTree(src, dst string) error {
  	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
  		if err != nil {
  			return err
  		}
  		rel, err := filepath.Rel(src, path)
  		if err != nil {
  			return err
  		}
  		target := filepath.Join(dst, rel)
  		if d.IsDir() {
  			return os.MkdirAll(target, 0o755)
  		}
  		data, err := os.ReadFile(path)
  		if err != nil {
  			return err
  		}
  		return os.WriteFile(target, data, 0o644)
  	})
  }
  ```

---

## Task 7 — `eng init`

- [x] **7.1** Create `cli/init_cmd.go`:

  ```go
  package main

  import (
  	"flag"
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/detect"
  	"eng/internal/project"
  )

  func cmdInit(args []string) {
  	flagset := flag.NewFlagSet("init", flag.ExitOnError)
  	flagset.Parse(args)

  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	if _, err := os.Stat(filepath.Join(dir, project.ConfigPath)); err == nil {
  		fmt.Println(".agent/project.yaml already exists — not overwritten")
  		return
  	}

  	det := detect.Detect(dir)
  	mode := project.DetectMode(dir)
  	switch mode {
  	case "none":
  		mode = "modern"
  	case "legacy":
  		mode = "hybrid" // opting into .agent/ moves a legacy project to hybrid
  	}

  	cfg := &project.Config{
  		ProjectName:    filepath.Base(dir),
  		Mode:           mode,
  		HarnessProfile: "software",
  		Stack: project.Stack{
  			Type:  det.Type,
  			Build: det.Build,
  			Test:  det.Test,
  			Run:   det.Run,
  			Lint:  det.Lint,
  		},
  		EnabledSkills: []string{"engineering/karpathy-guidelines"},
  	}

  	if err := project.Save(dir, cfg); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Printf("Created .agent/project.yaml — mode: %s, stack: %s\n", cfg.Mode, cfg.Stack.Type)
  	if mode == "hybrid" {
  		fmt.Println("Existing CLAUDE.md / .plans/ / skills/ were left untouched.")
  	}
  }
  ```

---

## Task 8 — `eng doctor`

- [x] **8.1** Create `cli/doctor.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"

  	"eng/internal/project"
  	"eng/internal/skills"
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

  	mode := project.DetectMode(dir)
  	switch mode {
  	case "legacy":
  		fmt.Println("Project mode:      legacy (CLAUDE.md/.plans found, no .agent/) — fully compatible, no action required")
  	case "none":
  		fmt.Println("Project mode:      none — not yet initialized (`eng init` to enable)")
  	default:
  		fmt.Printf("Project mode:      %s (.agent/project.yaml present)\n", mode)
  	}

  	if cfg, err := project.Load(dir); err == nil {
  		fmt.Printf("Detected stack:    %s\n", cfg.Stack.Type)
  	}

  	resolved, err := skills.Resolve(filepath.Join(hDir, "skills"), filepath.Join(dir, "skills"))
  	if err == nil {
  		fmt.Printf("Skills resolved:   %d\n", len(resolved))
  		for _, s := range resolved {
  			fmt.Printf("  - %-30s [%s] %s\n", s.Name, s.Source, s.Description)
  		}
  	}
  }
  ```

---

## Task 9 — `eng scan`

- [x] **9.1** Create `cli/scan.go`:

  ```go
  package main

  import (
  	"bufio"
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"

  	"eng/internal/detect"
  )

  func cmdScan(args []string) {
  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	det := detect.Detect(dir)
  	fmt.Printf("Stack: %s\n", det.Type)
  	if det.Build != "" {
  		fmt.Printf("Build: %s\n", det.Build)
  	}
  	if det.Test != "" {
  		fmt.Printf("Test:  %s\n", det.Test)
  	}

  	ignore := loadAgentIgnore(dir)
  	counts := map[string]int{}
  	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
  		if err != nil {
  			return nil
  		}
  		rel, _ := filepath.Rel(dir, path)
  		if rel == "." {
  			return nil
  		}
  		if d.IsDir() && matchesIgnore(rel, ignore) {
  			return filepath.SkipDir
  		}
  		if !d.IsDir() {
  			if ext := filepath.Ext(path); ext != "" {
  				counts[ext]++
  			}
  		}
  		return nil
  	})

  	fmt.Println("\nFile counts by extension:")
  	for ext, n := range counts {
  		fmt.Printf("  %-10s %d\n", ext, n)
  	}
  }

  func loadAgentIgnore(dir string) []string {
  	f, err := os.Open(filepath.Join(dir, ".agentignore"))
  	if err != nil {
  		return []string{".git", "node_modules", "dist", "build", "vendor", "target"}
  	}
  	defer f.Close()
  	var lines []string
  	scanner := bufio.NewScanner(f)
  	for scanner.Scan() {
  		line := strings.TrimSpace(scanner.Text())
  		if line != "" && !strings.HasPrefix(line, "#") {
  			lines = append(lines, strings.TrimSuffix(line, "/"))
  		}
  	}
  	return lines
  }

  func matchesIgnore(rel string, ignore []string) bool {
  	base := filepath.Base(rel)
  	for _, pattern := range ignore {
  		if base == pattern {
  			return true
  		}
  	}
  	return false
  }
  ```

---

## Task 10 — `eng skills list`

- [x] **10.1** Create `cli/skills_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/skills"
  )

  func cmdSkills(args []string) {
  	if len(args) == 0 || args[0] != "list" {
  		fmt.Println("Usage: eng skills list")
  		os.Exit(1)
  	}

  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	resolved, err := skills.Resolve(filepath.Join(harnessDir(), "skills"), filepath.Join(dir, "skills"))
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	for _, s := range resolved {
  		fmt.Printf("%-30s [%-6s] domain=%-12s %s\n", s.Name, s.Source, s.Domain, s.Description)
  	}
  }
  ```

- [x] **10.2** Run `cd cli && go mod tidy && go build ./...`. This is the first point all ten
  files compile together — fix any import/typo errors surfaced here before proceeding.

---

## Task 11 — Docs and repo integration (last task)

- [x] **11.1** In `README.md`, after the existing `## Repository structure` section, add a
  new `## V2 harness (preview)` section (do not edit any existing section):

  ```markdown
  ---

  ## V2 harness (preview)

  This repository is evolving from a clone-per-project template into a globally-installable
  harness. The V1 workflow above (`scripts/init.sh`, per-project `skills/`, `.plans/`) is
  unaffected and continues to work exactly as documented — nothing here requires migrating.

  ```bash
  cd cli && go build -o eng .
  ./eng install --from ..              # populate ~/.engineering-harness/
  cd /path/to/any/project
  /path/to/eng init                    # writes .agent/project.yaml only
  /path/to/eng doctor                  # reports legacy / hybrid / modern + resolved skills
  ```

  See `.plans/2026-08-24-v2-harness-foundation/spec.md` for the full design.
  ```

- [x] **11.2** In `ROADMAP.md`, after the title block (before `## Phase 1`), add:

  ```markdown
  > **2026-08-24:** Phases below describe V1 template evolution. The global-install /
  > multi-project direction they were pointing at is now superseded by
  > `.plans/2026-08-24-v2-harness-foundation/` — see that plan for the current architecture.
  ```

- [x] **11.3** In `docs/src-map.md`, replace the `_Nothing documented yet...` placeholder
  paragraph under `## Modules` with:

  ```markdown
  ### `cli/` — the `eng` CLI

  What it does: Go CLI (`eng install`, `eng init`, `eng doctor`, `eng scan`,
  `eng skills list`) that installs the harness payload globally and links a thin
  `.agent/project.yaml` into any project.

  Key files: `cli/main.go` (dispatch), `cli/internal/project/project.go` (mode detection),
  `cli/internal/skills/skills.go` (global+local skill resolution)

  From: `.plans/2026-08-24-v2-harness-foundation/`

  ### `harness/` — the installable harness payload

  What it does: source tree copied by `eng install` into `~/.engineering-harness/` — core
  Planner/Executor methodology, the first skill (`engineering/karpathy-guidelines`), the
  `software` profile, and the plan templates.

  Notable: skills here use YAML frontmatter for metadata; project-local skills without
  frontmatter still resolve via the legacy `# Skill:` heading convention — see
  `cli/internal/skills/skills.go`.

  From: `.plans/2026-08-24-v2-harness-foundation/`
  ```
