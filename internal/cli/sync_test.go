package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestPlanAll_WithPackages_OnlyPlansSpecified(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	// Create flat packages: fish/ and git/
	writeTestFile(t, filepath.Join(dotfiles, "fish", ".config", "fish", "config.fish"), "# fish")
	writeTestFile(t, filepath.Join(dotfiles, "git", ".gitconfig"), "[user]")

	// Only request fish
	result := planAll(dotfiles, target, []string{"fish"})

	assert.Equal(t, 1, result.total, "should only plan fish package")
	assert.Equal(t, 1, result.pending, "fish config should be pending link")
}

func TestPlanAll_EmptyPackages_AppliesAll(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	writeTestFile(t, filepath.Join(dotfiles, "fish", ".config", "fish", "config.fish"), "# fish")
	writeTestFile(t, filepath.Join(dotfiles, "git", ".gitconfig"), "[user]")

	// Empty packages → all subdirs
	result := planAll(dotfiles, target, nil)

	assert.Equal(t, 2, result.total, "should plan all packages")
	assert.Equal(t, 2, result.pending, "all configs should be pending")
}

func TestDetectDriftFlat_FiltersToPackages(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	// Set up two packages in dotfiles
	fishSrc := filepath.Join(dotfiles, "fish", ".config", "fish", "config.fish")
	writeTestFile(t, fishSrc, "# managed")
	gitSrc := filepath.Join(dotfiles, "git", ".gitconfig")
	writeTestFile(t, gitSrc, "[user]")

	// Create correct symlinks for fish, but overwrite git
	require.NoError(t, os.MkdirAll(filepath.Join(target, ".config", "fish"), 0o755))
	require.NoError(t, os.Symlink(fishSrc, filepath.Join(target, ".config", "fish", "config.fish")))
	writeTestFile(t, filepath.Join(target, ".gitconfig"), "# overwritten")

	// Only check fish — git drift should be excluded
	driftEntries := detectDriftFlat(dotfiles, target, []string{"fish"}, nil)

	assert.Empty(t, driftEntries, "should have no drift for fish (symlink correct)")
}

func TestDetectDriftFlat_EmptyPackages_DetectsAll(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	// Set up git package with overwritten target
	writeTestFile(t, filepath.Join(dotfiles, "git", ".gitconfig"), "[user]")
	writeTestFile(t, filepath.Join(target, ".gitconfig"), "# overwritten")

	driftEntries := detectDriftFlat(dotfiles, target, nil, nil)

	assert.Len(t, driftEntries, 1, "should detect git drift")
	assert.Equal(t, "git", driftEntries[0].Package)
}

func TestPlanAll_IgnoresNonPackageDirs(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	writeTestFile(t, filepath.Join(dotfiles, "fish", ".config", "fish", "config.fish"), "# fish")
	writeTestFile(t, filepath.Join(dotfiles, "git", ".gitconfig"), "[user]")

	// Request a package that doesn't exist — should be silently skipped
	result := planAll(dotfiles, target, []string{"fish", "nonexistent"})

	assert.Equal(t, 1, result.total, "should only plan existing packages")
}
