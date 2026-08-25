package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/capabilities"
	"eng/internal/project"
	"eng/internal/skillvalidate"
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

	if cfg, err := project.Load(dir); err == nil {
		fmt.Printf("Detected stack:    %s\n", cfg.Stack.Type)
	}

	report, err := skillvalidate.Validate(filepath.Join(hDir, "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
	if err == nil {
		broken := 0
		for _, issue := range report.Errors() {
			if strings.Contains(issue.Message, "cycle") || strings.Contains(issue.Message, "requires unknown") {
				broken++
			}
		}
		fmt.Println("\nSkills:")
		fmt.Printf("  %d discovered\n", report.Discovered)
		fmt.Printf("  %d valid\n", report.Discovered-len(issuedSkillNames(report.Issues)))
		fmt.Printf("  %d warning(s)\n", len(report.Warnings()))
		fmt.Printf("  %d broken dependency issue(s)\n", broken)
		fmt.Println("  (run `eng skills list` or `eng skills validate` for detail)")
	}

	adapters := registeredAdapters(dir)
	fmt.Println("\nTools:")
	for _, a := range adapters {
		status := "unavailable"
		if a.Available() {
			status = "available"
		}
		fmt.Printf("  %-10s %-12s [%d capabilities]\n", a.Name(), status, len(a.Capabilities()))
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

// issuedSkillNames counts distinct skills with at least one issue, so
// doctor's "valid" count doesn't double-subtract a skill with two
// warnings.
func issuedSkillNames(issues []skillvalidate.Issue) map[string]bool {
	out := map[string]bool{}
	for _, i := range issues {
		if i.Skill != "(graph)" {
			out[i.Skill] = true
		}
	}
	return out
}
