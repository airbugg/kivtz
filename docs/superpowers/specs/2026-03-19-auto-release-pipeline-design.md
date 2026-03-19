# Auto Release Pipeline + Version Check

## Problem

Releases are manual (tag + push). No commit message enforcement. No PR preview builds. The repo is private, blocking self-update testing. No mechanism to notify users of available updates.

## Solution

### 1. Release Pipeline

**release-please** (Google) manages versioning via PR-based flow:

- Commits land on `main` with conventional commit messages (squash-merged)
- `release-please` auto-maintains a "Release PR" accumulating changes + changelog
- Merging the Release PR creates a git tag (e.g. `v0.3.0`)
- Tag triggers existing GoReleaser workflow, which builds binaries and publishes a GitHub Release

**Commit enforcement** via `action-semantic-pull-request` — lints PR titles. Pairs with squash-merge so the PR title becomes the commit message on `main`. Branch protection must enforce squash-merge-only to guarantee this.

**release-please configuration** (`release-please-config.json`):
- Release type: `go` (no file rewrites — version comes from git tag via ldflags; avoids `version.txt` that `simple` creates)
- Changelog sections: `feat` → Features, `fix` → Bug Fixes, `perf` → Performance
- Bump strategy: conventional commits (`feat` → minor, `fix` → patch, `feat!`/`BREAKING CHANGE` → major)

**GoReleaser changelog**: Set `skip: true` in `.goreleaser.yaml` since release-please manages the changelog and populates the GitHub Release body. Avoids duplication.

Pipeline flow:
```
PR opened -> lint title + snapshot build + tests
PR merged (squash) -> release-please updates Release PR
Release PR merged -> tag created -> GoReleaser builds + publishes
```

### 2. PR Preview Builds

GoReleaser `--snapshot` on PRs builds binaries without publishing. Uploaded as GitHub Actions artifacts (7-day retention). Sticky PR comment links to artifacts with checksums.

Slim snapshot config: darwin/arm64 + linux/amd64 only (fast CI). No Windows support (not a target platform).

Requires `pull-requests: write` permission for sticky comment.

### 3. Repo Visibility

Make repo public via `gh repo edit --visibility public`. Branch protection keeps write access restricted.

**Sequencing**: Merge all workflow changes first, then flip visibility. Public repos have different default `GITHUB_TOKEN` scoping — workflows must already be in place.

### 4. Version Check

Background check on every command run. Goroutine with 2-second timeout so it never blocks the CLI.

**Opt-out**: `KIVTZ_NO_UPDATE_CHECK=1` env var disables the check entirely (for CI, scripts, privacy).

Cache file at `~/.config/kivtz/update-check.json`:
```json
{"latest_version": "v0.3.0", "checked_at": "2026-03-19T12:00:00Z"}
```

Skip check if last check was within 24 hours.

Print one-liner if outdated:
```
  update available: v0.3.0 (current: v0.2.0)
  run `kivtz self-update` to upgrade
```

**Error handling**: All errors silently swallowed — network failures, malformed JSON, cache write errors. The version check must never interfere with the user's command.

**Version format**: Change GoReleaser ldflag from `{{.Version}}` to `{{.Tag}}` so `buildVersion` includes the `v` prefix (e.g. `v0.3.0`). This matches `release.TagName` from the GitHub API, eliminating prefix mismatch bugs. All comparisons use the `v`-prefixed form everywhere. The `PrintUpdateNotice` call in `root.go` passes `buildVersion` (set via `cli.SetVersion`).

## New Files

| File | Purpose |
|------|---------|
| `.github/workflows/release-please.yml` | Runs on push to main, manages Release PR. Permissions: `contents: write`, `pull-requests: write` |
| `.github/workflows/pr-build.yml` | Snapshot builds on PRs with sticky comment. Permissions: `pull-requests: write` |
| `.github/workflows/pr-lint.yml` | Conventional commit enforcement on PR titles. Permissions: default (read-only) |
| `.goreleaser-snapshot.yaml` | Slim config for PR previews (darwin/arm64 + linux/amd64) |
| `.release-please-manifest.json` | Version tracking (`{".": "0.2.0"}`) |
| `release-please-config.json` | release-please configuration (release type, changelog sections) |
| `internal/version/version.go` | Version check logic (check, cache, display) |
| `internal/version/version_test.go` | Tests for version check |

## Modified Files

| File | Change |
|------|--------|
| `internal/cli/root.go` | Call `version.PrintUpdateNotice(buildVersion, cacheDir)` after command execution via `PersistentPostRun`. Skip for `self-update` and `version` commands (self-update manages its own version display; version is informational only) |
| `internal/cli/selfupdate.go` | Remove inline GitHub API call and release struct. Reuse `version.CheckForUpdate` for latest version lookup. Keep download/replace logic (`downloadAndReplace`, tar extraction). Clear update cache after successful update so stale notice doesn't appear |
| `.goreleaser.yaml` | Set `changelog.skip: true`, change ldflag from `{{.Version}}` to `{{.Tag}}` for v-prefix consistency |
| `.github/workflows/release.yml` | Use `go-version-file: go.mod` instead of hardcoded `"1.24"` (bug fix — go.mod requires 1.25) |
| `.github/workflows/ci.yml` | Use `go-version-file: go.mod` instead of hardcoded `"1.24"` (bug fix — go.mod requires 1.25) |

## `internal/version` Package Design

```go
type UpdateInfo struct {
    LatestVersion string // always v-prefixed, e.g. "v0.3.0"
    Available     bool   // true if LatestVersion != currentVersion
}

// CheckForUpdate queries GitHub API for the latest release.
// Returns the latest version and whether an update is available.
func CheckForUpdate(currentVersion string) (*UpdateInfo, error)

// CachedCheck wraps CheckForUpdate with 24h file cache.
// cacheDir is typically ~/.config/kivtz/.
func CachedCheck(currentVersion, cacheDir string) (*UpdateInfo, error)

// PrintUpdateNotice runs CachedCheck in a goroutine with 2s timeout.
// Prints a one-liner if an update is available.
// All errors silently swallowed. Safe to call from any command.
// No-op if KIVTZ_NO_UPDATE_CHECK=1 is set.
func PrintUpdateNotice(currentVersion, cacheDir string)
```

Extracting into its own package keeps `selfupdate.go` focused on download/replace and makes the check independently testable.

### What moves from `selfupdate.go` to `internal/version`

| Moves to `version` | Stays in `selfupdate.go` |
|---------------------|--------------------------|
| `releasesAPI` const | `downloadAndReplace()` |
| `httpClient` var | `runSelfUpdate()` (calls `version.CheckForUpdate` then downloads) |
| Release struct (API response) | Tar extraction logic |
| Version comparison logic | Binary path resolution |
