# Tasks: V2 Harness Phase 2

Each task must be completed and its test (see `tests.md`) must pass before moving to the
next. Mark `[x]` when done. Read `spec.md` in full — especially "Design decisions" and
"Responsibilities" — before starting Task 1.

**Prerequisite:** `go version` must report 1.22+ (already confirmed during the V2 Foundation
plan) and `harness/`/`cli/eng` must be buildable from that plan's completed state.

---

## Task 1 — Preliminary fix: surface `.agent/project.yaml` parse errors; extend `Config`

- [x] **1.1** Replace the full contents of `cli/internal/project/project.go`:

  ```go
  package project

  import (
  	"fmt"
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

  type Workflow struct {
  	Triage     bool `yaml:"triage"`
  	PlanReview bool `yaml:"plan_review"`
  	Verifier   bool `yaml:"verifier"`
  }

  // enabled reports whether this Workflow struct was ever explicitly set.
  // An all-false zero value means "the workflow block was absent" — callers
  // treat that as "everything enabled" via EffectiveWorkflow.
  func (w Workflow) enabled() bool {
  	return w.Triage || w.PlanReview || w.Verifier
  }

  type RetryBudget struct {
  	Build           int `yaml:"build"`
  	UnitTest        int `yaml:"unit_test"`
  	IntegrationTest int `yaml:"integration_test"`
  }

  func (b RetryBudget) isZero() bool {
  	return b.Build == 0 && b.UnitTest == 0 && b.IntegrationTest == 0
  }

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

  // EffectiveWorkflow returns the configured Workflow, or all-enabled if this
  // project.yaml predates Phase 2 (no workflow block at all).
  func (c *Config) EffectiveWorkflow() Workflow {
  	if c.Workflow.enabled() {
  		return c.Workflow
  	}
  	return Workflow{Triage: true, PlanReview: true, Verifier: true}
  }

  // EffectiveRetryBudget returns the configured budget, or Phase 2's default
  // if the project.yaml doesn't declare one.
  func (c *Config) EffectiveRetryBudget() RetryBudget {
  	if !c.RetryBudget.isZero() {
  		return c.RetryBudget
  	}
  	return RetryBudget{Build: 2, UnitTest: 2, IntegrationTest: 1}
  }

  const ConfigPath = ".agent/project.yaml"

  func Load(dir string) (*Config, error) {
  	data, err := os.ReadFile(filepath.Join(dir, ConfigPath))
  	if err != nil {
  		return nil, err
  	}
  	var cfg Config
  	if err := yaml.Unmarshal(data, &cfg); err != nil {
  		return nil, fmt.Errorf("parsing %s: %w", ConfigPath, err)
  	}
  	if cfg.ConfigVersion == 0 {
  		cfg.ConfigVersion = 1 // Phase 1 files predate this field
  	}
  	return &cfg, nil
  }

  func Save(dir string, cfg *Config) error {
  	if cfg.ConfigVersion == 0 {
  		cfg.ConfigVersion = 2
  	}
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

  // ModeResult distinguishes "no .agent/ at all" from "project.yaml exists but
  // is broken" — the Phase 1 DetectMode collapsed these into "hybrid" silently.
  type ModeResult struct {
  	Mode     string // legacy | hybrid | modern | none | broken
  	ParseErr error  // non-nil only when Mode == "broken"
  }

  func DetectModeResult(dir string) ModeResult {
  	if _, err := os.Stat(filepath.Join(dir, ConfigPath)); err == nil {
  		cfg, loadErr := Load(dir)
  		if loadErr != nil {
  			return ModeResult{Mode: "broken", ParseErr: loadErr}
  		}
  		if cfg.Mode != "" {
  			return ModeResult{Mode: cfg.Mode}
  		}
  		return ModeResult{Mode: "hybrid"}
  	}
  	_, claudeErr := os.Stat(filepath.Join(dir, "CLAUDE.md"))
  	_, plansErr := os.Stat(filepath.Join(dir, ".plans"))
  	if claudeErr == nil || plansErr == nil {
  		return ModeResult{Mode: "legacy"}
  	}
  	return ModeResult{Mode: "none"}
  }

  // DetectMode is the Phase 1 string-only API, kept for existing callers
  // (cmdInit's flow never reaches the "broken" case — it stats ConfigPath
  // itself first and returns early when the file already exists).
  func DetectMode(dir string) string {
  	r := DetectModeResult(dir)
  	if r.Mode == "broken" {
  		return "hybrid"
  	}
  	return r.Mode
  }
  ```

- [x] **1.2** In `cli/doctor.go`, replace the mode-reporting block:

  Old:
  ```go
  	mode := project.DetectMode(dir)
  	switch mode {
  	case "legacy":
  		fmt.Println("Project mode:      legacy (CLAUDE.md/.plans found, no .agent/) — fully compatible, no action required")
  	case "none":
  		fmt.Println("Project mode:      none — not yet initialized (`eng init` to enable)")
  	default:
  		fmt.Printf("Project mode:      %s (.agent/project.yaml present)\n", mode)
  	}
  ```

  New:
  ```go
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
  ```

