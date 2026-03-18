package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAdd_NoDotfilesDir_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.toml")

	// Save config without dotfiles_dir
	require.NoError(t, config.Save(config.Config{}, configPath))

	var out bytes.Buffer
	err := runAdd("/some/path", configPath, &out)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dotfiles directory not configured")
}

func TestRunAdd_AdoptsAndUpdatesConfig(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := filepath.Join(home, "dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesDir, 0o755))

	configPath := filepath.Join(home, "config.toml")
	require.NoError(t, config.Save(config.Config{
		DotfilesDir: dotfilesDir,
	}, configPath))

	// Create a config dir to adopt
	nvimDir := filepath.Join(home, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- nvim config"), 0o644))

	var out bytes.Buffer
	err := runAdd(nvimDir, configPath, &out)
	require.NoError(t, err)

	// Config should have nvim in packages
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Contains(t, cfg.Packages, "nvim")

	// Source should be a symlink now
	info, err := os.Lstat(nvimDir)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)
}

func TestRunAdd_ShowsSummaryWithStats(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := filepath.Join(home, "dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesDir, 0o755))

	configPath := filepath.Join(home, "config.toml")
	require.NoError(t, config.Save(config.Config{
		DotfilesDir: dotfilesDir,
	}, configPath))

	// Create a config dir with multiple files
	fishDir := filepath.Join(home, ".config", "fish")
	require.NoError(t, os.MkdirAll(filepath.Join(fishDir, "conf.d"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "config.fish"), make([]byte, 1024), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "conf.d", "aliases.fish"), make([]byte, 512), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "conf.d", "env.fish"), make([]byte, 600), 0o644))

	var out bytes.Buffer
	err := runAdd(fishDir, configPath, &out)
	require.NoError(t, err)

	// Summary should mention package name, file count, and size
	output := out.String()
	assert.Contains(t, output, "fish")
	assert.Contains(t, output, "3 files")
	assert.Contains(t, output, "2.1 KB")
}

func TestRunAdd_SingleFile_AutoDetectsPackageName(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := filepath.Join(home, "dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesDir, 0o755))

	configPath := filepath.Join(home, "config.toml")
	require.NoError(t, config.Save(config.Config{
		DotfilesDir: dotfilesDir,
	}, configPath))

	// Create a traditional dotfile
	gitconfig := filepath.Join(home, ".gitconfig")
	require.NoError(t, os.WriteFile(gitconfig, []byte("[user]\n\tname = Test"), 0o644))

	var out bytes.Buffer
	err := runAdd(gitconfig, configPath, &out)
	require.NoError(t, err)

	// Should detect "git" as package name
	assert.Contains(t, out.String(), "Adopted git")

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Contains(t, cfg.Packages, "git")
}

func TestRunAdd_SourceDoesNotExist_ReturnsError(t *testing.T) {
	home := t.TempDir()
	dotfilesDir := filepath.Join(home, "dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesDir, 0o755))

	configPath := filepath.Join(home, "config.toml")
	require.NoError(t, config.Save(config.Config{
		DotfilesDir: dotfilesDir,
	}, configPath))

	var out bytes.Buffer
	err := runAdd("/nonexistent/path", configPath, &out)
	assert.Error(t, err)
}
