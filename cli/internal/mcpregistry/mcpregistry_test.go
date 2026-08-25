package mcpregistry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.yaml")
	os.WriteFile(path, []byte("servers:\n  - name: docs-search\n    transport: mock\n    capabilities: [docs.search]\n    permission_category: read\n"), 0o644)
	servers, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].Name != "docs-search" || servers[0].Transport != "mock" {
		t.Fatalf("got %+v", servers)
	}
}

func TestLoadMissingFileIsNotError(t *testing.T) {
	servers, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil || len(servers) != 0 {
		t.Fatalf("expected empty, no error; got %+v, %v", servers, err)
	}
}

func TestLoadRealHarnessRegistry(t *testing.T) {
	servers, err := Load(filepath.Join("..", "..", "..", "harness", "mcp", "servers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range servers {
		if s.Name == "docs-search" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the real harness/mcp/servers.yaml to declare docs-search")
	}
}
