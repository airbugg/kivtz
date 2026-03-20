// Package cli implements the kivtz command-line interface.
package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/airbugg/kivtz/internal/command"
	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/airbugg/kivtz/internal/stow"
	"github.com/airbugg/kivtz/internal/version"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
	verbose      bool
)

// SetVersion sets the build metadata from ldflags.
func SetVersion(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date
}

// Styles — single source of truth for CLI output.
var (
	bold    = lipgloss.NewStyle().Bold(true)
	dim     = lipgloss.NewStyle().Faint(true)
	success = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warning = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

var rootCmd = &cobra.Command{
	Use:   "kivtz",
	Short: "הדָּקֻנְ יֵצְבָק — cross-platform dotfiles manager",
	RunE:  runStatus,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "debug logging")
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
		name := cmd.Name()
		if name == "self-update" || name == "version" {
			return
		}
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}
		cacheDir := filepath.Dir(config.DefaultPath(homeDir))
		version.PrintUpdateNotice(buildVersion, cacheDir, "", os.Stderr)
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run:   func(_ *cobra.Command, _ []string) { fmt.Printf("kivtz %s (%s) built %s\n", buildVersion, buildCommit, buildDate) },
}

// Execute runs the root command.
func Execute() error { return rootCmd.Execute() }

// runStatus is the default command — shows a dashboard.
func runStatus(_ *cobra.Command, _ []string) error {
	pinfo, err := platform.Detect()
	if err != nil {
		return err
	}

	cfg, _ := config.Load(config.DefaultPath(pinfo.HomeDir))
	dotfilesDir := cfg.DotfilesDir

	fmt.Println()
	fmt.Printf("  %s\n", bold.Render("kivtzeynekuda"))
	fmt.Printf("  %s\n\n", dim.Render("הדָּקֻנְ יֵצְבָק"))

	fmt.Printf("  %s  %s/%s (%s)\n", dim.Render("platform"), pinfo.OS, pinfo.Arch, pinfo.Hostname)

	if dotfilesDir == "" {
		fmt.Printf("\n  %s\n\n", dim.Render("run `kivtz init <url>` to set up your dotfiles"))
		return nil
	}

	// Repo state
	if gitStatus, err := gitRepoStatus(dotfilesDir); err == nil {
		if gitStatus.clean {
			fmt.Printf("  %s  %s\n", dim.Render("repo"), success.Render("clean"))
		} else {
			fmt.Printf("  %s  %s\n", dim.Render("repo"), warning.Render(fmt.Sprintf("%d changed", gitStatus.changed)))
		}
	}

	// Stow status
	result := planAll(dotfilesDir, pinfo.HomeDir, cfg.Packages)
	if result.total > 0 {
		fmt.Printf("  %s  %s", dim.Render("stow"), success.Render(fmt.Sprintf("%d linked", result.current)))
		if result.pending > 0 {
			fmt.Printf(", %s", infoStyle.Render(fmt.Sprintf("%d pending", result.pending)))
		}
		if result.conflicts > 0 {
			fmt.Printf(", %s", warning.Render(fmt.Sprintf("%d conflicts", result.conflicts)))
		}
		fmt.Println()
	}

	// Hints
	fmt.Println()
	if result.pending > 0 || result.conflicts > 0 {
		fmt.Printf("  %s\n", dim.Render("run `kivtz sync` to apply changes"))
	}
	fmt.Println()

	return nil
}

// --- shared helpers ---

type planResult struct {
	entries   []stow.Entry
	pending   int
	current   int
	conflicts int
	total     int
}

// planAll plans stow operations for packages in dotfilesDir. If packages is
// non-empty, only those packages are planned; otherwise all subdirectories
// of dotfilesDir are treated as packages (backward compatible).
func planAll(dotfilesDir, targetDir string, packages []string) planResult {
	if len(packages) == 0 {
		packages = discoverPackages(dotfilesDir)
	}

	var result planResult
	for _, pkg := range packages {
		pkgDir := filepath.Join(dotfilesDir, pkg)
		pkgEntries, err := stow.Plan(pkgDir, targetDir)
		if err != nil {
			continue
		}
		for _, pe := range pkgEntries {
			result.total++
			switch pe.Action {
			case stow.Link:
				result.pending++
			case stow.Skip:
				result.current++
			case stow.Conflict:
				result.conflicts++
			}
		}
		result.entries = append(result.entries, pkgEntries...)
	}
	return result
}

// discoverPackages returns all subdirectory names in dotfilesDir.
func discoverPackages(dotfilesDir string) []string {
	entries, err := os.ReadDir(dotfilesDir)
	if err != nil {
		return nil
	}
	var pkgs []string
	for _, e := range entries {
		if e.IsDir() {
			pkgs = append(pkgs, e.Name())
		}
	}
	return pkgs
}


// planMachine plans stow operations for a single machine directory.
// The machine dir is a direct mirror of $HOME — no package nesting.
func planMachine(dotfilesDir, targetDir, machine string) (planResult, error) {
	machineDir := filepath.Join(dotfilesDir, machine)

	if _, err := os.Stat(machineDir); os.IsNotExist(err) {
		return planResult{}, fmt.Errorf("machine directory %q not found in %s", machine, dotfilesDir)
	}

	entries, err := stow.Plan(machineDir, targetDir)
	if err != nil {
		return planResult{}, err
	}

	var result planResult
	result.entries = entries

	for _, e := range entries {
		result.total++

		switch e.Action {
		case stow.Link:
			result.pending++
		case stow.Skip:
			result.current++
		case stow.Conflict:
			result.conflicts++
		}
	}

	return result, nil
}

// resolveMachine determines which machine directory to use.
// Priority: config value > hostname match > error.
func resolveMachine(dotfilesDir, configMachine, hostname string) (string, error) {
	if configMachine != "" {
		machineDir := filepath.Join(dotfilesDir, configMachine)

		if _, err := os.Stat(machineDir); os.IsNotExist(err) {
			return "", fmt.Errorf("configured machine %q not found in %s", configMachine, dotfilesDir)
		}

		return configMachine, nil
	}

	machineDir := filepath.Join(dotfilesDir, hostname)

	if _, err := os.Stat(machineDir); err == nil {
		return hostname, nil
	}

	return "", fmt.Errorf("no machine directory matches hostname %q — set machine in config.toml", hostname)
}

// formatDryRun produces a human-readable summary of a stow plan.
func formatDryRun(entries []stow.Entry) string {
	var b strings.Builder

	for _, e := range entries {
		target := shortPath(e.Target)
		var action string

		switch e.Action {
		case stow.Link:
			action = "link"
		case stow.Skip:
			action = "skip"
		case stow.Conflict:
			action = "conflict"
		}

		fmt.Fprintf(&b, "  %-10s %s\n", action, target)
	}

	return b.String()
}

type repoStatus struct {
	clean   bool
	changed int
}

func gitRepoStatus(dir string) (repoStatus, error) {
	out, err := command.New("git", "status", "--porcelain").Dir(dir).Run()
	if err != nil {
		return repoStatus{}, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return repoStatus{clean: true}, nil
	}
	return repoStatus{changed: len(lines)}, nil
}

func isOnline() bool {
	conn, err := net.DialTimeout("tcp", "github.com:443", 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func isYes(answer string) bool {
	a := strings.TrimSpace(answer)
	return a == "" || a == "y" || a == "Y"
}
