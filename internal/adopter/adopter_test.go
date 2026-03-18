package adopter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// --- PackageName tests ---

func TestPackageName_XDGConfigDir(t *testing.T) {
	// ~/.config/fish/ → fish
	name := PackageName(filepath.Join("/home/user", ".config", "fish"))
	assert.Equal(t, "fish", name)
}

func TestPackageName_XDGConfigNvim(t *testing.T) {
	// ~/.config/nvim/ → nvim
	name := PackageName(filepath.Join("/home/user", ".config", "nvim"))
	assert.Equal(t, "nvim", name)
}

func TestPackageName_TraditionalDotfile(t *testing.T) {
	// ~/.gitconfig → git
	name := PackageName("/home/user/.gitconfig")
	assert.Equal(t, "git", name)
}

func TestPackageName_TraditionalDotfileRC(t *testing.T) {
	// ~/.bashrc → bash
	name := PackageName("/home/user/.bashrc")
	assert.Equal(t, "bash", name)
}

func TestPackageName_TraditionalDotDir(t *testing.T) {
	// ~/.tmux → tmux
	name := PackageName("/home/user/.tmux")
	assert.Equal(t, "tmux", name)
}

func TestPackageName_PlainName(t *testing.T) {
	// ~/.config/ghostty → ghostty
	name := PackageName(filepath.Join("/home/user", ".config", "ghostty"))
	assert.Equal(t, "ghostty", name)
}

// --- Adopt directory tests ---

func TestAdopt_Directory_MovesAndSymlinks(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := t.TempDir()

	// Create ~/.config/fish/ with conf.d/config.fish inside
	fishDir := filepath.Join(home, ".config", "fish")
	writeFile(t, filepath.Join(fishDir, "config.fish"), "set -x PATH /usr/bin")
	writeFile(t, filepath.Join(fishDir, "conf.d", "aliases.fish"), "alias ll='ls -la'")

	err := Adopt(fishDir, dotfilesDir)
	require.NoError(t, err)

	// Files should exist in dotfiles dir under fish/
	adopted := filepath.Join(dotfilesDir, "fish", "config.fish")
	assert.FileExists(t, adopted)
	adoptedNested := filepath.Join(dotfilesDir, "fish", "conf.d", "aliases.fish")
	assert.FileExists(t, adoptedNested)

	// Original location should be a symlink
	info, err := os.Lstat(fishDir)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "original should be a symlink")

	// Symlink should point to the adopted location
	target, err := os.Readlink(fishDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dotfilesDir, "fish"), target)
}

func TestAdopt_Directory_PreservesContent(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := t.TempDir()

	fishDir := filepath.Join(home, ".config", "fish")
	writeFile(t, filepath.Join(fishDir, "config.fish"), "set -x PATH /usr/bin")

	err := Adopt(fishDir, dotfilesDir)
	require.NoError(t, err)

	// Content should be preserved after move
	content, err := os.ReadFile(filepath.Join(dotfilesDir, "fish", "config.fish"))
	require.NoError(t, err)
	assert.Equal(t, "set -x PATH /usr/bin", string(content))
}

// --- Adopt single file tests ---

func TestAdopt_SingleFile_MovesAndSymlinks(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := t.TempDir()

	gitconfig := filepath.Join(home, ".gitconfig")
	writeFile(t, gitconfig, "[user]\n\tname = Test")

	err := Adopt(gitconfig, dotfilesDir)
	require.NoError(t, err)

	// File should exist in dotfiles dir under git/
	adopted := filepath.Join(dotfilesDir, "git", ".gitconfig")
	assert.FileExists(t, adopted)

	// Original should be a symlink
	info, err := os.Lstat(gitconfig)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "original should be a symlink")

	// Symlink points to adopted file
	target, err := os.Readlink(gitconfig)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dotfilesDir, "git", ".gitconfig"), target)
}

func TestAdopt_SingleFile_PreservesContent(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := t.TempDir()

	gitconfig := filepath.Join(home, ".gitconfig")
	writeFile(t, gitconfig, "[user]\n\tname = Test")

	err := Adopt(gitconfig, dotfilesDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dotfilesDir, "git", ".gitconfig"))
	require.NoError(t, err)
	assert.Equal(t, "[user]\n\tname = Test", string(content))
}

// --- Error cases ---

func TestAdopt_SourceDoesNotExist_ReturnsError(t *testing.T) {
	dotfilesDir := t.TempDir()
	err := Adopt("/nonexistent/path", dotfilesDir)
	assert.Error(t, err)
}

func TestAdopt_AlreadySymlinked_ReturnsError(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := t.TempDir()

	// Create the adopted target
	fishAdopted := filepath.Join(dotfilesDir, "fish")
	require.NoError(t, os.MkdirAll(fishAdopted, 0o755))
	writeFile(t, filepath.Join(fishAdopted, "config.fish"), "content")

	// Create symlink at source pointing to adopted location
	fishDir := filepath.Join(home, ".config", "fish")
	require.NoError(t, os.MkdirAll(filepath.Dir(fishDir), 0o755))
	require.NoError(t, os.Symlink(fishAdopted, fishDir))

	err := Adopt(fishDir, dotfilesDir)
	assert.Error(t, err)
}
