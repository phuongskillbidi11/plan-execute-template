package tooladapter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"eng/internal/toolcap"
)

// ReferenceMCPAdapter is the Phase 7 deterministic mock/reference
// MCP-style adapter (Phase 7 spec.md Decision 9) — no network, no
// external process, always available when docsRoot exists. It proves
// the adapter lifecycle for an MCP-style integration without
// implementing real MCP transport. Its one capability, docs.search,
// greps docsRoot's *.md files for a query and returns bounded, matching
// file paths.
type ReferenceMCPAdapter struct {
	docsRoot string
}

func NewReferenceMCPAdapter(docsRoot string) ReferenceMCPAdapter {
	return ReferenceMCPAdapter{docsRoot: docsRoot}
}

func (a ReferenceMCPAdapter) Name() string     { return "mcp-docs" }
func (a ReferenceMCPAdapter) Provider() string { return "mcp" }
func (a ReferenceMCPAdapter) Version() string  { return "1.0.0" }

func (a ReferenceMCPAdapter) Available() bool {
	info, err := os.Stat(a.docsRoot)
	return err == nil && info.IsDir()
}

func (a ReferenceMCPAdapter) Capabilities() []toolcap.Capability {
	return []toolcap.Capability{{Name: "docs.search", Risk: toolcap.RiskRead}}
}

func (a ReferenceMCPAdapter) Doctor() (string, error) {
	if !a.Available() {
		return "", fmt.Errorf("docs root %s not found", a.docsRoot)
	}
	return "mock MCP server — docs root found at " + a.docsRoot, nil
}

func (a ReferenceMCPAdapter) Invoke(capability string, args []string, dir string) (string, error) {
	if capability != "docs.search" {
		return "", fmt.Errorf("mcp-docs adapter does not support capability %q", capability)
	}
	if len(args) == 0 {
		return "", fmt.Errorf("docs.search requires a query argument")
	}
	if !a.Available() {
		return "", fmt.Errorf("docs root %s not found", a.docsRoot)
	}
	query := strings.ToLower(strings.Join(args, " "))

	var matches []string
	filepath.WalkDir(a.docsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(string(data)), query) {
			matches = append(matches, path)
		}
		return nil
	})

	if len(matches) == 0 {
		return fmt.Sprintf("no matches for %q under %s", query, a.docsRoot), nil
	}
	const maxMatches = 10
	if len(matches) > maxMatches {
		matches = matches[:maxMatches]
	}
	return "matches for " + query + ":\n- " + strings.Join(matches, "\n- "), nil
}
