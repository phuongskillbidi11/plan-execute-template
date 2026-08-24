# Tasks: V2 Harness Phase 3

Each task must be completed and its test (see `tests.md`) must pass before moving to the
next. Mark `[x]` when done. Read `spec.md` in full — especially "Design decisions" and the
Decision 6 transition table — before starting Task 1.

**Prerequisite:** Go 1.22+, and Phase 1/2's completed `cli/`/`harness/` trees (already
committed as of this plan).

---

## Task 1 — Structured-or-shell command execution (`internal/executil`)

This is foundational: `project.Stack` and `hooks.Config.Commands` both switch to this type in
later tasks, so it must exist first.

- [x] **1.1** Create `cli/internal/executil/executil.go`:

  ```go
  package executil

  import (
  	"os/exec"

  	"gopkg.in/yaml.v3"
  )

  // Command describes how to run one shell command, either as a plain string
  // (compatibility mode, run via `sh -c` — the only mode Phase 1/2 ever used)
  // or as a structured argv (no shell at all). A plain YAML scalar unmarshals
  // into Shell; a mapping with `command`/`args` keys unmarshals into the
  // structured form. This is what makes the change backward compatible: every
  // existing `build_cmd: "npm run build"`-style string keeps parsing exactly
  // as before.
  type Command struct {
  	Shell   string
  	Program string
  	Args    []string
  }

  func (c *Command) UnmarshalYAML(value *yaml.Node) error {
  	if value.Kind == yaml.ScalarNode {
  		return value.Decode(&c.Shell)
  	}
  	var structured struct {
  		Command string   `yaml:"command"`
  		Args    []string `yaml:"args"`
  	}
  	if err := value.Decode(&structured); err != nil {
  		return err
  	}
  	c.Program = structured.Command
  	c.Args = structured.Args
  	return nil
  }

  func (c Command) MarshalYAML() (interface{}, error) {
  	if c.Program != "" {
  		return map[string]interface{}{"command": c.Program, "args": c.Args}, nil
  	}
  	return c.Shell, nil
  }

  // Empty reports whether no command was configured.
  func (c Command) Empty() bool {
  	return c.Shell == "" && c.Program == ""
  }

  // String returns a human-readable form for logging/printing.
  func (c Command) String() string {
  	if c.Program != "" {
  		s := c.Program
  		for _, a := range c.Args {
  			s += " " + a
  		}
  		return s
  	}
  	return c.Shell
  }

  // Run executes c in dir and returns combined stdout+stderr.
  func Run(c Command, dir string) (string, error) {
  	var cmd *exec.Cmd
  	if c.Program != "" {
  		cmd = exec.Command(c.Program, c.Args...)
  	} else {
  		cmd = exec.Command("sh", "-c", c.Shell)
  	}
  	cmd.Dir = dir
  	out, err := cmd.CombinedOutput()
  	return string(out), err
  }
  ```

- [x] **1.2** Create `cli/internal/executil/executil_test.go`:

  ```go
  package executil

  import (
  	"strings"
  	"testing"

  	"gopkg.in/yaml.v3"
  )

  func TestUnmarshalScalarIsShell(t *testing.T) {
  	var c Command
  	if err := yaml.Unmarshal([]byte(`"echo hi"`), &c); err != nil {
  		t.Fatal(err)
  	}
  	if c.Shell != "echo hi" || c.Program != "" {
  		t.Fatalf("got %+v", c)
  	}
  }

  func TestUnmarshalStructuredForm(t *testing.T) {
  	var c Command
  	if err := yaml.Unmarshal([]byte("command: cmake\nargs: [--build, build]\n"), &c); err != nil {
  		t.Fatal(err)
  	}
  	if c.Program != "cmake" || len(c.Args) != 2 || c.Args[1] != "build" {
  		t.Fatalf("got %+v", c)
  	}
  }

  func TestRunShellMode(t *testing.T) {
  	out, err := Run(Command{Shell: "echo hello"}, ".")
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !strings.Contains(out, "hello") {
  		t.Fatalf("got %q", out)
  	}
  }

  func TestRunStructuredMode(t *testing.T) {
  	// Uses `go version` rather than `echo` — Windows has no standalone echo.exe,
  	// but Go is already a hard prerequisite for this entire repository.
  	out, err := Run(Command{Program: "go", Args: []string{"version"}}, ".")
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !strings.Contains(out, "go version") {
  		t.Fatalf("got %q", out)
  	}
  }

  func TestEmpty(t *testing.T) {
  	if !(Command{}).Empty() {
  		t.Fatal("expected zero value to be Empty")
  	}
  	if (Command{Shell: "x"}).Empty() {
  		t.Fatal("expected non-empty Shell to not be Empty")
  	}
  }
  ```

---

## Task 2 — `plan.yaml` lifecycle fields and event log (`internal/planmeta`)

- [x] **2.1** Replace the full contents of `cli/internal/planmeta/planmeta.go`:

  ```go
  package planmeta

  import (
  	"encoding/json"
  	"os"
  	"path/filepath"
  	"time"

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

  type Review struct {
  	Verdict        string `yaml:"verdict,omitempty"` // PASS | REJECT
  	BlockingIssues int    `yaml:"blocking_issues,omitempty"`
  	ReviewedAt     string `yaml:"reviewed_at,omitempty"`
  }

  type Verification struct {
  	Verdict    string `yaml:"verdict,omitempty"` // PASS | FAIL
  	VerifiedAt string `yaml:"verified_at,omitempty"`
  }

  type Meta struct {
  	Plan        string        `yaml:"plan"`
  	RiskLevel   string        `yaml:"risk_level"` // quick-fix | bug | feature | architecture | high-risk
  	PlannedAt   PlannedAt     `yaml:"planned_at"`
  	Status      string        `yaml:"status,omitempty"` // deprecated (Phase 2) — see State
  	State       string        `yaml:"state"`
  	WriteScope  []string      `yaml:"write_scope"`
  	Retry       RetryCounters `yaml:"retry"`
  	RetryBudget RetryBudget   `yaml:"retry_budget"`

  	RequiresApproval bool         `yaml:"requires_approval"`
  	ApprovedAt       string       `yaml:"approved_at,omitempty"`
  	ApprovedBy       string       `yaml:"approved_by,omitempty"`
  	Review           Review       `yaml:"review,omitempty"`
  	Verification     Verification `yaml:"verification,omitempty"`
  }

  const FileName = "plan.yaml"
  const EventsFileName = "events.jsonl"

  // legacyStatusToState migrates Phase 2's write-once `status` field to the
  // richer Phase 3 `state` enum for plan.yaml files created before Phase 3.
  var legacyStatusToState = map[string]string{
  	"planned":   "PLANNED",
  	"reviewed":  "REVIEWED",
  	"executing": "EXECUTING",
  	"verified":  "COMPLETED",
  	"failed":    "FAILED",
  }

  func Load(planDir string) (*Meta, error) {
  	data, err := os.ReadFile(filepath.Join(planDir, FileName))
  	if err != nil {
  		return nil, err
  	}
  	var m Meta
  	if err := yaml.Unmarshal(data, &m); err != nil {
  		return nil, err
  	}
  	if m.State == "" {
  		if mapped, ok := legacyStatusToState[m.Status]; ok {
  			m.State = mapped
  		} else {
  			m.State = "NEW"
  		}
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

  type Event struct {
  	Type   string `json:"type"`
  	At     string `json:"at"`
  	Detail string `json:"detail,omitempty"`
  }

  // AppendEvent records one line to <planDir>/events.jsonl. Append-only by
  // design — plan.yaml stays a small, current-state snapshot; the full
  // history of every transition lives here instead, per Phase 3's "preserve
  // history rather than overwriting evidence" requirement.
  func AppendEvent(planDir, eventType, detail string) error {
  	f, err := os.OpenFile(filepath.Join(planDir, EventsFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
  	if err != nil {
  		return err
  	}
  	defer f.Close()

  	ev := Event{Type: eventType, At: time.Now().UTC().Format(time.RFC3339), Detail: detail}
  	data, err := json.Marshal(ev)
  	if err != nil {
  		return err
  	}
  	_, err = f.Write(append(data, '\n'))
  	return err
  }
  ```

