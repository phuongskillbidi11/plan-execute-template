package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/detect"
	"eng/internal/executil"
	"eng/internal/project"
)

func cmdInit(args []string) {
	flagset := flag.NewFlagSet("init", flag.ExitOnError)
	flagset.Parse(args)

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if _, err := os.Stat(filepath.Join(dir, project.ConfigPath)); err == nil {
		fmt.Println(".agent/project.yaml already exists — not overwritten")
		return
	}

	det := detect.Detect(dir)
	mode := project.DetectMode(dir)
	switch mode {
	case "none":
		mode = "modern"
	case "legacy":
		mode = "hybrid" // opting into .agent/ moves a legacy project to hybrid
	}

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

	if err := project.Save(dir, cfg); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	ensureGitignoreEntry(dir, ".agent/logs/")

	fmt.Printf("Created .agent/project.yaml — mode: %s, stack: %s\n", cfg.Mode, cfg.Stack.Type)
	if mode == "hybrid" {
		fmt.Println("Existing CLAUDE.md / .plans/ / skills/ were left untouched.")
	}
}

// ensureGitignoreEntry appends entry to the project's .gitignore if it
// isn't already present, creating the file if none exists. Never
// overwrites or reorders existing content — purely additive, matching
// this repo's own "never touch what you don't need to" convention.
func ensureGitignoreEntry(dir, entry string) {
	path := filepath.Join(dir, ".gitignore")
	existing, _ := os.ReadFile(path)
	if strings.Contains(string(existing), entry) {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}
	f.WriteString(prefix + entry + "\n")
}
