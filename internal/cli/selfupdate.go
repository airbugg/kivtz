package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/airbugg/kivtz/internal/platform"
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

const releasesAPI = "https://api.github.com/repos/airbugg/kivtz/releases/latest"

func runSelfUpdate(_ *cobra.Command, _ []string) error {
	pinfo, err := platform.Detect()
	if err != nil {
		return err
	}

	binaryPath := filepath.Join(pinfo.HomeDir, ".local", "bin", "kivtz")

	fmt.Printf("\n  current: %s\n", buildVersion)

	resp, err := http.Get(releasesAPI)
	if err != nil {
		return fmt.Errorf("checking releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Printf("  %s\n\n", warning.Render("no releases yet — tag a version first"))
		return nil
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}

	fmt.Printf("  latest:  %s\n", release.TagName)
	if release.TagName == buildVersion {
		fmt.Printf("  %s\n\n", success.Render("already up to date"))
		return nil
	}

	assetName := fmt.Sprintf("kivtz_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.URL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no binary for %s/%s in %s", runtime.GOOS, runtime.GOARCH, release.TagName)
	}

	fmt.Printf("  %s\n", dim.Render("downloading "+assetName+"..."))
	if err := downloadAndReplace(downloadURL, binaryPath); err != nil {
		return err
	}

	fmt.Printf("  %s %s → %s\n\n", success.Render("updated:"), buildVersion, release.TagName)
	return nil
}

func downloadAndReplace(url, dest string) error {
	resp, err := http.Get(url)
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
