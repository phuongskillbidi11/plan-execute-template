package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/project"
	"eng/internal/skills"
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

	mode := project.DetectMode(dir)
	switch mode {
	case "legacy":
		fmt.Println("Project mode:      legacy (CLAUDE.md/.plans found, no .agent/) — fully compatible, no action required")
	case "none":
		fmt.Println("Project mode:      none — not yet initialized (`eng init` to enable)")
	default:
		fmt.Printf("Project mode:      %s (.agent/project.yaml present)\n", mode)
	}

	if cfg, err := project.Load(dir); err == nil {
		fmt.Printf("Detected stack:    %s\n", cfg.Stack.Type)
	}

	resolved, err := skills.Resolve(filepath.Join(hDir, "skills"), filepath.Join(dir, "skills"))
	if err == nil {
		fmt.Printf("Skills resolved:   %d\n", len(resolved))
		for _, s := range resolved {
			fmt.Printf("  - %-30s [%s] %s\n", s.Name, s.Source, s.Description)
		}
	}
}
