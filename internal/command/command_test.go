package command_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/airbugg/kivtz/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestString_RendersFullCommand(t *testing.T) {
	cmd := command.New("git", "clone", "https://example.com", "/tmp/repo")
	assert.Equal(t, "git clone https://example.com /tmp/repo", cmd.String())
}

func TestString_WithDir(t *testing.T) {
	cmd := command.New("git", "status").Dir("/some/path")
	assert.Equal(t, "git status", cmd.String())
}

func TestRun_SuccessfulCommand(t *testing.T) {
	cmd := command.New("echo", "hello")
	out, err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, "hello\n", out)
}

func TestRun_ReturnsEnrichedErrorOnFailure(t *testing.T) {
	cmd := command.New("git", "clone", "https://invalid.example.com/nope.git", t.TempDir())
	_, err := cmd.Run()
	require.Error(t, err)

	var cmdErr *command.Error
	require.True(t, errors.As(err, &cmdErr))
	assert.Contains(t, cmdErr.Command, "git clone")
	assert.NotEmpty(t, cmdErr.Stderr)
}

func TestRun_UsesDir(t *testing.T) {
	dir := t.TempDir()
	cmd := command.New("pwd").Dir(dir)
	out, err := cmd.Run()
	require.NoError(t, err)
	assert.Equal(t, dir, strings.TrimSpace(out))
}

func TestDryRun_PrintsWithoutExecuting(t *testing.T) {
	var buf bytes.Buffer
	cmd := command.New("rm", "-rf", "/").Output(&buf)
	cmd.DryRun()
	assert.Contains(t, buf.String(), "rm -rf /")
}

func TestPrompt_YesExecutes(t *testing.T) {
	input := strings.NewReader("Y\n")
	var output bytes.Buffer
	cmd := command.New("echo", "ran").Input(input).Output(&output)
	out, err := cmd.Prompt()
	require.NoError(t, err)
	assert.Equal(t, "ran\n", out)
}

func TestPrompt_NoSkips(t *testing.T) {
	input := strings.NewReader("n\n")
	var output bytes.Buffer
	cmd := command.New("echo", "should-not-run").Input(input).Output(&output)
	out, err := cmd.Prompt()
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestPrompt_ManualShowsCommand(t *testing.T) {
	input := strings.NewReader("m\n")
	var output bytes.Buffer
	cmd := command.New("git", "push").Input(input).Output(&output)
	out, err := cmd.Prompt()
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Contains(t, output.String(), "git push")
}

func TestPrompt_DefaultIsYes(t *testing.T) {
	input := strings.NewReader("\n")
	var output bytes.Buffer
	cmd := command.New("echo", "default").Input(input).Output(&output)
	out, err := cmd.Prompt()
	require.NoError(t, err)
	assert.Equal(t, "default\n", out)
}
