package main

import (
	"fmt"
	"os"

	"eng/internal/agent"
)

func cmdAdapter(args []string) {
	if len(args) < 2 || args[0] != "prompt" {
		fmt.Println("Usage: eng adapter prompt <planner|plan-reviewer|executor|verifier> <plan-dir>")
		os.Exit(1)
	}
	role := agent.Role(args[1])
	if len(args) < 3 {
		fmt.Println("Usage: eng adapter prompt <role> <plan-dir>")
		os.Exit(1)
	}
	planDir := args[2]

	a := agent.ClaudeCodeAdapter{HarnessDir: harnessDir()}
	if !a.Available() {
		fmt.Println("note: `claude` was not found on PATH — printing the prompt for manual use anyway.")
	}

	prompt, err := a.RolePrompt(role, planDir)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	fmt.Println(prompt)
}
