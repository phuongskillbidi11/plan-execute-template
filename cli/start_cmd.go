package main

import (
	"fmt"
	"os"
	"os/exec"

	"eng/internal/capabilities"
)

func cmdStart(args []string) {
	fmt.Println("eng start")
	fmt.Println()
	cmdDoctor(nil)

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
