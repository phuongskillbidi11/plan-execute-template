# Tasks: V2 Harness Phase 5 (Runtime Integration)

Each task must be completed and its test (see `tests.md`) must pass before moving to the
next. Mark `[x]` when done. Read `spec.md` in full — especially "Design decisions" — before
starting Task 1.

**Prerequisite:** Go 1.22+, and Phase 1–4's completed `cli/`/`harness/` trees (already
committed as of this plan).

---

## Task 1 — Lifecycle state machine: spec-approval and quick-fix fast path

- [x] **1.1** In `cli/internal/workflow/workflow.go`, add two new state constants:

  Old:
  ```go
  	StateNeedsFix      = "NEEDS_FIX"
  	StateCancelled     = "CANCELLED"
  )
  ```

  New:
  ```go
  	StateNeedsFix      = "NEEDS_FIX"
  	StateCancelled     = "CANCELLED"

  	StateNeedsSpecApproval = "NEEDS_SPEC_APPROVAL"
  	StateSpecApproved      = "SPEC_APPROVED"
  )
  ```

- [x] **1.2** Extend the `Facts` struct:

  Old:
  ```go
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
  ```

  New:
  ```go
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

  	IsQuickFix          bool   // risk_level == "quick-fix"
  	PlanningMode        string // "auto_plan" (default/legacy) | "spec_first"
  	SpecReady           bool   // spec.md exists, non-empty, placeholder-free
  	SpecApproved        bool   // plan.yaml's spec_approved_at is set
  	RequireSpecApproval bool
  	TasksAndTestsReady  bool // tasks.md AND tests.md exist, non-empty, placeholder-free
  }
  ```

- [x] **1.3** Replace the `StateTriaged` case in `Decide` and add two new cases immediately
  after it:

  Old:
  ```go
  	switch f.State {
  	case StateTriaged:
  		if f.PlanFilesReady {
  			return Decision{NextState: StatePlanned, Reason: "spec.md/tasks.md/tests.md are present"}
  		}
  		return Decision{NextState: StateTriaged, Reason: "waiting on Planner to write spec.md/tasks.md/tests.md"}

  	case StatePlanned:
  ```

  New:
  ```go
  	switch f.State {
  	case StateTriaged:
  		if f.IsQuickFix {
  			if f.PlanFilesReady {
  				return Decision{NextState: StateExecuting, Reason: "quick-fix: minimal plan present, skipping review/approval"}
  			}
  			return Decision{NextState: StateTriaged, Reason: "waiting on the minimal quick-fix plan (spec.md + tasks.md + tests.md)"}
  		}
  		if f.PlanningMode == "spec_first" {
  			if !f.SpecReady {
  				return Decision{NextState: StateTriaged, Reason: "waiting on Planner to write spec.md"}
  			}
  			if f.RequireSpecApproval {
  				return Decision{NextState: StateNeedsSpecApproval, Reason: "spec.md written — waiting on `eng plan approve-spec`"}
  			}
  			return Decision{NextState: StateSpecApproved, Reason: "spec.md written, spec approval not required"}
  		}
  		// auto_plan (default when PlanningMode is unset) — Phase 3 behavior, unchanged.
  		if f.PlanFilesReady {
  			return Decision{NextState: StatePlanned, Reason: "spec.md/tasks.md/tests.md are present"}
  		}
  		return Decision{NextState: StateTriaged, Reason: "waiting on Planner to write spec.md/tasks.md/tests.md"}

  	case StateNeedsSpecApproval:
  		if f.SpecApproved {
  			return Decision{NextState: StateSpecApproved, Reason: "spec approved"}
  		}
  		return Decision{NextState: StateNeedsSpecApproval, Reason: "still waiting on `eng plan approve-spec`"}

  	case StateSpecApproved:
  		if f.TasksAndTestsReady {
  			return Decision{NextState: StatePlanned, Reason: "tasks.md/tests.md are present"}
  		}
  		return Decision{NextState: StateSpecApproved, Reason: "waiting on Planner to write tasks.md/tests.md"}

  	case StatePlanned:
  ```

- [x] **1.4** Append new tests to `cli/internal/workflow/workflow_test.go` (do not modify any
  existing test — they must keep passing unchanged, proving the `auto_plan` path is untouched):

  ```go
  func TestQuickFixSkipsStraightToExecuting(t *testing.T) {
  	waiting := Decide(Facts{State: StateTriaged, IsQuickFix: true, PlanFilesReady: false})
  	if waiting.NextState != StateTriaged {
  		t.Fatalf("expected to stay TRIAGED until minimal plan exists, got %+v", waiting)
  	}
  	ready := Decide(Facts{State: StateTriaged, IsQuickFix: true, PlanFilesReady: true})
  	if ready.NextState != StateExecuting {
  		t.Fatalf("expected EXECUTING directly, got %+v", ready)
  	}
  }

  func TestSpecFirstRequiresSpecApprovalBeforeTasks(t *testing.T) {
  	waitingSpec := Decide(Facts{State: StateTriaged, PlanningMode: "spec_first", SpecReady: false})
  	if waitingSpec.NextState != StateTriaged {
  		t.Fatalf("expected to stay TRIAGED, got %+v", waitingSpec)
  	}
  	needsApproval := Decide(Facts{State: StateTriaged, PlanningMode: "spec_first", SpecReady: true, RequireSpecApproval: true})
  	if needsApproval.NextState != StateNeedsSpecApproval {
  		t.Fatalf("expected NEEDS_SPEC_APPROVAL, got %+v", needsApproval)
  	}
  	skipApproval := Decide(Facts{State: StateTriaged, PlanningMode: "spec_first", SpecReady: true, RequireSpecApproval: false})
  	if skipApproval.NextState != StateSpecApproved {
  		t.Fatalf("expected SPEC_APPROVED when approval not required, got %+v", skipApproval)
  	}
  }

  func TestNeedsSpecApprovalGate(t *testing.T) {
  	blocked := Decide(Facts{State: StateNeedsSpecApproval, SpecApproved: false})
  	if blocked.NextState != StateNeedsSpecApproval {
  		t.Fatalf("expected to stay blocked, got %+v", blocked)
  	}
  	approved := Decide(Facts{State: StateNeedsSpecApproval, SpecApproved: true})
  	if approved.NextState != StateSpecApproved {
  		t.Fatalf("expected SPEC_APPROVED, got %+v", approved)
  	}
  }

  func TestSpecApprovedWaitsForTasksAndTests(t *testing.T) {
  	waiting := Decide(Facts{State: StateSpecApproved, TasksAndTestsReady: false})
  	if waiting.NextState != StateSpecApproved {
  		t.Fatalf("expected to stay SPEC_APPROVED, got %+v", waiting)
  	}
  	ready := Decide(Facts{State: StateSpecApproved, TasksAndTestsReady: true})
  	if ready.NextState != StatePlanned {
  		t.Fatalf("expected PLANNED, got %+v", ready)
  	}
  }

  func TestAutoPlanPathUnaffectedByNewFields(t *testing.T) {
  	// Zero-value PlanningMode/IsQuickFix must reproduce Phase 3's exact behavior.
  	d := Decide(Facts{State: StateTriaged, PlanFilesReady: true})
  	if d.NextState != StatePlanned {
  		t.Fatalf("expected PLANNED (auto_plan, unchanged), got %+v", d)
  	}
  }
  ```

---

## Task 2 — `project.Workflow` planning-mode fields

