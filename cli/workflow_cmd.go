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
	"eng/internal/rolestate"
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

	isQuickFix := meta.RiskLevel == "quick-fix"
	rs, _ := rolestate.Load(planDir)
	activeRole := "none"
	if rs != nil && rs.CurrentRole != "" {
		activeRole = rs.CurrentRole
	}
	nextRole := rolestate.NextRole(meta.State, isQuickFix)
	if nextRole == "" {
		nextRole = "none"
	}
	fmt.Printf("Active role:   %s\n", activeRole)
	fmt.Printf("Next role:     %s\n", nextRole)

	facts, err := gatherFacts(planDir, meta)
	if err != nil {
		fmt.Println("error gathering state:", err)
		return
	}
	decision := workflow.Decide(facts)
	fmt.Printf("Next:          %s\n", decision.Reason)

	if meta.State == workflow.StateVerifying {
		fmt.Printf("Mechanical verification: %s\n", verdictOrPending(meta.Verification.Verdict))
		if facts.RoleVerificationRequired {
			fmt.Printf("Role verification:       %s\n", verdictOrPending(meta.RoleVerification.Verdict))
		} else {
			fmt.Println("Role verification:       not required (workflow.verifier disabled)")
		}
	}
	printUncheckedChecklistLines(meta.State, planDir)
}

// verdictOrPending renders an empty verdict string as "pending" rather
// than a blank line — used by workflowStatus's VERIFYING-state detail.
func verdictOrPending(verdict string) string {
	if verdict == "" {
		return "pending"
	}
	return verdict
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
		printUncheckedChecklistLines(meta.State, planDir)
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

	// Phase 10: a stale role activation from before a replan cycle must not
	// satisfy the executor-activation gate afterward — see spec.md's
	// role-state invalidation rule.
	if meta.State == workflow.StateNeedsReplan {
		if err := rolestate.Reset(planDir); err != nil {
			fmt.Println("warning: failed to reset role-state.yaml on replan:", err)
		}
	}

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

// printUncheckedChecklistLines names the specific tasks.md line(s) blocking
// advance out of EXECUTING/NEEDS_FIX, instead of leaving "tasks.md still
// has unchecked items" to be figured out by re-reading the whole file.
// Phase 9 spec.md P2-1 — the bottom Completion checklist stays the sole
// authoritative source; this only improves what's printed about it.
func printUncheckedChecklistLines(state, planDir string) {
	if state != workflow.StateExecuting && state != workflow.StateNeedsFix {
		return
	}
	lines := uncheckedChecklistLines(planDir)
	for _, l := range lines {
		fmt.Printf("               %s\n", l)
	}
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
	planningMode := "auto_plan"
	requireSpecApproval := true
	roleVerificationRequired := true
	if cfg, err := project.Load(repoRoot); err == nil {
		reviewRequired = reviewRequired || cfg.Workflow.PlanReviewEnabled()
		planningMode = cfg.Workflow.PlanningModeOrDefault()
		requireSpecApproval = cfg.Workflow.RequireSpecApprovalOrDefault()
		roleVerificationRequired = cfg.Workflow.VerifierEnabled()
	}

	drifted, _, _ := checkDrift(planDir)

	isQuickFix := meta.RiskLevel == "quick-fix"
	executorActivated := false
	if rs, err := rolestate.Load(planDir); err == nil {
		executorActivated = rs.CurrentRole == "executor"
	}

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

		IsQuickFix:          isQuickFix,
		PlanningMode:        planningMode,
		SpecReady:           specReady(planDir),
		SpecApproved:        meta.SpecApprovedAt != "",
		RequireSpecApproval: requireSpecApproval,
		TasksAndTestsReady:  tasksAndTestsReady(planDir),

		ExecutorActivated:        executorActivated,
		RoleVerificationRequired: roleVerificationRequired,
		RoleVerificationVerdict:  meta.RoleVerification.Verdict,
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

func tasksComplete(planDir string) bool {
	return len(uncheckedChecklistLines(planDir)) == 0
}

// uncheckedChecklistLines returns every "- [ ]" line in tasks.md, trimmed —
// the same scan tasksComplete has always gated on (the bottom Completion
// checklist remains the sole machine-authoritative source, per Phase 9
// spec.md P2-1/DECISION_LOG.md Decision 5), surfaced so `eng workflow
// advance`'s "still has unchecked items" message can name the actual
// blocking line(s) instead of leaving an Executor to guess.
func uncheckedChecklistLines(planDir string) []string {
	f, err := os.Open(filepath.Join(planDir, "tasks.md"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Same exact detection tasksComplete has always used — untrimmed,
		// so this is byte-identical to the pre-Phase-9 gating logic. Only
		// the appended copy is trimmed, for display.
		if line := scanner.Text(); strings.HasPrefix(line, "- [ ]") {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}
