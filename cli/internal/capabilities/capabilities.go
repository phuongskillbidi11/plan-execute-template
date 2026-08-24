package capabilities

import "os/exec"

// Known is the fixed set of capabilities eng can detect today. Device/
// protocol capabilities (serial, Modbus, OPC UA, ...) are explicitly
// deferred to a later phase alongside the MCP adapter layer they'd serve.
var Known = []string{"git", "claude", "codex", "docker"}

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
