package capabilities

import (
	"os/exec"
	"strings"
)

// Known is the fixed set of capabilities eng can detect today. Device/
// protocol capabilities (serial, Modbus, OPC UA, ...) are explicitly
// deferred to a later phase alongside the MCP adapter layer they'd serve.
var Known = []string{"git", "claude", "codex", "docker", "gh"}

// Detect reports whether name's executable is found on PATH.
func Detect(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// DetectAll returns every Known capability mapped to its availability.
func DetectAll() map[string]bool {
	out := make(map[string]bool, len(Known))
	for _, name := range Known {
		out[name] = Detect(name)
	}
	return out
}

// Capability is the richer, additive schema Phase 5 needs — Detect/DetectAll
// above are unchanged and untouched by any existing caller.
type Capability struct {
	Name      string
	Available bool
	Provider  string
	Version   string // best-effort; "" if unknown or unavailable
}

func Describe(name string) Capability {
	c := Capability{Name: name, Available: Detect(name), Provider: "local-binary"}
	if c.Available {
		c.Version = detectVersion(name)
	}
	return c
}

func DescribeAll() []Capability {
	out := make([]Capability, 0, len(Known))
	for _, name := range Known {
		out = append(out, Describe(name))
	}
	return out
}

// detectVersion is best-effort and only implemented for tools with a
// well-known "--version" flag — not every CLI uses one uniformly, and
// per-tool version-string parsing beyond this is a later improvement.
func detectVersion(name string) string {
	switch name {
	case "git", "docker":
		out, err := exec.Command(name, "--version").Output()
		if err != nil {
			return ""
		}
		lines := strings.SplitN(string(out), "\n", 2)
		return strings.TrimSpace(lines[0])
	default:
		return ""
	}
}
