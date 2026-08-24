package executil

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestUnmarshalScalarIsShell(t *testing.T) {
	var c Command
	if err := yaml.Unmarshal([]byte(`"echo hi"`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Shell != "echo hi" || c.Program != "" {
		t.Fatalf("got %+v", c)
	}
}

func TestUnmarshalStructuredForm(t *testing.T) {
	var c Command
	if err := yaml.Unmarshal([]byte("command: cmake\nargs: [--build, build]\n"), &c); err != nil {
		t.Fatal(err)
	}
	if c.Program != "cmake" || len(c.Args) != 2 || c.Args[1] != "build" {
		t.Fatalf("got %+v", c)
	}
}

func TestRunShellMode(t *testing.T) {
	out, err := Run(Command{Shell: "echo hello"}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("got %q", out)
	}
}

func TestRunStructuredMode(t *testing.T) {
	// Uses `go version` rather than `echo` — Windows has no standalone echo.exe,
	// but Go is already a hard prerequisite for this entire repository.
	out, err := Run(Command{Program: "go", Args: []string{"version"}}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "go version") {
		t.Fatalf("got %q", out)
	}
}

func TestEmpty(t *testing.T) {
	if !(Command{}).Empty() {
		t.Fatal("expected zero value to be Empty")
	}
	if (Command{Shell: "x"}).Empty() {
		t.Fatal("expected non-empty Shell to not be Empty")
	}
}
