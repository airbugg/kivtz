package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --yes flag is wired to initCmd
func TestInitCmd_hasYesFlag(t *testing.T) {
	flag := initCmd.Flags().Lookup("yes")
	require.NotNil(t, flag, "init command should have --yes flag")
	assert.Equal(t, "false", flag.DefValue)
}

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

// Clone flow uses command module and saves config
func TestCloneFlow_existingDirSavesConfig(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")
	repoURL := "https://github.com/test/dotfiles.git"

	// Pre-create dotfiles dir so clone is skipped
	require.NoError(t, os.MkdirAll(dotfilesDir, 0o755))

	var out bytes.Buffer
	opts := cloneOpts{
		repoURL:     repoURL,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		hostname:    "test-host",
		platform:    "darwin",
		out:         &out,
		in:          strings.NewReader("n\n"), // decline apply
	}

	err := runCloneFlowWithOpts(opts)
	require.NoError(t, err)

	// Config should be saved
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, dotfilesDir, cfg.DotfilesDir)
	assert.Equal(t, repoURL, cfg.RepoURL)

	// Should show "directory exists" message
	assert.Contains(t, out.String(), "directory exists")
}

// Clone flow shows transparent command string via command module
func TestCloneFlow_showsCloneCommand(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	// Use a local bare repo to avoid network calls.
	localRepo := filepath.Join(root, "repo")
	require.NoError(t, os.MkdirAll(localRepo, 0o755))
	// Init a bare git repo for cloning
	exec_cmd := exec.Command("git", "init", "--bare", localRepo)
	require.NoError(t, exec_cmd.Run())

	var out bytes.Buffer
	opts := cloneOpts{
		repoURL:     localRepo,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		hostname:    "test-host",
		platform:    "darwin",
		out:         &out,
		in:          strings.NewReader("n\n"),
	}

	err := runCloneFlowWithOpts(opts)
	require.NoError(t, err)

	// Output should show the git clone command transparently
	output := out.String()
	assert.Contains(t, output, "git clone")
	assert.Contains(t, output, localRepo)
}

func TestRunDiscovery_statusToStderr(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".config", "fish"), 0o755))

	var stdout, stderr bytes.Buffer
	err := runDiscovery(root, &stdout, &stderr, "list")

	require.NoError(t, err)
	assert.NotEmpty(t, stderr.String())
}
