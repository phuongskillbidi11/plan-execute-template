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
