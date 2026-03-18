package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/airbugg/kivtz/internal/command"
)

// gitSetupCommands returns the sequence of commands for git/GitHub setup after adoption.
// Each command is configured with the given output writer and input reader for Prompt().
func gitSetupCommands(dotfilesDir string, out io.Writer, in io.Reader) []*command.Cmd {
	repoName := filepath.Base(dotfilesDir)
	cmd := func(name string, args ...string) *command.Cmd {
		return command.New(name, args...).Dir(dotfilesDir).Output(out).Input(in)
	}
	return []*command.Cmd{
		cmd("git", "init"),
		cmd("gh", "repo", "create", repoName, "--private", "--source", dotfilesDir, "--push"),
		cmd("git", "add", "."),
		cmd("git", "commit", "-m", "Initial dotfiles commit"),
	}
}

// runGitSetup offers git init, GitHub repo creation, and initial commit via interactive prompts.
func runGitSetup(dotfilesDir string, out io.Writer, in io.Reader) error {
	fmt.Fprintf(out, "\n  %s\n\n", bold.Render("Set up git"))

	cmds := gitSetupCommands(dotfilesDir, out, in)
	for _, c := range cmds {
		if _, err := c.Prompt(); err != nil {
			fmt.Fprintf(out, "  %s %s\n", warning.Render("warning:"), err)
		}
	}
	return nil
}

// runGitSetupAuto runs git setup commands without prompts (--yes mode).
func runGitSetupAuto(dotfilesDir string, out io.Writer) error {
	fmt.Fprintf(out, "\n  %s\n\n", bold.Render("Set up git"))

	cmds := gitSetupCommands(dotfilesDir, out, nil)
	for _, c := range cmds {
		if _, err := c.Run(); err != nil {
			fmt.Fprintf(out, "  %s %s\n", warning.Render("warning:"), err)
		}
	}
	return nil
}