- [x] **2.2** Replace the full contents of `cli/internal/planmeta/planmeta_test.go`:

  ```go
  package planmeta

  import (
  	"bufio"
  	"os"
  	"path/filepath"
  	"testing"
  )

  func TestSaveLoadRoundTrip(t *testing.T) {
  	dir := t.TempDir()
  	m := &Meta{
  		Plan:       "2026-08-24-example",
  		RiskLevel:  "feature",
  		PlannedAt:  PlannedAt{GitSHA: "abc123"},
  		State:      "PLANNED",
  		WriteScope: []string{"src/api/**"},
  	}
  	if err := Save(dir, m); err != nil {
  		t.Fatal(err)
  	}
  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got.PlannedAt.GitSHA != "abc123" || got.State != "PLANNED" {
  		t.Fatalf("round-trip mismatch: %+v", got)
  	}
  }

  func TestDefaultBudget(t *testing.T) {
  	b := DefaultBudget()
  	if b.Build != 2 || b.UnitTest != 2 || b.IntegrationTest != 1 {
  		t.Fatalf("unexpected default budget: %+v", b)
  	}
  }

  func TestLegacyStatusMigratesToState(t *testing.T) {
  	dir := t.TempDir()
  	// Simulates a Phase-2-created plan.yaml: has `status`, no `state`.
  	os.WriteFile(filepath.Join(dir, FileName), []byte("plan: x\nstatus: executing\n"), 0o644)

  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got.State != "EXECUTING" {
  		t.Fatalf("expected EXECUTING, got %q", got.State)
  	}
  }

  func TestNoStatusOrStateDefaultsToNew(t *testing.T) {
  	dir := t.TempDir()
  	os.WriteFile(filepath.Join(dir, FileName), []byte("plan: x\n"), 0o644)

  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if got.State != "NEW" {
  		t.Fatalf("expected NEW, got %q", got.State)
  	}
  }

  func TestAppendEventWritesJSONLine(t *testing.T) {
  	dir := t.TempDir()
  	if err := AppendEvent(dir, "triaged", "feature"); err != nil {
  		t.Fatal(err)
  	}
  	if err := AppendEvent(dir, "approved", "alice"); err != nil {
  		t.Fatal(err)
  	}

  	f, err := os.Open(filepath.Join(dir, EventsFileName))
  	if err != nil {
  		t.Fatal(err)
  	}
  	defer f.Close()

  	var lines []string
  	scanner := bufio.NewScanner(f)
  	for scanner.Scan() {
  		lines = append(lines, scanner.Text())
  	}
  	if len(lines) != 2 {
  		t.Fatalf("expected 2 event lines, got %d: %v", len(lines), lines)
  	}
  }
  ```

---

## Task 3 — `project.Stack` adopts `executil.Command`