- [x] **2.1** In `cli/internal/project/project.go`, extend `Workflow` and add two accessor
  methods:

  Old:
  ```go
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
  ```

  New:
  ```go
  type Workflow struct {
  	Triage     bool `yaml:"triage"`
  	PlanReview bool `yaml:"plan_review"`
  	Verifier   bool `yaml:"verifier"`

  	// PlanningMode: "" (unset) | "auto_plan" | "spec_first". Empty means
  	// "this project predates Phase 5, or never set it" — PlanningModeOrDefault
  	// resolves that to "auto_plan", the exact behavior every plan created
  	// under Phases 1-4 already has. Only a fresh `eng init` writes
  	// "spec_first" explicitly for a brand-new project.
  	PlanningMode string `yaml:"planning_mode,omitempty"`

  	// RequireSpecApproval is a pointer so YAML can distinguish "not set"
  	// (nil, defaults to true) from "explicitly false" — yaml.v3 handles
  	// pointer fields natively, no custom unmarshaling needed here.
  	RequireSpecApproval *bool `yaml:"require_spec_approval,omitempty"`
  }

  // enabled reports whether this Workflow struct was ever explicitly set.
  // An all-false zero value means "the workflow block was absent" — callers
  // treat that as "everything enabled" via EffectiveWorkflow.
  func (w Workflow) enabled() bool {
  	return w.Triage || w.PlanReview || w.Verifier
  }

  // PlanningModeOrDefault returns "auto_plan" when unset — the behavior
  // every project.yaml written before Phase 5 already has.
  func (w Workflow) PlanningModeOrDefault() string {
  	if w.PlanningMode == "" {
  		return "auto_plan"
  	}
  	return w.PlanningMode
  }

  // RequireSpecApprovalOrDefault returns true when unset.
  func (w Workflow) RequireSpecApprovalOrDefault() bool {
  	if w.RequireSpecApproval == nil {
  		return true
  	}
  	return *w.RequireSpecApproval
  }
  ```

- [x] **2.2** Append tests to `cli/internal/project/project_test.go`:

  ```go
  func TestPlanningModeDefaultsToAutoPlan(t *testing.T) {
  	w := Workflow{}
  	if got := w.PlanningModeOrDefault(); got != "auto_plan" {
  		t.Fatalf("expected auto_plan, got %q", got)
  	}
  }

  func TestPlanningModeExplicitSpecFirst(t *testing.T) {
  	w := Workflow{PlanningMode: "spec_first"}
  	if got := w.PlanningModeOrDefault(); got != "spec_first" {
  		t.Fatalf("expected spec_first, got %q", got)
  	}
  }

  func TestRequireSpecApprovalDefaultsTrue(t *testing.T) {
  	w := Workflow{}
  	if !w.RequireSpecApprovalOrDefault() {
  		t.Fatal("expected default true")
  	}
  }

  func TestRequireSpecApprovalExplicitFalse(t *testing.T) {
  	f := false
  	w := Workflow{RequireSpecApproval: &f}
  	if w.RequireSpecApprovalOrDefault() {
  		t.Fatal("expected explicit false to be respected")
  	}
  }

  func TestLegacyProjectYAMLWithoutPlanningModeStillLoads(t *testing.T) {
  	dir := t.TempDir()
  	os.MkdirAll(filepath.Join(dir, ".agent"), 0o755)
  	// Simulates a Phase 1-4 project.yaml: no planning_mode/require_spec_approval keys at all.
  	content := "project_name: x\nmode: modern\nworkflow:\n  triage: true\n  plan_review: true\n  verifier: true\n"
  	os.WriteFile(filepath.Join(dir, ConfigPath), []byte(content), 0o644)

  	cfg, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if cfg.Workflow.PlanningModeOrDefault() != "auto_plan" {
  		t.Fatalf("expected a pre-Phase-5 project.yaml to resolve to auto_plan, got %q", cfg.Workflow.PlanningModeOrDefault())
  	}
  }
  ```

---

## Task 3 — `plan.yaml` spec-approval fields and structured events

- [x] **3.1** In `cli/internal/planmeta/planmeta.go`, add fields to `Meta` and a new function:

  Old:
  ```go
  	RequiresApproval bool         `yaml:"requires_approval"`
  	ApprovedAt       string       `yaml:"approved_at,omitempty"`
  	ApprovedBy       string       `yaml:"approved_by,omitempty"`
  	Review           Review       `yaml:"review,omitempty"`
  	Verification     Verification `yaml:"verification,omitempty"`
  }
  ```

  New:
  ```go
  	RequiresApproval bool         `yaml:"requires_approval"`
  	ApprovedAt       string       `yaml:"approved_at,omitempty"`
  	ApprovedBy       string       `yaml:"approved_by,omitempty"`
  	Review           Review       `yaml:"review,omitempty"`
  	Verification     Verification `yaml:"verification,omitempty"`

  	// SpecApprovedAt/By are entirely separate from ApprovedAt/By above:
  	// this pair means "the requirements in spec.md are approved," never
  	// "this risky execution may proceed" — see Phase 5 spec.md Decision 5.
  	SpecApprovedAt string `yaml:"spec_approved_at,omitempty"`
  	SpecApprovedBy string `yaml:"spec_approved_by,omitempty"`
  }
  ```

  Then append this function after `AppendEvent`:

  ```go
  // AppendStructuredEvent records one line to events.jsonl with an arbitrary
  // flat payload merged with {type, at} — used for compact records like the
  // Quick Fix completion event, which needs more than AppendEvent's single
  // "detail" string.
  func AppendStructuredEvent(planDir, eventType string, data map[string]interface{}) error {
  	f, err := os.OpenFile(filepath.Join(planDir, EventsFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
  	if err != nil {
  		return err
  	}
  	defer f.Close()

  	payload := map[string]interface{}{
  		"type": eventType,
  		"at":   time.Now().UTC().Format(time.RFC3339),
  	}
  	for k, v := range data {
  		payload[k] = v
  	}
  	line, err := json.Marshal(payload)
  	if err != nil {
  		return err
  	}
  	_, err = f.Write(append(line, '\n'))
  	return err
  }
  ```

- [x] **3.2** Append a test to `cli/internal/planmeta/planmeta_test.go`:

  ```go
  func TestAppendStructuredEventWritesFlatJSON(t *testing.T) {
  	dir := t.TempDir()
  	err := AppendStructuredEvent(dir, "quick_fix", map[string]interface{}{
  		"summary":      "Changed timeout",
  		"files":        []string{"src/connection.cpp"},
  		"verification": "PASS",
  	})
  	if err != nil {
  		t.Fatal(err)
  	}

  	data, err := os.ReadFile(filepath.Join(dir, EventsFileName))
  	if err != nil {
  		t.Fatal(err)
  	}
  	var decoded map[string]interface{}
  	if err := json.Unmarshal(bytesTrimNewline(data), &decoded); err != nil {
  		t.Fatalf("line is not valid flat JSON: %v (%s)", err, data)
  	}
  	if decoded["type"] != "quick_fix" || decoded["verification"] != "PASS" {
  		t.Fatalf("unexpected payload: %+v", decoded)
  	}
  }

  func bytesTrimNewline(b []byte) []byte {
  	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
  		b = b[:len(b)-1]
  	}
  	return b
  }
  ```

  Add `"encoding/json"` to the test file's import block if not already present.

---

## Task 4 — `workflow_cmd.go`: compute the new facts

