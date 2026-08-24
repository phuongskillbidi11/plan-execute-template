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

// copyTree is defined in install.go and reused here — see plan_cmd.go's own
// history in tasks.md Task 6.2 for why this comment exists instead of a
// second definition.

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
