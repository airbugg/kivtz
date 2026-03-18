// Package cli implements the kivtz command-line interface.
package cli

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/airbugg/kivtz/internal/stow"
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
	dotfilesDir := resolveDotfilesDir(cfg, pinfo)

	fmt.Println()
	fmt.Printf("  %s\n", bold.Render("kivtzeynekuda"))
	fmt.Printf("  %s\n\n", dim.Render("הדָּקֻנְ יֵצְבָק"))

	fmt.Printf("  %s  %s/%s (%s)\n", dim.Render("platform"), pinfo.OS, pinfo.Arch, pinfo.Hostname)

	if isOnline() {
		fmt.Printf("  %s  %s\n", dim.Render("network"), success.Render("online"))
	} else {
		fmt.Printf("  %s  %s\n", dim.Render("network"), warning.Render("offline"))
	}

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
	result := planAll(pinfo, dotfilesDir, "")
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

func planAll(pinfo platform.Info, dotfilesDir, targetOverride string) planResult {
	targetDir := pinfo.HomeDir
	if targetOverride != "" {
		targetDir = targetOverride
	}

	var result planResult
	for _, group := range pinfo.Groups() {
		groupDir := filepath.Join(dotfilesDir, group)
		entries, err := os.ReadDir(groupDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			pkgEntries, err := stow.Plan(filepath.Join(groupDir, e.Name()), targetDir)
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
	}
	return result
}

func resolveDotfilesDir(cfg config.Config, _ platform.Info) string {
	if cfg.DotfilesDir != "" {
		return cfg.DotfilesDir
	}
	return ""
}

type repoStatus struct {
	clean   bool
	changed int
}

func gitRepoStatus(dir string) (repoStatus, error) {
	out, err := runGit(dir, "status", "--porcelain")
	if err != nil {
		return repoStatus{}, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return repoStatus{clean: true}, nil
	}
	return repoStatus{changed: len(lines)}, nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
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