- [x] **1.3** Append to `cli/internal/project/project_test.go` (add `"gopkg.in/yaml.v3"` to the
  import block):

  ```go
  func TestDetectModeResultBroken(t *testing.T) {
  	dir := t.TempDir()
  	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
  	os.WriteFile(filepath.Join(dir, ConfigPath), []byte(": not valid yaml :: ["), 0o644)
  	r := DetectModeResult(dir)
  	if r.Mode != "broken" || r.ParseErr == nil {
  		t.Fatalf("expected broken with a parse error, got %+v", r)
  	}
  }

  func TestEffectiveWorkflowDefaultsAllTrue(t *testing.T) {
  	cfg := &Config{}
  	w := cfg.EffectiveWorkflow()
  	if !w.Triage || !w.PlanReview || !w.Verifier {
  		t.Fatalf("expected all-true default, got %+v", w)
  	}
  }

  func TestEffectiveRetryBudgetDefault(t *testing.T) {
  	cfg := &Config{}
  	b := cfg.EffectiveRetryBudget()
  	if b.Build != 2 || b.UnitTest != 2 || b.IntegrationTest != 1 {
  		t.Fatalf("expected default budget, got %+v", b)
  	}
  }

  func TestConfigVersionDefaultsToOneOnLoad(t *testing.T) {
  	dir := t.TempDir()
  	cfg := &Config{ProjectName: "x", Mode: "modern"}
  	data, _ := yaml.Marshal(cfg) // simulates a Phase-1 file with no config_version field
  	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
  	os.WriteFile(filepath.Join(dir, ConfigPath), data, 0o644)

  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got.ConfigVersion != 1 {
  		t.Fatalf("expected ConfigVersion=1 for a pre-Phase-2 file, got %d", got.ConfigVersion)
  	}
  }
  ```

---

## Task 2 — `plan.yaml` schema (`internal/planmeta`)

- [x] **2.1** Create `cli/internal/planmeta/planmeta.go`:

  ```go
  package planmeta

  import (
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"
  )

  type RetryCounters struct {
  	Build           int `yaml:"build"`
  	UnitTest        int `yaml:"unit_test"`
  	IntegrationTest int `yaml:"integration_test"`
  }

  type RetryBudget struct {
  	Build           int `yaml:"build"`
  	UnitTest        int `yaml:"unit_test"`
  	IntegrationTest int `yaml:"integration_test"`
  }

  type PlannedAt struct {
  	GitSHA string `yaml:"git_sha"`
  }

  type Meta struct {
  	Plan        string        `yaml:"plan"`
  	RiskLevel   string        `yaml:"risk_level"` // quick-fix | bug | feature | architecture | high-risk
  	PlannedAt   PlannedAt     `yaml:"planned_at"`
  	Status      string        `yaml:"status"` // planned | reviewed | executing | verified | failed
  	WriteScope  []string      `yaml:"write_scope"`
  	Retry       RetryCounters `yaml:"retry"`
  	RetryBudget RetryBudget   `yaml:"retry_budget"`
  }

  const FileName = "plan.yaml"

  func Load(planDir string) (*Meta, error) {
  	data, err := os.ReadFile(filepath.Join(planDir, FileName))
  	if err != nil {
  		return nil, err
  	}
  	var m Meta
  	if err := yaml.Unmarshal(data, &m); err != nil {
  		return nil, err
  	}
  	return &m, nil
  }

  func Save(planDir string, m *Meta) error {
  	data, err := yaml.Marshal(m)
  	if err != nil {
  		return err
  	}
  	return os.WriteFile(filepath.Join(planDir, FileName), data, 0o644)
  }

  // DefaultBudget is used when neither plan.yaml nor .agent/project.yaml
  // declares a retry_budget.
  func DefaultBudget() RetryBudget {
  	return RetryBudget{Build: 2, UnitTest: 2, IntegrationTest: 1}
  }
  ```

- [x] **2.2** Create `cli/internal/planmeta/planmeta_test.go`:

  ```go
  package planmeta

  import "testing"

  func TestSaveLoadRoundTrip(t *testing.T) {
  	dir := t.TempDir()
  	m := &Meta{
  		Plan:      "2026-08-24-example",
  		RiskLevel: "feature",
  		PlannedAt: PlannedAt{GitSHA: "abc123"},
  		Status:    "planned",
  		WriteScope: []string{"src/api/**"},
  	}
  	if err := Save(dir, m); err != nil {
  		t.Fatal(err)
  	}
  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got.PlannedAt.GitSHA != "abc123" || got.RiskLevel != "feature" {
  		t.Fatalf("round-trip mismatch: %+v", got)
  	}
  }

  func TestDefaultBudget(t *testing.T) {
  	b := DefaultBudget()
  	if b.Build != 2 || b.UnitTest != 2 || b.IntegrationTest != 1 {
  		t.Fatalf("unexpected default budget: %+v", b)
  	}
  }
  ```

---

## Task 3 — Git helpers (`internal/gitutil`)

- [x] **3.1** Create `cli/internal/gitutil/gitutil.go`:

  ```go
  package gitutil

  import (
  	"os/exec"
  	"strings"
  )

  func run(dir string, args ...string) (string, error) {
  	cmd := exec.Command("git", args...)
  	cmd.Dir = dir
  	out, err := cmd.Output()
  	return strings.TrimSpace(string(out)), err
  }

  // HeadSHA returns the current commit hash of dir's repository.
  func HeadSHA(dir string) (string, error) {
  	return run(dir, "rev-parse", "HEAD")
  }

  // ChangedFilesSince returns paths (relative to dir) that differ between sha
  // and the current working tree, including uncommitted changes — this is
  // deliberately "diff against a fixed point," not "commits since sha."
  func ChangedFilesSince(dir, sha string) ([]string, error) {
  	out, err := run(dir, "diff", "--name-only", sha)
  	if err != nil {
  		return nil, err
  	}
  	if out == "" {
  		return nil, nil
  	}
  	return strings.Split(out, "\n"), nil
  }
  ```

