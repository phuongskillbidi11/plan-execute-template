package main

import (
	"fmt"
	"os"
	"path/filepath"

	"eng/internal/skills"
	"eng/internal/skillvalidate"
)

func cmdSkills(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: eng skills <list|validate>")
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		skillsList(args[1:])
	case "validate":
		skillsValidate(args[1:])
	default:
		fmt.Println("Usage: eng skills <list|validate>")
		os.Exit(1)
	}
}

func skillsList(args []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	resolved, err := skills.ResolveWithPrivate(filepath.Join(harnessDir(), "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	for _, s := range resolved {
		level := s.Level
		if level == "" {
			level = "-"
		}
		fmt.Printf("%-30s [%-7s] domain=%-12s level=%-11s %s\n", s.Name, s.Source, s.Domain, level, s.Description)
	}
}

func skillsValidate(args []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	report, err := skillvalidate.Validate(filepath.Join(harnessDir(), "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Printf("%d skill(s) discovered, %d error(s), %d warning(s)\n\n", report.Discovered, len(report.Errors()), len(report.Warnings()))
	for _, issue := range report.Issues {
		fmt.Printf("[%s] %-30s %s\n", issue.Severity, issue.Skill, issue.Message)
	}

	if len(report.Errors()) > 0 {
		os.Exit(1)
	}
}
