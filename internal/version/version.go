// Package version checks for available updates via the GitHub releases API.
package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"path/filepath"
	"time"
)

const ReleasesAPI = "https://api.github.com/repos/airbugg/kivtz/releases/latest"

var httpClient = &http.Client{Timeout: 10 * time.Second}

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type UpdateInfo struct {
	LatestVersion string
	Available     bool
	Assets        []Asset
}

type releaseResponse struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// CheckForUpdate queries apiURL for the latest release and compares against currentVersion.
// apiURL defaults to ReleasesAPI if empty.
func CheckForUpdate(currentVersion, apiURL string) (*UpdateInfo, error) {
	if apiURL == "" {
		apiURL = ReleasesAPI
	}

	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &UpdateInfo{}, nil
	}
	if resp.StatusCode != 200 {
		return nil, &httpError{StatusCode: resp.StatusCode}
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &UpdateInfo{
		LatestVersion: release.TagName,
		Available:     isUpdateAvailable(release.TagName, currentVersion),
		Assets:        release.Assets,
	}, nil
}

func isUpdateAvailable(latest, current string) bool {
	if latest == "" || latest == current {
		return false
	}

	return compareSemver(latest, current) > 0
}

// compareSemver returns >0 if a > b, <0 if a < b, 0 if equal.
// Expects "vMAJOR.MINOR.PATCH" format.
func compareSemver(a, b string) int {
	av := parseSemver(a)
	bv := parseSemver(b)

	for i := range av {
		if d := av[i] - bv[i]; d != 0 {
			return d
		}
	}

	return 0
}

func parseSemver(v string) [3]int {
	parts := strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3)
	var result [3]int

	for i, s := range parts {
		result[i], _ = strconv.Atoi(s)
	}

	return result
}

// FindAssetURL searches the UpdateInfo assets for the named asset and returns its download URL.
func FindAssetURL(info *UpdateInfo, assetName string) (string, error) {
	for _, a := range info.Assets {
		if a.Name == assetName {
			return a.URL, nil
		}
	}
	return "", fmt.Errorf("no binary %s in release %s", assetName, info.LatestVersion)
}

const cacheTTL = 24 * time.Hour
const cacheFileName = "update-check.json"

type CacheEntry struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// CachedCheck wraps CheckForUpdate with a 24-hour file cache.
// Only caches version availability — Assets will be nil on cache hits.
// Use CheckForUpdate directly when asset download URLs are needed.
// apiURL defaults to ReleasesAPI if empty.
func CachedCheck(currentVersion, cacheDir, apiURL string) (*UpdateInfo, error) {
	cachePath := filepath.Join(cacheDir, cacheFileName)

	if data, err := os.ReadFile(cachePath); err == nil {
		var entry CacheEntry
		if json.Unmarshal(data, &entry) == nil && time.Since(entry.CheckedAt) < cacheTTL {
			return &UpdateInfo{
				LatestVersion: entry.LatestVersion,
				Available:     isUpdateAvailable(entry.LatestVersion, currentVersion),
			}, nil
		}
	}

	info, err := CheckForUpdate(currentVersion, apiURL)
	if err != nil {
		return nil, err
	}

	entry := CacheEntry{LatestVersion: info.LatestVersion, CheckedAt: time.Now()}
	if data, err := json.Marshal(entry); err == nil {
		os.MkdirAll(cacheDir, 0o755)
		os.WriteFile(cachePath, data, 0o644)
	}

	return info, nil
}

// ClearCache removes the update check cache file.
func ClearCache(cacheDir string) error {
	err := os.Remove(filepath.Join(cacheDir, cacheFileName))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// PrintUpdateNotice checks for updates and prints a notice if one is available.
// Runs the check in a goroutine with a 2-second timeout.
// All errors are silently swallowed. No-op if KIVTZ_NO_UPDATE_CHECK=1 or version is "dev".
// Output is written to w.
func PrintUpdateNotice(currentVersion, cacheDir, apiURL string, w io.Writer) {
	if os.Getenv("KIVTZ_NO_UPDATE_CHECK") == "1" {
		return
	}
	if currentVersion == "dev" {
		return
	}

	type result struct {
		info *UpdateInfo
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		info, err := CachedCheck(currentVersion, cacheDir, apiURL)
		ch <- result{info, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil || res.info == nil || !res.info.Available {
			return
		}
		fmt.Fprintf(w, "\n  update available: %s (current: %s)\n", res.info.LatestVersion, currentVersion)
		fmt.Fprintf(w, "  run `kivtz self-update` to upgrade\n")
	case <-time.After(2 * time.Second):
		return
	}
}

type httpError struct {
	StatusCode int
}

func (e *httpError) Error() string {
	return "GitHub API returned " + http.StatusText(e.StatusCode)
}
