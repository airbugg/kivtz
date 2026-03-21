// Package drift detects configuration drift — files that have been modified
// or added outside of kivtz's management.
package drift

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Kind classifies a drift entry.
type Kind int

const (
	Overwritten Kind = iota // Managed symlink replaced by a regular file
	New                     // Untracked file appeared in a managed directory
)

// Entry represents a file that has drifted from its managed state.
type Entry struct {
	Path    string // Absolute path of the drifted file
	Package string // Name of the stow package that manages this directory
	Kind    Kind
}

// Detect scans a dotfiles group directory for drift. The group dir contains
// stow packages (subdirectories), each mirroring a target directory layout.
//
// It detects:
//   - Managed symlinks that have been overwritten by regular files
//   - New files in managed subdirectories (not in the target root — too noisy)
//
// It skips:
//   - Symlinks (managed by stow, possibly from another group)
//   - Files matching the hardcoded deny list (secrets, history, runtime state)
//   - Files matching the ignore patterns (.syncignore format: "package/rel/path")
func Detect(groupDir, targetDir string, ignorePatterns []string) ([]Entry, error) {
	var results []Entry

	pkgEntries, err := os.ReadDir(groupDir)
	if err != nil {
		return nil, err
	}

	for _, pkgEntry := range pkgEntries {
		if !pkgEntry.IsDir() {
			continue
		}
		pkgName := pkgEntry.Name()
		pkgDir := filepath.Join(groupDir, pkgName)

		managed, managedDirs := indexPackage(pkgDir)

		results = append(results, checkOverwritten(pkgName, managed, targetDir, ignorePatterns)...)
		results = append(results, checkNew(pkgName, managed, managedDirs, targetDir, ignorePatterns)...)
	}

	return results, nil
}

// indexPackage walks a package directory and returns the set of managed
// file paths (relative to the package root) and their parent directories.
func indexPackage(pkgDir string) (managed map[string]bool, dirs map[string]bool) {
	managed = map[string]bool{}
	dirs = map[string]bool{}

	filepath.WalkDir(pkgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(pkgDir, path)
		managed[rel] = true
		if dir := filepath.Dir(rel); dir != "." {
			dirs[dir] = true
		}
		return nil
	})
	return
}

func checkOverwritten(pkgName string, managed map[string]bool, targetDir string, ignore []string) []Entry {
	var results []Entry
	for rel := range managed {
		if isDenied(filepath.Base(rel)) {
			continue
		}
		if isIgnored(pkgName+"/"+filepath.ToSlash(rel), ignore) {
			continue
		}
		targetPath := filepath.Join(targetDir, rel)
		info, err := os.Lstat(targetPath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue // Doesn't exist or is still a symlink — no drift
		}
		results = append(results, Entry{Path: targetPath, Package: pkgName, Kind: Overwritten})
	}
	return results
}

func checkNew(pkgName string, managed, dirs map[string]bool, targetDir string, ignore []string) []Entry {
	var results []Entry
	for dir := range dirs {
		targetSubDir := filepath.Join(targetDir, dir)
		entries, err := os.ReadDir(targetSubDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fullPath := filepath.Join(targetSubDir, e.Name())
			if isSymlink(fullPath) {
				continue
			}
			if isDenied(e.Name()) {
				continue
			}
			rel := filepath.Join(dir, e.Name())
			if managed[rel] {
				continue
			}
			if isIgnored(pkgName+"/"+filepath.ToSlash(rel), ignore) {
				continue
			}
			results = append(results, Entry{Path: fullPath, Package: pkgName, Kind: New})
		}
	}
	return results
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func isIgnored(path string, patterns []string) bool {
	// path format: "package/relative/file"
	// Strip the package prefix to get just the relative path
	relPath := path
	if idx := strings.Index(path, "/"); idx >= 0 {
		relPath = path[idx+1:]
	}

	for _, p := range patterns {
		p = strings.TrimSpace(p)

		if p == path || p == relPath {
			return true
		}
	}

	return false
}

// denyList contains filename patterns that should never be flagged as drift.
var denyList = []string{
	".npmrc", ".netrc", ".env", ".boto",
	".pem", ".key", ".p12", "id_rsa", "id_ed25519",
	"_history", ".bash_history", ".zsh_history", ".python_history", ".node_repl_history",
	".viminfo", ".lesshst",
	".DS_Store", ".CFUserTextEncoding",
	".claude.json", "history.jsonl", "stats-cache.json",
	"mcp-needs-auth-cache.json", "install-counts-cache.json",
	"settings.local.json", "settings.json.bak",
}

func isDenied(filename string) bool {
	for _, pattern := range denyList {
		if filename == pattern || strings.HasSuffix(filename, pattern) {
			return true
		}
	}
	return false
}

// ParseIgnoreFile reads a .syncignore file and returns non-empty, non-comment lines.
func ParseIgnoreFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns, nil
}