- [x] **3.1** In `cli/internal/project/project.go`, replace the `Stack` struct and its import
  block:

  Old:
  ```go
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
  ```

  New:
  ```go
  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"

  	"eng/internal/executil"
  )

  type Stack struct {
  	Type  string           `yaml:"type"`
  	Build executil.Command `yaml:"build_cmd"`
  	Test  executil.Command `yaml:"test_cmd"`
  	Run   executil.Command `yaml:"run_cmd"`
  	Lint  executil.Command `yaml:"lint_cmd"`
  }
  ```

  (A plain string like `build_cmd: "go build ./..."` in an existing `.agent/project.yaml`
  still parses correctly — `executil.Command`'s `UnmarshalYAML` treats a scalar as `.Shell`.)

- [x] **3.2** In `cli/internal/project/project_test.go`, update `TestSaveLoadRoundTrip` (the
  original Phase 1 test) to use the new field type:

  Old:
  ```go
  	cfg := &Config{ProjectName: "x", Mode: "modern", Stack: Stack{Type: "go"}}
  ```

  New:
  ```go
  	cfg := &Config{ProjectName: "x", Mode: "modern", Stack: Stack{Type: "go", Test: executil.Command{Shell: "go test ./..."}}}
  ```

  Add `"eng/internal/executil"` to the file's import block, and extend the assertion below it:

  Old:
  ```go
  	if got.Mode != "modern" || got.Stack.Type != "go" {
  		t.Fatalf("round-trip mismatch: %+v", got)
  	}
  ```

  New:
  ```go
  	if got.Mode != "modern" || got.Stack.Type != "go" || got.Stack.Test.Shell != "go test ./..." {
  		t.Fatalf("round-trip mismatch: %+v", got)
  	}
  ```

- [x] **3.3** Add a new test to `cli/internal/project/project_test.go` confirming a
  Phase-1/2-style plain-string command still parses:

  ```go
  func TestPlainStringStackCommandStillParses(t *testing.T) {
  	dir := t.TempDir()
  	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
  	content := "project_name: x\nmode: modern\nstack:\n  type: go\n  build_cmd: \"go build ./...\"\n"
  	os.WriteFile(filepath.Join(dir, ConfigPath), []byte(content), 0o644)

  	cfg, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if cfg.Stack.Build.Shell != "go build ./..." {
  		t.Fatalf("expected plain string to parse as Shell, got %+v", cfg.Stack.Build)
  	}
  }
  ```

- [x] **3.4** In `cli/init_cmd.go`, update the `Stack` construction and add the import:

  Old:
  ```go
  import (
  	"flag"
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/detect"
  	"eng/internal/project"
  )
  ```
  ```go
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
  ```

  New:
  ```go
  import (
  	"flag"
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/detect"
  	"eng/internal/executil"
  	"eng/internal/project"
  )
  ```
  ```go
  	cfg := &project.Config{
  		ProjectName:    filepath.Base(dir),
  		Mode:           mode,
  		HarnessProfile: "software",
  		Stack: project.Stack{
  			Type:  det.Type,
  			Build: executil.Command{Shell: det.Build},
  			Test:  executil.Command{Shell: det.Test},
  			Run:   executil.Command{Shell: det.Run},
  			Lint:  executil.Command{Shell: det.Lint},
  		},
  		EnabledSkills: []string{"engineering/karpathy-guidelines"},
  	}
  ```

---

## Task 4 — Lifecycle state machine (`internal/workflow`)

- [x] **4.1** Create `cli/internal/workflow/workflow.go`:

  ```go
  package workflow

  const (
  	StateNew           = "NEW"
  	StateTriaged       = "TRIAGED"
  	StatePlanned       = "PLANNED"
  	StateReviewed      = "REVIEWED"
  	StateApproved      = "APPROVED"
  	StateExecuting     = "EXECUTING"
  	StateVerifying     = "VERIFYING"
  	StateCompleted     = "COMPLETED"
  	StateBlocked       = "BLOCKED"
  	StateFailed        = "FAILED"
  	StateNeedsReplan   = "NEEDS_REPLAN"
  	StateNeedsApproval = "NEEDS_APPROVAL"
  	StateNeedsFix      = "NEEDS_FIX"
  	StateCancelled     = "CANCELLED"
  )

  // Terminal reports whether a state has no further automatic transitions —
  // eng workflow advance refuses to do anything once a plan reaches one of
  // these, requiring an explicit human action instead.
  func Terminal(state string) bool {
  	switch state {
  	case StateCompleted, StateFailed, StateCancelled, StateBlocked:
  		return true
  	default:
  		return false
  	}
  }

  // Facts is everything Decide needs, gathered by the caller from plan.yaml,
  // tasks.md, and .agent/project.yaml. Decide itself does no I/O, which makes
  // every transition rule independently testable.
  type Facts struct {
  	State               string
  	PlanFilesReady      bool // spec.md/tasks.md/tests.md all exist and are non-empty
  	ReviewRequired      bool
  	ReviewVerdict       string // "" | PASS | REJECT
  	RequiresApproval    bool
  	Approved            bool
  	DriftDetected       bool
  	TasksComplete       bool   // zero remaining "- [ ]" lines in tasks.md
  	VerificationVerdict string // "" | PASS | FAIL
  	RetryExhausted      bool
  }

  // Decision is the one transition Decide recommends for the current Facts,
  // plus a side effect hint the caller may need to perform (e.g. running
  // `eng verify`) before the next Decide call would see updated Facts.
  type Decision struct {
  	NextState string
  	Reason    string
  	Action    string // "" | "run_verify"
  }

  // Decide implements the Phase 3 spec.md "Design decisions / Decision 6"
  // transition table exactly. It applies at most one transition per call —
  // the caller (eng workflow advance) never chains multiple automatic
  // transitions silently past a state a human should see.
  func Decide(f Facts) Decision {
  	switch f.State {
  	case StateTriaged:
  		if f.PlanFilesReady {
  			return Decision{NextState: StatePlanned, Reason: "spec.md/tasks.md/tests.md are present"}
  		}
  		return Decision{NextState: StateTriaged, Reason: "waiting on Planner to write spec.md/tasks.md/tests.md"}

  	case StatePlanned:
  		if !f.ReviewRequired {
  			return Decision{NextState: StateReviewed, Reason: "plan review not required for this project/risk level"}
  		}
  		switch f.ReviewVerdict {
  		case "PASS":
  			return Decision{NextState: StateReviewed, Reason: "review verdict PASS"}
  		case "REJECT":
  			return Decision{NextState: StateNeedsReplan, Reason: "review verdict REJECT"}
  		default:
  			return Decision{NextState: StatePlanned, Reason: "waiting on `eng plan review`"}
  		}

  	case StateReviewed:
  		if !f.RequiresApproval || f.Approved {
  			return Decision{NextState: StateApproved, Reason: "approval not required or already granted"}
  		}
  		return Decision{NextState: StateNeedsApproval, Reason: "run `eng plan approve` before execution can begin"}

  	case StateNeedsApproval:
  		if f.Approved {
  			return Decision{NextState: StateApproved, Reason: "approval granted"}
  		}
  		return Decision{NextState: StateNeedsApproval, Reason: "still waiting on `eng plan approve`"}

  	case StateApproved:
  		if f.DriftDetected {
  			return Decision{NextState: StateNeedsReplan, Reason: "PLAN_DRIFT_DETECTED before execution started"}
  		}
  		return Decision{NextState: StateExecuting, Reason: "no drift detected — Executor may begin"}

  	case StateExecuting, StateNeedsFix:
  		if f.TasksComplete {
  			return Decision{NextState: StateVerifying, Reason: "all tasks.md items checked off", Action: "run_verify"}
  		}
  		return Decision{NextState: f.State, Reason: "tasks.md still has unchecked items"}

  	case StateVerifying:
  		switch f.VerificationVerdict {
  		case "PASS":
  			return Decision{NextState: StateCompleted, Reason: "eng verify reported PASS"}
  		case "FAIL":
  			if f.RetryExhausted {
  				return Decision{NextState: StateFailed, Reason: "eng verify FAILed and the retry budget is exhausted"}
  			}
  			return Decision{NextState: StateNeedsFix, Reason: "eng verify FAILed — retry budget remains"}
  		default:
  			return Decision{NextState: StateVerifying, Reason: "waiting on `eng verify`"}
  		}

  	case StateNeedsReplan:
  		return Decision{NextState: StatePlanned, Reason: "replanning acknowledged — re-entering review"}

  	default:
  		return Decision{NextState: f.State, Reason: "no automatic transition from this state"}
  	}
  }
  ```

- [x] **4.2** Create `cli/internal/workflow/workflow_test.go`:

  ```go
  package workflow

  import "testing"

  func TestTriagedWaitsForPlanFiles(t *testing.T) {
  	d := Decide(Facts{State: StateTriaged, PlanFilesReady: false})
  	if d.NextState != StateTriaged {
  		t.Fatalf("expected to stay TRIAGED, got %+v", d)
  	}
  	d = Decide(Facts{State: StateTriaged, PlanFilesReady: true})
  	if d.NextState != StatePlanned {
  		t.Fatalf("expected PLANNED, got %+v", d)
  	}
  }

  func TestPlannedReviewPassAndReject(t *testing.T) {
  	pass := Decide(Facts{State: StatePlanned, ReviewRequired: true, ReviewVerdict: "PASS"})
  	if pass.NextState != StateReviewed {
  		t.Fatalf("expected REVIEWED, got %+v", pass)
  	}
  	reject := Decide(Facts{State: StatePlanned, ReviewRequired: true, ReviewVerdict: "REJECT"})
  	if reject.NextState != StateNeedsReplan {
  		t.Fatalf("expected NEEDS_REPLAN, got %+v", reject)
  	}
  	skip := Decide(Facts{State: StatePlanned, ReviewRequired: false})
  	if skip.NextState != StateReviewed {
  		t.Fatalf("expected REVIEWED when review not required, got %+v", skip)
  	}
  }

  func TestApprovalGate(t *testing.T) {
  	blocked := Decide(Facts{State: StateReviewed, RequiresApproval: true, Approved: false})
  	if blocked.NextState != StateNeedsApproval {
  		t.Fatalf("expected NEEDS_APPROVAL, got %+v", blocked)
  	}
  	approved := Decide(Facts{State: StateReviewed, RequiresApproval: true, Approved: true})
  	if approved.NextState != StateApproved {
  		t.Fatalf("expected APPROVED, got %+v", approved)
  	}
  	notNeeded := Decide(Facts{State: StateReviewed, RequiresApproval: false})
  	if notNeeded.NextState != StateApproved {
  		t.Fatalf("expected APPROVED when not required, got %+v", notNeeded)
  	}
  }

  func TestDriftBeforeExecutionForcesReplan(t *testing.T) {
  	d := Decide(Facts{State: StateApproved, DriftDetected: true})
  	if d.NextState != StateNeedsReplan {
  		t.Fatalf("expected NEEDS_REPLAN, got %+v", d)
  	}
  	ok := Decide(Facts{State: StateApproved, DriftDetected: false})
  	if ok.NextState != StateExecuting {
  		t.Fatalf("expected EXECUTING, got %+v", ok)
  	}
  }

  func TestExecutingToVerifyingTriggersVerify(t *testing.T) {
  	d := Decide(Facts{State: StateExecuting, TasksComplete: true})
  	if d.NextState != StateVerifying || d.Action != "run_verify" {
  		t.Fatalf("expected VERIFYING with run_verify action, got %+v", d)
  	}
  	notDone := Decide(Facts{State: StateExecuting, TasksComplete: false})
  	if notDone.NextState != StateExecuting {
  		t.Fatalf("expected to stay EXECUTING, got %+v", notDone)
  	}
  }

  func TestVerifyingRoutesOnVerdict(t *testing.T) {
  	pass := Decide(Facts{State: StateVerifying, VerificationVerdict: "PASS"})
  	if pass.NextState != StateCompleted {
  		t.Fatalf("expected COMPLETED, got %+v", pass)
  	}
  	failRetry := Decide(Facts{State: StateVerifying, VerificationVerdict: "FAIL", RetryExhausted: false})
  	if failRetry.NextState != StateNeedsFix {
  		t.Fatalf("expected NEEDS_FIX, got %+v", failRetry)
  	}
  	failExhausted := Decide(Facts{State: StateVerifying, VerificationVerdict: "FAIL", RetryExhausted: true})
  	if failExhausted.NextState != StateFailed {
  		t.Fatalf("expected FAILED, got %+v", failExhausted)
  	}
  }

  func TestTerminalStates(t *testing.T) {
  	for _, s := range []string{StateCompleted, StateFailed, StateCancelled, StateBlocked} {
  		if !Terminal(s) {
  			t.Fatalf("%s should be terminal", s)
  		}
  	}
  	for _, s := range []string{StateNew, StateTriaged, StateExecuting} {
  		if Terminal(s) {
  			t.Fatalf("%s should not be terminal", s)
  		}
  	}
  }
  ```

---

## Task 5 — Workflow profiles

- [x] **5.1** Create `cli/internal/workflow/profile.go`:

  ```go
  package workflow

  import (
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"
  )

  // Profile is a named, ordered, descriptive list of stages a plan's risk
  // level routes through. It is informational (used by `eng workflow status`
  // to show the human the whole path) — the authoritative gating logic is
  // Decide's transition table, not this file.
  type Profile struct {
  	Name   string   `yaml:"name"`
  	Stages []string `yaml:"stages"`
  }

  func LoadProfile(harnessDir, name string) (*Profile, error) {
  	path := filepath.Join(harnessDir, "workflows", name+".yaml")
  	data, err := os.ReadFile(path)
  	if err != nil {
  		return nil, err
  	}
  	var p Profile
  	if err := yaml.Unmarshal(data, &p); err != nil {
  		return nil, err
  	}
  	return &p, nil
  }

  // ProfileForRiskLevel maps a risk_level (per core/triage/METHOD.md) to its
  // workflow profile's file name under harness/workflows/.
  func ProfileForRiskLevel(riskLevel string) string {
  	switch riskLevel {
  	case "quick-fix":
  		return "quick-fix"
  	case "bug":
  		return "bug-fix"
  	case "architecture":
  		return "architecture"
  	case "high-risk":
  		return "high-risk"
  	default:
  		return "feature"
  	}
  }
  ```

- [x] **5.2** Create `cli/internal/workflow/profile_test.go`:

  ```go
  package workflow

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func TestLoadProfile(t *testing.T) {
  	harness := t.TempDir()
  	os.MkdirAll(filepath.Join(harness, "workflows"), 0o755)
  	os.WriteFile(filepath.Join(harness, "workflows", "feature.yaml"),
  		[]byte("name: feature\nstages: [triage, plan, review, execute, verify]\n"), 0o644)

  	p, err := LoadProfile(harness, "feature")
  	if err != nil {
  		t.Fatal(err)
  	}
  	if p.Name != "feature" || len(p.Stages) != 5 {
  		t.Fatalf("got %+v", p)
  	}
  }

  func TestProfileForRiskLevel(t *testing.T) {
  	cases := map[string]string{
  		"quick-fix":    "quick-fix",
  		"bug":          "bug-fix",
  		"architecture": "architecture",
  		"high-risk":    "high-risk",
  		"feature":      "feature",
  		"":             "feature",
  	}
  	for risk, want := range cases {
  		if got := ProfileForRiskLevel(risk); got != want {
  			t.Errorf("ProfileForRiskLevel(%q) = %q, want %q", risk, got, want)
  		}
  	}
  }
  ```

- [x] **5.3** Create the five profile files:

  `harness/workflows/quick-fix.yaml`:
  ```yaml
  name: quick-fix
  stages: [triage, execute, verify]
  ```

  `harness/workflows/bug-fix.yaml`:
  ```yaml
  name: bug-fix
  stages: [triage, plan, execute, verify]
  ```

  `harness/workflows/feature.yaml`:
  ```yaml
  name: feature
  stages: [triage, plan, review, execute, verify]
  ```

  `harness/workflows/architecture.yaml`:
  ```yaml
  name: architecture
  stages: [triage, plan, review, approval, execute, verify]
  ```

  `harness/workflows/high-risk.yaml`:
  ```yaml
  name: high-risk
  stages: [triage, plan, review, approval, execute, verify]
  ```

---

## Task 6 — Capability registry (`internal/capabilities`)

- [x] **6.1** Create `cli/internal/capabilities/capabilities.go`:

  ```go
  package capabilities

  import "os/exec"

  // Known is the fixed set of capabilities eng can detect today. Device/
  // protocol capabilities (serial, Modbus, OPC UA, ...) are explicitly
  // deferred to a later phase alongside the MCP adapter layer they'd serve.
  var Known = []string{"git", "claude", "codex", "docker"}

  // Detect reports whether name's executable is found on PATH.
  func Detect(name string) bool {
  	_, err := exec.LookPath(name)
  	return err == nil
  }

  // DetectAll returns every Known capability mapped to its availability.
  func DetectAll() map[string]bool {
  	out := make(map[string]bool, len(Known))
  	for _, name := range Known {
  		out[name] = Detect(name)
  	}
  	return out
  }
  ```

- [x] **6.2** Create `cli/internal/capabilities/capabilities_test.go`:

  ```go
  package capabilities

  import "testing"

  func TestDetectMissingBinary(t *testing.T) {
  	if Detect("definitely-not-a-real-binary-xyz") {
  		t.Fatal("expected false for a nonexistent binary")
  	}
  }

  func TestDetectAllCoversKnownSet(t *testing.T) {
  	all := DetectAll()
  	if len(all) != len(Known) {
  		t.Fatalf("expected %d entries, got %d", len(Known), len(all))
  	}
  	for _, name := range Known {
  		if _, ok := all[name]; !ok {
  			t.Fatalf("missing %q in DetectAll result", name)
  		}
  	}
  }

  func TestDetectGitIsUsuallyPresent(t *testing.T) {
  	// This repository is a git repo being worked on right now — git must be
  	// on PATH for that to be possible at all.
  	if !Detect("git") {
  		t.Skip("git not found on PATH in this environment")
  	}
  }
  ```

---

## Task 7 — Agent adapter interface (`internal/agent`)

- [x] **7.1** Create `cli/internal/agent/agent.go`:

  ```go
  package agent

  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/capabilities"
  )

  type Role string

  const (
  	RolePlanner  Role = "planner"
  	RoleReviewer Role = "plan-reviewer"
  	RoleExecutor Role = "executor"
  	RoleVerifier Role = "verifier"
  )

  // Adapter lets eng talk to a specific AI coding agent. Implementations
  // detect availability and assemble role-specific prompts; none of them
  // drive an agent unattended — see Phase 3 DECISION_LOG for why.
  type Adapter interface {
  	Name() string
  	Available() bool
  	RolePrompt(role Role, planDir string) (string, error)
  }

  // ClaudeCodeAdapter is the reference implementation — the only one Phase 3
  // implements, per the explicit "prioritize Claude Code, don't deeply
  // integrate every agent" instruction.
  type ClaudeCodeAdapter struct {
  	HarnessDir string
  }

  func (a ClaudeCodeAdapter) Name() string { return "claude-code" }

  func (a ClaudeCodeAdapter) Available() bool { return capabilities.Detect("claude") }

  func (a ClaudeCodeAdapter) RolePrompt(role Role, planDir string) (string, error) {
  	methodPath := filepath.Join(a.HarnessDir, "core", string(role), "METHOD.md")
  	method, err := os.ReadFile(methodPath)
  	if err != nil {
  		return "", fmt.Errorf("reading %s: %w", methodPath, err)
  	}

  	abs, err := filepath.Abs(planDir)
  	if err != nil {
  		abs = planDir
  	}

  	return fmt.Sprintf(`%s

  ---

  You are acting in the %s role for the plan at: %s

  Read spec.md, tasks.md, tests.md, and plan.yaml in that folder before doing anything else.
  `, string(method), role, abs), nil
  }
  ```

- [x] **7.2** Create `cli/internal/agent/agent_test.go`:

  ```go
  package agent

  import (
  	"os"
  	"path/filepath"
  	"strings"
  	"testing"
  )

  func TestRolePromptIncludesMethodAndPlanPath(t *testing.T) {
  	harness := t.TempDir()
  	methodDir := filepath.Join(harness, "core", "executor")
  	os.MkdirAll(methodDir, 0o755)
  	os.WriteFile(filepath.Join(methodDir, "METHOD.md"), []byte("# Core Method: Executor\nDo the thing."), 0o644)

  	a := ClaudeCodeAdapter{HarnessDir: harness}
  	prompt, err := a.RolePrompt(RoleExecutor, "/some/plan/dir")
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !strings.Contains(prompt, "Do the thing.") {
  		t.Fatalf("prompt missing method content: %s", prompt)
  	}
  	if !strings.Contains(prompt, "executor") {
  		t.Fatalf("prompt missing role name: %s", prompt)
  	}
  }

  func TestRolePromptErrorsOnMissingMethod(t *testing.T) {
  	a := ClaudeCodeAdapter{HarnessDir: t.TempDir()}
  	if _, err := a.RolePrompt(RolePlanner, "."); err == nil {
  		t.Fatal("expected an error for a missing METHOD.md")
  	}
  }

  func TestAvailableReturnsWithoutPanicking(t *testing.T) {
  	a := ClaudeCodeAdapter{}
  	_ = a.Available()
  }
  ```

- [x] **7.3** Create `harness/adapters/claude-code/ADAPTER.md`:

  ```markdown
  # Adapter: Claude Code

  Reference implementation of the `internal/agent.Adapter` interface.

  ## Capability

  Detected via `claude` on PATH (see `eng capabilities list`).

  ## Responsibilities implemented

  - **Detect** — `Available()` checks the capability registry.
  - **Provide role instructions** — `RolePrompt(role, planDir)` reads
    `core/<role>/METHOD.md` and prepends it to a short block naming the plan directory and
    which files to read first.

  ## Responsibilities NOT implemented (by design — see Phase 3 DECISION_LOG)

  - **Launch agent (non-interactive)** — `eng start` launches an *interactive* `claude`
    session attached to the current terminal; it does not pipe a prompt in and run unattended.
  - **Collect result** — meaningless without non-interactive invocation; deferred.

  ## Adding a new adapter

  Implement `internal/agent.Adapter` (`Name`, `Available`, `RolePrompt`) and register it where
  `eng adapter`/`eng start` select an adapter. No other file in this repository should need to
  change — this is exactly the boundary the interface exists to draw.
  ```

---

## Task 8 — `plan_cmd.go`: review/approve/block/cancel, drift extraction, event logging

- [x] **8.1** Replace the full contents of `cli/plan_cmd.go`:

  ```go
  package main

  import (
  	"flag"
  	"fmt"
  	"os"
  	"path/filepath"
  	"time"

  	"eng/internal/gitutil"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/workflow"
  )

  func cmdPlan(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng plan <new|drift|retry|review|approve|block|cancel> ...")
  		os.Exit(1)
  	}
  	switch args[0] {
  	case "new":
  		planNew(args[1:])
  	case "drift":
  		planDrift(args[1:])
  	case "retry":
  		planRetry(args[1:])
  	case "review":
  		planReview(args[1:])
  	case "approve":
  		planApprove(args[1:])
  	case "block":
  		planBlock(args[1:])
  	case "cancel":
  		planCancel(args[1:])
  	default:
  		fmt.Println("Usage: eng plan <new|drift|retry|review|approve|block|cancel> ...")
  		os.Exit(1)
  	}
  }

  func planNew(args []string) {
  	flagset := flag.NewFlagSet("plan new", flag.ExitOnError)
  	risk := flagset.String("risk", "feature", "quick-fix|bug|feature|architecture|high-risk")
  	requiresApproval := flagset.Bool("requires-approval", false, "force an approval gate regardless of risk level")
  	flagset.Parse(args)
  	rest := flagset.Args()
  	if len(rest) == 0 {
  		fmt.Println("Usage: eng plan new <name> [--risk <level>] [--requires-approval]")
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

  	needsApproval := *requiresApproval || *risk == "high-risk"

  	meta := &planmeta.Meta{
  		Plan:             filepath.Base(planDir),
  		RiskLevel:        *risk,
  		PlannedAt:        planmeta.PlannedAt{GitSHA: sha},
  		State:            workflow.StateTriaged,
  		RetryBudget:      budget,
  		RequiresApproval: needsApproval,
  	}
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	planmeta.AppendEvent(planDir, "triaged", *risk)

  	fmt.Printf("Scaffolded %s — risk: %s, git_sha: %s, requires_approval: %v\n", planDir, *risk, sha, needsApproval)
  }

  // copyTree is defined in install.go and reused here.

  // checkDrift is the pure logic behind `eng plan drift`, factored out so
  // `eng workflow advance` can consult it without re-printing anything.
  func checkDrift(planDir string) (bool, []string, error) {
  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		return false, nil, fmt.Errorf("no %s found in %s", planmeta.FileName, planDir)
  	}
  	repoRoot, err := os.Getwd()
  	if err != nil {
  		return false, nil, err
  	}
  	changed, err := gitutil.ChangedFilesSince(repoRoot, meta.PlannedAt.GitSHA)
  	if err != nil {
  		return false, nil, err
  	}
  	if len(changed) == 0 {
  		return false, nil, nil
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
  	return len(relevant) > 0, relevant, nil
  }

  func planDrift(args []string) {
  	dir := "."
  	if len(args) > 0 {
  		dir = args[0]
  	}
  	planDir, _ := filepath.Abs(dir)

  	drifted, files, err := checkDrift(planDir)
  	if err != nil {
  		fmt.Println(err)
  		os.Exit(1)
  	}
  	if !drifted {
  		fmt.Println("OK — no changes since plan was created")
  		return
  	}

  	fmt.Println("PLAN_DRIFT_DETECTED — the following files changed since this plan was created:")
  	for _, f := range files {
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
  	planmeta.AppendEvent(planDir, "retry", stage)

  	if *count > *limit {
  		fmt.Printf("RETRY BUDGET EXHAUSTED for %s (%d/%d) — escalate to Planner or human\n", stage, *count, *limit)
  		os.Exit(1)
  	}
  	fmt.Printf("RETRY %d/%d for %s — proceed\n", *count, *limit, stage)
  }

  func planReview(args []string) {
  	flagset := flag.NewFlagSet("plan review", flag.ExitOnError)
  	verdict := flagset.String("verdict", "", "PASS|REJECT")
  	blocking := flagset.Int("blocking-issues", 0, "number of blocking issues found")
  	flagset.Parse(args)
  	rest := flagset.Args()
  	if len(rest) == 0 || (*verdict != "PASS" && *verdict != "REJECT") {
  		fmt.Println("Usage: eng plan review <plan-dir> --verdict PASS|REJECT [--blocking-issues N]")
  		os.Exit(1)
  	}
  	planDir, _ := filepath.Abs(rest[0])

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	meta.Review = planmeta.Review{
  		Verdict:        *verdict,
  		BlockingIssues: *blocking,
  		ReviewedAt:     time.Now().UTC().Format(time.RFC3339),
  	}
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	planmeta.AppendEvent(planDir, "reviewed", *verdict)
  	fmt.Printf("Recorded review verdict: %s (%d blocking issues)\n", *verdict, *blocking)
  }

  func planApprove(args []string) {
  	flagset := flag.NewFlagSet("plan approve", flag.ExitOnError)
  	by := flagset.String("by", "", "who is approving")
  	flagset.Parse(args)
  	rest := flagset.Args()
  	if len(rest) == 0 {
  		fmt.Println("Usage: eng plan approve <plan-dir> [--by <name>]")
  		os.Exit(1)
  	}
  	planDir, _ := filepath.Abs(rest[0])

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	meta.ApprovedAt = time.Now().UTC().Format(time.RFC3339)
  	meta.ApprovedBy = *by
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	planmeta.AppendEvent(planDir, "approved", *by)
  	fmt.Printf("Approved by %q at %s\n", *by, meta.ApprovedAt)
  }

  func planBlock(args []string) {
  	flagset := flag.NewFlagSet("plan block", flag.ExitOnError)
  	reason := flagset.String("reason", "", "why this plan is blocked")
  	flagset.Parse(args)
  	rest := flagset.Args()
  	if len(rest) == 0 {
  		fmt.Println(`Usage: eng plan block <plan-dir> --reason "..."`)
  		os.Exit(1)
  	}
  	setTerminalState(rest[0], workflow.StateBlocked, "blocked", *reason)
  }

  func planCancel(args []string) {
  	flagset := flag.NewFlagSet("plan cancel", flag.ExitOnError)
  	reason := flagset.String("reason", "", "why this plan is cancelled")
  	flagset.Parse(args)
  	rest := flagset.Args()
  	if len(rest) == 0 {
  		fmt.Println(`Usage: eng plan cancel <plan-dir> [--reason "..."]`)
  		os.Exit(1)
  	}
  	setTerminalState(rest[0], workflow.StateCancelled, "cancelled", *reason)
  }

  func setTerminalState(dir, state, eventType, reason string) {
  	planDir, _ := filepath.Abs(dir)
  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}
  	meta.State = state
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	planmeta.AppendEvent(planDir, eventType, reason)
  	suffix := ""
  	if reason != "" {
  		suffix = fmt.Sprintf(" (%s)", reason)
  	}
  	fmt.Printf("State set to %s%s\n", state, suffix)
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

---

## Task 9 — `eng verify` persists a machine-readable verdict and uses `executil`

- [x] **9.1** Replace the full contents of `cli/verify_cmd.go`:

  ```go
  package main

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

  	pass, report, err := runVerify(planDir)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Println(report)
  	if !pass {
  		os.Exit(1)
  	}
  }

  // runVerify performs the actual verification and returns pass/fail plus the
  // report text, without calling os.Exit — factored out so `eng workflow
  // advance` can call it directly and decide what to do with the result
  // itself, rather than having the whole orchestrator process die on FAIL.
  func runVerify(planDir string) (bool, string, error) {
  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		return false, "", fmt.Errorf("no %s found in %s — nothing to verify", planmeta.FileName, planDir)
  	}

  	repoRoot, err := os.Getwd()
  	if err != nil {
  		return false, "", err
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

  	if cfg, err := project.Load(repoRoot); err == nil && !cfg.Stack.Test.Empty() {
  		fmt.Fprintf(&report, "\n## Test run\n\nCommand: `%s`\n\n", cfg.Stack.Test.String())
  		out, testErr := executil.Run(cfg.Stack.Test, repoRoot)
  		fmt.Fprintf(&report, "```\n%s\n```\n\n", out)
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

  	meta.Verification = planmeta.Verification{Verdict: verdict, VerifiedAt: time.Now().UTC().Format(time.RFC3339)}
  	if err := planmeta.Save(planDir, meta); err != nil {
  		return pass, report.String(), fmt.Errorf("verification ran but failed to persist to plan.yaml: %w", err)
  	}
  	planmeta.AppendEvent(planDir, "verified", verdict)

  	if err := os.WriteFile(filepath.Join(planDir, "verify-report.md"), []byte(report.String()), 0o644); err != nil {
  		return pass, report.String(), err
  	}

  	return pass, report.String(), nil
  }
  ```

---

## Task 10 — `eng hooks run` adopts `executil.Command`

- [x] **10.1** In `cli/internal/hooks/hooks.go`, change the `Commands` field type and add the
  import:

  Old:
  ```go
  import (
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"
  )
  ```
  ```go
  	Commands      map[string]string `yaml:"commands"`
  ```

  New:
  ```go
  import (
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"

  	"eng/internal/executil"
  )
  ```
  ```go
  	Commands      map[string]executil.Command `yaml:"commands"`
  ```

- [x] **10.2** Replace the full contents of `cli/hooks_cmd.go`:

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

---

## Task 11 — The orchestrator: `eng workflow start/status/advance`

- [x] **11.1** In `cli/triage_cmd.go`, extract the pure heuristic so `eng workflow start` can
  reuse it. Replace the full contents:

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

  // triageLevel is the pure heuristic, factored out so `eng workflow start`
  // can call it directly instead of going through cmdTriage's print/exit path.
  func triageLevel(text string) (level, workflowDesc string) {
  	lower := strings.ToLower(text)
  	for _, k := range triageKeywords {
  		for _, w := range k.words {
  			if strings.Contains(lower, w) {
  				return k.level, k.workflow
  			}
  		}
  	}
  	return "feature", "full spec + tasks + tests"
  }

  func cmdTriage(args []string) {
  	if len(args) == 0 {
  		fmt.Println(`Usage: eng triage "<request text>"`)
  		os.Exit(1)
  	}
  	level, wf := triageLevel(strings.Join(args, " "))
  	fmt.Printf("Suggested level: %s\n", level)
  	fmt.Printf("Suggested workflow: %s\n", wf)
  	fmt.Println("\n(heuristic hint only — the Planner makes the final call)")
  }
  ```

- [x] **11.2** Create `cli/workflow_cmd.go`:

  ```go
  package main

  import (
  	"bufio"
  	"fmt"
  	"os"
  	"path/filepath"
  	"regexp"
  	"strings"
  	"time"

  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/workflow"
  )

  func cmdWorkflow(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng workflow <start|status|advance> ...")
  		os.Exit(1)
  	}
  	switch args[0] {
  	case "start":
  		workflowStart(args[1:])
  	case "status":
  		workflowStatus(args[1:])
  	case "advance":
  		workflowAdvance(args[1:])
  	default:
  		fmt.Println("Usage: eng workflow <start|status|advance> ...")
  		os.Exit(1)
  	}
  }

  var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

  func slugify(text string) string {
  	s := strings.ToLower(text)
  	s = slugRe.ReplaceAllString(s, "-")
  	s = strings.Trim(s, "-")
  	if len(s) > 40 {
  		s = s[:40]
  	}
  	if s == "" {
  		s = "request"
  	}
  	return s
  }

  func workflowStart(args []string) {
  	if len(args) == 0 {
  		fmt.Println(`Usage: eng workflow start "<requirement text>"`)
  		os.Exit(1)
  	}
  	text := strings.Join(args, " ")

  	level, _ := triageLevel(text)
  	name := slugify(text)

  	fmt.Printf("Triage suggests level: %s\n", level)
  	planNew([]string{"--risk", level, name})

  	repoRoot, _ := os.Getwd()
  	planDir := filepath.Join(repoRoot, ".plans", time.Now().Format("2006-01-02")+"-"+name)
  	workflowStatus([]string{planDir})
  }

  func workflowStatus(args []string) {
  	dir := "."
  	if len(args) > 0 {
  		dir = args[0]
  	}
  	planDir, _ := filepath.Abs(dir)

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	profileName := workflow.ProfileForRiskLevel(meta.RiskLevel)
  	profile, perr := workflow.LoadProfile(harnessDir(), profileName)

  	fmt.Printf("Plan:          %s\n", meta.Plan)
  	fmt.Printf("Risk level:    %s\n", meta.RiskLevel)
  	if perr == nil {
  		fmt.Printf("Profile:       %s (%s)\n", profile.Name, strings.Join(profile.Stages, " -> "))
  	}
  	fmt.Printf("State:         %s\n", meta.State)
  	fmt.Printf("Requires approval: %v", meta.RequiresApproval)
  	if meta.RequiresApproval {
  		if meta.ApprovedAt != "" {
  			fmt.Printf(" (approved by %q at %s)", meta.ApprovedBy, meta.ApprovedAt)
  		} else {
  			fmt.Print(" (NOT yet approved)")
  		}
  	}
  	fmt.Println()

  	facts, err := gatherFacts(planDir, meta)
  	if err != nil {
  		fmt.Println("error gathering state:", err)
  		return
  	}
  	decision := workflow.Decide(facts)
  	fmt.Printf("Next:          %s\n", decision.Reason)
  }

  func workflowAdvance(args []string) {
  	dir := "."
  	if len(args) > 0 {
  		dir = args[0]
  	}
  	planDir, _ := filepath.Abs(dir)

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	if workflow.Terminal(meta.State) {
  		fmt.Printf("Plan is in terminal state %s — nothing to advance\n", meta.State)
  		return
  	}

  	facts, err := gatherFacts(planDir, meta)
  	if err != nil {
  		fmt.Println("error gathering state:", err)
  		os.Exit(1)
  	}
  	decision := workflow.Decide(facts)

  	if decision.NextState == meta.State {
  		fmt.Printf("Still in %s — %s\n", meta.State, decision.Reason)
  		printNextAction(meta.State, planDir)
  		return
  	}

  	fmt.Printf("%s -> %s (%s)\n", meta.State, decision.NextState, decision.Reason)
  	meta.State = decision.NextState
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	planmeta.AppendEvent(planDir, "state_changed", meta.State)

  	if decision.Action == "run_verify" {
  		fmt.Println("Running eng verify automatically...")
  		_, report, verr := runVerify(planDir)
  		if verr != nil {
  			fmt.Println("error running verify:", verr)
  			os.Exit(1)
  		}
  		fmt.Println(report)

  		// One additional Decide call is safe here — and only here — because
  		// it is reacting to the fresh Verification fact runVerify just wrote,
  		// not chaining further speculative transitions.
  		meta, _ = planmeta.Load(planDir)
  		facts, _ = gatherFacts(planDir, meta)
  		decision = workflow.Decide(facts)
  		if decision.NextState != meta.State {
  			fmt.Printf("%s -> %s (%s)\n", meta.State, decision.NextState, decision.Reason)
  			meta.State = decision.NextState
  			planmeta.Save(planDir, meta)
  			planmeta.AppendEvent(planDir, "state_changed", meta.State)
  		}
  	}

  	printNextAction(meta.State, planDir)
  }

  func printNextAction(state, planDir string) {
  	switch state {
  	case workflow.StateTriaged, workflow.StateNeedsReplan:
  		fmt.Printf("Next action: run `eng adapter prompt planner %s`\n", planDir)
  	case workflow.StatePlanned:
  		fmt.Printf("Next action: run `eng adapter prompt plan-reviewer %s`, then `eng plan review %s --verdict ...`\n", planDir, planDir)
  	case workflow.StateNeedsApproval:
  		fmt.Printf("Next action: run `eng plan approve %s`\n", planDir)
  	case workflow.StateExecuting, workflow.StateNeedsFix:
  		fmt.Printf("Next action: run `eng adapter prompt executor %s`\n", planDir)
  	case workflow.StateCompleted:
  		fmt.Println("Next action: none — plan is complete")
  	case workflow.StateFailed, workflow.StateBlocked, workflow.StateCancelled:
  		fmt.Println("Next action: human decision required — this plan will not advance automatically")
  	}
  }

  // gatherFacts reads plan.yaml, tasks.md, and .agent/project.yaml to build the
  // pure Facts workflow.Decide needs — this is the only place doing I/O.
  func gatherFacts(planDir string, meta *planmeta.Meta) (workflow.Facts, error) {
  	repoRoot, err := os.Getwd()
  	if err != nil {
  		return workflow.Facts{}, err
  	}

  	reviewRequired := meta.RiskLevel == "architecture" || meta.RiskLevel == "high-risk"
  	if cfg, err := project.Load(repoRoot); err == nil {
  		reviewRequired = reviewRequired || cfg.EffectiveWorkflow().PlanReview
  	}

  	drifted, _, _ := checkDrift(planDir)

  	return workflow.Facts{
  		State:               meta.State,
  		PlanFilesReady:      filesReady(planDir),
  		ReviewRequired:      reviewRequired,
  		ReviewVerdict:       meta.Review.Verdict,
  		RequiresApproval:    meta.RequiresApproval,
  		Approved:            meta.ApprovedAt != "",
  		DriftDetected:       drifted,
  		TasksComplete:       tasksComplete(planDir),
  		VerificationVerdict: meta.Verification.Verdict,
  		RetryExhausted:      meta.Retry.UnitTest > meta.RetryBudget.UnitTest || meta.Retry.Build > meta.RetryBudget.Build || meta.Retry.IntegrationTest > meta.RetryBudget.IntegrationTest,
  	}, nil
  }

  // filesReady checks more than existence: eng plan new scaffolds spec.md/
  // tasks.md/tests.md from harness/templates/plan/, which are non-empty
  // *template* content, not a real plan. Treating that as "ready" would let
  // TRIAGED jump straight to PLANNED before the Planner ever did anything —
  // so this also requires spec.md's placeholder title to have been replaced.
  func filesReady(planDir string) bool {
  	for _, n := range []string{"spec.md", "tasks.md", "tests.md"} {
  		info, err := os.Stat(filepath.Join(planDir, n))
  		if err != nil || info.Size() == 0 {
  			return false
  		}
  	}
  	specData, err := os.ReadFile(filepath.Join(planDir, "spec.md"))
  	if err != nil {
  		return false
  	}
  	return !strings.Contains(string(specData), "[Feature Name]")
  }

  func tasksComplete(planDir string) bool {
  	f, err := os.Open(filepath.Join(planDir, "tasks.md"))
  	if err != nil {
  		return false
  	}
  	defer f.Close()
  	scanner := bufio.NewScanner(f)
  	for scanner.Scan() {
  		if strings.HasPrefix(scanner.Text(), "- [ ]") {
  			return false
  		}
  	}
  	return true
  }
  ```

---

## Task 12 — `eng adapter prompt`

- [x] **12.1** Create `cli/adapter_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/agent"
  )

  func cmdAdapter(args []string) {
  	if len(args) < 2 || args[0] != "prompt" {
  		fmt.Println("Usage: eng adapter prompt <planner|plan-reviewer|executor|verifier> <plan-dir>")
  		os.Exit(1)
  	}
  	role := agent.Role(args[1])
  	if len(args) < 3 {
  		fmt.Println("Usage: eng adapter prompt <role> <plan-dir>")
  		os.Exit(1)
  	}
  	planDir := args[2]

  	a := agent.ClaudeCodeAdapter{HarnessDir: harnessDir()}
  	if !a.Available() {
  		fmt.Println("note: `claude` was not found on PATH — printing the prompt for manual use anyway.")
  	}

  	prompt, err := a.RolePrompt(role, planDir)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	fmt.Println(prompt)
  }

  var _ = filepath.Join // silence unused import if role validation changes later
  ```

  Remove the trailing `var _ = filepath.Join` line and the `"path/filepath"` import if you did
  not end up needing them — check with `go vet` in Task 17.3.

---

## Task 13 — `eng capabilities list`

- [x] **13.1** Create `cli/capabilities_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"

  	"eng/internal/capabilities"
  )

  func cmdCapabilities(args []string) {
  	if len(args) == 0 || args[0] != "list" {
  		fmt.Println("Usage: eng capabilities list")
  		os.Exit(1)
  	}
  	for _, name := range capabilities.Known {
  		status := "unavailable"
  		if capabilities.Detect(name) {
  			status = "available"
  		}
  		fmt.Printf("%-10s %s\n", name, status)
  	}
  }
  ```

---

## Task 14 — `eng doctor` gains a Capabilities section

- [x] **14.1** In `cli/doctor.go`, add the import and append a new block at the end of
  `cmdDoctor` (after the existing "Skills resolved" loop):

  Old (end of file):
  ```go
  	resolved, err := skills.Resolve(filepath.Join(hDir, "skills"), filepath.Join(dir, "skills"))
  	if err == nil {
  		fmt.Printf("Skills resolved:   %d\n", len(resolved))
  		for _, s := range resolved {
  			fmt.Printf("  - %-30s [%s] %s\n", s.Name, s.Source, s.Description)
  		}
  	}
  }
  ```

  New:
  ```go
  	resolved, err := skills.Resolve(filepath.Join(hDir, "skills"), filepath.Join(dir, "skills"))
  	if err == nil {
  		fmt.Printf("Skills resolved:   %d\n", len(resolved))
  		for _, s := range resolved {
  			fmt.Printf("  - %-30s [%s] %s\n", s.Name, s.Source, s.Description)
  		}
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
  ```

  And add `"eng/internal/capabilities"` to the import block at the top of the file.

---

## Task 15 — `eng install` copies the binary and gains `--add-to-path`

- [x] **15.1** Replace the full contents of `cli/install.go`:

  ```go
  package main

  import (
  	"flag"
  	"fmt"
  	"io/fs"
  	"os"
  	"os/exec"
  	"path/filepath"
  	"runtime"
  	"strings"
  )

  func harnessDir() string {
  	home, err := os.UserHomeDir()
  	if err != nil {
  		fmt.Println("error: cannot resolve home directory:", err)
  		os.Exit(1)
  	}
  	return filepath.Join(home, ".engineering-harness")
  }

  func binDir() string {
  	return filepath.Join(harnessDir(), "bin")
  }

  func cmdInstall(args []string) {
  	flagset := flag.NewFlagSet("install", flag.ExitOnError)
  	from := flagset.String("from", ".", "path to a checkout containing a harness/ directory")
  	addToPath := flagset.Bool("add-to-path", false, "also add the harness bin/ directory to PATH")
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

  	if err := installBinary(); err != nil {
  		fmt.Println("warning: could not copy eng binary to bin/:", err)
  	} else {
  		fmt.Printf("Copied eng binary to %s\n", binDir())
  	}

  	printPathInstructions()
  	if *addToPath {
  		if err := applyPathSetup(); err != nil {
  			fmt.Println("warning: could not apply PATH setup automatically:", err)
  		}
  	}
  }

  func installBinary() error {
  	self, err := os.Executable()
  	if err != nil {
  		return err
  	}
  	if err := os.MkdirAll(binDir(), 0o755); err != nil {
  		return err
  	}
  	data, err := os.ReadFile(self)
  	if err != nil {
  		return err
  	}
  	name := "eng"
  	if runtime.GOOS == "windows" {
  		name = "eng.exe"
  	}
  	return os.WriteFile(filepath.Join(binDir(), name), data, 0o755)
  }

  func printPathInstructions() {
  	dir := binDir()
  	fmt.Println("\nTo use `eng` from any terminal, add this to your PATH:")
  	if runtime.GOOS == "windows" {
  		fmt.Printf("  setx PATH \"%%PATH%%;%s\"\n", dir)
  		fmt.Println("  (open a new terminal afterward — setx only affects new sessions)")
  	} else {
  		fmt.Printf("  export PATH=\"%s:$PATH\"\n", dir)
  		fmt.Println("  (add that line to ~/.bashrc or ~/.zshrc to make it permanent)")
  	}
  	fmt.Println("Or re-run `eng install --add-to-path` to apply this automatically.")
  }

  func applyPathSetup() error {
  	dir := binDir()
  	if runtime.GOOS == "windows" {
  		current := os.Getenv("PATH")
  		if len(current)+len(dir)+1 > 1024 {
  			fmt.Println("warning: PATH is already near setx's 1024-character limit — add it manually instead")
  			return nil
  		}
  		cmd := exec.Command("setx", "PATH", current+";"+dir)
  		return cmd.Run()
  	}

  	line := fmt.Sprintf("export PATH=\"%s:$PATH\"\n", dir)
  	for _, profile := range []string{".bashrc", ".zshrc"} {
  		home, err := os.UserHomeDir()
  		if err != nil {
  			continue
  		}
  		path := filepath.Join(home, profile)
  		if _, err := os.Stat(path); err != nil {
  			continue
  		}
  		existing, _ := os.ReadFile(path)
  		if strings.Contains(string(existing), dir) {
  			continue // already present
  		}
  		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
  		if err != nil {
  			return err
  		}
  		_, err = f.WriteString("\n# added by `eng install --add-to-path`\n" + line)
  		f.Close()
  		if err != nil {
  			return err
  		}
  	}
  	return nil
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

  Note: `installBinary` reads and rewrites the *currently running* binary's own bytes — this
  works because `os.Executable()` resolves before the write, and OSes allow reading a running
  executable's file freely; only in-place overwriting the running file itself would be a
  problem, which this avoids by writing to a different path (`bin/eng`).

---

## Task 16 — `eng start`

- [x] **16.1** Create `cli/start_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"os/exec"

  	"eng/internal/capabilities"
  )

  func cmdStart(args []string) {
  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	fmt.Println("eng start")
  	fmt.Println()
  	cmdDoctor(nil)
  	_ = dir

  	if capabilities.Detect("claude") {
  		fmt.Println("\nLaunching Claude Code...")
  		c := exec.Command("claude")
  		c.Stdin = os.Stdin
  		c.Stdout = os.Stdout
  		c.Stderr = os.Stderr
  		if err := c.Run(); err != nil {
  			fmt.Println("\nCould not launch `claude` automatically:", err)
  			fmt.Println("Run it yourself in this directory.")
  		}
  		return
  	}

  	fmt.Println("\n`claude` was not found on PATH. Configure an agent in .agent/project.yaml,")
  	fmt.Println("or install one and re-run `eng start`.")
  }
  ```

---

## Task 17 — Wire up dispatch in `main.go`

- [x] **17.1** In `cli/main.go`, update the `switch` in `main()`:

  Old:
  ```go
  	case "plan":
  		cmdPlan(os.Args[2:])
  	case "verify":
  		cmdVerify(os.Args[2:])
  	case "hooks":
  		cmdHooks(os.Args[2:])
  	case "triage":
  		cmdTriage(os.Args[2:])
  	default:
  ```

  New:
  ```go
  	case "plan":
  		cmdPlan(os.Args[2:])
  	case "verify":
  		cmdVerify(os.Args[2:])
  	case "hooks":
  		cmdHooks(os.Args[2:])
  	case "triage":
  		cmdTriage(os.Args[2:])
  	case "workflow":
  		cmdWorkflow(os.Args[2:])
  	case "adapter":
  		cmdAdapter(os.Args[2:])
  	case "capabilities":
  		cmdCapabilities(os.Args[2:])
  	case "start":
  		cmdStart(os.Args[2:])
  	default:
  ```

- [x] **17.2** In `cli/main.go`'s `usage()` function, append to the printed command list
  (after the existing `triage` line):

  ```
    workflow start "<text>"            Triage + create a plan, then report its status
    workflow status [dir]              Report a plan's lifecycle state and next action
    workflow advance [dir]             Mechanically apply the next safe transition
    adapter prompt <role> <dir>        Print the assembled prompt for an agent session
    capabilities list                  Report which known tools are on PATH
    start                               Run doctor, then launch the configured agent`)
  ```

