package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