- [x] **3.2** Create `cli/internal/gitutil/gitutil_test.go`:

  ```go
  package gitutil

  import (
  	"os"
  	"os/exec"
  	"path/filepath"
  	"testing"
  )

  func initRepo(t *testing.T) string {
  	t.Helper()
  	dir := t.TempDir()
  	run := func(args ...string) {
  		cmd := exec.Command("git", args...)
  		cmd.Dir = dir
  		if out, err := cmd.CombinedOutput(); err != nil {
  			t.Fatalf("git %v: %v\n%s", args, err, out)
  		}
  	}
  	run("init", "-q")
  	run("config", "user.email", "test@example.com")
  	run("config", "user.name", "test")
  	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one"), 0o644)
  	run("add", "a.txt")
  	run("commit", "-q", "-m", "init")
  	return dir
  }

  func TestHeadSHA(t *testing.T) {
  	dir := initRepo(t)
  	sha, err := HeadSHA(dir)
  	if err != nil || len(sha) < 7 {
  		t.Fatalf("unexpected sha %q, err %v", sha, err)
  	}
  }

  func TestChangedFilesSince(t *testing.T) {
  	dir := initRepo(t)
  	sha, _ := HeadSHA(dir)

  	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("two"), 0o644)
  	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new"), 0o644)

  	changed, err := ChangedFilesSince(dir, sha)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(changed) != 1 || changed[0] != "a.txt" {
  		// b.txt is untracked, not part of `git diff` until added — expected.
  		t.Fatalf("expected only a.txt (tracked, modified), got %+v", changed)
  	}
  }
  ```

---

## Task 4 — Hooks config (`internal/hooks`)

- [x] **4.1** Create `cli/internal/hooks/hooks.go`:

  ```go
  package hooks

  import (
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"
  )

  type Config struct {
  	BeforePlan    []string          `yaml:"before_plan"`
  	AfterPlan     []string          `yaml:"after_plan"`
  	BeforeExecute []string          `yaml:"before_execute"`
  	AfterTask     []string          `yaml:"after_task"`
  	AfterExecute  []string          `yaml:"after_execute"`
  	OnFailure     []string          `yaml:"on_failure"`
  	Commands      map[string]string `yaml:"commands"`
  }

  // Load reads .agent/hooks.yaml if present, else globalDefaultPath. A
  // project-local file fully replaces the global default — no partial merge.
  func Load(projectDir, globalDefaultPath string) (*Config, error) {
  	path := globalDefaultPath
  	local := filepath.Join(projectDir, ".agent", "hooks.yaml")
  	if _, err := os.Stat(local); err == nil {
  		path = local
  	}
  	data, err := os.ReadFile(path)
  	if err != nil {
  		return nil, err
  	}
  	var cfg Config
  	if err := yaml.Unmarshal(data, &cfg); err != nil {
  		return nil, err
  	}
  	return &cfg, nil
  }

  // Stage returns the ordered hook names for a lifecycle stage.
  func (c *Config) Stage(name string) []string {
  	switch name {
  	case "before_plan":
  		return c.BeforePlan
  	case "after_plan":
  		return c.AfterPlan
  	case "before_execute":
  		return c.BeforeExecute
  	case "after_task":
  		return c.AfterTask
  	case "after_execute":
  		return c.AfterExecute
  	case "on_failure":
  		return c.OnFailure
  	default:
  		return nil
  	}
  }
  ```

- [x] **4.2** Create `cli/internal/hooks/hooks_test.go`:

  ```go
  package hooks

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func writeYAML(t *testing.T, path, content string) {
  	t.Helper()
  	os.MkdirAll(filepath.Dir(path), 0o755)
  	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
  		t.Fatal(err)
  	}
  }

  func TestLoadGlobalDefault(t *testing.T) {
  	global := filepath.Join(t.TempDir(), "default.yaml")
  	writeYAML(t, global, "before_plan: [project_scan]\ncommands:\n  project_scan: eng scan\n")

  	project := t.TempDir()
  	cfg, err := Load(project, global)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got := cfg.Stage("before_plan"); len(got) != 1 || got[0] != "project_scan" {
  		t.Fatalf("got %+v", got)
  	}
  }

  func TestProjectOverrideReplacesGlobal(t *testing.T) {
  	global := filepath.Join(t.TempDir(), "default.yaml")
  	writeYAML(t, global, "before_plan: [project_scan]\n")

  	project := t.TempDir()
  	writeYAML(t, filepath.Join(project, ".agent", "hooks.yaml"), "before_plan: [custom_check]\n")

  	cfg, err := Load(project, global)
  	if err != nil {
  		t.Fatal(err)
  	}
  	got := cfg.Stage("before_plan")
  	if len(got) != 1 || got[0] != "custom_check" {
  		t.Fatalf("expected project override only, got %+v", got)
  	}
  }
  ```

---

## Task 5 — Plan templates and `harness/hooks/default.yaml`

