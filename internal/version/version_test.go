package version_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/airbugg/kivtz/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeReleasesServer(tagName string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name": tagName,
			"assets": []map[string]string{
				{"name": "kivtz_darwin_arm64.tar.gz", "browser_download_url": "https://example.com/kivtz_darwin_arm64.tar.gz"},
			},
		})
	}))
}

func TestCheckForUpdate_Available(t *testing.T) {
	srv := fakeReleasesServer("v0.3.0")
	defer srv.Close()

	info, err := version.CheckForUpdate("v0.2.0", srv.URL)
	require.NoError(t, err)
	assert.True(t, info.Available)
	assert.Equal(t, "v0.3.0", info.LatestVersion)
}

func TestCheckForUpdate_UpToDate(t *testing.T) {
	srv := fakeReleasesServer("v0.2.0")
	defer srv.Close()

	info, err := version.CheckForUpdate("v0.2.0", srv.URL)
	require.NoError(t, err)
	assert.False(t, info.Available)
	assert.Equal(t, "v0.2.0", info.LatestVersion)
}

func TestCheckForUpdate_IncludesAssets(t *testing.T) {
	srv := fakeReleasesServer("v0.3.0")
	defer srv.Close()

	info, err := version.CheckForUpdate("v0.2.0", srv.URL)
	require.NoError(t, err)
	require.Len(t, info.Assets, 1)
	assert.Equal(t, "kivtz_darwin_arm64.tar.gz", info.Assets[0].Name)
}

func TestFindAssetURL_Found(t *testing.T) {
	info := &version.UpdateInfo{
		LatestVersion: "v0.3.0",
		Assets: []version.Asset{
			{Name: "kivtz_darwin_arm64.tar.gz", URL: "https://example.com/kivtz_darwin_arm64.tar.gz"},
			{Name: "kivtz_linux_amd64.tar.gz", URL: "https://example.com/kivtz_linux_amd64.tar.gz"},
		},
	}
	url, err := version.FindAssetURL(info, "kivtz_linux_amd64.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/kivtz_linux_amd64.tar.gz", url)
}

func TestFindAssetURL_NotFound(t *testing.T) {
	info := &version.UpdateInfo{LatestVersion: "v0.3.0"}
	_, err := version.FindAssetURL(info, "kivtz_windows_amd64.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no binary")
}

func TestCheckForUpdate_NoReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	info, err := version.CheckForUpdate("v0.2.0", srv.URL)
	require.NoError(t, err)
	assert.False(t, info.Available)
	assert.Empty(t, info.LatestVersion)
}

func TestCachedCheck_FreshCache_SkipsAPI(t *testing.T) {
	apiCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.3.0", "assets": []any{}})
	}))
	defer srv.Close()

	cacheDir := t.TempDir()

	// First call hits API
	info, err := version.CachedCheck("v0.2.0", cacheDir, srv.URL)
	require.NoError(t, err)
	assert.True(t, info.Available)
	assert.Equal(t, 1, apiCalls)

	// Second call uses cache
	info2, err := version.CachedCheck("v0.2.0", cacheDir, srv.URL)
	require.NoError(t, err)
	assert.True(t, info2.Available)
	assert.Equal(t, 1, apiCalls) // no additional API call
}

func TestCachedCheck_ExpiredCache_CallsAPI(t *testing.T) {
	srv := fakeReleasesServer("v0.3.0")
	defer srv.Close()

	cacheDir := t.TempDir()

	// Write an expired cache file (25 hours ago)
	cache := version.CacheEntry{
		LatestVersion: "v0.2.5",
		CheckedAt:     time.Now().Add(-25 * time.Hour),
	}
	data, _ := json.Marshal(cache)
	os.WriteFile(filepath.Join(cacheDir, "update-check.json"), data, 0o644)

	info, err := version.CachedCheck("v0.2.0", cacheDir, srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "v0.3.0", info.LatestVersion) // fresh from API, not cached v0.2.5
}

func TestPrintUpdateNotice_PrintsWhenAvailable(t *testing.T) {
	srv := fakeReleasesServer("v0.3.0")
	defer srv.Close()

	cacheDir := t.TempDir()
	var buf bytes.Buffer

	version.PrintUpdateNotice("v0.2.0", cacheDir, srv.URL, &buf)

	output := buf.String()
	assert.Contains(t, output, "v0.3.0")
	assert.Contains(t, output, "self-update")
}

func TestPrintUpdateNotice_SilentWhenUpToDate(t *testing.T) {
	srv := fakeReleasesServer("v0.2.0")
	defer srv.Close()

	cacheDir := t.TempDir()
	var buf bytes.Buffer

	version.PrintUpdateNotice("v0.2.0", cacheDir, srv.URL, &buf)

	assert.Empty(t, buf.String())
}

func TestPrintUpdateNotice_SilentOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	var buf bytes.Buffer

	version.PrintUpdateNotice("v0.2.0", cacheDir, srv.URL, &buf)

	assert.Empty(t, buf.String())
}

func TestPrintUpdateNotice_SilentWhenEnvSet(t *testing.T) {
	srv := fakeReleasesServer("v0.3.0")
	defer srv.Close()

	t.Setenv("KIVTZ_NO_UPDATE_CHECK", "1")
	cacheDir := t.TempDir()
	var buf bytes.Buffer

	version.PrintUpdateNotice("v0.2.0", cacheDir, srv.URL, &buf)

	assert.Empty(t, buf.String())
}

func TestPrintUpdateNotice_SilentForDevVersion(t *testing.T) {
	srv := fakeReleasesServer("v0.3.0")
	defer srv.Close()

	cacheDir := t.TempDir()
	var buf bytes.Buffer

	version.PrintUpdateNotice("dev", cacheDir, srv.URL, &buf)

	assert.Empty(t, buf.String())
}
