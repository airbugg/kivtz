// Package stow manages symlinks between a source directory (dotfiles package)
// and a target directory (typically $HOME). It plans what symlinks need to be
// created, detects conflicts, and applies changes.
package stow

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Action classifies what should happen for a managed file.
type Action int

const (
	Link     Action = iota // Target doesn't exist or has same content — create symlink
	Skip                   // Target is already a correct symlink
	Conflict               // Target exists with different content
)

// Entry represents a single file in a stow plan.
type Entry struct {
	Source string // Absolute path in the dotfiles package
	Target string // Absolute path in the target directory (e.g. $HOME)
	Action Action
	Diff   string // Unified diff for conflicts, empty otherwise
}

// Plan scans srcDir and determines what actions are needed to stow it
// into targetDir. Creates file-level symlinks only (no directory folding).
func Plan(srcDir, targetDir string) ([]Entry, error) {
	var entries []Entry

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, rel)
		action := classify(path, targetPath)

		var diff string
		if action == Conflict {
			diff = generateDiff(targetPath, path)
		}

		entries = append(entries, Entry{
			Source: path,
			Target: targetPath,
			Action: action,
			Diff:   diff,
		})
		return nil
	})

	return entries, err
}

// Apply executes a plan, creating symlinks for Link entries.
// Conflict and Skip entries are left untouched.
func Apply(entries []Entry) error {
	for _, e := range entries {
		if e.Action != Link {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(e.Target), 0o755); err != nil {
			return fmt.Errorf("creating dir for %s: %w", e.Target, err)
		}
		// Remove existing file if present (same-content replacement)
		if _, err := os.Lstat(e.Target); err == nil {
			if err := os.Remove(e.Target); err != nil {
				return fmt.Errorf("removing %s: %w", e.Target, err)
			}
		}
		if err := os.Symlink(e.Source, e.Target); err != nil {
			return fmt.Errorf("symlinking %s: %w", e.Target, err)
		}
	}
	return nil
}

func classify(source, target string) Action {
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return Link
	}
	if err != nil {
		return Conflict
	}

	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := resolveLink(target)
		if err != nil {
			return Conflict
		}
		if resolved == filepath.Clean(source) {
			return Skip
		}
		// Dangling symlink (target no longer exists) — safe to replace
		if _, err := os.Stat(target); os.IsNotExist(err) {
			return Link
		}
		return Conflict
	}

	if info.Mode().IsRegular() {
		if contentEqual(source, target) {
			return Link // Same content — safe to replace with symlink
		}
		return Conflict
	}

	return Conflict
}

func resolveLink(path string) (string, error) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	return filepath.Clean(target), nil
}

func generateDiff(local, repo string) string {
	localContent, err := os.ReadFile(local)
	if err != nil {
		return ""
	}
	repoContent, err := os.ReadFile(repo)
	if err != nil {
		return ""
	}

	dmp := diffmatchpatch.New()
	a, b, lines := dmp.DiffLinesToChars(string(localContent), string(repoContent))
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)
	diffs = dmp.DiffCleanupSemantic(diffs)

	var buf strings.Builder
	buf.WriteString("--- local (current)\n+++ repo (incoming)\n")
	for _, d := range diffs {
		for _, line := range strings.Split(strings.TrimRight(d.Text, "\n"), "\n") {
			switch d.Type {
			case diffmatchpatch.DiffDelete:
				buf.WriteString("-" + line + "\n")
			case diffmatchpatch.DiffInsert:
				buf.WriteString("+" + line + "\n")
			case diffmatchpatch.DiffEqual:
				buf.WriteString(" " + line + "\n")
			}
		}
	}
	return buf.String()
}

func contentEqual(a, b string) bool {
	ac, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	bc, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ac, bc)
}
