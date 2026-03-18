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
	assert.Equal(t, original, loaded)
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := config.Load("/nonexistent/config.toml")
	require.NoError(t, err)
	assert.Equal(t, config.Config{}, cfg)
}

func TestSave_CreatesParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "config.toml")
	require.NoError(t, config.Save(config.Config{}, path))
	_, err := os.Stat(path)
	assert.NoError(t, err)
}
