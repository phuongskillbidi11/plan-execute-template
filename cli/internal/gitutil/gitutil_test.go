package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one"), 0o644)
	run("add", "a.txt")
	run("commit", "-q", "-m", "init")
	return dir
}

func TestHeadSHA(t *testing.T) {
	dir := initRepo(t)
	sha, err := HeadSHA(dir)
	if err != nil || len(sha) < 7 {
		t.Fatalf("unexpected sha %q, err %v", sha, err)
	}
}

func TestChangedFilesSince(t *testing.T) {
	dir := initRepo(t)
	sha, _ := HeadSHA(dir)

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("two"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new"), 0o644)

	changed, err := ChangedFilesSince(dir, sha)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != "a.txt" {
		// b.txt is untracked, not part of `git diff` until added — expected.
		t.Fatalf("expected only a.txt (tracked, modified), got %+v", changed)
	}
}
