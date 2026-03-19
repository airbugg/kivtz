# Auto Release Pipeline + Version Check — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automate releases on merge to main, add PR preview builds, and notify users of available updates.

**Architecture:** GitHub Actions workflows handle CI/CD (release-please for versioning, GoReleaser for builds, semantic PR linting). A new `internal/version` package handles update checks with 24h caching and 2s timeout. `selfupdate.go` is refactored to reuse the version package.

**Tech Stack:** GitHub Actions, release-please, GoReleaser v2, `action-semantic-pull-request`, Go stdlib (`net/http`, `encoding/json`, `os`, `time`)

**Spec:** `docs/superpowers/specs/2026-03-19-auto-release-pipeline-design.md`

---

## Task 1: Fix Go version in existing workflows

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Update ci.yml to use go-version-file**

Replace the hardcoded `go-version: "1.24"` with `go-version-file: go.mod` in `.github/workflows/ci.yml`:

```yaml
name: CI
on:
  pull_request:
  push:
    branches: [main]
jobs:
  test:
    strategy:
      matrix:
        os: [macos-latest, ubuntu-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test ./...
      - run: go vet ./...
```

- [ ] **Step 2: Update release.yml to use go-version-file**

Replace the hardcoded `go-version: "1.24"` with `go-version-file: go.mod` in `.github/workflows/release.yml`:

```yaml
name: Release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml .github/workflows/release.yml
git commit -m "fix: use go-version-file in CI and release workflows"
```

---

## Task 2: Update GoReleaser config

**Files:**
- Modify: `.goreleaser.yaml`

- [ ] **Step 1: Change ldflag to use Tag and skip changelog**

Update `.goreleaser.yaml`:
- Change `-X main.version={{.Version}}` to `-X main.version={{.Tag}}` (includes `v` prefix)
- Set `changelog.skip: true` (release-please manages changelog)

Result:

```yaml
version: 2
project_name: kivtz
builds:
  - main: ./cmd/kivtz
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - arm64
      - amd64
    ldflags:
      - -s -w
      - -X main.version={{.Tag}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}

archives:
  - format: tar.gz
    name_template: "kivtz_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: checksums.txt

release:
  github:
    owner: airbugg
    name: kivtz

changelog:
  skip: true
```

- [ ] **Step 2: Commit**

```bash
git add .goreleaser.yaml
git commit -m "chore: use Tag ldflag for v-prefix, skip changelog (release-please manages it)"
```

---

## Task 3: Add release-please workflow and config

**Files:**
- Create: `.github/workflows/release-please.yml`
- Create: `.release-please-manifest.json`
- Create: `release-please-config.json`

- [ ] **Step 1: Create release-please workflow**

Create `.github/workflows/release-please.yml`:

```yaml
name: Release Please
on:
  push:
    branches: [main]
permissions:
  contents: write
  pull-requests: write
jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          release-type: go
```

- [ ] **Step 2: Create release-please manifest**

Create `.release-please-manifest.json`:

```json
{
  ".": "0.2.0"
}
```

- [ ] **Step 3: Create release-please config**

Create `release-please-config.json`:

```json
{
  "packages": {
    ".": {
      "release-type": "go",
      "changelog-sections": [
        { "type": "feat", "section": "Features" },
        { "type": "fix", "section": "Bug Fixes" },
        { "type": "perf", "section": "Performance" },
        { "type": "docs", "section": "Documentation", "hidden": true },
        { "type": "chore", "section": "Miscellaneous", "hidden": true },
        { "type": "ci", "section": "Miscellaneous", "hidden": true },
        { "type": "test", "section": "Miscellaneous", "hidden": true },
        { "type": "refactor", "section": "Miscellaneous", "hidden": true }
      ]
    }
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release-please.yml .release-please-manifest.json release-please-config.json
git commit -m "ci: add release-please for automated versioning"
```

---

## Task 4: Add PR title lint workflow

**Files:**
- Create: `.github/workflows/pr-lint.yml`

