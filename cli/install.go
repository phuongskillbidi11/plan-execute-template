package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func harnessDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("error: cannot resolve home directory:", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".engineering-harness")
}

func cmdInstall(args []string) {
	flagset := flag.NewFlagSet("install", flag.ExitOnError)
	from := flagset.String("from", ".", "path to a checkout containing a harness/ directory")
	flagset.Parse(args)

	src := filepath.Join(*from, "harness")
	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		fmt.Printf("error: %s does not contain a harness/ directory\n", *from)
		os.Exit(1)
	}

	dst := harnessDir()
	if err := copyTree(src, dst); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Printf("Installed harness to %s\n", dst)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
