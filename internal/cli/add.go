package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/airbugg/kivtz/internal/adopter"
	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Adopt a single config into the dotfiles repo",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		pinfo, err := platform.Detect()
		if err != nil {
			return err
		}
		return runAdd(args[0], config.DefaultPath(pinfo.HomeDir), os.Stdout)
	},
}

// runAdd is the testable core of the add command.
func runAdd(sourcePath, configPath string, out io.Writer) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.DotfilesDir == "" {
		return fmt.Errorf("dotfiles directory not configured — run `kivtz init` first")
	}

	pkgName := adopter.PackageName(sourcePath)

	if err := adopter.Adopt(sourcePath, cfg.DotfilesDir); err != nil {
		return fmt.Errorf("adopting %s: %w", pkgName, err)
	}

	// Compute stats from adopted location
	pkgDir := filepath.Join(cfg.DotfilesDir, pkgName)
	fileCount, totalSize := packageStats(pkgDir)

	// Update packages list
	if !slices.Contains(cfg.Packages, pkgName) {
		cfg.Packages = append(cfg.Packages, pkgName)
	}
	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintf(out, "Adopted %s (%d files, %s)\n", pkgName, fileCount, formatSize(totalSize))
	return nil
}

// packageStats walks a directory and returns file count and total size.
func packageStats(dir string) (int, int64) {
	var count int
	var size int64
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		count++
		if info, err := d.Info(); err == nil {
			size += info.Size()
		}
		return nil
	})
	return count, size
}

// formatSize formats bytes into a human-readable string.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
