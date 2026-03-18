package scanner_test

import (
	"fmt"
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

func TestScan_ExcludesDeniedEntries(t *testing.T) {
	root := t.TempDir()

	// Denied entries
	writeFile(t, filepath.Join(root, ".ssh", "id_rsa"), "secret key")
	writeFile(t, filepath.Join(root, ".gnupg", "pubring.kbx"), "keyring")
	writeFile(t, filepath.Join(root, ".cache", "something"), "cached")
	writeFile(t, filepath.Join(root, ".local", "share", "data"), "data")
	writeFile(t, filepath.Join(root, ".config", "node_modules", "pkg", "index.js"), "module")
	writeFile(t, filepath.Join(root, ".npmrc"), "registry=https://example.com")
	writeFile(t, filepath.Join(root, ".bash_history"), "ls\ncd\n")
	writeFile(t, filepath.Join(root, ".zsh_history"), "ls\ncd\n")
	writeFile(t, filepath.Join(root, ".Trash", "deleted.txt"), "trash")

	// Allowed entry
	writeFile(t, filepath.Join(root, ".gitconfig"), "[user]\nname = test")
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}

	assert.True(t, names[".gitconfig"], "should include .gitconfig")
	assert.True(t, names["fish"], "should include fish")
	assert.False(t, names[".ssh"], "should exclude .ssh")
	assert.False(t, names[".gnupg"], "should exclude .gnupg")
	assert.False(t, names[".cache"], "should exclude .cache")
	assert.False(t, names[".local"], "should exclude .local")
	assert.False(t, names["node_modules"], "should exclude node_modules")
	assert.False(t, names[".npmrc"], "should exclude .npmrc")
	assert.False(t, names[".bash_history"], "should exclude .bash_history")
	assert.False(t, names[".zsh_history"], "should exclude .zsh_history")
	assert.False(t, names[".Trash"], "should exclude .Trash")
}

func TestScan_DenyListIsExported(t *testing.T) {
	assert.NotEmpty(t, scanner.DenyList, "DenyList should be exported and non-empty")
	assert.Contains(t, scanner.DenyList, ".ssh")
	assert.Contains(t, scanner.DenyList, ".cache")
}

func TestScan_ExcludesDirectoryWithTooManyFiles(t *testing.T) {
	root := t.TempDir()

	// Directory with 150 files — should be excluded
	bloatedDir := filepath.Join(root, ".config", "bloated")
	require.NoError(t, os.MkdirAll(bloatedDir, 0o755))
	for i := range 150 {
		writeFile(t, filepath.Join(bloatedDir, fmt.Sprintf("file%d.conf", i)), "data")
	}

	// Small directory — should be included
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	assert.False(t, names["bloated"], "should exclude directory with >100 files")
	assert.True(t, names["fish"], "should include small directory")
}

func TestScan_ExcludesLargeFile(t *testing.T) {
	root := t.TempDir()

	// 2MB file — should be excluded
	largeContent := make([]byte, 2*1024*1024)
	writeFile(t, filepath.Join(root, ".largeconfig"), string(largeContent))

	// Small file — should be included
	writeFile(t, filepath.Join(root, ".gitconfig"), "[user]\nname = test")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	assert.False(t, names[".largeconfig"], "should exclude file >1MB")
	assert.True(t, names[".gitconfig"], "should include small file")
}

func TestScan_ExcludesLargeDirectory(t *testing.T) {
	root := t.TempDir()

	// Directory with total size >1MB
	bigDir := filepath.Join(root, ".config", "bigdir")
	require.NoError(t, os.MkdirAll(bigDir, 0o755))
	bigContent := make([]byte, 600*1024) // 600KB each, 2 files = 1.2MB
	writeFile(t, filepath.Join(bigDir, "a.dat"), string(bigContent))
	writeFile(t, filepath.Join(bigDir, "b.dat"), string(bigContent))

	// Small directory
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	assert.False(t, names["bigdir"], "should exclude directory with total size >1MB")
	assert.True(t, names["fish"], "should include small directory")
}

func TestScan_EmptyRootReturnsEmpty(t *testing.T) {
	root := t.TempDir()

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