- [x] **4.1** In `cli/workflow_cmd.go`, replace `filesReady` and `gatherFacts`:

  Old:
  ```go
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
  ```

  New:
  ```go
  // gatherFacts reads plan.yaml, tasks.md, and .agent/project.yaml to build the
  // pure Facts workflow.Decide needs — this is the only place doing I/O.
  func gatherFacts(planDir string, meta *planmeta.Meta) (workflow.Facts, error) {
  	repoRoot, err := os.Getwd()
  	if err != nil {
  		return workflow.Facts{}, err
  	}

  	reviewRequired := meta.RiskLevel == "architecture" || meta.RiskLevel == "high-risk"
  	planningMode := "auto_plan"
  	requireSpecApproval := true
  	if cfg, err := project.Load(repoRoot); err == nil {
  		reviewRequired = reviewRequired || cfg.EffectiveWorkflow().PlanReview
  		planningMode = cfg.Workflow.PlanningModeOrDefault()
  		requireSpecApproval = cfg.Workflow.RequireSpecApprovalOrDefault()
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

  		IsQuickFix:          meta.RiskLevel == "quick-fix",
  		PlanningMode:        planningMode,
  		SpecReady:           specReady(planDir),
  		SpecApproved:        meta.SpecApprovedAt != "",
  		RequireSpecApproval: requireSpecApproval,
  		TasksAndTestsReady:  tasksAndTestsReady(planDir),
  	}, nil
  }

  // specReady checks spec.md alone: exists, non-empty, placeholder-free.
  func specReady(planDir string) bool {
  	data, err := os.ReadFile(filepath.Join(planDir, "spec.md"))
  	if err != nil || len(data) == 0 {
  		return false
  	}
  	return !strings.Contains(string(data), "[Feature Name]")
  }

  // tasksAndTestsReady checks tasks.md and tests.md: both templates share
  // spec.md's "[Feature Name]" placeholder marker (see Phase 5 spec.md
  // Decision 12), so the same check applies.
  func tasksAndTestsReady(planDir string) bool {
  	for _, n := range []string{"tasks.md", "tests.md"} {
  		data, err := os.ReadFile(filepath.Join(planDir, n))
  		if err != nil || len(data) == 0 {
  			return false
  		}
  		if strings.Contains(string(data), "[Feature Name]") {
  			return false
  		}
  	}
  	return true
  }

  // filesReady is the auto_plan-mode fact: all three files ready at once.
  // Decomposed from specReady/tasksAndTestsReady — identical semantics to
  // Phase 3/4's original filesReady, zero behavior change.
  func filesReady(planDir string) bool {
  	return specReady(planDir) && tasksAndTestsReady(planDir)
  }
  ```

---

## Task 5 — `eng plan approve-spec`, `eng plan escalate`, and quick-fix templates in `eng plan new`

- [x] **5.1** In `cli/plan_cmd.go`, update `cmdPlan`'s switch and `planNew`'s template
  selection:

  Old:
  ```go
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
  ```

  New:
  ```go
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
  	case "approve-spec":
  		planApproveSpec(args[1:])
  	case "escalate":
  		planEscalate(args[1:])
  	case "block":
  		planBlock(args[1:])
  	case "cancel":
  		planCancel(args[1:])
  	default:
  		fmt.Println("Usage: eng plan <new|drift|retry|review|approve|approve-spec|escalate|block|cancel> ...")
  		os.Exit(1)
  	}
  ```

- [x] **5.2** In `planNew`, select the quick-fix template directory when `--risk quick-fix`:

  Old:
  ```go
  	tmplDir := filepath.Join(harnessDir(), "templates", "plan")
  	if err := copyTree(tmplDir, planDir); err != nil {
  ```

  New:
  ```go
  	tmplDir := filepath.Join(harnessDir(), "templates", "plan")
  	if *risk == "quick-fix" {
  		tmplDir = filepath.Join(harnessDir(), "templates", "quickfix")
  	}
  	if err := copyTree(tmplDir, planDir); err != nil {
  ```

- [x] **5.3** Append two new functions to `cli/plan_cmd.go`:

  ```go
  func planApproveSpec(args []string) {
  	flagset := flag.NewFlagSet("plan approve-spec", flag.ExitOnError)
  	by := flagset.String("by", "", "who is approving the spec")
  	flagset.Parse(reorderFlagsFirst(args, map[string]bool{}))
  	rest := flagset.Args()
  	if len(rest) == 0 {
  		fmt.Println("Usage: eng plan approve-spec <plan-dir> [--by <name>]")
  		os.Exit(1)
  	}
  	planDir, _ := filepath.Abs(rest[0])

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	meta.SpecApprovedAt = time.Now().UTC().Format(time.RFC3339)
  	meta.SpecApprovedBy = *by
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	planmeta.AppendEvent(planDir, "spec_approved", *by)
  	fmt.Printf("Spec approved by %q at %s — this is a requirements approval, not an execution approval.\n", *by, meta.SpecApprovedAt)
  }

  func planEscalate(args []string) {
  	flagset := flag.NewFlagSet("plan escalate", flag.ExitOnError)
  	to := flagset.String("to", "", "bug|feature|architecture|high-risk")
  	reason := flagset.String("reason", "", "why this is being escalated")
  	flagset.Parse(reorderFlagsFirst(args, map[string]bool{}))
  	rest := flagset.Args()
  	if len(rest) == 0 || *to == "" {
  		fmt.Println(`Usage: eng plan escalate <plan-dir> --to <bug|feature|architecture|high-risk> [--reason "..."]`)
  		os.Exit(1)
  	}
  	planDir, _ := filepath.Abs(rest[0])

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}
  	if meta.RiskLevel != "quick-fix" {
  		fmt.Println("error: only a quick-fix plan can be escalated with this command")
  		os.Exit(1)
  	}

  	from := meta.RiskLevel
  	meta.RiskLevel = *to
  	meta.State = workflow.StateTriaged
  	meta.RequiresApproval = *to == "high-risk"
  	if err := planmeta.Save(planDir, meta); err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	planmeta.AppendEvent(planDir, "escalated", fmt.Sprintf("%s -> %s: %s", from, *to, *reason))
  	fmt.Printf("Escalated %s -> %s — state reset to TRIAGED.\n", from, *to)
  	fmt.Println("Flesh out spec.md/tasks.md/tests.md into the full format before continuing — this command only records the fact, it does not regenerate plan content.")
  }
  ```

---

## Task 6 — Minimal quick-fix templates

- [x] **6.1** Create `harness/templates/quickfix/spec.md`:

  ```markdown
  # Quick Fix: [Feature Name]

  ## Goal

  [One sentence: what does this quick fix change, and why?]
  ```

- [x] **6.2** Create `harness/templates/quickfix/tasks.md`:

  ```markdown
  # Tasks: [Feature Name]

  ## Task 1 — Apply the fix

  - [ ] **1.1** [Exact file + change.]
  ```

- [x] **6.3** Create `harness/templates/quickfix/tests.md`:

  ```markdown
  # Tests: [Feature Name]

  ## T1 — Verify the fix

  ```bash
  [exact command]
  ```

  **Pass:** [condition]
  **Fail:** [what to report]
  ```

- [x] **6.4** Create `harness/templates/quickfix/plan.yaml`:

  ```yaml
  plan: YYYY-MM-DD-feature-name
  risk_level: quick-fix
  planned_at:
    git_sha: ""
  state: TRIAGED
  write_scope: []
  retry:
    build: 0
    unit_test: 0
    integration_test: 0
  retry_budget:
    build: 2
    unit_test: 2
    integration_test: 1
  requires_approval: false
  ```

  (`eng plan new` overwrites this file's stamped fields immediately after copying — its
  content here only needs to be valid YAML for `copyTree`.)

---

## Task 7 — `eng init` sets `planning_mode: spec_first` for new projects only