- [x] **5.1** Create `harness/templates/plan/plan.yaml`:

  ```yaml
  plan: YYYY-MM-DD-feature-name
  risk_level: feature   # quick-fix | bug | feature | architecture | high-risk
  planned_at:
    git_sha: ""
  status: planned       # planned | reviewed | executing | verified | failed
  write_scope: []       # e.g. ["src/api/**", "tests/api/**"] — from spec.md's Affected files
  retry:
    build: 0
    unit_test: 0
    integration_test: 0
  retry_budget:
    build: 2
    unit_test: 2
    integration_test: 1
  ```

- [x] **5.2** Create `harness/templates/plan/review.md`:

  ```markdown
  # Plan Review — [Feature Name]

  > Written by the Plan Reviewer, independently of the Planner. Read-only with respect to
  > `spec.md`/`tasks.md`/`tests.md` — this file is the only output of this role.

  ## Verdict

  [ ] APPROVED
  [ ] CHANGES REQUESTED

  ## Checklist

  | Check | Finding |
  |---|---|
  | Missing requirements | |
  | Incorrect assumptions | |
  | Architecture inconsistencies | |
  | Missing edge cases | |
  | Missing tests | |
  | Dependency problems | |
  | Security / hardware impact | |

  ## Notes

  [Anything else worth flagging before this plan reaches the Executor.]
  ```

- [x] **5.3** Create `harness/templates/plan/verify-report.md`:

  ```markdown
  # Verify Report — [Feature Name]

  > Generated by `eng verify`. Do not hand-edit — re-run `eng verify` to refresh.

  ## Verdict

  PENDING

  ## Git diff since plan was created

  ## Test run

  ## Unexpected changes outside write_scope
  ```

- [x] **5.4** Create `harness/hooks/default.yaml`:

  ```yaml
  before_plan:
    - project_scan
  after_plan:
    - plan_review
  before_execute:
    - drift_check
  after_task:
    - test
  after_execute:
    - regression_test
    - verify
  on_failure:
    - collect_logs

  commands:
    project_scan: "eng scan"
    plan_review: ""              # performed by the Plan Reviewer role, not a shell command
    drift_check: "eng plan drift ."
    test: "${test_cmd}"
    regression_test: "${test_cmd}"
    verify: "eng verify ."
    collect_logs: ""             # performed by the Executor writing errors.log, not a shell command
  ```

---

## Task 6 — `eng plan new/drift/retry`

