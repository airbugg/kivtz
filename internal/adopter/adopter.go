// Package adopter moves config files/directories into a flat dotfiles directory
// and creates symlinks back to the original locations.
package adopter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PackageName derives a short package name from a config path.
// ~/.config/fish/ → fish, ~/.gitconfig → git, ~/.bashrc → bash
func PackageName(path string) string {
	// If inside .config/, use the last component directly
	dir, base := filepath.Split(filepath.Clean(path))
	if filepath.Base(filepath.Clean(dir)) == ".config" {
		return base
	}

	// Traditional dotfile: strip leading dot and known suffixes
	name := strings.TrimPrefix(base, ".")
	for _, suffix := range []string{"config", "rc"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

// Adopt moves source into dotfilesDir/<packageName>/ and creates a symlink
// at the original location pointing back.
//
// For directories: moves the entire directory tree.
// For single files: moves the file into a package subdirectory.
func Adopt(source, dotfilesDir string) error {
	source = filepath.Clean(source)

	// Check source exists
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("source does not exist: %w", err)
	}

	// If source is already a symlink, it's already adopted
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("source %s is already a symlink", source)
	}

	pkgName := PackageName(source)
	pkgDir := filepath.Join(dotfilesDir, pkgName)

	if info.IsDir() {
		return adoptDir(source, pkgDir)
	}
	return adoptFile(source, pkgDir, info.Name())
}

func adoptDir(source, pkgDir string) error {
	// Move directory to dotfiles
	if err := os.Rename(source, pkgDir); err != nil {
		return fmt.Errorf("moving directory: %w", err)
	}

	// Create symlink at original location
	if err := os.Symlink(pkgDir, source); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}
	return nil
}

func adoptFile(source, pkgDir, filename string) error {
	dest := filepath.Join(pkgDir, filename)

	// Create package directory
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		return fmt.Errorf("creating package dir: %w", err)
	}

	// Move file
	if err := os.Rename(source, dest); err != nil {
		return fmt.Errorf("moving file: %w", err)
	}

	// Create symlink at original location
	if err := os.Symlink(dest, source); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}
	return nil
}
