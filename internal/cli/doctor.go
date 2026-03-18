package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and diagnose issues",
	RunE:  runDoctor,
}

func runDoctor(_ *cobra.Command, _ []string) error {
	pinfo, err := platform.Detect()
	if err != nil {
		return err
	}

	cfg, _ := config.Load(config.DefaultPath(pinfo.HomeDir))
	dotfilesDir := resolveDotfilesDir(cfg, pinfo)

	fmt.Println()

	check("platform", "pass", fmt.Sprintf("%s/%s (%s)", pinfo.OS, pinfo.Arch, pinfo.Hostname))

	// Tools
	for _, tool := range []struct{ name, cmd string }{
		{"git", "git"}, {"fish", "fish"},
	} {
		if out := versionOf(tool.cmd); out != "" {
			check(tool.name, "pass", out)
		} else {
			check(tool.name, "warn", "not installed")
		}
	}

	// Dotfiles
	if dotfilesDir == "" {
		check("dotfiles", "warn", "not configured — run `kivtz init <url>`")
	} else if s, err := gitRepoStatus(dotfilesDir); err != nil {
		check("dotfiles", "fail", "not found")
	} else if s.clean {
		check("dotfiles", "pass", "clean")
	} else {
		check("dotfiles", "warn", fmt.Sprintf("%d uncommitted changes", s.changed))
	}

	// Stow
	if dotfilesDir != "" {
		result := planAll(pinfo, dotfilesDir, "")
		detail := fmt.Sprintf("%d/%d linked", result.current, result.total)
		if result.pending > 0 || result.conflicts > 0 {
			check("stow", "warn", fmt.Sprintf("%s, %d pending, %d conflicts", detail, result.pending, result.conflicts))
		} else {
			check("stow", "pass", detail)
		}
	}

	// Credentials (macOS)
	if pinfo.OS == platform.Darwin {
		sock := filepath.Join(pinfo.HomeDir, "Library", "Group Containers", "2BUA8C4S2C.com.1password", "t", "agent.sock")
		if _, err := os.Stat(sock); err == nil {
			check("1password", "pass", "SSH agent connected")
		} else {
			check("1password", "warn", "SSH agent not found")
		}
	}

	// Key tools
	for _, tool := range []string{"starship", "zoxide", "fzf"} {
		if platform.HasCommand(tool) {
			check(tool, "pass", "installed")
		} else {
			check(tool, "warn", "not installed")
		}
	}

	fmt.Println()
	return nil
}

func check(name, status, detail string) {
	var icon string
	switch status {
	case "pass":
		icon = success.Render("[pass]")
	case "warn":
		icon = warning.Render("[warn]")
	case "fail":
		icon = errStyle.Render("[fail]")
	}
	fmt.Printf("  %s %s: %s\n", icon, name, detail)
}

func versionOf(cmd string) string {
	path, err := exec.LookPath(cmd)
	if err != nil {
		return ""
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