- [x] **6.1** Create `cli/plan_cmd.go`:

  ```go
  package main

  import (
  	"flag"
  	"fmt"
  	"io/fs"
  	"os"
  	"path/filepath"
  	"time"

  	"eng/internal/gitutil"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  )

  func cmdPlan(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng plan <new|drift|retry> ...")
  		os.Exit(1)
  	}
  	switch args[0] {
  	case "new":
  		planNew(args[1:])
  	case "drift":
  		planDrift(args[1:])
  	case "retry":
  		planRetry(args[1:])
  	default:
  		fmt.Println("Usage: eng plan <new|drift|retry> ...")
  		os.Exit(1)
  	}
  }

  func planNew(args []string) {
  	flagset := flag.NewFlagSet("plan new", flag.ExitOnError)
  	risk := flagset.String("risk", "feature", "quick-fix|bug|feature|architecture|high-risk")
  	flagset.Parse(args)
  	rest := flagset.Args()
  	if len(rest) == 0 {
  		fmt.Println("Usage: eng plan new <name> [--risk <level>]")
  		os.Exit(1)
  	}
  	name := rest[0]

  	repoRoot, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	modeResult := project.DetectModeResult(repoRoot)
  	if modeResult.Mode == "legacy" || modeResult.Mode == "none" {
  		fmt.Println("error: run `eng init` first — eng plan new requires .agent/project.yaml")
  		os.Exit(1)
  	}
  	if modeResult.Mode == "broken" {
  		fmt.Printf("error: %s is broken: %v\n", project.ConfigPath, modeResult.ParseErr)
  		os.Exit(1)
  	}

  	planDir := filepath.Join(repoRoot, ".plans", time.Now().Format("2006-01-02")+"-"+name)
  	if _, err := os.Stat(planDir); err == nil {
  		fmt.Println("error: plan folder already exists:", planDir)
  		os.Exit(1)
  	}

  	tmplDir := filepath.Join(harnessDir(), "templates", "plan")
  	if err := copyTree(tmplDir, planDir); err != nil {
  		fmt.Println("error copying templates:", err)
  		os.Exit(1)
  	}

  	sha, err := gitutil.HeadSHA(repoRoot)
  	if err != nil {
  		fmt.Println("error: cannot resolve HEAD sha — is this a git repo?", err)
  		os.Exit(1)
  	}

  	budget := planmeta.DefaultBudget()
  	if cfg, err := project.Load(repoRoot); err == nil {
  		eb := cfg.EffectiveRetryBudget()
  		budget = planmeta.RetryBudget{Build: eb.Build, UnitTest: eb.UnitTest, IntegrationTest: eb.IntegrationTest}
  	}

  	meta := &planmeta.Meta{
  		Plan:        filepath.Base(planDir),
  		RiskLevel:   *risk,
  		PlannedAt:   planmeta.PlannedAt{GitSHA: sha},
  		Status:      "planned",
  		RetryBudget: budget,
  	}
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Printf("Scaffolded %s — risk: %s, git_sha: %s\n", planDir, *risk, sha)
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

  func planDrift(args []string) {
  	dir := "."
  	if len(args) > 0 {
  		dir = args[0]
  	}
  	planDir, _ := filepath.Abs(dir)

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s — nothing to check\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	repoRoot, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	changed, err := gitutil.ChangedFilesSince(repoRoot, meta.PlannedAt.GitSHA)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	if len(changed) == 0 {
  		fmt.Println("OK — no changes since plan was created")
  		return
  	}

  	relevant := changed
  	if len(meta.WriteScope) > 0 {
  		relevant = nil
  		for _, f := range changed {
  			if matchesAnyGlob(f, meta.WriteScope) {
  				relevant = append(relevant, f)
  			}
  		}
  	}
  	if len(relevant) == 0 {
  		fmt.Println("OK — unrelated files changed, no drift in this plan's scope")
  		return
  	}

  	fmt.Println("PLAN_DRIFT_DETECTED — the following files changed since this plan was created:")
  	for _, f := range relevant {
  		fmt.Printf("  - %s\n", f)
  	}
  	fmt.Println("\nRevalidate the plan against current source before executing further.")
  	os.Exit(1)
  }

  func planRetry(args []string) {
  	if len(args) < 2 {
  		fmt.Println("Usage: eng plan retry <plan-dir> <build|unit_test|integration_test>")
  		os.Exit(1)
  	}
  	planDir, _ := filepath.Abs(args[0])
  	stage := args[1]

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s — cannot track retries\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	var count, limit *int
  	switch stage {
  	case "build":
  		count, limit = &meta.Retry.Build, &meta.RetryBudget.Build
  	case "unit_test":
  		count, limit = &meta.Retry.UnitTest, &meta.RetryBudget.UnitTest
  	case "integration_test":
  		count, limit = &meta.Retry.IntegrationTest, &meta.RetryBudget.IntegrationTest
  	default:
  		fmt.Println("Unknown stage:", stage, "(expected build|unit_test|integration_test)")
  		os.Exit(1)
  	}

  	*count++
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	if *count > *limit {
  		fmt.Printf("RETRY BUDGET EXHAUSTED for %s (%d/%d) — escalate to Planner or human\n", stage, *count, *limit)
  		os.Exit(1)
  	}
  	fmt.Printf("RETRY %d/%d for %s — proceed\n", *count, *limit, stage)
  }

  // matchesAnyGlob supports filepath.Match patterns plus a "prefix/**" suffix
  // convention for directory-scope matches (filepath.Match has no "**").
  func matchesAnyGlob(path string, patterns []string) bool {
  	for _, p := range patterns {
  		if ok, _ := filepath.Match(p, path); ok {
  			return true
  		}
  		if trimmed, isDirGlob := cutSuffix(p, "/**"); isDirGlob {
  			if path == trimmed || len(path) > len(trimmed) && path[:len(trimmed)+1] == trimmed+"/" {
  				return true
  			}
  		}
  	}
  	return false
  }

  func cutSuffix(s, suffix string) (string, bool) {
  	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
  		return s[:len(s)-len(suffix)], true
  	}
  	return s, false
  }
  ```

  Note: `cli/install.go` already defines a `copyTree` function. Rename **this task's** copy to
  avoid a duplicate-symbol build error — see Task 6.2.

- [x] **6.2** Since `cli/install.go` already has `func copyTree(src, dst string) error` in
  package `main`, delete the duplicate `copyTree` function from `cli/plan_cmd.go` (the one just
  added in 6.1) and reuse the existing one from `install.go` instead — they are identical in
  behavior. `plan_cmd.go` should have no `copyTree` definition after this edit.

---

## Task 7 — `eng verify`

- [x] **7.1** Create `cli/verify_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"os/exec"
  	"path/filepath"
  	"strings"

  	"eng/internal/gitutil"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  )

  func cmdVerify(args []string) {
  	dir := "."
  	if len(args) > 0 {
  		dir = args[0]
  	}
  	planDir, err := filepath.Abs(dir)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s — nothing to verify\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	repoRoot, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	var report strings.Builder
  	fmt.Fprintf(&report, "# Verify Report — %s\n\n", meta.Plan)
  	pass := true

  	changed, err := gitutil.ChangedFilesSince(repoRoot, meta.PlannedAt.GitSHA)
  	if err != nil {
  		fmt.Fprintf(&report, "## Git diff\n\nERROR: %v\n\n", err)
  		pass = false
  	} else {
  		fmt.Fprintf(&report, "## Git diff since %s\n\n", meta.PlannedAt.GitSHA)
  		var unexpected []string
  		for _, f := range changed {
  			fmt.Fprintf(&report, "- %s\n", f)
  			if len(meta.WriteScope) > 0 && !matchesAnyGlob(f, meta.WriteScope) {
  				unexpected = append(unexpected, f)
  			}
  		}
  		if len(unexpected) > 0 {
  			fmt.Fprintf(&report, "\n**UNEXPECTED CHANGES outside write_scope:**\n")
  			for _, f := range unexpected {
  				fmt.Fprintf(&report, "- %s\n", f)
  			}
  			pass = false
  		}
  	}

  	if cfg, err := project.Load(repoRoot); err == nil && cfg.Stack.Test != "" {
  		fmt.Fprintf(&report, "\n## Test run\n\nCommand: `%s`\n\n", cfg.Stack.Test)
  		c := exec.Command("sh", "-c", cfg.Stack.Test)
  		c.Dir = repoRoot
  		out, testErr := c.CombinedOutput()
  		fmt.Fprintf(&report, "```\n%s\n```\n\n", string(out))
  		if testErr != nil {
  			fmt.Fprintf(&report, "Test command exited with error: %v\n\n", testErr)
  			pass = false
  		}
  	}

  	verdict := "PASS"
  	if !pass {
  		verdict = "FAIL"
  	}
  	fmt.Fprintf(&report, "## Verdict: %s\n", verdict)

  	if err := os.WriteFile(filepath.Join(planDir, "verify-report.md"), []byte(report.String()), 0o644); err != nil {
  		fmt.Println("error writing report:", err)
  		os.Exit(1)
  	}

  	fmt.Println(report.String())
  	if !pass {
  		os.Exit(1)
  	}
  }
  ```

