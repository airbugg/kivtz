// Package command provides a transparent execution engine for external commands.
// Every command is shown as a copyable string before execution, with enriched
// error messages and dry-run support.
package command

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Error is an enriched error returned when a command fails.
type Error struct {
	Command string
	Stderr  string
	Err     error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Command, e.Stderr)
}

func (e *Error) Unwrap() error { return e.Err }

// Cmd represents an external command with builder-pattern configuration.
type Cmd struct {
	name   string
	args   []string
	dir    string
	input  io.Reader
	output io.Writer
}

// New creates a command from a program name and arguments.
func New(name string, args ...string) *Cmd {
	return &Cmd{
		name:   name,
		args:   args,
		output: os.Stdout,
		input:  os.Stdin,
	}
}

// Dir sets the working directory for the command.
func (c *Cmd) Dir(dir string) *Cmd {
	c.dir = dir
	return c
}

// Input sets the reader for interactive prompt input.
func (c *Cmd) Input(r io.Reader) *Cmd {
	c.input = r
	return c
}

// Output sets the writer for dry-run and prompt output.
func (c *Cmd) Output(w io.Writer) *Cmd {
	c.output = w
	return c
}

// String returns the full command as a copyable string.
func (c *Cmd) String() string {
	parts := make([]string, 0, 1+len(c.args))
	parts = append(parts, c.name)
	parts = append(parts, c.args...)
	return strings.Join(parts, " ")
}

// Run executes the command and returns stdout. On failure, returns an enriched Error.
func (c *Cmd) Run() (string, error) {
	cmd := exec.Command(c.name, c.args...)
	if c.dir != "" {
		cmd.Dir = c.dir
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", &Error{
			Command: c.String(),
			Stderr:  strings.TrimSpace(stderr.String()),
			Err:     err,
		}
	}
	return stdout.String(), nil
}

// DryRun prints the command without executing it.
func (c *Cmd) DryRun() {
	fmt.Fprintf(c.output, "  → %s\n", c.String())
}

// Prompt asks the user whether to run the command, then acts accordingly.
// Returns (stdout, error). "Y" or empty runs the command, "n" skips,
// "m" (manual / "I'll do it myself") shows the command and moves on.
func (c *Cmd) Prompt() (string, error) {
	fmt.Fprintf(c.output, "  → %s\n", c.String())
	fmt.Fprintf(c.output, "  Run this? [Y/n/m(anual)] ")

	scanner := bufio.NewScanner(c.input)
	if !scanner.Scan() {
		return "", nil
	}
	answer := strings.TrimSpace(scanner.Text())

	switch strings.ToLower(answer) {
	case "", "y":
		return c.Run()
	case "n":
		return "", nil
	default: // "m" or anything else = manual
		fmt.Fprintf(c.output, "  Run manually: %s\n", c.String())
		return "", nil
	}
}
