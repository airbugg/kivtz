package drift_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/drift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(target, link))
}

func TestDetect_OverwrittenSymlink(t *testing.T) {
	group, target := t.TempDir(), t.TempDir()

	src := filepath.Join(group, "fish", ".config", "fish", "config.fish")
	writeFile(t, src, "# managed")

	tgt := filepath.Join(target, ".config", "fish", "config.fish")
	writeFile(t, tgt, "# overwritten") // regular file where symlink should be

	entries, err := drift.Detect(group, target, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, drift.Overwritten, entries[0].Kind)
	assert.Equal(t, "fish", entries[0].Package)
}

func TestDetect_NewFileInSubdir(t *testing.T) {
	group, target := t.TempDir(), t.TempDir()

	src := filepath.Join(group, "fish", ".config", "fish", "config.fish")
	writeFile(t, src, "# managed")
	symlink(t, src, filepath.Join(target, ".config", "fish", "config.fish"))

	// New file in managed subdir
	writeFile(t, filepath.Join(target, ".config", "fish", "local.fish"), "# new")

	entries, err := drift.Detect(group, target, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, drift.New, entries[0].Kind)
}

func TestDetect_RootDirNotScanned(t *testing.T) {
	group, target := t.TempDir(), t.TempDir()

	// Package manages ~/.gitconfig (file in target root)
	writeFile(t, filepath.Join(group, "git", ".gitconfig"), "[user]")
	symlink(t, filepath.Join(group, "git", ".gitconfig"), filepath.Join(target, ".gitconfig"))

	// Unrelated file in target root — should NOT be detected
	writeFile(t, filepath.Join(target, ".boto"), "")

	entries, err := drift.Detect(group, target, nil)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDetect_SymlinksFromOtherGroupsSkipped(t *testing.T) {
	group, target := t.TempDir(), t.TempDir()

	src := filepath.Join(group, "fish", ".config", "fish", "config.fish")
	writeFile(t, src, "# common")
	symlink(t, src, filepath.Join(target, ".config", "fish", "config.fish"))

	// Symlink from another group (e.g. macos/fish)
	symlink(t, "/other/macos/fish/macos.fish", filepath.Join(target, ".config", "fish", "macos.fish"))

	entries, err := drift.Detect(group, target, nil)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDetect_DenyListBlocks(t *testing.T) {
	group, target := t.TempDir(), t.TempDir()

	writeFile(t, filepath.Join(group, "app", ".config", "app", "config"), "managed")
	symlink(t, filepath.Join(group, "app", ".config", "app", "config"), filepath.Join(target, ".config", "app", "config"))

	// Denied files in managed directory
	writeFile(t, filepath.Join(target, ".config", "app", ".DS_Store"), "")
	writeFile(t, filepath.Join(target, ".config", "app", "settings.local.json"), "{}")
	writeFile(t, filepath.Join(target, ".config", "app", ".env"), "SECRET=x")

	entries, err := drift.Detect(group, target, nil)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDetect_IgnorePatterns(t *testing.T) {
	group, target := t.TempDir(), t.TempDir()

	writeFile(t, filepath.Join(group, "bin", ".local", "bin", "gh"), "#!/bin/sh")
	symlink(t, filepath.Join(group, "bin", ".local", "bin", "gh"), filepath.Join(target, ".local", "bin", "gh"))

	writeFile(t, filepath.Join(target, ".local", "bin", "claude"), "#!/bin/sh")

	entries, err := drift.Detect(group, target, []string{"bin/.local/bin/claude"})
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDetect_CleanState(t *testing.T) {
	group, target := t.TempDir(), t.TempDir()

	src := filepath.Join(group, "fish", ".config", "fish", "config.fish")
	writeFile(t, src, "# managed")
	symlink(t, src, filepath.Join(target, ".config", "fish", "config.fish"))

	entries, err := drift.Detect(group, target, nil)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestParseIgnoreFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".syncignore")
	writeFile(t, path, "# comment\nbin/.local/bin/claude\n\nclaude/.claude/x.json\n")

	patterns, err := drift.ParseIgnoreFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"bin/.local/bin/claude", "claude/.claude/x.json"}, patterns)
}

func TestParseIgnoreFile_Missing(t *testing.T) {
	patterns, err := drift.ParseIgnoreFile("/nonexistent")
	require.NoError(t, err)
	assert.Nil(t, patterns)
}
