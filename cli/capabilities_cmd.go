package main

import (
	"fmt"
	"os"

	"eng/internal/agent"
	"eng/internal/capabilities"
)

func cmdCapabilities(args []string) {
	if len(args) == 0 || args[0] != "list" {
		fmt.Println("Usage: eng capabilities list [--verbose] [--role <role>]")
		os.Exit(1)
	}

	verbose := false
	role := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--verbose":
			verbose = true
		case "--role":
			if i+1 < len(args) {
				role = args[i+1]
			}
		}
	}

	for _, c := range capabilities.DescribeAll() {
		if role != "" && !agent.RoleMayUse(role, c.Name) {
			continue
		}
		status := "unavailable"
		if c.Available {
			status = "available"
		}
		if verbose {
			fmt.Printf("%-10s %-12s provider=%-14s version=%s\n", c.Name, status, c.Provider, c.Version)
		} else {
			fmt.Printf("%-10s %s\n", c.Name, status)
		}
	}
}
