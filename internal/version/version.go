// Package version checks for available updates via the GitHub releases API.
package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
		Available:     release.TagName != "" && release.TagName != currentVersion,
		Assets:        release.Assets,
	}, nil
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
// apiURL defaults to ReleasesAPI if empty.
func CachedCheck(currentVersion, cacheDir, apiURL string) (*UpdateInfo, error) {
	cachePath := filepath.Join(cacheDir, cacheFileName)

	if data, err := os.ReadFile(cachePath); err == nil {
		var entry CacheEntry
		if json.Unmarshal(data, &entry) == nil && time.Since(entry.CheckedAt) < cacheTTL {
			return &UpdateInfo{
				LatestVersion: entry.LatestVersion,
				Available:     entry.LatestVersion != "" && entry.LatestVersion != currentVersion,
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
func ClearCache(cacheDir string) {
	os.Remove(filepath.Join(cacheDir, cacheFileName))
}

type httpError struct {
	StatusCode int
}

func (e *httpError) Error() string {
	return "GitHub API returned " + http.StatusText(e.StatusCode)
}
