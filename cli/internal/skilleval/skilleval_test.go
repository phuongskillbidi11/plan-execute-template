package skilleval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadScenariosParsesFields(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "example.yaml"), []byte("name: example\nrequest: \"debug a c++ build\"\nexpected_skills: [debugging, cpp]\n"), 0o644)
	scenarios, err := LoadScenarios(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenarios) != 1 || scenarios[0].Request != "debug a c++ build" || len(scenarios[0].ExpectedSkills) != 2 {
		t.Fatalf("got %+v", scenarios)
	}
}

func TestLoadScenariosMissingRootIsNotError(t *testing.T) {
	scenarios, err := LoadScenarios(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(scenarios) != 0 {
		t.Fatalf("expected empty, no error; got %+v, %v", scenarios, err)
	}
}

func TestLoadScenariosDefaultsNameToFilename(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "unnamed.yaml"), []byte("request: \"x\"\nexpected_skills: []\n"), 0o644)
	scenarios, err := LoadScenarios(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenarios) != 1 || scenarios[0].Name != "unnamed" {
		t.Fatalf("got %+v", scenarios)
	}
}