- [x] **17.3** Run `cd cli && go vet ./... && go build ./... 2>&1`. Remove the placeholder
  `var _ = filepath.Join` line from `adapter_cmd.go` (Task 12.1) if `go vet`/`go build` show
  `path/filepath` as unused after your final version of that file — fix any other compile
  errors before proceeding.

---

## Task 18 — Docs integration and VERSION bump (last task)

- [x] **18.1** Update `harness/VERSION`:

  ```
  0.3.0-phase3
  ```

- [x] **18.2** In `docs/gotchas.md`, append a note to the existing `eng hooks run` entry
  (added by Phase 2) rather than deleting it — history stays, but the fix is now documented:

  ```markdown

  **Resolved in Phase 3:** `eng install` now copies the running binary to
  `~/.engineering-harness/bin/` and prints (or, with `--add-to-path`, applies) the correct
  PATH setup for the current platform. See
  `.plans/2026-08-24-v2-harness-phase3/spec.md` Decision 8.
  ```

- [x] **18.3** In `README.md`, immediately after the Phase 2 section added previously (before
  the following `---`), add:

  ```markdown

  Phase 3 adds a lightweight orchestrator on top of Phases 1–2's primitives:

  ```bash
  cd cli && go build -o eng .
  ./eng workflow start "add a recommendations endpoint"   # triage + eng plan new + status
  ./eng workflow status .plans/2026-08-24-add-a-recommendations-endpoint
  ./eng workflow advance .plans/2026-08-24-add-a-recommendations-endpoint
  ./eng plan review .plans/2026-08-24-add-a-recommendations-endpoint --verdict PASS
  ./eng plan approve .plans/2026-08-24-add-a-recommendations-endpoint   # only if required
  ./eng adapter prompt executor .plans/2026-08-24-add-a-recommendations-endpoint
  ./eng capabilities list
  ./eng start
  ```

  See `.plans/2026-08-24-v2-harness-phase3/spec.md` for the full design.
  ```

