package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func harnessDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("error: cannot resolve home directory:", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".engineering-harness")
}

func binDir() string {
	return filepath.Join(harnessDir(), "bin")
}

// harnessVersion reads ~/.engineering-harness/VERSION, trimmed — "" if the
// harness isn't installed or the file is missing. Shared by doctor.go and
// start_cmd.go so there's one read of this file, not two.
func harnessVersion() string {
	data, err := os.ReadFile(filepath.Join(harnessDir(), "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func cmdInstall(args []string) {
	flagset := flag.NewFlagSet("install", flag.ExitOnError)
	from := flagset.String("from", ".", "path to a checkout containing a harness/ directory")
	addToPath := flagset.Bool("add-to-path", false, "also add the harness bin/ directory to PATH")
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

	if err := installBinary(); err != nil {
		fmt.Println("warning: could not copy eng binary to bin/:", err)
	} else {
		fmt.Printf("Copied eng binary to %s\n", binDir())
	}

	printPathInstructions()
	if *addToPath {
		if err := applyPathSetup(); err != nil {
			fmt.Println("warning: could not apply PATH setup automatically:", err)
		}
	}
}

func installBinary() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(binDir(), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	name := "eng"
	if runtime.GOOS == "windows" {
		name = "eng.exe"
	}
	return os.WriteFile(filepath.Join(binDir(), name), data, 0o755)
}

func printPathInstructions() {
	dir := binDir()
	fmt.Println("\nTo use `eng` from any terminal, add this to your PATH:")
	if runtime.GOOS == "windows" {
		fmt.Printf("  setx PATH \"%%PATH%%;%s\"\n", dir)
		fmt.Println("  (open a new terminal afterward — setx only affects new sessions)")
	} else {
		fmt.Printf("  export PATH=\"%s:$PATH\"\n", dir)
		fmt.Println("  (add that line to ~/.bashrc or ~/.zshrc to make it permanent)")
	}
	fmt.Println("Or re-run `eng install --add-to-path` to apply this automatically.")
}

func applyPathSetup() error {
	dir := binDir()
	if runtime.GOOS == "windows" {
		current := os.Getenv("PATH")
		if len(current)+len(dir)+1 > 1024 {
			fmt.Println("warning: PATH is already near setx's 1024-character limit — add it manually instead")
			return nil
		}
		cmd := exec.Command("setx", "PATH", current+";"+dir)
		return cmd.Run()
	}

	line := fmt.Sprintf("export PATH=\"%s:$PATH\"\n", dir)
	for _, profile := range []string{".bashrc", ".zshrc"} {
		home, err := os.UserHomeDir()
		if err != nil {
			continue
		}
		path := filepath.Join(home, profile)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		existing, _ := os.ReadFile(path)
		if strings.Contains(string(existing), dir) {
			continue // already present
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, err = f.WriteString("\n# added by `eng install --add-to-path`\n" + line)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
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
