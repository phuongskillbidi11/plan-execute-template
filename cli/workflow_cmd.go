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