---

## Task 8 — `eng hooks run`

- [x] **8.1** Create `cli/hooks_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"os/exec"
  	"path/filepath"
  	"strings"

  	"eng/internal/hooks"
  	"eng/internal/project"
  )

  func cmdHooks(args []string) {
  	if len(args) < 2 || args[0] != "run" {
  		fmt.Println("Usage: eng hooks run <stage>")
  		os.Exit(1)
  	}
  	stage := args[1]

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
  		testCmd = pcfg.Stack.Test
  	}

  	for _, name := range names {
  		cmdStr := strings.ReplaceAll(cfg.Commands[name], "${test_cmd}", testCmd)
  		if cmdStr == "" {
  			fmt.Printf("[%s] %-16s manual step — no shell command; perform via the documented role\n", stage, name)
  			continue
  		}
  		fmt.Printf("[%s] %-16s -> %s\n", stage, name, cmdStr)
  		c := exec.Command("sh", "-c", cmdStr)
  		c.Dir = dir
  		c.Stdout = os.Stdout
  		c.Stderr = os.Stderr
  		if err := c.Run(); err != nil {
  			fmt.Printf("HOOK FAILED: %s (%v)\n", name, err)
  			os.Exit(1)
  		}
  	}
  }
  ```

---

## Task 9 — `eng triage`

- [x] **9.1** Create `cli/triage_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"strings"
  )

  var triageKeywords = []struct {
  	level    string
  	workflow string
  	words    []string
  }{
  	{"high-risk", "high-risk workflow — requires human approval before executing",
  		[]string{"production", "deploy", "migration", "flash", "firmware", "plc write", "delete data", "drop table"}},
  	{"architecture", "research + ADR + full spec/tasks/tests",
  		[]string{"architecture", "redesign", "migrate to", "replace", "rewrite"}},
  	{"bug", "bug workflow — reproduce, fix, regression test",
  		[]string{"bug", "fix", "broken", "error", "crash", "fails"}},
  	{"quick-fix", "quick workflow — skip full spec, single-file plan",
  		[]string{"typo", "rename", "comment", "formatting", "small change"}},
  }

  func cmdTriage(args []string) {
  	if len(args) == 0 {
  		fmt.Println(`Usage: eng triage "<request text>"`)
  		os.Exit(1)
  	}
  	text := strings.ToLower(strings.Join(args, " "))

  	for _, k := range triageKeywords {
  		for _, w := range k.words {
  			if strings.Contains(text, w) {
  				printTriage(k.level, k.workflow)
  				return
  			}
  		}
  	}
  	printTriage("feature", "full spec + tasks + tests")
  }

  func printTriage(level, workflow string) {
  	fmt.Printf("Suggested level: %s\n", level)
  	fmt.Printf("Suggested workflow: %s\n", workflow)
  	fmt.Println("\n(heuristic hint only — the Planner makes the final call)")
  }
  ```

---

## Task 10 — Wire up dispatch in `main.go`

- [x] **10.1** In `cli/main.go`, update the `switch` in `main()`:

  Old:
  ```go
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
  ```

  New:
  ```go
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
  	case "plan":
  		cmdPlan(os.Args[2:])
  	case "verify":
  		cmdVerify(os.Args[2:])
  	case "hooks":
  		cmdHooks(os.Args[2:])
  	case "triage":
  		cmdTriage(os.Args[2:])
  	default:
  		usage()
  		os.Exit(1)
  	}
  ```

- [x] **10.2** In `cli/main.go`'s `usage()` function, append four lines to the printed command
  list (after the `skills list` line, before the closing backtick):

  ```
    plan new <name> [--risk <level>]   Scaffold a plan and stamp it with the current git SHA
    plan drift [dir]                   Check whether relevant files changed since planning
    plan retry <dir> <stage>           Track a retry against this plan's budget
    verify [dir]                       Run tests, check the git diff, write verify-report.md
    hooks run <stage>                  Run the configured hooks for a lifecycle stage
    triage "<text>"                    Heuristic risk-level hint (not authoritative)
  ```

- [x] **10.3** Run `cd cli && go build ./... 2>&1` and fix any compile errors (expected: none,
  if Tasks 1–10.2 were followed exactly) before proceeding.

---

## Task 11 — Core methodology docs

