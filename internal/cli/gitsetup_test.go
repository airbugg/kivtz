package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tracer bullet: git init command is created with correct dir and uses Prompt
func TestGitSetup_gitInitCommand(t *testing.T) {
	var out bytes.Buffer
	cmds := gitSetupCommands("/tmp/dotfiles", &out, strings.NewReader("n\nn\n"))

	assert.GreaterOrEqual(t, len(cmds), 1)
	assert.Equal(t, "git init", cmds[0].String())
}

// gh repo create command is included
func TestGitSetup_ghRepoCreateCommand(t *testing.T) {
	var out bytes.Buffer
	cmds := gitSetupCommands("/tmp/dotfiles", &out, strings.NewReader(""))

	require.GreaterOrEqual(t, len(cmds), 2)
	assert.Equal(t, "gh repo create dotfiles --private --source /tmp/dotfiles --push", cmds[1].String())
}

// git add, commit, push commands are included
func TestGitSetup_gitCommitAndPushCommands(t *testing.T) {
	var out bytes.Buffer
	cmds := gitSetupCommands("/tmp/dotfiles", &out, strings.NewReader(""))

	require.GreaterOrEqual(t, len(cmds), 4)
	assert.Equal(t, "git add .", cmds[2].String())
	assert.Equal(t, "git commit -m Initial dotfiles commit", cmds[3].String())
}

// runGitSetup prompts all commands and shows each in output
func TestRunGitSetup_promptsAllCommands(t *testing.T) {
	var out bytes.Buffer
	// "n" to skip each command (4 commands)
	in := strings.NewReader("n\nn\nn\nn\n")

	err := runGitSetup("/tmp/dotfiles", &out, in)
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "git init")
	assert.Contains(t, output, "gh repo create")
	assert.Contains(t, output, "git add .")
	assert.Contains(t, output, "git commit")
}

// runGitSetup shows header before commands
func TestRunGitSetup_showsHeader(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("n\nn\nn\nn\n")

	err := runGitSetup("/tmp/dotfiles", &out, in)
	require.NoError(t, err)

	assert.Contains(t, out.String(), "git")
}

// Discovery flow offers git setup after adoption
func TestDiscoveryFlow_offersGitSetupAfterAdoption(t *testing.T) {
	root := t.TempDir()
	dotfilesDir := filepath.Join(root, ".dotfiles")
	configPath := filepath.Join(root, ".config", "kivtz", "config.toml")

	// Create real source files for adoption
	fishDir := filepath.Join(root, ".config", "fish")
	require.NoError(t, os.MkdirAll(fishDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fishDir, "config.fish"), []byte("# fish"), 0o644))

	fish := fishEntry(root)

	var out bytes.Buffer
	// "y" for adoption confirm, then "n" for all 4 git commands
	opts := discoveryOpts{
		homeDir:     root,
		dotfilesDir: dotfilesDir,
		configPath:  configPath,
		out:         &out,
		in:          strings.NewReader("y\nn\nn\nn\nn\n"),
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fish}, nil
		},
		selectFn: func(entries, _ []scanner.Entry) ([]scanner.Entry, error) {
			return entries, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	output := out.String()
	// Should show git setup prompts after adoption
	assert.Contains(t, output, "git init")
	assert.Contains(t, output, "gh repo create")
}

// Discovery flow does NOT offer git setup when user aborts adoption
func TestDiscoveryFlow_noGitSetupOnAbort(t *testing.T) {
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
		in:          strings.NewReader("n\n"), // decline adoption
		scan: func(_ string) ([]scanner.Entry, error) {
			return []scanner.Entry{fishEntry(root)}, nil
		},
		selectFn: func(e, _ []scanner.Entry) ([]scanner.Entry, error) {
			return e, nil
		},
	}

	err := runDiscoveryFlow(opts)
	require.NoError(t, err)

	output := out.String()
	assert.NotContains(t, output, "git init")
}
