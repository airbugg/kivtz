package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fishEntry(root string) scanner.Entry {
	return scanner.Entry{
		Name:      "fish",
		Path:      filepath.Join(root, ".config", "fish"),
		Size:      512,
		ModTime:   time.Now(),
		FileCount: 3,
		IsDir:     true,
	}
}

func gitEntry(root string) scanner.Entry {
	return scanner.Entry{
		Name:      ".gitconfig",
		Path:      filepath.Join(root, ".gitconfig"),
		Size:      128,
		ModTime:   time.Now(),
		FileCount: 1,
		IsDir:     false,
	}
}

// Tracer bullet: selected entries get adopted and config is written
func TestDiscoveryFlow_adoptsSelectedAndWritesConfig(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	// Create real source files for adoption
	fishDir := filepath.Join(root, ".config", "fish")
	require.NoError(t, os.MkdirAll(fishDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "config.fish"), []byte("# fish"), 0o644))

	fish := fishEntry(root)

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader("y\n"), // confirm adoption
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fish}, nil
		},
		selectFn: func(entries, _ []scanner.Entry) ([]scanner.Entry, error) {
			return entries, nil // select all
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	// Verify fish was adopted (symlink exists at original location)
	info, err := os.Lstat(fishDir)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "fish should be symlinked")

	// Verify config was written with packages
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Contains(t, cfg.Packages, "fish")
	assert.Equal(t, dotfilesDir, cfg.DotfilesDir)
}

// Summary shows correct counts before adoption
func TestDiscoveryFlow_showsSummary(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	fishDir := filepath.Join(root, ".config", "fish")
	require.NoError(t, os.MkdirAll(fishDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "config.fish"), []byte("# fish"), 0o644))

	gitFile := filepath.Join(root, ".gitconfig")
	require.NoError(t, os.WriteFile(gitFile, []byte("[user]"), 0o644))

	entries := []scanner.Entry{fishEntry(root), gitEntry(root)}

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader("y\n"),
		scan: func(_ string) ([]scanner.Entry, error) {
			return entries, nil
		},
		selectFn: func(e, _ []scanner.Entry) ([]scanner.Entry, error) {
			return e, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "2 configs")
	assert.Contains(t, output, dotfilesDir)
}

// User says "n" at confirmation — nothing adopted
func TestDiscoveryFlow_abortOnNo(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	fishDir := filepath.Join(root, ".config", "fish")
	require.NoError(t, os.MkdirAll(fishDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "config.fish"), []byte("# fish"), 0o644))

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader("n\n"), // decline
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fishEntry(root)}, nil
		},
		selectFn: func(e, _ []scanner.Entry) ([]scanner.Entry, error) {
			return e, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	// Fish should NOT be a symlink — still original dir
	info, err := os.Lstat(fishDir)
	require.NoError(t, err)
	assert.False(t, info.Mode()&os.ModeSymlink != 0, "fish should not be adopted")

	// Config should not exist
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))
}

// Empty selection (user quits TUI) — aborts gracefully
func TestDiscoveryFlow_emptySelectionAborts(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader(""),
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fishEntry(root)}, nil
		},
		selectFn: func(_, _ []scanner.Entry) ([]scanner.Entry, error) {
			return nil, nil // user quit TUI
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "No configs selected")
}

// No configs found during scan
func TestDiscoveryFlow_noConfigsFound(t *testing.T) {
	root := t.TempDir()

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: filepath.Join(root, ".dotfiles"),
		configPath:  filepath.Join(root, ".config", "kivtz", "config.toml"),
		out:         &out,
		in:          strings.NewReader(""),
		scan: func(_ string) ([]scanner.Entry, error) {
			return nil, nil
		},
		selectFn: func(_, _ []scanner.Entry) ([]scanner.Entry, error) {
			return nil, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	assert.Contains(t, out.String(), "No configs found")
}

// Config written with dotfiles dir and platform info
func TestDiscoveryFlow_writesConfigWithDotfilesDir(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, "my-dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	gitFile := filepath.Join(root, ".gitconfig")
	require.NoError(t, os.WriteFile(gitFile, []byte("[user]"), 0o644))

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader("y\n"),
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{gitEntry(root)}, nil
		},
		selectFn: func(e, _ []scanner.Entry) ([]scanner.Entry, error) {
			return e, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, dotfilesDir, cfg.DotfilesDir)
	assert.Contains(t, cfg.Packages, "git")
}