- [x] **11.1** Create `harness/core/triage/METHOD.md`:

  ```markdown
  # Core Method: Triage

  Domain-agnostic request classification, run before any plan is written.

  ## Levels

  | Level | Examples | Workflow |
  |---|---|---|
  | quick-fix | typo, rename, comment, formatting | Single-file plan — skip spec/tasks/tests split |
  | bug | reproducible defect, broken behavior | Reproduce → fix → regression test |
  | feature | new capability, no architecture change | Full spec.md + tasks.md + tests.md |
  | architecture | crosses module boundaries, changes a decision | Research first, consult ADRs, full plan + Plan Reviewer required |
  | high-risk | production deploy, data migration, firmware flash, PLC write, destructive operation | Full plan + Plan Reviewer + explicit human approval before Executor touches anything real |

  ## Role

  Triage is a classification step, not a gate — it decides which workflow a request follows,
  not whether the request is allowed. `eng triage "<text>"` provides a keyword-based hint;
  the Planner makes the actual determination using full context the heuristic doesn't have
  (the project's own conventions, `docs/gotchas.md`, prior `DECISION_LOG.md` entries).

  ## Before writing spec.md

  1. Determine the level using the table above.
  2. Record it as `plan.yaml`'s `risk_level` (via `eng plan new --risk <level>`).
  3. If the level is `architecture` or `high-risk`, the Plan Reviewer step is mandatory even if
     `.agent/project.yaml`'s `workflow.plan_review` is otherwise optional for this project.
  4. If the level is `high-risk`, cross-check `.agent/project.yaml`'s `require_approval` list —
     if the request matches a listed category, say so explicitly in `spec.md` and flag every
     task that touches it with `**Requires approval:**`.
  ```

- [x] **11.2** Create `harness/core/plan-reviewer/METHOD.md`:

  ```markdown
  # Core Method: Plan Reviewer

  Independent review between Planner and Executor. Read-only with respect to `spec.md`,
  `tasks.md`, and `tests.md` — the Reviewer's only output is `review.md`.

  ## Role

  Read the plan the same way the Executor will, and try to find what the Planner missed before
  an Executor spends real time on it.

  ## Checklist (from `harness/templates/plan/review.md`)

  - **Missing requirements** — does `spec.md`'s Goal fully cover what the request asked for?
  - **Incorrect assumptions** — does the plan assume something about the codebase that
    `docs/src-map.md` or the actual source contradicts?
  - **Architecture inconsistencies** — does this plan conflict with a decision recorded in a
    prior `DECISION_LOG.md` or ADR?
  - **Missing edge cases** — what input/state does `tests.md` not cover?
  - **Missing tests** — does every task group have a corresponding test, per this repo's own
    Goal-Driven Execution principle?
  - **Dependency problems** — does a task assume an earlier task's output that doesn't exist
    yet, or an external dependency the project doesn't have?
  - **Security or hardware impact** — does any task touch auth, secrets, or (for embedded/
    automation profiles) physical hardware state?

  ## Verdict

  `APPROVED` or `CHANGES REQUESTED` (with specific findings against the checklist above),
  written to `review.md`. `CHANGES REQUESTED` means the Planner revises `spec.md`/`tasks.md`;
  the Reviewer does not edit them directly.

  ## Constraint

  Never modify `spec.md`, `tasks.md`, `tests.md`, or any source file. This role's only write is
  `review.md`.
  ```

- [x] **11.3** Create `harness/core/verifier/METHOD.md`:

  ```markdown
  # Core Method: Verifier

  Independent check that an Executor's own PASS/FAIL self-report is not the only authority on
  whether a plan is actually done.

  ## Role

  Run `eng verify [plan-dir]` after the Executor reports all tasks `[x]`. This:

  1. Diffs the repository against `plan.yaml`'s recorded `planned_at.git_sha`.
  2. Flags any changed file outside the plan's declared `write_scope`.
  3. Runs the project's test command (`.agent/project.yaml`'s `stack.test_cmd`).
  4. Writes `verify-report.md` with a PASS/FAIL verdict.

  ## Constraint

  Never modify source files. If verification FAILs, report it — do not attempt a fix. That is
  the Executor's job, working from the Verifier's report, within its retry budget
  (`eng plan retry`).

  ## Definition of Done

  A plan is not done because its own Executor said so. It is done when:
  - Every task in `tasks.md` is `[x]`
  - Every test in `tests.md` passes
  - `eng verify` reports PASS
  - `docs/src-map.md` is updated if the plan added a new module (per this repo's existing
    convention)
  ```

- [x] **11.4** In `harness/core/planner/METHOD.md`, replace the "Before writing spec.md"
  section:

  Old:
  ```markdown
  ## Before writing spec.md

  1. Read the project's own context docs (`docs/src-map.md`, `docs/gotchas.md`, or
     `docs/context/*` if present) — do not re-invent what's already documented.
  2. Resolve enabled skills (`eng skills list`) and load only the ones relevant to the request.
  3. Read prior `.plans/*/DECISION_LOG.md` entries touching the same area.
  ```

  New:
  ```markdown
  ## Before writing spec.md

  1. Run Triage (see `core/triage/METHOD.md`) to determine the risk level.
  2. Read the project's own context docs (`docs/src-map.md`, `docs/gotchas.md`, or
     `docs/context/*` if present) — do not re-invent what's already documented.
  3. Resolve enabled skills (`eng skills list`) and load only the ones relevant to the request.
  4. Read prior `.plans/*/DECISION_LOG.md` entries touching the same area.

  ## After spec.md is confirmed

  1. Run `eng plan new <name> --risk <level>` to scaffold the plan folder and stamp
     `plan.yaml` with the current git SHA.
  2. Fill in `plan.yaml`'s `write_scope` from `spec.md`'s "Affected files" table.
  3. If `.agent/project.yaml`'s `workflow.plan_review` is enabled (or the risk level is
     `architecture`/`high-risk`, which makes it mandatory regardless), hand the plan to a
     Plan Reviewer session before an Executor starts.
  ```

