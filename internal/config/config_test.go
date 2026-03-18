package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSave_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	original := config.Config{
		DotfilesDir: "~/.dotfiles",
		RepoURL:     "github.com/airbugg/kivtzeynekuda",
		Platform:    "darwin",
		Hostname:    "test-host",
	}

	require.NoError(t, config.Save(original, path))

	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.DotfilesDir, loaded.DotfilesDir)
	assert.Equal(t, original.RepoURL, loaded.RepoURL)
	assert.Equal(t, original.Platform, loaded.Platform)
	assert.Equal(t, original.Hostname, loaded.Hostname)
	assert.NotNil(t, loaded.Packages)
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := config.Load("/nonexistent/config.toml")
	require.NoError(t, err)
	assert.Empty(t, cfg.DotfilesDir)
	assert.NotNil(t, cfg.Packages)
	assert.Empty(t, cfg.Packages)
}

func TestSave_CreatesParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "config.toml")
	require.NoError(t, config.Save(config.Config{}, path))
	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestPackages_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	original := config.Config{
		DotfilesDir: "~/.dotfiles",
		Platform:    "darwin",
		Packages:    []string{"fish", "git"},
	}

	require.NoError(t, config.Save(original, path))

	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"fish", "git"}, loaded.Packages)
}

func TestPackages_LegacyConfigReturnsEmptySlice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	// Write a legacy config without packages field
	legacy := `dotfiles_dir = "~/.dotfiles"
platform = "darwin"
hostname = "test-host"
`
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o644))

	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.NotNil(t, loaded.Packages, "Packages should be empty slice, not nil")
	assert.Empty(t, loaded.Packages)
}