- [x] **18.4** In `ROADMAP.md`, extend the note:

  Old:
  ```markdown
  > `.plans/2026-08-24-v2-harness-foundation/` (global install foundation) and
  > `.plans/2026-08-24-v2-harness-phase2/` (Triage/Plan Reviewer/Verifier/hooks/drift/retry) —
  > see those plans for the current architecture.
  ```

  New:
  ```markdown
  > `.plans/2026-08-24-v2-harness-foundation/` (global install foundation),
  > `.plans/2026-08-24-v2-harness-phase2/` (Triage/Plan Reviewer/Verifier/hooks/drift/retry),
  > and `.plans/2026-08-24-v2-harness-phase3/` (orchestrator, lifecycle state machine, Claude
  > Code adapter) — see those plans for the current architecture.
  ```

- [x] **18.5** In `docs/src-map.md`, add a final module section after the Phase 2 entries:

  ```markdown

  ### `cli/internal/workflow/`, `cli/internal/agent/`, `cli/internal/capabilities/`, `cli/internal/executil/` — Phase 3 orchestration

  What it does: `workflow` holds the lifecycle state enum and the pure transition table
  (`Decide`); `agent` defines the `Adapter` interface with `ClaudeCodeAdapter` as the only
  implementation; `capabilities` detects which known CLI tools are on PATH; `executil` runs a
  command either as a shell string (compatibility mode) or a structured argv (no shell).

  Key files: `cli/workflow_cmd.go` (`eng workflow start/status/advance`), `cli/adapter_cmd.go`
  (`eng adapter prompt`), `cli/internal/workflow/workflow.go` (the transition table)

  Notable: `eng workflow advance` never writes plan content and never invokes an agent
  unattended — every human/AI-driven stage ends with a printed next command and a stop. The
  approval gate (`requires_approval`/`approved_at` on `plan.yaml`) is enforced here: a plan
  cannot reach `EXECUTING` while it's set and unapproved.

  From: `.plans/2026-08-24-v2-harness-phase3/`
  ```
