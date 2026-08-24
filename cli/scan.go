package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/detect"
)

func cmdScan(args []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	det := detect.Detect(dir)
	fmt.Printf("Stack: %s\n", det.Type)
	if det.Build != "" {
		fmt.Printf("Build: %s\n", det.Build)
	}
	if det.Test != "" {
		fmt.Printf("Test:  %s\n", det.Test)
	}

	ignore := loadAgentIgnore(dir)
	counts := map[string]int{}
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		if d.IsDir() && matchesIgnore(rel, ignore) {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			if ext := filepath.Ext(path); ext != "" {
				counts[ext]++
			}
		}
		return nil
	})

	fmt.Println("\nFile counts by extension:")
	for ext, n := range counts {
		fmt.Printf("  %-10s %d\n", ext, n)
	}
}

func loadAgentIgnore(dir string) []string {
	f, err := os.Open(filepath.Join(dir, ".agentignore"))
	if err != nil {
		return []string{".git", "node_modules", "dist", "build", "vendor", "target"}
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, strings.TrimSuffix(line, "/"))
		}
	}
	return lines
}

func matchesIgnore(rel string, ignore []string) bool {
	base := filepath.Base(rel)
	for _, pattern := range ignore {
		if base == pattern {
			return true
		}
	}
	return false
}
