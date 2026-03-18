package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/airbugg/kivtz/internal/tui"
	"github.com/spf13/cobra"
)

var (
	initList bool
	initJSON bool
)

func init() {
	initCmd.Flags().BoolVar(&initList, "list", false, "output discovered configs as name<tab>path, one per line")
	initCmd.Flags().BoolVar(&initJSON, "json", false, "output discovered configs as a JSON array")
	initCmd.MarkFlagsMutuallyExclusive("list", "json")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [url]",
	Short: "Clone a dotfiles repo and set up this machine",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func runInit(_ *cobra.Command, args []string) error {
	if initList || initJSON {
		pinfo, err := platform.Detect()
		if err != nil {
			return err
		}
		mode := "list"
		if initJSON {
			mode = "json"
		}
		return runDiscovery(pinfo.HomeDir, os.Stdout, os.Stderr, mode)
	}

	pinfo, err := platform.Detect()
	if err != nil {
		return err
	}

	configPath := config.DefaultPath(pinfo.HomeDir)
	cfg, _ := config.Load(configPath)

	fmt.Println()
	fmt.Printf("  %s\n", bold.Render("kivtz init"))
	fmt.Printf("  %s %s/%s\n\n", dim.Render("detected:"), pinfo.OS, pinfo.Arch)

	// No URL arg and no existing repo → discovery flow
	if len(args) == 0 && cfg.RepoURL == "" {
		defaultDir := filepath.Join(pinfo.HomeDir, ".dotfiles")
		return runDiscoveryFlow(discoveryOpts{
			homeDir:     pinfo.HomeDir,
			dotfilesDir: defaultDir,
			configPath:  configPath,
			out:         os.Stdout,
			in:          os.Stdin,
			scan:        scanner.Scan,
			selectFn:    tui.RunSelector,
		})
	}

	// Clone flow (existing behavior)
	return runCloneFlow(pinfo, cfg, configPath, args)
}

func runCloneFlow(pinfo platform.Info, cfg config.Config, configPath string, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Determine repo URL
	var repoURL string
	if len(args) > 0 {
		repoURL = args[0]
	} else if cfg.RepoURL != "" {
		fmt.Printf("  repo URL [%s]: ", dim.Render(cfg.RepoURL))
		answer, _ := reader.ReadString('\n')
		if a := strings.TrimSpace(answer); a != "" {
			repoURL = a
		} else {
			repoURL = cfg.RepoURL
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
	if a := strings.TrimSpace(answer); a != "" {
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

// runDiscovery scans root for configs and writes results to stdout in the given mode ("list" or "json").
// Status messages go to stderr.
func runDiscovery(root string, stdout, stderr io.Writer, mode string) error {
	fmt.Fprintf(stderr, "Scanning %s for configs...\n", root)

	entries, err := scanner.Scan(root)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	fmt.Fprintf(stderr, "Found %d configs\n", len(entries))

	switch mode {
	case "json":
		return scanner.WriteJSON(stdout, entries)
	default:
		return scanner.WriteList(stdout, entries)
	}
}

