package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestScan_FindsXDGAndTraditionalDotfiles(t *testing.T) {
	root := t.TempDir()

	// XDG: ~/.config/fish/config.fish
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish config")
	// Traditional: ~/.gitconfig
	writeFile(t, filepath.Join(root, ".gitconfig"), "[user]\nname = test")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	assert.True(t, names["fish"], "should find fish under .config")
	assert.True(t, names[".gitconfig"], "should find .gitconfig as traditional dotfile")
}

func TestScan_EntryMetadata_Directory(t *testing.T) {
	root := t.TempDir()

	// Directory with 2 files
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")
	writeFile(t, filepath.Join(root, ".config", "fish", "conf.d", "aliases.fish"), "# aliases")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "fish", e.Name)
	assert.Equal(t, filepath.Join(root, ".config", "fish"), e.Path)
	assert.True(t, e.IsDir)
	assert.Equal(t, 2, e.FileCount)
	assert.Greater(t, e.Size, int64(0))
	assert.False(t, e.ModTime.IsZero())
}

func TestScan_EntryMetadata_SingleFile(t *testing.T) {
	root := t.TempDir()

	content := "[user]\nname = test"
	writeFile(t, filepath.Join(root, ".gitconfig"), content)

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, ".gitconfig", e.Name)
	assert.Equal(t, filepath.Join(root, ".gitconfig"), e.Path)
	assert.False(t, e.IsDir)
	assert.Equal(t, 1, e.FileCount)
	assert.Equal(t, int64(len(content)), e.Size)
	assert.False(t, e.ModTime.IsZero())
}

func TestScan_OnlyTopLevelEntries(t *testing.T) {
	root := t.TempDir()

	// XDG entries: fish and nvim (top-level under .config)
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")
	writeFile(t, filepath.Join(root, ".config", "nvim", "init.lua"), "-- nvim")
	// Nested dir inside nvim — should NOT appear as separate entry
	writeFile(t, filepath.Join(root, ".config", "nvim", "lua", "plugins.lua"), "-- plugins")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	assert.ElementsMatch(t, []string{"fish", "nvim"}, names, "should only return top-level .config children")
}

func TestScan_DoesNotIncludeConfigDirItself(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")
	writeFile(t, filepath.Join(root, ".bashrc"), "# bash")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	for _, e := range entries {
		assert.NotEqual(t, ".config", e.Name, ".config itself should not appear as an entry")
	}
}

func TestScan_EmptyRootReturnsEmpty(t *testing.T) {
	root := t.TempDir()

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
