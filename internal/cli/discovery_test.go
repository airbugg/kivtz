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
