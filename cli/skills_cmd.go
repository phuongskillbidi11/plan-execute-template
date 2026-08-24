package main

import (
	"fmt"
	"os"
	"path/filepath"

	"eng/internal/skills"
)

func cmdSkills(args []string) {
	if len(args) == 0 || args[0] != "list" {
		fmt.Println("Usage: eng skills list")
		os.Exit(1)
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	resolved, err := skills.Resolve(filepath.Join(harnessDir(), "skills"), filepath.Join(dir, "skills"))
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	for _, s := range resolved {
		fmt.Printf("%-30s [%-6s] domain=%-12s %s\n", s.Name, s.Source, s.Domain, s.Description)
	}
}
