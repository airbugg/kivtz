package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/stow"
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

func TestPlanMachine_PlansFromMachineDir(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	// Create machine dir with home-relative files (no package nesting)
	writeTestFile(t, filepath.Join(dotfiles, "macbook", ".config", "fish", "config.fish"), "# fish")
	writeTestFile(t, filepath.Join(dotfiles, "macbook", ".gitconfig"), "[user]")
	writeTestFile(t, filepath.Join(dotfiles, "macbook", ".config", "fish", "conf.d", "git.fish"), "# aliases")

	result, err := planMachine(dotfiles, target, "macbook")

	require.NoError(t, err)
	assert.Equal(t, 3, result.total, "should plan all 3 files")
	assert.Equal(t, 3, result.pending, "all should be pending")
}

func TestPlanMachine_ErrorsOnMissingMachineDir(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	_, err := planMachine(dotfiles, target, "nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveMachine_FromConfig(t *testing.T) {
	dotfiles := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dotfiles, "macbook"), 0o755))

	machine, err := resolveMachine(dotfiles, "macbook", "other-hostname")

	require.NoError(t, err)
	assert.Equal(t, "macbook", machine)
}

func TestResolveMachine_FallbackToHostname(t *testing.T) {
	dotfiles := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dotfiles, "my-laptop"), 0o755))

	machine, err := resolveMachine(dotfiles, "", "my-laptop")

	require.NoError(t, err)
	assert.Equal(t, "my-laptop", machine)
}

func TestResolveMachine_ErrorsWhenNoMatch(t *testing.T) {
	dotfiles := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dotfiles, "macbook"), 0o755))

	_, err := resolveMachine(dotfiles, "", "unknown-host")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown-host")
}

func TestPlanMachine_DryRunDoesNotCreateSymlinks(t *testing.T) {
	dotfiles := t.TempDir()
	target := t.TempDir()

	writeTestFile(t, filepath.Join(dotfiles, "macbook", ".gitconfig"), "[user]")
	writeTestFile(t, filepath.Join(dotfiles, "macbook", ".config", "fish", "config.fish"), "# fish")

	result, err := planMachine(dotfiles, target, "macbook")
	require.NoError(t, err)
	assert.Equal(t, 2, result.pending)

	// Key assertion: plan was created but NO symlinks exist in target
	_, err = os.Lstat(filepath.Join(target, ".gitconfig"))
	assert.True(t, os.IsNotExist(err), "dry-run should not create symlinks")
	_, err = os.Lstat(filepath.Join(target, ".config", "fish", "config.fish"))
	assert.True(t, os.IsNotExist(err), "dry-run should not create symlinks")
}

func TestFormatDryRun_ShowsEntries(t *testing.T) {
	entries := []stow.Entry{
		{Source: "/dotfiles/macbook/.gitconfig", Target: "/home/user/.gitconfig", Action: stow.Link},
		{Source: "/dotfiles/macbook/.zshrc", Target: "/home/user/.zshrc", Action: stow.Skip},
		{Source: "/dotfiles/macbook/.bashrc", Target: "/home/user/.bashrc", Action: stow.Conflict},
	}

	output := formatDryRun(entries)

	assert.Contains(t, output, ".gitconfig")
	assert.Contains(t, output, "link")
	assert.Contains(t, output, ".zshrc")
	assert.Contains(t, output, "skip")
	assert.Contains(t, output, ".bashrc")
	assert.Contains(t, output, "conflict")
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