- [x] **7.1** In `cli/init_cmd.go`, add the import and update the `Config` construction:

  Old:
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

  New:
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
  		// planning_mode is written explicitly only here, at eng init time for
  		// a brand-new project — an existing .agent/project.yaml is never
  		// touched (see the early-return above), so no project that has
  		// already run eng init under any prior phase is affected by this.
  		Workflow: project.Workflow{PlanningMode: "spec_first"},
  	}
  ```

---

## Task 8 — `context_cmd.go`: `buildContextBundle`, fallback-to-full, `eng context manifest`

- [x] **8.1** Replace the full contents of `cli/context_cmd.go`:

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
  	"eng/internal/skillmatch"
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

  // selectSkills is the pure core behind `eng context skills`, reused by
  // buildContextBundle so the manifest can record exactly what was chosen.
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

  func writeSkillSelection(w io.Writer, selected []skills.Skill, total int, cfg contextcfg.Config) {
  	fmt.Fprintf(w, "Selected %d/%d skills (strategy: %s, max_skills: %d)\n\n", len(selected), total, cfg.Strategy, cfg.MaxSkills)
  	for _, s := range selected {
  		fmt.Fprintf(w, "- %-30s [%s] %s\n", s.Name, s.Domain, s.Description)
  	}
  	if len(selected) < total {
  		fmt.Fprintf(w, "\n%d skill(s) omitted as not relevant to this request.\n", total-len(selected))
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
  	writeSkillSelection(os.Stdout, selected, total, cfg)
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
  		selected, total, _ := selectSkills(repoRoot, request, cfg)
  		if allSectionsEmpty(byFile) && len(selected) == 0 && cfg.Strategy != "full" {
  			fbCfg := cfg
  			fbCfg.Strategy = "full"
  			byFile = selectProjectContext(repoRoot, request, fbCfg)
  			selected, total, _ = selectSkills(repoRoot, request, fbCfg)
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
  		writeSkillSelection(&out, selected, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for _, s := range selected {
  			fmt.Fprintf(&manifest, "  - %s\n", s.Name)
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
  		selected, total, _ := selectSkills(repoRoot, request, cfg)
  		fmt.Fprintf(&out, "\n## Skills\n")
  		writeSkillSelection(&out, selected, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for _, s := range selected {
  			fmt.Fprintf(&manifest, "  - %s\n", s.Name)
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

---

## Task 9 — `eng adapter prompt` folds in the context bundle

- [x] **9.1** Replace the full contents of `cli/adapter_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"strings"

  	"eng/internal/agent"
  )

  func cmdAdapter(args []string) {
  	if len(args) < 2 || args[0] != "prompt" {
  		fmt.Println(`Usage: eng adapter prompt <planner|plan-reviewer|executor|verifier> <plan-dir> ["<request text>"]`)
  		os.Exit(1)
  	}
  	role := agent.Role(args[1])
  	if len(args) < 3 {
  		fmt.Println("Usage: eng adapter prompt <role> <plan-dir>")
  		os.Exit(1)
  	}
  	planDir := args[2]
  	request := ""
  	if len(args) > 3 {
  		request = strings.Join(args[3:], " ")
  	}

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

  	// Phase 5: the context bundle is now always folded in here — the
  	// Context Manager's output is the authoritative prepared context for
  	// this role, not a separate manual step (Phase 5 spec.md Decision 2).
  	bundle, err := buildContextBundle(string(role), planDir, request)
  	if err != nil {
  		fmt.Println("(no project-specific context bundle available:", err, ")")
  		return
  	}
  	fmt.Println(bundle)
  }
  ```

---

## Task 10 — Quick-fix structured event and log pruning in `eng verify`

- [x] **10.1** In `cli/verify_cmd.go`, add imports and call `logprune.Prune` after
  `writeFullLog`:

  Old:
  ```go
  import (
  	"eng/internal/executil"
  	"eng/internal/gitutil"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  )
  ```

  New:
  ```go
  import (
  	"eng/internal/executil"
  	"eng/internal/gitutil"
  	"eng/internal/logprune"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/taskscope"
  )
  ```

  Old:
  ```go
  		ctxCfg := loadContextConfig(repoRoot)
  		logPath, logErr := writeFullLog(repoRoot, "verify", out)
  		display := out
  		if ctxCfg.SummarizeToolOutput {
  			display = summarizeOutput(out, ctxCfg.MaxLogLines)
  		}
  ```

  New:
  ```go
  		ctxCfg := loadContextConfig(repoRoot)
  		logPath, logErr := writeFullLog(repoRoot, "verify", out)
  		if logErr == nil {
  			logprune.Prune(filepath.Join(repoRoot, ".agent", "logs"), ctxCfg.MaxLogFiles, ctxCfg.MaxLogAgeDays, ctxCfg.MaxLogTotalMB, false)
  		}
  		display := out
  		if ctxCfg.SummarizeToolOutput {
  			display = summarizeOutput(out, ctxCfg.MaxLogLines)
  		}
  ```

- [x] **10.2** In `runVerify`, immediately before the final `return pass, report.String(), nil`
  line, add the quick-fix structured event:

  Old:
  ```go
  	if err := os.WriteFile(filepath.Join(planDir, "verify-report.md"), []byte(report.String()), 0o644); err != nil {
  		return pass, report.String(), err
  	}

  	return pass, report.String(), nil
  }
  ```

  New:
  ```go
  	if err := os.WriteFile(filepath.Join(planDir, "verify-report.md"), []byte(report.String()), 0o644); err != nil {
  		return pass, report.String(), err
  	}

  	if meta.RiskLevel == "quick-fix" && pass {
  		summary, _ := taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
  		planmeta.AppendStructuredEvent(planDir, "quick_fix", map[string]interface{}{
  			"summary":      summary,
  			"files":        changed,
  			"verification": verdict,
  		})
  	}

  	return pass, report.String(), nil
  }
  ```

  Note: `changed` and `verdict` are already in scope at this point in `runVerify` from the
  git-diff and verdict-computation steps earlier in the function.

---

## Task 11 — Log retention (`internal/logprune`)

- [x] **11.1** Create `cli/internal/logprune/logprune.go`:

  ```go
  package logprune

  import (
  	"os"
  	"path/filepath"
  	"sort"
  	"time"
  )

  type Result struct {
  	Deleted        []string
  	KeptMostRecent string
  }

  // Prune deletes *.log files in dir beyond maxFiles/maxAgeDays/maxTotalMB,
  // oldest first, but never deletes the single most recently modified file —
  // a cheap approximation of "don't delete logs an active plan might still
  // need" (see Phase 5 spec.md Decision 8). Any limit <= 0 is treated as
  // "no limit" for that dimension.
  func Prune(dir string, maxFiles, maxAgeDays, maxTotalMB int, dryRun bool) (Result, error) {
  	entries, err := os.ReadDir(dir)
  	if err != nil {
  		if os.IsNotExist(err) {
  			return Result{}, nil
  		}
  		return Result{}, err
  	}

  	type fileInfo struct {
  		path string
  		mod  time.Time
  		size int64
  	}
  	var files []fileInfo
  	for _, e := range entries {
  		if e.IsDir() || filepath.Ext(e.Name()) != ".log" {
  			continue
  		}
  		info, err := e.Info()
  		if err != nil {
  			continue
  		}
  		files = append(files, fileInfo{filepath.Join(dir, e.Name()), info.ModTime(), info.Size()})
  	}
  	if len(files) == 0 {
  		return Result{}, nil
  	}

  	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
  	mostRecent := files[len(files)-1].path

  	var totalBytes int64
  	for _, f := range files {
  		totalBytes += f.size
  	}
  	maxBytes := int64(maxTotalMB) * 1024 * 1024
  	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
  	remaining := len(files)

  	var result Result
  	result.KeptMostRecent = mostRecent
  	for _, f := range files {
  		if f.path == mostRecent {
  			continue
  		}
  		tooMany := maxFiles > 0 && remaining > maxFiles
  		tooOld := maxAgeDays > 0 && f.mod.Before(cutoff)
  		tooBig := maxTotalMB > 0 && totalBytes > maxBytes
  		if tooMany || tooOld || tooBig {
  			if !dryRun {
  				os.Remove(f.path)
  			}
  			result.Deleted = append(result.Deleted, f.path)
  			totalBytes -= f.size
  			remaining--
  		}
  	}
  	return result, nil
  }
  ```

- [x] **11.2** Create `cli/internal/logprune/logprune_test.go`:

  ```go
  package logprune

  import (
  	"os"
  	"path/filepath"
  	"testing"
  	"time"
  )

  func writeLog(t *testing.T, dir, name string, age time.Duration, size int) {
  	t.Helper()
  	path := filepath.Join(dir, name)
  	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
  		t.Fatal(err)
  	}
  	mod := time.Now().Add(-age)
  	if err := os.Chtimes(path, mod, mod); err != nil {
  		t.Fatal(err)
  	}
  }

  func TestPruneRespectsMaxFilesButKeepsMostRecent(t *testing.T) {
  	dir := t.TempDir()
  	writeLog(t, dir, "a.log", 3*time.Hour, 10)
  	writeLog(t, dir, "b.log", 2*time.Hour, 10)
  	writeLog(t, dir, "c.log", 1*time.Hour, 10)

  	result, err := Prune(dir, 1, 0, 0, false)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(result.Deleted) != 2 {
  		t.Fatalf("expected 2 deleted, got %d: %v", len(result.Deleted), result.Deleted)
  	}
  	if _, err := os.Stat(filepath.Join(dir, "c.log")); err != nil {
  		t.Fatal("the most recent file must never be deleted")
  	}
  }

  func TestPruneDryRunDeletesNothing(t *testing.T) {
  	dir := t.TempDir()
  	writeLog(t, dir, "a.log", 3*time.Hour, 10)
  	writeLog(t, dir, "b.log", 1*time.Hour, 10)

  	result, err := Prune(dir, 1, 0, 0, true)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(result.Deleted) != 1 {
  		t.Fatalf("expected 1 reported-would-delete, got %d", len(result.Deleted))
  	}
  	if _, err := os.Stat(filepath.Join(dir, "a.log")); err != nil {
  		t.Fatal("dry-run must not actually delete anything")
  	}
  }

  func TestPruneMissingDirIsNotAnError(t *testing.T) {
  	result, err := Prune(filepath.Join(t.TempDir(), "nope"), 10, 30, 250, false)
  	if err != nil || len(result.Deleted) != 0 {
  		t.Fatalf("expected no error and no deletions, got %+v, %v", result, err)
  	}
  }

  func TestPruneAgeLimit(t *testing.T) {
  	dir := t.TempDir()
  	writeLog(t, dir, "old.log", 40*24*time.Hour, 10)
  	writeLog(t, dir, "new.log", time.Hour, 10)

  	result, err := Prune(dir, 0, 30, 0, false)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(result.Deleted) != 1 || result.Deleted[0] != filepath.Join(dir, "old.log") {
  		t.Fatalf("expected only old.log deleted, got %+v", result.Deleted)
  	}
  }
  ```

---

## Task 12 — `eng logs prune`

- [x] **12.1** Create `cli/logs_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/logprune"
  )

  func cmdLogs(args []string) {
  	if len(args) == 0 || args[0] != "prune" {
  		fmt.Println("Usage: eng logs prune [--dry-run]")
  		os.Exit(1)
  	}
  	dryRun := len(args) > 1 && args[1] == "--dry-run"

  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	cfg := loadContextConfig(dir)

  	result, err := logprune.Prune(filepath.Join(dir, ".agent", "logs"), cfg.MaxLogFiles, cfg.MaxLogAgeDays, cfg.MaxLogTotalMB, dryRun)
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	verb := "Deleted"
  	if dryRun {
  		verb = "Would delete"
  	}
  	fmt.Printf("%s %d log file(s); kept most recent: %s\n", verb, len(result.Deleted), result.KeptMostRecent)
  	for _, f := range result.Deleted {
  		fmt.Println("  -", f)
  	}
  }
  ```

---

## Task 13 — Log-retention budget fields on `contextcfg.Config`

- [x] **13.1** In `cli/internal/contextcfg/contextcfg.go`, extend `Config`, `override`, and
  `Default`:

  Old:
  ```go
  type Config struct {
  	Strategy              string // full | selective
  	MaxSkills             int
  	MaxDocs               int
  	MaxLogLines           int
  	IncludeCompletedTasks bool
  	SummarizeToolOutput   bool
  }
  ```

  New:
  ```go
  type Config struct {
  	Strategy              string // full | selective
  	MaxSkills             int
  	MaxDocs               int
  	MaxLogLines           int
  	IncludeCompletedTasks bool
  	SummarizeToolOutput   bool

  	MaxLogFiles   int // .agent/logs/ retention — see internal/logprune
  	MaxLogAgeDays int
  	MaxLogTotalMB int
  }
  ```

  Old:
  ```go
  type override struct {
  	Strategy              *string `yaml:"strategy"`
  	MaxSkills             *int    `yaml:"max_skills"`
  	MaxDocs               *int    `yaml:"max_docs"`
  	MaxLogLines           *int    `yaml:"max_log_lines"`
  	IncludeCompletedTasks *bool   `yaml:"include_completed_tasks"`
  	SummarizeToolOutput   *bool   `yaml:"summarize_tool_output"`
  }
  ```

  New:
  ```go
  type override struct {
  	Strategy              *string `yaml:"strategy"`
  	MaxSkills             *int    `yaml:"max_skills"`
  	MaxDocs               *int    `yaml:"max_docs"`
  	MaxLogLines           *int    `yaml:"max_log_lines"`
  	IncludeCompletedTasks *bool   `yaml:"include_completed_tasks"`
  	SummarizeToolOutput   *bool   `yaml:"summarize_tool_output"`

  	MaxLogFiles   *int `yaml:"max_log_files"`
  	MaxLogAgeDays *int `yaml:"max_log_age_days"`
  	MaxLogTotalMB *int `yaml:"max_log_total_mb"`
  }
  ```

  Old:
  ```go
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
  ```

  New:
  ```go
  func Default() Config {
  	return Config{
  		Strategy:              "selective",
  		MaxSkills:             5,
  		MaxDocs:               8,
  		MaxLogLines:           300,
  		IncludeCompletedTasks: false,
  		SummarizeToolOutput:   true,
  		MaxLogFiles:           100,
  		MaxLogAgeDays:         30,
  		MaxLogTotalMB:         250,
  	}
  }
  ```

  Finally, in `Load`, add three more `if o.X != nil` blocks alongside the existing ones:

  Old:
  ```go
  	if o.SummarizeToolOutput != nil {
  		cfg.SummarizeToolOutput = *o.SummarizeToolOutput
  	}
  	return cfg, nil
  }
  ```

  New:
  ```go
  	if o.SummarizeToolOutput != nil {
  		cfg.SummarizeToolOutput = *o.SummarizeToolOutput
  	}
  	if o.MaxLogFiles != nil {
  		cfg.MaxLogFiles = *o.MaxLogFiles
  	}
  	if o.MaxLogAgeDays != nil {
  		cfg.MaxLogAgeDays = *o.MaxLogAgeDays
  	}
  	if o.MaxLogTotalMB != nil {
  		cfg.MaxLogTotalMB = *o.MaxLogTotalMB
  	}
  	return cfg, nil
  }
  ```

- [x] **13.2** Append a test to `cli/internal/contextcfg/contextcfg_test.go`:

  ```go
  func TestLogRetentionDefaults(t *testing.T) {
  	cfg := Default()
  	if cfg.MaxLogFiles != 100 || cfg.MaxLogAgeDays != 30 || cfg.MaxLogTotalMB != 250 {
  		t.Fatalf("unexpected log retention defaults: %+v", cfg)
  	}
  }
  ```

---

## Task 14 — Capability Registry: `Describe`/`DescribeAll`

- [x] **14.1** In `cli/internal/capabilities/capabilities.go`, add the import and new code:

  Old:
  ```go
  package capabilities

  import "os/exec"
  ```

  New:
  ```go
  package capabilities

  import (
  	"os/exec"
  	"strings"
  )
  ```

  Append after `DetectAll`:

  ```go
  // Capability is the richer, additive schema Phase 5 needs — Detect/DetectAll
  // above are unchanged and untouched by any existing caller.
  type Capability struct {
  	Name      string
  	Available bool
  	Provider  string
  	Version   string // best-effort; "" if unknown or unavailable
  }

  func Describe(name string) Capability {
  	c := Capability{Name: name, Available: Detect(name), Provider: "local-binary"}
  	if c.Available {
  		c.Version = detectVersion(name)
  	}
  	return c
  }

  func DescribeAll() []Capability {
  	out := make([]Capability, 0, len(Known))
  	for _, name := range Known {
  		out = append(out, Describe(name))
  	}
  	return out
  }

  // detectVersion is best-effort and only implemented for tools with a
  // well-known "--version" flag — not every CLI uses one uniformly, and
  // per-tool version-string parsing beyond this is a later improvement.
  func detectVersion(name string) string {
  	switch name {
  	case "git", "docker":
  		out, err := exec.Command(name, "--version").Output()
  		if err != nil {
  			return ""
  		}
  		lines := strings.SplitN(string(out), "\n", 2)
  		return strings.TrimSpace(lines[0])
  	default:
  		return ""
  	}
  }
  ```

- [x] **14.2** Append tests to `cli/internal/capabilities/capabilities_test.go`:

  ```go
  func TestDescribeAllCoversKnownSet(t *testing.T) {
  	all := DescribeAll()
  	if len(all) != len(Known) {
  		t.Fatalf("expected %d entries, got %d", len(Known), len(all))
  	}
  }

  func TestDescribeUnavailableHasNoVersion(t *testing.T) {
  	c := Describe("definitely-not-a-real-binary-xyz")
  	if c.Available || c.Version != "" {
  		t.Fatalf("expected unavailable with no version, got %+v", c)
  	}
  }

  func TestDescribeGitHasProviderSet(t *testing.T) {
  	c := Describe("git")
  	if c.Provider != "local-binary" {
  		t.Fatalf("expected provider local-binary, got %q", c.Provider)
  	}
  }
  ```

---

## Task 15 — `eng capabilities list --verbose --role`

- [x] **15.1** Replace the full contents of `cli/capabilities_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"

  	"eng/internal/agent"
  	"eng/internal/capabilities"
  )

  func cmdCapabilities(args []string) {
  	if len(args) == 0 || args[0] != "list" {
  		fmt.Println("Usage: eng capabilities list [--verbose] [--role <role>]")
  		os.Exit(1)
  	}

  	verbose := false
  	role := ""
  	for i := 1; i < len(args); i++ {
  		switch args[i] {
  		case "--verbose":
  			verbose = true
  		case "--role":
  			if i+1 < len(args) {
  				role = args[i+1]
  			}
  		}
  	}

  	for _, c := range capabilities.DescribeAll() {
  		if role != "" && !agent.RoleMayUse(role, c.Name) {
  			continue
  		}
  		status := "unavailable"
  		if c.Available {
  			status = "available"
  		}
  		if verbose {
  			fmt.Printf("%-10s %-12s provider=%-14s version=%s\n", c.Name, status, c.Provider, c.Version)
  		} else {
  			fmt.Printf("%-10s %s\n", c.Name, status)
  		}
  	}
  }
  ```

---

## Task 16 — Role-based tool permissions (`internal/agent`)

- [x] **16.1** Create `cli/internal/agent/permissions.go`:

  ```go
  package agent

  // RolePermissions is a static, reporting-only table — nothing yet enforces
  // this against a real tool invocation (see Phase 5 spec.md Decision 11).
  var RolePermissions = map[Role][]string{
  	RolePlanner:  {"git"},
  	RoleReviewer: {"git"},
  	RoleExecutor: {"git", "claude", "codex", "docker"},
  	RoleVerifier: {"git", "docker"},
  }

  // RoleMayUse reports whether role is permitted to consider capability.
  func RoleMayUse(role, capability string) bool {
  	for _, c := range RolePermissions[Role(role)] {
  		if c == capability {
  			return true
  		}
  	}
  	return false
  }
  ```

- [x] **16.2** Create `cli/internal/agent/permissions_test.go`:

  ```go
  package agent

  import "testing"

  func TestPlannerMayUseGitOnly(t *testing.T) {
  	if !RoleMayUse("planner", "git") {
  		t.Fatal("expected planner to be permitted git")
  	}
  	if RoleMayUse("planner", "docker") {
  		t.Fatal("expected planner NOT to be permitted docker")
  	}
  }

  func TestExecutorMayUseDocker(t *testing.T) {
  	if !RoleMayUse("executor", "docker") {
  		t.Fatal("expected executor to be permitted docker")
  	}
  }

  func TestUnknownRoleMayUseNothing(t *testing.T) {
  	if RoleMayUse("not-a-real-role", "git") {
  		t.Fatal("expected an unknown role to be permitted nothing")
  	}
  }
  ```

---

## Task 17 — Tool Adapter interface foundation (`internal/tooladapter`)

- [x] **17.1** Create `cli/internal/tooladapter/tooladapter.go`:

  ```go
  package tooladapter

  import "fmt"

  // Adapter exposes one external tool/capability to the harness. Distinct
  // from internal/agent.Adapter (which launches/talks to a coding agent) —
  // see Phase 5 spec.md Decision 10 for why these stay separate.
  type Adapter interface {
  	Name() string
  	Capability() string // matches a capabilities.Known entry
  	Available() bool
  	PermissionLevel() string // "read" | "read-write" | "execute" | "high-risk"
  	Doctor() (string, error)
  }

  // GitAdapter is the only reference implementation in Phase 5 — it exists
  // to prove the interface compiles and is testable, not as a real
  // capability gate (git access is already unconditional throughout this
  // harness).
  type GitAdapter struct {
  	available bool
  }

  func NewGitAdapter(available bool) GitAdapter { return GitAdapter{available: available} }

  func (g GitAdapter) Name() string            { return "git" }
  func (g GitAdapter) Capability() string      { return "git" }
  func (g GitAdapter) Available() bool         { return g.available }
  func (g GitAdapter) PermissionLevel() string { return "read-write" }

  func (g GitAdapter) Doctor() (string, error) {
  	if g.available {
  		return "git is on PATH", nil
  	}
  	return "", fmt.Errorf("git not found on PATH")
  }
  ```

- [x] **17.2** Create `cli/internal/tooladapter/tooladapter_test.go`:

  ```go
  package tooladapter

  import "testing"

  func TestGitAdapterImplementsAdapter(t *testing.T) {
  	var _ Adapter = GitAdapter{}
  }

  func TestGitAdapterAvailable(t *testing.T) {
  	g := NewGitAdapter(true)
  	if !g.Available() || g.PermissionLevel() != "read-write" {
  		t.Fatalf("unexpected adapter state: %+v", g)
  	}
  	if _, err := g.Doctor(); err != nil {
  		t.Fatalf("expected no error when available, got %v", err)
  	}
  }

  func TestGitAdapterUnavailable(t *testing.T) {
  	g := NewGitAdapter(false)
  	if _, err := g.Doctor(); err == nil {
  		t.Fatal("expected an error when unavailable")
  	}
  }
  ```

---

## Task 18 — Tool Router foundation (`internal/toolrouter`)

- [x] **18.1** Create `cli/internal/toolrouter/toolrouter.go`:

  ```go
  package toolrouter

  import "eng/internal/tooladapter"

  // Filter returns the subset of adapters matching a required capability
  // list and currently available. This is the entire Tool Router for
  // Phase 5 — it exposes nothing to any agent session because no session
  // object exists in this architecture to expose into yet (Requirement 16:
  // foundation only).
  func Filter(required []string, adapters []tooladapter.Adapter) []tooladapter.Adapter {
  	want := map[string]bool{}
  	for _, r := range required {
  		want[r] = true
  	}
  	var out []tooladapter.Adapter
  	for _, a := range adapters {
  		if want[a.Capability()] && a.Available() {
  			out = append(out, a)
  		}
  	}
  	return out
  }
  ```

- [x] **18.2** Create `cli/internal/toolrouter/toolrouter_test.go`:

  ```go
  package toolrouter

  import (
  	"testing"

  	"eng/internal/tooladapter"
  )

  func TestFilterMatchesRequiredAndAvailable(t *testing.T) {
  	adapters := []tooladapter.Adapter{
  		tooladapter.NewGitAdapter(true),
  	}
  	got := Filter([]string{"git"}, adapters)
  	if len(got) != 1 {
  		t.Fatalf("expected 1 match, got %d", len(got))
  	}
  }

  func TestFilterExcludesUnavailable(t *testing.T) {
  	adapters := []tooladapter.Adapter{
  		tooladapter.NewGitAdapter(false),
  	}
  	got := Filter([]string{"git"}, adapters)
  	if len(got) != 0 {
  		t.Fatalf("expected 0 matches for an unavailable adapter, got %d", len(got))
  	}
  }

  func TestFilterExcludesUnrequested(t *testing.T) {
  	adapters := []tooladapter.Adapter{
  		tooladapter.NewGitAdapter(true),
  	}
  	got := Filter([]string{"docker"}, adapters)
  	if len(got) != 0 {
  		t.Fatalf("expected 0 matches for an unrequested capability, got %d", len(got))
  	}
  }
  ```

---

## Task 19 — Adapter directory reorganization

- [x] **19.1** Create `harness/adapters/agents/claude-code/ADAPTER.md` with the exact content
  currently at `harness/adapters/claude-code/ADAPTER.md`, then delete the old file and its now-
  empty `harness/adapters/claude-code/` directory. Nothing in any Go source reads the old path
  — `internal/agent.ClaudeCodeAdapter` only ever reads `core/<role>/METHOD.md` — so this move
  has zero code impact.

- [x] **19.2** Create `harness/adapters/tools/README.md`:

  ```markdown
  # Tool/MCP Adapters (placeholder)

  This directory is the intended home for future external-tool adapters — the
  `internal/tooladapter.Adapter` implementations beyond the one reference implementation
  (`GitAdapter`) that ships with Phase 5.

  Nothing here is implemented yet. Per Phase 5's explicit scope constraint, no docker/ssh/
  github/database adapter, and certainly no live PLC/Modbus/OPC UA/industrial-control adapter,
  is built in this phase — this is architectural foundation only.

  See `cli/internal/tooladapter/tooladapter.go` for the interface and
  `.plans/2026-08-24-v2-harness-phase5-runtime/spec.md` Decision 10 for why Tool Adapters are
  kept structurally separate from Agent Adapters (`harness/adapters/agents/`).
  ```

---

## Task 20 — `eng start`: safe first-run handling and a runtime-method banner

- [x] **20.1** Replace the full contents of `cli/start_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"os/exec"
  	"path/filepath"

  	"eng/internal/capabilities"
  	"eng/internal/project"
  )

  func cmdStart(args []string) {
  	doInit := false
  	for _, a := range args {
  		if a == "--init" {
  			doInit = true
  		}
  	}

  	dir, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	if project.DetectModeResult(dir).Mode == "none" {
  		if doInit {
  			cmdInit(nil)
  		} else {
  			fmt.Println("This project isn't initialized for the harness yet.")
  			fmt.Println("Run `eng init` first, or `eng start --init` to initialize now and continue.")
  			fmt.Println("(eng init only ever creates .agent/project.yaml — nothing else is touched.)")
  			return
  		}
  	}

  	fmt.Println("eng start")
  	fmt.Println()
  	cmdDoctor(nil)

  	fmt.Println("\nFor natural-language requests, this session should consult:")
  	fmt.Printf("  %s\n", filepath.Join(harnessDir(), "core", "runtime", "METHOD.md"))

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

## Task 21 — Triage: a documented-gotcha match never lowers, only holds-or-raises, the suggested level

- [x] **21.1** In `cli/triage_cmd.go`, add the import and extend `triageLevel`:

  Old:
  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"strings"
  )
  ```

  New:
  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"

  	"eng/internal/docsearch"
  )
  ```

  Old:
  ```go
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
  ```

  New:
  ```go
  // levelRank orders risk levels so a gotcha match can "hold or raise" but
  // never lower the keyword-based suggestion (Requirement 21: a heuristic
  // must stay observable and must never claim more authority than it has —
  // this only ever nudges upward, it does not reclassify).
  var levelRank = map[string]int{
  	"quick-fix":    0,
  	"bug":          1,
  	"feature":      2,
  	"architecture": 3,
  	"high-risk":    4,
  }

  // triageLevel is the pure heuristic, factored out so `eng workflow start`
  // can call it directly instead of going through cmdTriage's print/exit path.
  func triageLevel(text string) (level, workflowDesc string) {
  	level, workflowDesc = keywordLevel(text)

  	dir, err := os.Getwd()
  	if err != nil {
  		return level, workflowDesc
  	}
  	sections, err := docsearch.ParseSections(filepath.Join(dir, "docs", "gotchas.md"))
  	if err != nil {
  		return level, workflowDesc
  	}
  	if matched := docsearch.Match(sections, text, 1); len(matched) > 0 {
  		if levelRank["architecture"] > levelRank[level] {
  			return "architecture", "matched a documented gotcha (" + matched[0].Title + ") — research + ADR + full spec/tasks/tests, elevated from the keyword-only suggestion"
  		}
  	}
  	return level, workflowDesc
  }

  func keywordLevel(text string) (level, workflowDesc string) {
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
  ```

  (This is deliberately conservative: a gotcha match raises at most to `architecture`, never
  to `high-risk` — high-risk stays a purely explicit, keyword-or-Planner-declared classification,
  never inferred from a gotcha entry.)

