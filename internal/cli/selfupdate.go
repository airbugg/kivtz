package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/airbugg/kivtz/internal/config"
	"github.com/airbugg/kivtz/internal/platform"
	"github.com/airbugg/kivtz/internal/version"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
}

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Download and install the latest kivtz release",
	RunE:  runSelfUpdate,
}

func runSelfUpdate(_ *cobra.Command, _ []string) error {
	pinfo, err := platform.Detect()
	if err != nil {
		return err
	}

	binaryPath := filepath.Join(pinfo.HomeDir, ".local", "bin", "kivtz")
	cacheDir := filepath.Dir(config.DefaultPath(pinfo.HomeDir))

	fmt.Printf("\n  current: %s\n", buildVersion)

	info, err := version.CheckForUpdate(buildVersion, "")
	if err != nil {
		return fmt.Errorf("checking releases: %w", err)
	}

	if info.LatestVersion == "" {
		fmt.Printf("  %s\n\n", warning.Render("no releases yet — tag a version first"))
		return nil
	}

	fmt.Printf("  latest:  %s\n", info.LatestVersion)
	if !info.Available {
		fmt.Printf("  %s\n\n", success.Render("already up to date"))
		return nil
	}

	assetName := fmt.Sprintf("kivtz_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	downloadURL, err := version.FindAssetURL(info, assetName)
	if err != nil {
		return err
	}

	fmt.Printf("  %s\n", dim.Render("downloading "+assetName+"..."))
	if err := downloadAndReplace(downloadURL, binaryPath); err != nil {
		return err
	}

	if err := version.ClearCache(cacheDir); err != nil {
		fmt.Printf("  %s %v\n", warning.Render("cache:"), err)
	}
	fmt.Printf("  %s %s → %s\n\n", success.Render("updated:"), buildVersion, info.LatestVersion)
	return nil
}

func downloadAndReplace(url, dest string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.TrimPrefix(hdr.Name, "./") == "kivtz" {
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			tmp := dest + ".tmp"
			f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				os.Remove(tmp)
				return err
			}
			f.Close()
			if err := os.Rename(tmp, dest); err != nil {
				os.Remove(tmp)
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("kivtz binary not found in archive")
}
