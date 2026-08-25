package logprune

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Result struct {
	Deleted        []string
	KeptMostRecent string
}

// Prune deletes *.log files in dir beyond maxFiles/maxAgeDays/maxTotalMB,
// oldest first, but never deletes the single most recently modified file —
// a cheap approximation of "don't delete logs an active plan might still
// need" (see Phase 5 spec.md Decision 8). Any limit <= 0 is treated as
// "no limit" for that dimension.
func Prune(dir string, maxFiles, maxAgeDays, maxTotalMB int, dryRun bool) (Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil
		}
		return Result{}, err
	}

	type fileInfo struct {
		path string
		mod  time.Time
		size int64
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".log" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{filepath.Join(dir, e.Name()), info.ModTime(), info.Size()})
	}
	if len(files) == 0 {
		return Result{}, nil
	}

	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	mostRecent := files[len(files)-1].path

	var totalBytes int64
	for _, f := range files {
		totalBytes += f.size
	}
	maxBytes := int64(maxTotalMB) * 1024 * 1024
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	remaining := len(files)

	var result Result
	result.KeptMostRecent = mostRecent
	for _, f := range files {
		if f.path == mostRecent {
			continue
		}
		tooMany := maxFiles > 0 && remaining > maxFiles
		tooOld := maxAgeDays > 0 && f.mod.Before(cutoff)
		tooBig := maxTotalMB > 0 && totalBytes > maxBytes
		if tooMany || tooOld || tooBig {
			if !dryRun {
				os.Remove(f.path)
			}
			result.Deleted = append(result.Deleted, f.path)
			totalBytes -= f.size
			remaining--
		}
	}
	return result, nil
}