---

## Task 22 — The Runtime Router methodology

- [x] **22.1** Create `harness/core/runtime/METHOD.md`:

  ```markdown
  # Core Method: Runtime Router

  The missing piece between "the harness's tools exist" and "a user just describes what they
  want." Not an AI role with its own responsibilities — a documented command sequence any
  Claude Code session follows automatically when a human describes an engineering requirement
  in plain language, instead of typing low-level `eng` commands themselves.

  ## When this applies

  The human describes a requirement in natural language (not a raw `eng` command). Low-level
  commands (`eng plan new`, `eng workflow advance`, `eng context bundle`, ...) remain available
  for debugging, CI, and advanced manual use — they are never removed, only no longer the
  expected default typing surface for a normal request.

  ## The sequence

  1. Run `eng doctor` once per session to confirm harness/project state.
  2. Run `eng workflow start "<the exact requirement text>"`. This triages the request (see
     `core/triage/METHOD.md`), scaffolds a plan via `eng plan new`, and reports the initial
     state and risk level.
  3. Read the reported risk level.
     - `quick-fix` → follow **Quick Fix path**, below.
     - anything else → follow **Spec-First path**, below (the default `planning_mode` for any
       project initialized under Phase 5 onward; a project whose `.agent/project.yaml` predates
       Phase 5, or was never given `planning_mode: spec_first`, instead follows the single-step
       `TRIAGED → PLANNED` path documented in `core/planner/METHOD.md` — check
       `eng workflow status <plan-dir>`'s reported `Profile:` line if unsure).
  4. At every state, use `eng workflow advance <plan-dir>` for the mechanical transition — never
     decide a transition by judgment. Before invoking a role, run
     `eng adapter prompt <role> <plan-dir> "<request text>"`, which now folds in that role's
     `eng context bundle` output automatically.
  5. Never skip a printed gate (`NEEDS_SPEC_APPROVAL`, `NEEDS_APPROVAL`) — stop and ask the
     human explicitly, in the conversation, before proceeding past one.

  ## Quick Fix path

  1. If, while working, the change turns out to be broader than it looked (touches more than
     the one localized area, needs a schema/API change, needs review), **do not continue as a
     quick fix** — run `eng plan escalate <plan-dir> --to bug|feature|architecture --reason
     "..."`, then flesh out `spec.md`/`tasks.md`/`tests.md` into the full format and resume via
     the Spec-First or auto_plan path from `TRIAGED`.
  2. Otherwise, edit `tasks.md`'s one task block, make the localized change, and run
     `eng verify <plan-dir>`. On `PASS`, a compact `quick_fix` event is recorded automatically —
     no further documentation is needed for a genuine quick fix.

  ## Spec-First path (the Phase 5 default for new projects)

  1. After `eng workflow start`, the state is `TRIAGED`. Write **only** `spec.md` — do not
     write `tasks.md`/`tests.md` yet.
  2. Run `eng workflow advance <plan-dir>` — moves to `NEEDS_SPEC_APPROVAL`.
  3. Stop. Show the human `spec.md`'s Goal in the conversation and ask for explicit approval or
     revision — the same way this repository's Planner has always asked for spec confirmation.
  4. Once approved, run `eng plan approve-spec <plan-dir>` — this is a **requirements**
     approval, distinct from the execution-risk approval gate later in the lifecycle.
  5. Run `eng workflow advance <plan-dir>` — moves to `SPEC_APPROVED`.
  6. Now write `tasks.md` and `tests.md`.
  7. Run `eng workflow advance <plan-dir>` — moves to `PLANNED`, and the rest of the lifecycle
     continues exactly as documented in `core/plan-reviewer/METHOD.md`,
     `core/executor/METHOD.md`, and `core/verifier/METHOD.md`.

  ## Constraint

  This document never authorizes skipping a state, inventing a transition, or treating a
  heuristic (Triage's keyword match) as authoritative. Every state change still flows through
  `eng workflow advance`'s deterministic `Facts → Decide → Decision` — see
  `core/context-manager/METHOD.md`'s fail-safe rule for the same principle applied to context
  selection.
  ```

