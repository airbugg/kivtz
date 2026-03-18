package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/airbugg/kivtz/internal/adopter"
	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/scanner"
)

// discoveryOpts holds injectable dependencies for the discovery-first init flow.
type discoveryOpts struct {
	homeDir     string
	dotfilesDir string
	configPath  string
	out         io.Writer
	in          io.Reader
	scan        func(root string) ([]scanner.Entry, error)
	selectFn    func(entries, preSelected []scanner.Entry) ([]scanner.Entry, error)
}

// runDiscoveryFlow orchestrates: scan → score → TUI select → summary → confirm → adopt → config.
func runDiscoveryFlow(opts discoveryOpts) error {
	// Scan
	entries, err := opts.scan(opts.homeDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(opts.out, "  No configs found to manage.")
		return nil
	}

	fmt.Fprintf(opts.out, "  Found %d configs\n\n", len(entries))

	// Score and pre-select
	preSelected := scanner.PreSelected(entries, 4)

	// TUI selection
	selected, err := opts.selectFn(entries, preSelected)
	if err != nil {
		return fmt.Errorf("selection failed: %w", err)
	}
	if len(selected) == 0 {
		fmt.Fprintln(opts.out, "  No configs selected.")
		return nil
	}

	// Summary
	fmt.Fprintf(opts.out, "\n  Moving %d configs into %s\n", len(selected), opts.dotfilesDir)
	for _, e := range selected {
		fmt.Fprintf(opts.out, "    %s %s\n", adopter.PackageName(e.Path), dim.Render(e.Path))
	}

	// Confirm
	fmt.Fprintf(opts.out, "\n  Proceed? [Y/n] ")
	reader := bufio.NewReader(opts.in)
	answer, _ := reader.ReadString('\n')
	if !isYes(answer) {
		fmt.Fprintln(opts.out, "  Aborted.")
		return nil
	}

	// Ensure dotfiles dir exists
	if err := os.MkdirAll(opts.dotfilesDir, 0o755); err != nil {
		return fmt.Errorf("creating dotfiles dir: %w", err)
	}

	// Adopt each selected entry
	var adopted []string
	for _, e := range selected {
		pkgName := adopter.PackageName(e.Path)
		if err := adopter.Adopt(e.Path, opts.dotfilesDir); err != nil {
			fmt.Fprintf(opts.out, "  %s skipping %s: %s\n", warning.Render("warning:"), pkgName, err)
			continue
		}
		adopted = append(adopted, pkgName)
		fmt.Fprintf(opts.out, "  %s %s\n", success.Render("adopted"), pkgName)
	}

	if len(adopted) == 0 {
		fmt.Fprintln(opts.out, "  No configs were adopted.")
		return nil
	}

	// Write config
	cfg, _ := config.Load(opts.configPath)
	cfg.DotfilesDir = opts.dotfilesDir
	for _, pkg := range adopted {
		if !slices.Contains(cfg.Packages, pkg) {
			cfg.Packages = append(cfg.Packages, pkg)
		}
	}
	if err := config.Save(cfg, opts.configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Fprintf(opts.out, "\n  %s %s\n", success.Render("config saved:"), opts.configPath)

	return nil
}
