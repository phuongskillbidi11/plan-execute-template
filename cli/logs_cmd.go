package main

import (
	"fmt"
	"os"
	"path/filepath"

	"eng/internal/logprune"
)

func cmdLogs(args []string) {
	if len(args) == 0 || args[0] != "prune" {
		fmt.Println("Usage: eng logs prune [--dry-run]")
		os.Exit(1)
	}
	dryRun := len(args) > 1 && args[1] == "--dry-run"

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	cfg := loadContextConfig(dir)

	result, err := logprune.Prune(filepath.Join(dir, ".agent", "logs"), cfg.MaxLogFiles, cfg.MaxLogAgeDays, cfg.MaxLogTotalMB, dryRun)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	verb := "Deleted"
	if dryRun {
		verb = "Would delete"
	}
	fmt.Printf("%s %d log file(s); kept most recent: %s\n", verb, len(result.Deleted), result.KeptMostRecent)
	for _, f := range result.Deleted {
		fmt.Println("  -", f)
	}
}
