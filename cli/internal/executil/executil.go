package executil

import (
	"os/exec"

	"gopkg.in/yaml.v3"
)

// Command describes how to run one shell command, either as a plain string
// (compatibility mode, run via `sh -c` — the only mode Phase 1/2 ever used)
// or as a structured argv (no shell at all). A plain YAML scalar unmarshals
// into Shell; a mapping with `command`/`args` keys unmarshals into the
// structured form. This is what makes the change backward compatible: every
// existing `build_cmd: "npm run build"`-style string keeps parsing exactly
// as before.
type Command struct {
	Shell   string
	Program string
	Args    []string
}

func (c *Command) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return value.Decode(&c.Shell)
	}
	var structured struct {
		Command string   `yaml:"command"`
		Args    []string `yaml:"args"`
	}
	if err := value.Decode(&structured); err != nil {
		return err
	}
	c.Program = structured.Command
	c.Args = structured.Args
	return nil
}

func (c Command) MarshalYAML() (interface{}, error) {
	if c.Program != "" {
		return map[string]interface{}{"command": c.Program, "args": c.Args}, nil
	}
	return c.Shell, nil
}

// Empty reports whether no command was configured.
func (c Command) Empty() bool {
	return c.Shell == "" && c.Program == ""
}

// String returns a human-readable form for logging/printing.
func (c Command) String() string {
	if c.Program != "" {
		s := c.Program
		for _, a := range c.Args {
			s += " " + a
		}
		return s
	}
	return c.Shell
}

// Run executes c in dir and returns combined stdout+stderr.
func Run(c Command, dir string) (string, error) {
	var cmd *exec.Cmd
	if c.Program != "" {
		cmd = exec.Command(c.Program, c.Args...)
	} else {
		cmd = exec.Command("sh", "-c", c.Shell)
	}
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
