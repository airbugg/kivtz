package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [url]",
	Short: "Clone a dotfiles repo and set up this machine",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func runInit(_ *cobra.Command, args []string) error {
	pinfo, err := platform.Detect()
	if err != nil {
		return err
	}

	configPath := config.DefaultPath(pinfo.HomeDir)
	cfg, _ := config.Load(configPath)

	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Printf("  %s\n", bold.Render("kivtz init"))
	fmt.Printf("  %s %s/%s\n\n", dim.Render("detected:"), pinfo.OS, pinfo.Arch)

	// Determine repo URL
	var repoURL string
	if len(args) > 0 {
		repoURL = args[0]
	} else if cfg.RepoURL != "" {
		fmt.Printf("  repo URL [%s]: ", dim.Render(cfg.RepoURL))
		answer, _ := reader.ReadString('\n')
		if a := trimLine(answer); a != "" {
			repoURL = a
		} else {
			repoURL = cfg.RepoURL
		}
	} else {
		fmt.Printf("  repo URL: ")
		answer, _ := reader.ReadString('\n')
		repoURL = trimLine(answer)
		if repoURL == "" {
			return fmt.Errorf("repo URL is required")
		}
	}

	// Determine clone path
	defaultPath := cfg.DotfilesDir
	if defaultPath == "" {
		defaultPath = pinfo.HomeDir + "/.dotfiles"
	}
	fmt.Printf("  clone to [%s]: ", dim.Render(defaultPath))
	answer, _ := reader.ReadString('\n')
	dotfilesDir := defaultPath
	if a := trimLine(answer); a != "" {
		dotfilesDir = a
	}

	// Clone if needed
	if _, err := os.Stat(dotfilesDir); os.IsNotExist(err) {
		fmt.Printf("  %s\n", dim.Render("cloning..."))
		if _, err := runGit("", "clone", repoURL, dotfilesDir); err != nil {
			return fmt.Errorf("clone failed: %w", err)
		}
		fmt.Printf("  %s\n", success.Render("cloned"))
	} else {
		fmt.Printf("  %s\n", dim.Render("directory exists, skipping clone"))
	}

	// Save config
	cfg.DotfilesDir = dotfilesDir
	cfg.RepoURL = repoURL
	cfg.Platform = pinfo.OS.String()
	cfg.Hostname = pinfo.Hostname
	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("  %s %s\n", success.Render("config saved:"), configPath)

	// Offer to apply
	fmt.Printf("\n  apply configs now? [Y/n] ")
	answer, _ = reader.ReadString('\n')
	if isYes(answer) {
		return runSync(nil, nil)
	}

	fmt.Println()
	return nil
}

func trimLine(s string) string {
	s = s[:max(0, len(s)-1)] // strip trailing newline
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
