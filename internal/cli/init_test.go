package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDiscovery_listMode(t *testing.T) {
	root := t.TempDir()
	// Create some config entries
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".config", "fish"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".config", "fish", "config.fish"), []byte("# fish"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitconfig"), []byte("[user]"), 0o644))

	var stdout, stderr bytes.Buffer
	err := runDiscovery(root, &stdout, &stderr, "list")

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "fish\t")
	assert.Contains(t, stdout.String(), ".gitconfig\t")
	// Status goes to stderr, not stdout
	assert.NotContains(t, stdout.String(), "Scanning")
}

func TestRunDiscovery_jsonMode(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".config", "nvim"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".config", "nvim", "init.lua"), []byte("-- nvim"), 0o644))

	var stdout, stderr bytes.Buffer
	err := runDiscovery(root, &stdout, &stderr, "json")

	require.NoError(t, err)

	var parsed []scanner.JSONEntry
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "nvim", parsed[0].Name)
}

func TestRunDiscovery_statusToStderr(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".config", "fish"), 0o755))

	var stdout, stderr bytes.Buffer
	err := runDiscovery(root, &stdout, &stderr, "list")

	require.NoError(t, err)
	assert.NotEmpty(t, stderr.String())
}
