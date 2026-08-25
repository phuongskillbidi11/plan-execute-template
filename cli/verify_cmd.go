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

		ctxCfg := loadContextConfig(repoRoot)
		logPath, logErr := writeFullLog(repoRoot, "verify", out)
		display := out
		if ctxCfg.SummarizeToolOutput {
			display = summarizeOutput(out, ctxCfg.MaxLogLines)
		}
		fmt.Fprintf(&report, "```\n%s\n```\n\n", display)
		if logErr == nil && display != out {
			fmt.Fprintf(&report, "Full output: `%s`\n\n", logPath)
		}
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

// writeFullLog persists the complete tool output to .agent/logs/, keeping
// the report/stdout bounded regardless of test-suite size (Requirement 8).
func writeFullLog(repoRoot, kind, content string) (string, error) {
	dir := filepath.Join(repoRoot, ".agent", "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%s.log", kind, time.Now().UTC().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// summarizeOutput bounds out to maxLines by keeping the head and tail —
// deliberately not head-only, since failures are conventionally reported
// at the end of test output. Line-count based, not token-based, per
// Requirement 7's explicit instruction not to hard-code token counts to a
// single model.
func summarizeOutput(out string, maxLines int) string {
	if maxLines <= 0 {
		return out
	}
	lines := strings.Split(out, "\n")
	if len(lines) <= maxLines {
		return out
	}
	half := maxLines / 2
	head := lines[:half]
	tail := lines[len(lines)-half:]
	omitted := len(lines) - len(head) - len(tail)
	return strings.Join(head, "\n") +
		fmt.Sprintf("\n... [%d lines omitted, see full log] ...\n", omitted) +
		strings.Join(tail, "\n")
}