---

## Task 23 — Wire up dispatch in `main.go`

- [x] **23.1** In `cli/main.go`, update the `switch` in `main()`:

  Old:
  ```go
  	case "context":
  		cmdContext(os.Args[2:])
  	default:
  ```

  New:
  ```go
  	case "context":
  		cmdContext(os.Args[2:])
  	case "logs":
  		cmdLogs(os.Args[2:])
  	default:
  ```

- [x] **23.2** In `cli/main.go`'s `usage()` function, append after the existing `context
  bundle` line:

  ```
    context manifest <plan-dir>        Pretty-print an existing context-manifest.yaml
    logs prune [--dry-run]             Apply .agent/logs/ retention (max_files/age/total size)`)
  ```

  (Adjust the trailing backtick placement so only the final line in the string closes it.)

- [x] **23.3** Run `cd cli && go vet ./... && go build -o eng . 2>&1` — fix any compile errors
  before proceeding.

---

## Task 24 — Version bump and docs integration (last task)

- [x] **24.1** Update `harness/VERSION`:

  ```
  0.5.0-phase5-runtime
  ```

- [x] **24.2** In `README.md`, immediately after the Phase 4 section added previously (before
  the following `---`), add:

  ```markdown

  Phase 5 wires the pieces above into a natural-language-first default experience:

  ```bash
  cd cli && go build -o eng .
  cd /path/to/any/project
  eng start   # doctor + a pointer to the runtime routing method, then launches your agent
  # inside that session, just describe the requirement — see
  # ~/.engineering-harness/core/runtime/METHOD.md for the exact command sequence it follows.
  ```

  Low-level commands (`eng plan new`, `eng workflow advance`, `eng context bundle`, ...) remain
  fully available for debugging, CI, and advanced manual workflows.

  See `.plans/2026-08-24-v2-harness-phase5-runtime/spec.md` for the full design.
  ```

