package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a discovered config file or directory.
type Entry struct {
	Name      string
	Path      string
	Size      int64
	ModTime   time.Time
	FileCount int
	IsDir     bool
}

// DenyList contains entry names that should never appear in scan results.
// Exported for testability.
var DenyList = []string{
	".ssh", ".gnupg", ".gpg",
	".npmrc", ".netrc", ".env", ".boto",
	".bash_history", ".zsh_history", ".python_history", ".node_repl_history",
	".cache", ".local", ".Trash",
	"node_modules",
	".DS_Store", ".CFUserTextEncoding",
	".viminfo", ".lesshst",
}

// MaxSize is the maximum total size (in bytes) for an entry to be included.
const MaxSize = 1 << 20 // 1 MB

// MaxFileCount is the maximum number of files for a directory to be included.
const MaxFileCount = 100

// Scan discovers config files under root (both ~/.config/* and ~/.* dotfiles).
// It collects only filesystem metadata — never reads file content.
// Entries matching the deny-list, exceeding MaxSize, or exceeding MaxFileCount are excluded.
func Scan(root string) ([]Entry, error) {
	var entries []Entry

	// Scan XDG config dir (~/.config/*)
	configDir := filepath.Join(root, ".config")
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		xdgEntries, err := os.ReadDir(configDir)
		if err != nil {
			return nil, err
		}
		for _, de := range xdgEntries {
			if isDenied(de.Name()) {
				continue
			}
			path := filepath.Join(configDir, de.Name())
			entry, err := buildEntry(de.Name(), path)
			if err != nil {
				continue
			}
			if isFiltered(entry) {
				continue
			}
			entries = append(entries, entry)
		}
	}

	// Scan traditional dotfiles (~/.*)
	rootEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, de := range rootEntries {
		name := de.Name()
		if !strings.HasPrefix(name, ".") || name == "." || name == ".." || name == ".config" {
			continue
		}
		if isDenied(name) {
			continue
		}
		path := filepath.Join(root, name)
		entry, err := buildEntry(name, path)
		if err != nil {
			continue
		}
		if isFiltered(entry) {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func isDenied(name string) bool {
	for _, d := range DenyList {
		if name == d {
			return true
		}
	}
	return false
}

func isFiltered(e Entry) bool {
	if e.Size > MaxSize {
		return true
	}
	if e.IsDir && e.FileCount > MaxFileCount {
		return true
	}
	return false
}

func buildEntry(name, path string) (Entry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Entry{}, err
	}

	entry := Entry{
		Name:    name,
		Path:    path,
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}

	if info.IsDir() {
		size, count, err := dirStats(path)
		if err != nil {
			return Entry{}, err
		}
		entry.Size = size
		entry.FileCount = count
	} else {
		entry.Size = info.Size()
		entry.FileCount = 1
	}

	return entry, nil
}

func dirStats(path string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	return totalSize, fileCount, err
}