- [x] **11.5** In `harness/core/executor/METHOD.md`, replace the "Constraints" section:

  Old:
  ```markdown
  ## Constraints

  No unplanned changes, no new files unless the task says to create one, no refactoring
  beyond the task, no premature abstraction, no skipping a verification command.
  ```

  New:
  ```markdown
  ## Before starting execution

  Run `eng plan drift <plan-dir>`. If it reports `PLAN_DRIFT_DETECTED`, stop — do not execute
  against a plan whose source has changed since it was written. Get the plan revalidated first.

  ## On each test failure

  Run `eng plan retry <plan-dir> <build|unit_test|integration_test>` before retrying. If it
  reports `RETRY BUDGET EXHAUSTED`, stop — escalate to the Planner or a human instead of
  attempting another fix.

  ## Stop conditions (hard stops, not judgment calls)

  Stop and report immediately, without attempting a workaround, when:
  - A requirement conflict appears that `spec.md` doesn't resolve
  - An unexpected schema change is discovered mid-task
  - A dependency mismatch blocks the task as written
  - Hardware configuration needed by the task is unknown or undocumented
  - `eng plan drift` reports `PLAN_DRIFT_DETECTED`
  - The current task is marked `**Requires approval:**` — get explicit human confirmation
    before performing it, every time, with no exception for a previously-approved similar task
  - Any operation matches a category in `.agent/project.yaml`'s `require_approval` list

  ## Constraints

  No unplanned changes, no new files unless the task says to create one, no refactoring
  beyond the task, no premature abstraction, no skipping a verification command.
  ```

---

## Task 12 — Version bump and docs integration (last task)

- [x] **12.1** Update `harness/VERSION`:

  ```
  0.2.0-phase2
  ```

- [x] **12.2** In `README.md`, immediately after the "V2 harness (preview)" section added by
  the Foundation plan (before the following `---`), add:

  ```markdown

  Phase 2 adds reliability primitives on top of the foundation above:

  ```bash
  cd cli && go build -o eng .
  ./eng plan new my-feature --risk feature   # scaffold + stamp plan.yaml with current git SHA
  ./eng plan drift .plans/2026-08-24-my-feature   # OK or PLAN_DRIFT_DETECTED
  ./eng verify .plans/2026-08-24-my-feature       # run tests, check diff, write verify-report.md
  ./eng hooks run before_execute                  # run configured lifecycle hooks
  ./eng triage "fix the login bug"                # heuristic risk-level hint
  ```

  See `.plans/2026-08-24-v2-harness-phase2/spec.md` for the full design.
  ```

- [x] **12.3** In `ROADMAP.md`, extend the note added by the Foundation plan:

  Old:
  ```markdown
  > **2026-08-24:** Phases below describe V1 template evolution. The global-install /
  > multi-project direction they were pointing at is now superseded by
  > `.plans/2026-08-24-v2-harness-foundation/` — see that plan for the current architecture.
  ```

  New:
  ```markdown
  > **2026-08-24:** Phases below describe V1 template evolution. The global-install /
  > multi-project direction they were pointing at is now superseded by
  > `.plans/2026-08-24-v2-harness-foundation/` (global install foundation) and
  > `.plans/2026-08-24-v2-harness-phase2/` (Triage/Plan Reviewer/Verifier/hooks/drift/retry) —
  > see those plans for the current architecture.
  ```

- [x] **12.4** In `docs/src-map.md`, add two new module sections after the `harness/` section
  added by the Foundation plan:

  ```markdown
  ### `cli/internal/planmeta/`, `cli/internal/gitutil/`, `cli/internal/hooks/` — Phase 2 plan lifecycle state

  What it does: `planmeta` reads/writes each plan's `plan.yaml` (git SHA, risk level, write
  scope, retry counters/budget); `gitutil` wraps `git rev-parse`/`git diff` for drift checks;
  `hooks` loads the lifecycle hook table (`harness/hooks/default.yaml`, project-overridable via
  `.agent/hooks.yaml`).

  Key files: `cli/plan_cmd.go` (`eng plan new/drift/retry`), `cli/verify_cmd.go` (`eng verify`),
  `cli/hooks_cmd.go` (`eng hooks run`)

  Notable: `eng verify` and `eng hooks run` shell out via `sh -c` — requires a POSIX shell on
  PATH (Git Bash's `sh.exe` on Windows), same dependency V1's own scripts always had.

  From: `.plans/2026-08-24-v2-harness-phase2/`

  ### `harness/core/triage/`, `harness/core/plan-reviewer/`, `harness/core/verifier/` — new roles

  What it does: methodology docs for the three roles Phase 2 adds around the existing
  Planner/Executor loop. None of them are enforced by code — `review.md` and
  `verify-report.md` are the enforcement surface (a file with a clear verdict), not a gate a
  script can close.

  From: `.plans/2026-08-24-v2-harness-phase2/`
  ```