- [ ] **Step 1: Create PR lint workflow**

Create `.github/workflows/pr-lint.yml`:

```yaml
name: PR Title Lint
on:
  pull_request:
    types: [opened, edited, synchronize, reopened]
jobs:
  lint-pr-title:
    runs-on: ubuntu-latest
    steps:
      - uses: amannn/action-semantic-pull-request@v5
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          types: |
            feat
            fix
            docs
            style
            refactor
            perf
            test
            build
            ci
            chore
            revert
          requireScope: false
          subjectPattern: ^(?![A-Z]).+$
          subjectPatternError: |
            The subject "{subject}" found in the pull request title "{title}"
            should start with a lowercase letter.
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/pr-lint.yml
git commit -m "ci: add PR title lint for conventional commits"
```

---

## Task 5: Add PR snapshot build workflow

**Files:**
- Create: `.goreleaser-snapshot.yaml`
- Create: `.github/workflows/pr-build.yml`

- [ ] **Step 1: Create slim snapshot GoReleaser config**

Create `.goreleaser-snapshot.yaml` (darwin/arm64 + linux/amd64 only):

```yaml
version: 2
project_name: kivtz
builds:
  - main: ./cmd/kivtz
    env:
      - CGO_ENABLED=0
    targets:
      - darwin_arm64
      - linux_amd64
    ldflags:
      - -s -w
      - -X main.version=snapshot
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}

archives:
  - format: tar.gz
    name_template: "kivtz_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: checksums.txt

changelog:
  skip: true
```

- [ ] **Step 2: Create PR build workflow**

Create `.github/workflows/pr-build.yml`:

```yaml
name: PR Build
on:
  pull_request:
    branches: [main]
permissions:
  pull-requests: write
jobs:
  snapshot:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --snapshot --clean --config .goreleaser-snapshot.yaml
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      - uses: actions/upload-artifact@v4
        with:
          name: snapshot-binaries
          path: dist/kivtz_*
          retention-days: 7
      - name: Generate artifact summary
        id: summary
        run: |
          RUN_URL="${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}"
          {
            echo "comment<<EOF"
            echo "### Snapshot Build"
            echo "Binaries available as [workflow artifacts](${RUN_URL})."
            echo ""
            echo "**Checksums:**"
            echo '```'
            cat dist/checksums.txt
            echo '```'
            echo "EOF"
          } >> "$GITHUB_OUTPUT"
      - uses: marocchino/sticky-pull-request-comment@v2
        with:
          header: snapshot-build
          message: ${{ steps.summary.outputs.comment }}
```

- [ ] **Step 3: Commit**

```bash
git add .goreleaser-snapshot.yaml .github/workflows/pr-build.yml
git commit -m "ci: add PR snapshot builds with sticky comment"
```

---

## Task 6: Create `internal/version` package — `CheckForUpdate`

**Files:**
- Create: `internal/version/version.go`
- Create: `internal/version/version_test.go`

This task builds the core version check function using TDD. The function queries the GitHub releases API and returns whether an update is available.

- [ ] **Step 1: Write failing test for CheckForUpdate**

Create `internal/version/version_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/version/ -v
```

Expected: compilation error — `version` package doesn't exist yet.

- [ ] **Step 3: Write minimal implementation**

Create `internal/version/version.go`:

```go
// Package version checks for available updates via the GitHub releases API.
package version

