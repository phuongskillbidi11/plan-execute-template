package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eng/internal/gitutil"
	"eng/internal/planmeta"
	"eng/internal/project"
	"eng/internal/workflow"
)

// reorderFlagsFirst moves "--flag"/"--flag=value" tokens (and, for a
// non-boolean flag, the value token immediately following it) ahead of any
// positional arguments, so a command reads correctly regardless of whether
// a user writes "eng plan review <dir> --verdict PASS" or
// "eng plan review --verdict PASS <dir>". Go's flag package only supports
// the latter natively — it stops parsing at the first non-flag token, which
// silently discarded every flag in this family of commands until this fix.
func reorderFlagsFirst(args []string, boolFlags map[string]bool) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") || boolFlags[name] {
			continue // value is embedded, or this flag takes no value
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}

func cmdPlan(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: eng plan <new|drift|retry|review|approve|approve-spec|verify-review|escalate|block|cancel> ...")
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
	case "approve-spec":
		planApproveSpec(args[1:])
	case "verify-review":
		planVerifyReview(args[1:])
	case "escalate":
		planEscalate(args[1:])
	case "block":
		planBlock(args[1:])
	case "cancel":
		planCancel(args[1:])
	default:
		fmt.Println("Usage: eng plan <new|drift|retry|review|approve|approve-spec|verify-review|escalate|block|cancel> ...")
		os.Exit(1)
	}
}

func planNew(args []string) {
	flagset := flag.NewFlagSet("plan new", flag.ExitOnError)
	risk := flagset.String("risk", "feature", "quick-fix|bug|feature|architecture|high-risk")
	requiresApproval := flagset.Bool("requires-approval", false, "force an approval gate regardless of risk level")
	flagset.Parse(reorderFlagsFirst(args, map[string]bool{"requires-approval": true}))
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
	if *risk == "quick-fix" {
		tmplDir = filepath.Join(harnessDir(), "templates", "quickfix")
	}
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
	flagset.Parse(reorderFlagsFirst(args, map[string]bool{}))
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

// planVerifyReview records the Verifier role's own independent verdict —
// deliberately separate from `eng verify`'s mechanical PASS/FAIL (Phase 10
// spec.md's "Mechanical verification vs. role verification"). Mirrors
// planReview's exact shape: this command only records the verdict flag;
// verifier-review.md (scaffolded by `eng plan new`, alongside review.md)
// is filled in by hand, the same way review.md always has been.
func planVerifyReview(args []string) {
	flagset := flag.NewFlagSet("plan verify-review", flag.ExitOnError)
	verdict := flagset.String("verdict", "", "PASS|FAIL")
	by := flagset.String("by", "", "who is verifying")
	notes := flagset.String("notes", "", "free-text notes")
	flagset.Parse(reorderFlagsFirst(args, map[string]bool{}))
	rest := flagset.Args()
	if len(rest) == 0 || (*verdict != "PASS" && *verdict != "FAIL") {
		fmt.Println(`Usage: eng plan verify-review <plan-dir> --verdict PASS|FAIL [--by <name>] [--notes "..."]`)
		os.Exit(1)
	}
	planDir, _ := filepath.Abs(rest[0])

	meta, err := planmeta.Load(planDir)
	if err != nil {
		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
		os.Exit(1)
	}

	meta.RoleVerification = planmeta.RoleVerification{
		Verdict:    *verdict,
		VerifiedAt: time.Now().UTC().Format(time.RFC3339),
		VerifiedBy: *by,
		Notes:      *notes,
	}
	if err := planmeta.Save(planDir, meta); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	planmeta.AppendEvent(planDir, "role_verification_recorded", *verdict)
	fmt.Printf("Recorded Verifier role verdict: %s\n", *verdict)
}

func planApprove(args []string) {
	flagset := flag.NewFlagSet("plan approve", flag.ExitOnError)
	by := flagset.String("by", "", "who is approving")
	flagset.Parse(reorderFlagsFirst(args, map[string]bool{}))
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
	flagset.Parse(reorderFlagsFirst(args, map[string]bool{}))
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
	flagset.Parse(reorderFlagsFirst(args, map[string]bool{}))
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
