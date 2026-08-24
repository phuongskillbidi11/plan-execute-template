package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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

	fmt.Printf("Created .agent/project.yaml — mode: %s, stack: %s\n", cfg.Mode, cfg.Stack.Type)
	if mode == "hybrid" {
		fmt.Println("Existing CLAUDE.md / .plans/ / skills/ were left untouched.")
	}
}
