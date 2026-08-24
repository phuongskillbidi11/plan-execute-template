package main

import (
	"fmt"
	"os"

	"eng/internal/capabilities"
)

func cmdCapabilities(args []string) {
	if len(args) == 0 || args[0] != "list" {
		fmt.Println("Usage: eng capabilities list")
		os.Exit(1)
	}
	for _, name := range capabilities.Known {
		status := "unavailable"
		if capabilities.Detect(name) {
			status = "available"
		}
		fmt.Printf("%-10s %s\n", name, status)
	}
}
