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

// Scan discovers config files under root (both ~/.config/* and ~/.* dotfiles).
// It collects only filesystem metadata — never reads file content.
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
			path := filepath.Join(configDir, de.Name())
			entry, err := buildEntry(de.Name(), path)
			if err != nil {
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
		path := filepath.Join(root, name)
		entry, err := buildEntry(name, path)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
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
