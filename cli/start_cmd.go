package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"eng/internal/capabilities"
	"eng/internal/project"
)

func cmdStart(args []string) {
	doInit := false
	for _, a := range args {
		if a == "--init" {
			doInit = true
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if project.DetectModeResult(dir).Mode == "none" {
		if doInit {
			cmdInit(nil)
		} else {
			fmt.Println("This project isn't initialized for the harness yet.")
			fmt.Println("Run `eng init` first, or `eng start --init` to initialize now and continue.")
			fmt.Println("(eng init only ever creates .agent/project.yaml — nothing else is touched.)")
			return
		}
	}

	fmt.Println("eng start")
	fmt.Println()
	cmdDoctor(nil)

	fmt.Println("\nFor natural-language requests, this session should consult:")
	fmt.Printf("  %s\n", filepath.Join(harnessDir(), "core", "runtime", "METHOD.md"))

	if capabilities.Detect("claude") {
		fmt.Println("\nLaunching Claude Code...")
		c := exec.Command("claude")
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Println("\nCould not launch `claude` automatically:", err)
			fmt.Println("Run it yourself in this directory.")
		}
		return
	}

	fmt.Println("\n`claude` was not found on PATH. Configure an agent in .agent/project.yaml,")
	fmt.Println("or install one and re-run `eng start`.")
}