- [x] **24.3** In `ROADMAP.md`, extend the note to include the Phase 5 plan link, following the
  same pattern as the Phase 4 addition.

- [x] **24.4** In `docs/src-map.md`, add a final module section after the Phase 4 entries:

  ```markdown

  ### `harness/core/runtime/`, Phase 5 workflow/context/log extensions — runtime integration

  What it does: `harness/core/runtime/METHOD.md` is the natural-language routing protocol a
  Claude Code session follows for a plain-language request. `eng adapter prompt` now folds in
  `eng context bundle`'s output automatically. `internal/workflow` gained `NEEDS_SPEC_APPROVAL`/
  `SPEC_APPROVED` (a distinct concept from the existing execution-approval gate) and a
  quick-fix fast path that skips straight from `TRIAGED` to `EXECUTING` with a minimal plan.
  `internal/logprune` bounds `.agent/logs/` growth. `internal/tooladapter`/`internal/toolrouter`
  are new, deliberately unpopulated foundations for future external-tool adapters, kept
  structurally separate from `internal/agent`'s coding-agent adapters.

  Key files: `harness/core/runtime/METHOD.md`, `cli/context_cmd.go` (`buildContextBundle`),
  `cli/internal/workflow/workflow.go` (the extended transition table)

  Notable: `planning_mode` defaults to `auto_plan` (Phase 1-4's exact behavior) whenever it is
  unset — only a project initialized by Phase 5's `eng init` gets `spec_first` explicitly; no
  existing project's state-machine behavior changes. Quick Fix's fast path deliberately skips
  review/approval by design — `eng plan escalate` is the correction mechanism if a request
  turns out to be broader than triage guessed.

  From: `.plans/2026-08-24-v2-harness-phase5-runtime/`
  ```