import (
	"encoding/json"
	"fmt"
	"net/http"
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

type httpError struct {
	StatusCode int
}

func (e *httpError) Error() string {
	return "GitHub API returned " + http.StatusText(e.StatusCode)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/version/ -v
```

Expected: all 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/version/version.go internal/version/version_test.go
git commit -m "feat: add version.CheckForUpdate with GitHub API integration"
```

---

## Task 7: Add `CachedCheck` to version package

**Files:**
- Modify: `internal/version/version.go`
- Modify: `internal/version/version_test.go`

- [ ] **Step 1: Write failing tests for CachedCheck**

Add to `internal/version/version_test.go`:

```go
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
```

Add these imports to the test file: `"os"`, `"path/filepath"`, `"time"`.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/version/ -run "CachedCheck" -v
```

Expected: compilation error — `CachedCheck` and `CacheEntry` don't exist.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/version/version.go`. Merge `"os"` and `"path/filepath"` into the existing import block:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/version/ -v
```

Expected: all 8 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/version/version.go internal/version/version_test.go
git commit -m "feat: add version.CachedCheck with 24h file cache"
```

---

## Task 8: Add `PrintUpdateNotice` to version package

**Files:**
- Modify: `internal/version/version.go`
- Modify: `internal/version/version_test.go`

- [ ] **Step 1: Write failing tests for PrintUpdateNotice**

Add to `internal/version/version_test.go`:

```go
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
```

Add `"bytes"` to test imports.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/version/ -run "PrintUpdateNotice" -v
```

Expected: compilation error — `PrintUpdateNotice` signature doesn't match.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/version/version.go`. Merge `"io"` into the existing import block (note: `"fmt"` was already added in Task 6):

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/version/ -v
```

Expected: all 13 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/version/version.go internal/version/version_test.go
git commit -m "feat: add version.PrintUpdateNotice with timeout and opt-out"
```

---

## Task 9: Refactor `selfupdate.go` to use version package

**Files:**
- Modify: `internal/cli/selfupdate.go`

- [ ] **Step 1: Run existing tests to establish baseline**

```bash
go test ./internal/cli/ -v
```

Expected: all existing tests PASS.

- [ ] **Step 2: Refactor selfupdate.go**

Replace the inline GitHub API logic with `version.CheckForUpdate`. Remove:
- `releasesAPI` const
- `httpClient` var
- Inline release struct
- API call and JSON decoding

Keep:
- `downloadAndReplace()` (tar extraction)
- `runSelfUpdate()` (orchestration, but simplified)

Updated `internal/cli/selfupdate.go`:

```go
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

	version.ClearCache(cacheDir)
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
```

Note: `version.FindAssetURL` was already added in Task 6. It operates on `*UpdateInfo` (no second API call) — the assets come from the same response as `CheckForUpdate`.

- [ ] **Step 3: Run tests to verify nothing is broken**

```bash
go test ./... -v
```

Expected: all tests PASS. The refactoring preserves behavior.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/selfupdate.go internal/version/version.go
git commit -m "refactor: extract GitHub API logic from selfupdate to version package"
```

---

## Task 10: Wire `PrintUpdateNotice` into root command

**Files:**
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Add PersistentPostRun to rootCmd**

Modify `internal/cli/root.go`. Add a `PersistentPostRun` hook that calls `version.PrintUpdateNotice`, skipping `self-update` and `version` commands.

In the `init()` function, add after the existing content:

```go
rootCmd.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
	name := cmd.Name()
	if name == "self-update" || name == "version" {
		return
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cacheDir := filepath.Dir(config.DefaultPath(homeDir))
	version.PrintUpdateNotice(buildVersion, cacheDir, "", os.Stderr)
}
```

Add import:

```go
"github.com/airbugg/kivtz/internal/version"
```

- [ ] **Step 2: Run tests to verify nothing is broken**

```bash
go test ./... -v
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat: show update notice after command execution"
```

---

## Task 11: Make repo public

This task is done last, after all workflow changes are merged to main.

- [ ] **Step 1: Verify all workflow files are committed**

```bash
git status
git log --oneline -10
```

Expected: all workflow files committed, clean working tree.

- [ ] **Step 2: Push all changes**

```bash
git push origin main
```

- [ ] **Step 3: Make repo public**

```bash
gh repo edit --visibility public
```

- [ ] **Step 4: Verify self-update works against public repo**

```bash
go run ./cmd/kivtz self-update
```

Expected: either "already up to date" or downloads latest release successfully.

- [ ] **Step 5: Verify update check works**

```bash
go run ./cmd/kivtz doctor
```

Expected: if a newer release exists, prints update notice at the end.
