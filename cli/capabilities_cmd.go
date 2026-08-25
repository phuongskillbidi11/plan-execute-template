package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/agent"
	"eng/internal/capabilities"
	"eng/internal/planmeta"
)

func cmdCapabilities(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: eng capabilities <list|explain> ...")
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		capabilitiesList(args[1:])
	case "explain":
		capabilitiesExplain(args[1:])
	default:
		fmt.Println("Usage: eng capabilities <list|explain> ...")
		os.Exit(1)
	}
}

func capabilitiesList(args []string) {
	verbose := false
	role := ""
	for i := 0; i < len(args); i++ {
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

func capabilitiesExplain(args []string) {
	if len(args) < 2 {
		fmt.Println(`Usage: eng capabilities explain <role> <plan-dir> ["<request text>"]`)
		os.Exit(1)
	}
	role := args[0]
	planDir, err := filepath.Abs(args[1])
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	request := ""
	if len(args) > 2 {
		request = strings.Join(args[2:], " ")
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	meta, err := planmeta.Load(planDir)
	if err != nil {
		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
		os.Exit(1)
	}

	result := routeTools(repoRoot, role, request, meta.ApprovedAt != "")

	fmt.Println("Tool routing for role:", role)
	for _, s := range result.Allowed {
		fmt.Printf("  %-24s ALLOWED        (%s) — %s\n", s.Capability, s.Adapter, s.Reason)
	}
	for _, b := range result.NeedsApproval {
		fmt.Printf("  %-24s NEEDS_APPROVAL (%s) — %s\n", b.Capability, b.Adapter, b.Reason)
	}
	for _, b := range result.Blocked {
		fmt.Printf("  %-24s BLOCKED        (%s) — %s\n", b.Capability, b.Adapter, b.Reason)
	}
	if len(result.Allowed)+len(result.NeedsApproval)+len(result.Blocked) == 0 {
		fmt.Println("  (no external capabilities requested by skills matching this request)")
	}
}
