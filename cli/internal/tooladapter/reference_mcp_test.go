package tooladapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"eng/internal/toolcap"
)

func TestReferenceMCPAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = ReferenceMCPAdapter{}
}

func TestReferenceMCPAdapterAvailableWhenDocsRootExists(t *testing.T) {
	a := NewReferenceMCPAdapter(t.TempDir())
	if !a.Available() {
		t.Fatal("expected available when docs root exists")
	}
}

func TestReferenceMCPAdapterUnavailableWhenDocsRootMissing(t *testing.T) {
	a := NewReferenceMCPAdapter(filepath.Join(t.TempDir(), "nope"))
	if a.Available() {
		t.Fatal("expected unavailable when docs root is missing")
	}
	if _, err := a.Doctor(); err == nil {
		t.Fatal("expected Doctor to error when unavailable")
	}
}

func TestReferenceMCPAdapterCapabilityIsRead(t *testing.T) {
	a := NewReferenceMCPAdapter(t.TempDir())
	caps := a.Capabilities()
	if len(caps) != 1 || caps[0].Name != "docs.search" || caps[0].Risk != toolcap.RiskRead {
		t.Fatalf("got %+v", caps)
	}
}

func TestReferenceMCPAdapterSearchFindsMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "example.md"), []byte("# Modbus addressing\nPDU address is not the same as 40001.\n"), 0o644)
	a := NewReferenceMCPAdapter(dir)
	out, err := a.Invoke("docs.search", []string{"modbus"}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "example.md") {
		t.Fatalf("expected match to reference example.md, got %q", out)
	}
}

func TestReferenceMCPAdapterSearchNoMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "example.md"), []byte("# Something else\n"), 0o644)
	a := NewReferenceMCPAdapter(dir)
	out, err := a.Invoke("docs.search", []string{"nonexistent-term-xyz"}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no matches") {
		t.Fatalf("expected a no-matches message, got %q", out)
	}
}

func TestReferenceMCPAdapterInvokeRequiresQuery(t *testing.T) {
	a := NewReferenceMCPAdapter(t.TempDir())
	if _, err := a.Invoke("docs.search", nil, "."); err == nil {
		t.Fatal("expected an error when no query is given")
	}
}
