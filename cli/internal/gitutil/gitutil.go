package gitutil

import (
	"os/exec"
	"strings"
)

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// HeadSHA returns the current commit hash of dir's repository.
func HeadSHA(dir string) (string, error) {
	return run(dir, "rev-parse", "HEAD")
}

// ChangedFilesSince returns paths (relative to dir) that differ between sha
// and the current working tree, including uncommitted changes — this is
// deliberately "diff against a fixed point," not "commits since sha."
func ChangedFilesSince(dir, sha string) ([]string, error) {
	out, err := run(dir, "diff", "--name-only", sha)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}
