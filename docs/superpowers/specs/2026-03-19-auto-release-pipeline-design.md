# Auto Release Pipeline + Version Check

## Problem

Releases are manual (tag + push). No commit message enforcement. No PR preview builds. The repo is private, blocking self-update testing. No mechanism to notify users of available updates.

## Solution

### 1. Release Pipeline

**release-please** (Google) manages versioning via PR-based flow:

- Commits land on `main` with conventional commit messages (squash-merged)
- `release-please` auto-maintains a "Release PR" accumulating changes + changelog
- Merging the Release PR creates a git tag
- Tag triggers existing GoReleaser workflow, which builds binaries and publishes a GitHub Release

**Commit enforcement** via `action-semantic-pull-request` — lints PR titles. Pairs with squash-merge so the PR title becomes the commit message on `main`.

Pipeline flow:
```
PR opened -> lint title + snapshot build + tests
PR merged (squash) -> release-please updates Release PR
Release PR merged -> tag created -> GoReleaser builds + publishes
```

### 2. PR Preview Builds

GoReleaser `--snapshot` on PRs builds binaries without publishing. Uploaded as GitHub Actions artifacts (7-day retention). Sticky PR comment links to artifacts with checksums.

Slim snapshot config: darwin/arm64 + linux/amd64 only (fast CI).

### 3. Repo Visibility

Make repo public via `gh repo edit --visibility public`. Branch protection keeps write access restricted.

### 4. Version Check

Background check on every command run. Goroutine with 2-second timeout so it never blocks the CLI.

Cache file at `~/.config/kivtz/update-check.json`:
```json
{"latest_version": "v0.3.0", "checked_at": "2026-03-19T12:00:00Z"}
```

Skip check if last check was within 24 hours.

Print one-liner if outdated:
```
  kivtz v0.2.0 — update available: v0.3.0
  run `kivtz self-update` to upgrade
```

## New Files

| File | Purpose |
|------|---------|
| `.github/workflows/release-please.yml` | Runs on push to main, manages Release PR |
| `.github/workflows/pr-build.yml` | Snapshot builds on PRs with sticky comment |
| `.github/workflows/pr-lint.yml` | Conventional commit enforcement on PR titles |
| `.goreleaser-snapshot.yaml` | Slim config for PR previews (darwin/arm64 + linux/amd64) |
| `.release-please-manifest.json` | Version tracking for release-please |
| `release-please-config.json` | release-please configuration |
| `internal/version/version.go` | Version check logic (check, cache, display) |
| `internal/version/version_test.go` | Tests for version check |

## Modified Files

| File | Change |
|------|--------|
| `internal/cli/root.go` | Call version check after command execution |
| `internal/cli/selfupdate.go` | Reuse version check from `internal/version` |

## Unchanged Files

| File | Why |
|------|-----|
| `.github/workflows/release.yml` | Already correct (tag-triggered GoReleaser) |
| `.github/workflows/ci.yml` | Stays as-is |

## `internal/version` Package Design

```go
// CheckForUpdate queries GitHub API for the latest release.
func CheckForUpdate(currentVersion string) (*UpdateInfo, error)

// CachedCheck wraps CheckForUpdate with 24h file cache.
func CachedCheck(currentVersion, cacheDir string) (*UpdateInfo, error)

// PrintUpdateNotice runs CachedCheck in a goroutine with 2s timeout.
// Prints a one-liner if an update is available. Safe to call from any command.
func PrintUpdateNotice(currentVersion, cacheDir string)
```

`UpdateInfo` contains `LatestVersion string` and `Available bool`.

Extracting into its own package keeps `selfupdate.go` focused on download/replace and makes the check independently testable.
