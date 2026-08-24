package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/executil"
	"eng/internal/hooks"
	"eng/internal/project"
)

func cmdHooks(args []string) {
	if len(args) < 2 || args[0] != "run" {
		fmt.Println("Usage: eng hooks run <stage>")
		os.Exit(1)
	}
	stage := args[1]

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	globalDefault := filepath.Join(harnessDir(), "hooks", "default.yaml")
	cfg, err := hooks.Load(dir, globalDefault)
	if err != nil {
		fmt.Println("error loading hooks:", err)
		os.Exit(1)
	}

	names := cfg.Stage(stage)
	if len(names) == 0 {
		fmt.Printf("No hooks configured for stage %q\n", stage)
		return
	}

	testCmd := ""
	if pcfg, err := project.Load(dir); err == nil {
		testCmd = pcfg.Stack.Test.String()
	}

	for _, name := range names {
		cmd := cfg.Commands[name]
		if cmd.Empty() {
			fmt.Printf("[%s] %-16s manual step — no shell command; perform via the documented role\n", stage, name)
			continue
		}
		if cmd.Shell != "" {
			cmd.Shell = strings.ReplaceAll(cmd.Shell, "${test_cmd}", testCmd)
		}
		fmt.Printf("[%s] %-16s -> %s\n", stage, name, cmd.String())
		out, err := executil.Run(cmd, dir)
		fmt.Print(out)
		if err != nil {
			fmt.Printf("HOOK FAILED: %s (%v)\n", name, err)
			os.Exit(1)
		}
	}
}
