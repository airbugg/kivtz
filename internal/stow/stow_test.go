package stow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/stow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestPlan_NewFile(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(src, ".config", "fish", "config.fish"), "# fish")

	entries, err := stow.Plan(src, target)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, stow.Link, entries[0].Action)
	assert.Equal(t, filepath.Join(src, ".config", "fish", "config.fish"), entries[0].Source)
	assert.Equal(t, filepath.Join(target, ".config", "fish", "config.fish"), entries[0].Target)
}

func TestPlan_CorrectSymlink(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	srcFile := filepath.Join(src, "config.fish")
	writeFile(t, srcFile, "# fish")

	targetFile := filepath.Join(target, "config.fish")
	require.NoError(t, os.Symlink(srcFile, targetFile))

	entries, err := stow.Plan(src, target)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, stow.Skip, entries[0].Action)
}

func TestPlan_RelativeSymlink(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	srcFile := filepath.Join(src, ".config", "app", "config")
	writeFile(t, srcFile, "content")

	targetFile := filepath.Join(target, ".config", "app", "config")
	require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
	rel, _ := filepath.Rel(filepath.Dir(targetFile), srcFile)
	require.NoError(t, os.Symlink(rel, targetFile))

	entries, err := stow.Plan(src, target)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, stow.Skip, entries[0].Action)
}

func TestPlan_ConflictingFile(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(src, "config"), "repo version")
	writeFile(t, filepath.Join(target, "config"), "local version")

	entries, err := stow.Plan(src, target)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, stow.Conflict, entries[0].Action)
}

func TestPlan_SameContent_ReturnsLink(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(src, "config"), "same")
	writeFile(t, filepath.Join(target, "config"), "same")

	entries, err := stow.Plan(src, target)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, stow.Link, entries[0].Action)
}

func TestPlan_WrongSymlink(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(src, "config"), "managed")

	targetFile := filepath.Join(target, "config")
	require.NoError(t, os.Symlink("/some/other/path", targetFile))

	entries, err := stow.Plan(src, target)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, stow.Conflict, entries[0].Action)
}

func TestApply_CreatesSymlinks(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	srcFile := filepath.Join(src, ".config", "app", "config")
	writeFile(t, srcFile, "content")

	entries := []stow.Entry{{Source: srcFile, Target: filepath.Join(target, ".config", "app", "config"), Action: stow.Link}}

	require.NoError(t, stow.Apply(entries))

	info, err := os.Lstat(filepath.Join(target, ".config", "app", "config"))
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)

	resolved, _ := os.Readlink(filepath.Join(target, ".config", "app", "config"))
	assert.Equal(t, srcFile, resolved)
}

func TestApply_SkipsNonLinkEntries(t *testing.T) {
	target := t.TempDir()
	writeFile(t, filepath.Join(target, "config"), "local")

	entries := []stow.Entry{{Source: "/src/config", Target: filepath.Join(target, "config"), Action: stow.Conflict}}

	require.NoError(t, stow.Apply(entries))

	// File should still be regular, not a symlink
	info, _ := os.Lstat(filepath.Join(target, "config"))
	assert.True(t, info.Mode().IsRegular())
}

func TestPlanThenApply_Idempotent(t *testing.T) {
	src, target := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(src, ".config", "fish", "config.fish"), "# fish")
	writeFile(t, filepath.Join(src, ".gitconfig"), "[user]")

	// First apply
	entries, err := stow.Plan(src, target)
	require.NoError(t, err)
	require.NoError(t, stow.Apply(entries))

	// Second plan should all be Skip
	entries2, err := stow.Plan(src, target)
	require.NoError(t, err)
	for _, e := range entries2 {
		assert.Equal(t, stow.Skip, e.Action, "should be skip after apply: %s", e.Target)
	}
}