// Re-scan: when all configs already managed, shows "no configs found"
func TestDiscoveryFlow_rescanAllManagedShowsNone(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	// Both fish and git already managed
	require.NoError(t, os.MkdirAll(filepath.Join(dotfilesDir, "fish"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dotfilesDir, "git"), 0o755))

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader(""),
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fishEntry(root), gitEntry(root)}, nil
		},
		selectFn: func(_, _ []scanner.Entry) ([]scanner.Entry, error) {
			t.Fatal("selectFn should not be called when all configs managed")
			return nil, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "No configs found")
}

// --yes flag: uses pre-selected, skips TUI and confirmation
func TestDiscoveryFlow_yesFlagUsesPreSelected(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	// Create real source files
	fishDir := filepath.Join(root, ".config", "fish")
	require.NoError(t, os.MkdirAll(fishDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "config.fish"), []byte("# fish"), 0o644))

	gitFile := filepath.Join(root, ".gitconfig")
	require.NoError(t, os.WriteFile(gitFile, []byte("[user]"), 0o644))

	fish := fishEntry(root)
	git := gitEntry(root)

	selectCalled := false
	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader(""), // no input — --yes means no prompts
		yes:         true,
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fish, git}, nil
		},
		selectFn: func(_, _ []scanner.Entry) ([]scanner.Entry, error) {
			selectCalled = true
			return nil, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	// TUI should NOT be called
	assert.False(t, selectCalled, "selectFn should not be called with --yes")

	// Config should be written with adopted packages
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Packages)
}

// Re-scan: already-managed packages are filtered out
func TestDiscoveryFlow_rescanFiltersManaged(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	// Create nvim source for adoption
	nvimDir := filepath.Join(root, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- nvim"), 0o644))

	// Simulate fish already managed: exists in dotfiles dir
	require.NoError(t, os.MkdirAll(filepath.Join(dotfilesDir, "fish"), 0o755))

	fish := fishEntry(root)
	nvim := scanner.Entry{
		Name:      "nvim",
		Path:      nvimDir,
		Size:      256,
		ModTime:   fish.ModTime,
		FileCount: 1,
		IsDir:     true,
	}

	var selectCalled []scanner.Entry
	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader("y\n"),
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fish, nvim}, nil
		},
		selectFn: func(entries, _ []scanner.Entry) ([]scanner.Entry, error) {
			selectCalled = entries
			return entries, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	// Only nvim should be presented to TUI (fish is already managed)
	require.Len(t, selectCalled, 1)
	assert.Equal(t, "nvim", selectCalled[0].Name)
}

// Adoption failure for one entry doesn't stop others
func TestDiscoveryFlow_partialAdoptionContinues(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	// Only create gitconfig, not fish — fish adoption will fail
	gitFile := filepath.Join(root, ".gitconfig")
	require.NoError(t, os.WriteFile(gitFile, []byte("[user]"), 0o644))

	fish := fishEntry(root) // path doesn't exist on disk
	git := gitEntry(root)

	var out bytes.Buffer
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader("y\n"),
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fish, git}, nil
		},
		selectFn: func(e, _ []scanner.Entry) ([]scanner.Entry, error) {
			return e, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Contains(t, cfg.Packages, "git")
	// fish should not be in packages since adoption failed
	assert.NotContains(t, cfg.Packages, "fish")
}
