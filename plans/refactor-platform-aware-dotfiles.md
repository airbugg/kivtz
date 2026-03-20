---
title: "Refactor: Platform-aware dotfiles with shared Claude profiles"
date: 2026-03-20
status: draft
repos:
  - ~/dev/kivtz (tool)
  - ~/kivtzeynekuda (dotfiles)
---

## Problem Statement

Two intertwined problems:

1. **kivtz doesn't understand the dotfiles repo structure.** The repo uses platform group directories (`common/`, `macos/`, `linux/`, `wsl/`) containing stow-able file trees, but kivtz treats top-level dirs as flat stow packages. Running `kivtz sync` created wrong directories (`~/claude/`, `~/fish/`, etc.) instead of symlinking into `$HOME` correctly. Additionally, the `common/` platform concept introduces merge complexity — when both `common/` and a platform dir contribute files to the same tool (e.g., fish), the resolution logic becomes brittle and unintuitive.

2. **Claude multi-profile config is fragile and duplicative.** Three directories (`~/.claude`, `~/.claude-personal`, `~/.claude-work`) exist with unclear boundaries. Skills, hooks, and commands are duplicated across profiles. The shared `~/.claude` dir was ambiguous — sometimes a third profile, sometimes a shared base. Research shows `CLAUDE_CONFIG_DIR` is incompletely implemented: skills and commands are hardcoded to `~/.claude/` regardless of the env var, while CLAUDE.md and settings.json are loaded from both locations.

## Solution

### Dotfiles: per-machine directories, no package abstraction

Replace the `common/` + `<platform>/` group structure with **per-machine directories**. Each machine's directory is a direct mirror of `$HOME` — no intermediate package grouping.

```
kivtzeynekuda/
  macbook/                    # Eugene's MacBook Pro
    .config/fish/config.fish
    .config/fish/conf.d/git.fish
    .config/fish/conf.d/macos.fish
    .config/fish/completions/bun.fish
    .config/fish/completions/kivtz.fish
    .config/fish/functions/proj.fish
    .config/ghostty/config
    .config/ccstatusline/personal.json
    .config/ccstatusline/work.json
    .config/ccstatusline/settings.json
    .config/git/ignore
    .config/gh-personal/hosts.yml
    .config/gh-personal/config.yml
    .config/gh-work/hosts.yml
    .config/gh-work/config.yml
    .gitconfig
    .gitconfig-personal
    .gitconfig-platform
    .asdfrc
    .bash_profile
    .zprofile
    .zshrc
    .claude/CLAUDE.md
    .claude/settings.json
    .claude/hooks/smart-approve.sh
    .claude/hooks/statusline.sh
    .claude/hooks/profile-widget.sh
    .claude/commands/commit-push-pr.md
    .claude/statusline-command.sh
    .claude-personal/CLAUDE.md
    .claude-personal/settings.json
    .claude-work/CLAUDE.md
    .claude-work/settings.json
    .claude-work/settings.local.json
    .claude-work/policy-limits.json
    .claude-work/statusline-command.sh
  homeserver/                 # Linux home server
    .config/fish/config.fish
    .config/fish/conf.d/git.fish
    .config/fish/conf.d/linux.fish
    ...
  windows-pc/                 # WSL
    ...
```

**Why per-machine over common/ + platform/:**
- Zero merge logic. Each machine's dir is exactly what it gets.
- No ambiguity about which files come from where.
- Adding a machine-specific tweak = edit that machine's dir. No platform guard needed.
- kivtz becomes trivially simple: `stow.Plan("<machine>/", "~/")`.

**Trade-off:** ~18 stable files duplicated across machines (git aliases, gitconfig, shell configs). These rarely change. When they do, updating N machine dirs is a minor cost vs. the complexity of merge/conflict resolution.

### Claude profiles: shared base + thin overlays

Leverage the `CLAUDE_CONFIG_DIR` behavior where both `~/.claude/` and `$CLAUDE_CONFIG_DIR/` are read:

```
~/.claude/                          # SHARED BASE — always read by Claude Code
  CLAUDE.md                         # shared instructions
  settings.json                     # shared hooks, shared permissions, deny rules
  hooks/                            # hook scripts
  commands/                         # shared slash commands
  skills/                           # ALL skills (npx skills installs here once)

~/.claude-work/                     # WORK OVERLAY (CLAUDE_CONFIG_DIR target)
  CLAUDE.md                         # work-only additions (vertical spacing, React/RN rules, CLI tools)
  settings.json                     # work-only: model, enabled plugins, effort level
  settings.local.json               # local overrides
  policy-limits.json

~/.claude-personal/                 # PERSONAL OVERLAY (CLAUDE_CONFIG_DIR target)
  CLAUDE.md                         # personal-only additions (currently empty, future use)
  settings.json                     # personal-only: plugins, marketplace configs
```

**Why this works:**
- Skills hardcoded to `~/.claude/skills/` — both profiles see them automatically.
- CLAUDE.md loaded from both locations — shared base + profile extras, additive.
- Hooks defined in `~/.claude/settings.json` — both profiles inherit them.
- No duplication of skills, hooks, or commands across profiles.

**What goes in each overlay settings.json:**
- `model` (if different per profile)
- `enabledPlugins` (work has slack, figma, etc.)
- `effortLevel`
- `extraKnownMarketplaces` (personal has ralph-marketplace)

**What goes in shared settings.json:**
- `hooks` (PreToolUse smart-approve)
- `statusLine` configuration
- `permissions.allow` and `permissions.deny`

### Fish aliases

```fish
alias ccw='CLAUDE_CONFIG_DIR=~/.claude-work GH_CONFIG_DIR=~/.config/gh-work command claude'
alias ccp='CLAUDE_CONFIG_DIR=~/.claude-personal GH_CONFIG_DIR=~/.config/gh-personal command claude'
alias claude='ccp'
alias claude-work='ccw'
alias claude-personal='ccp'
```

### kivtz changes

1. Replace package-based discovery with machine-based stowing.
2. Auto-detect current machine from hostname (mapped in config.toml).
3. Add `--dry-run` flag to `sync` that plans but does not apply.
4. Simplify config.toml to bare minimum.

New config.toml:
```toml
dotfiles_dir = "/Users/eugenel/kivtzeynekuda"
repo_url = "https://github.com/airbugg/kivtzeynekuda"
machine = "macbook"
```

`machine` maps to the directory name in the repo. Auto-detected from hostname with fallback to config.

## Commits

### Phase 0: Restore broken state (kivtz repo)

0a. **Add `--dry-run` flag to `kivtz sync`.** Modify `runSync` to accept a `--dry-run` flag. When set, call `stow.Plan` and print what would be linked, conflicted, or skipped — but do not call `stow.Apply`. Print colored output showing each entry with its action. This is the safety net for all subsequent work.

0b. **Add `machine` field to config.toml.** Add `Machine string` to the Config struct. Update `Load`/`Save`. No behavior change yet — just the field.

0c. **Replace `discoverPackages` with `resolveMachineDir`.** New function: if `config.Machine` is set, return `filepath.Join(dotfilesDir, config.Machine)`. If not set, fall back to trying `os.Hostname()` matched against directory names in the repo. If no match, error with a clear message.

0d. **Replace `planAll` package loop with single machine plan.** Instead of iterating over packages, call `stow.Plan(machineDir, targetDir)` once. Remove `discoverPackages` entirely. Update `planResult` accounting.

0e. **Update drift detection for machine-based layout.** `detectDriftFlat` currently expects package subdirs. Update to scan the single machine directory. The drift scanner walks managed subdirs looking for new/overwritten files — this should work with any directory structure.

0f. **Update tests.** Rewrite `sync_test.go` to use machine dirs instead of flat packages. Add test for `--dry-run` output. Add test for hostname fallback. Add test for missing machine dir error.

0g. **Build and install.** `go build -o ~/.local/bin/kivtz ./cmd/kivtz` to replace the current binary.

### Phase 1: Restructure dotfiles repo

1a. **Create `macbook/` directory in the dotfiles repo.** Copy all files from `common/` and `macos/` into a flat `macbook/` directory, stripping the package subdirectory layer. For example, `common/fish/.config/fish/config.fish` becomes `macbook/.config/fish/config.fish` and `common/git/.gitconfig` becomes `macbook/.gitconfig`. Exclude non-stowable files (`macos/Brewfile`, `macos/macos.sh`, `.DS_Store`). Move those to a top-level `scripts/` or `setup/` directory.

1b. **Verify with `kivtz sync --dry-run`.** Run dry-run and verify every file maps to the correct `$HOME` target. Compare against the audit of currently-working symlinks to ensure nothing is lost. Every OK symlink from the audit must appear in the dry-run output. Every BROKEN symlink must show as "would link."

1c. **Create `homeserver/` directory.** Copy the Linux-relevant files: `linux/` files + the shared files that apply to the home server. This can be done later when actually setting up the home server — start with just `macbook/`.

1d. **Remove old `common/`, `macos/`, `linux/`, `wsl/` directories.** Only after dry-run verification confirms the new structure is correct.

1e. **Update `.syncignore` and `.gitignore`.** Paths changed from `claude/.claude-personal/...` to `macbook/.claude-personal/...`. Update all patterns.

1f. **Update repo README.** Document the per-machine layout.

### Phase 2: Restructure Claude profiles

2a. **Split shared CLAUDE.md from work CLAUDE.md.** Extract the common sections (Communication, Bug Fixing) into `~/.claude/CLAUDE.md`. Leave only work-specific sections (Git, PRs, Code Style vertical spacing, React/RN, CLI Tools) in `~/.claude-work/CLAUDE.md`. Leave personal CLAUDE.md as-is (currently only has Communication + Bug Fixing — which will now be in shared, so personal overlay becomes empty or gets removed).

2b. **Split shared settings.json from profile settings.json.** Move hooks, statusLine, and shared permissions into `~/.claude/settings.json`. Keep only model, enabledPlugins, effortLevel, and extra marketplace config in each profile's settings.json. Remove duplicate permissions from profile settings.json files.

2c. **Verify with `kivtz sync --dry-run`.** Confirm all three directories (`.claude/`, `.claude-personal/`, `.claude-work/`) have the correct files mapped.

2d. **Remove duplicate skills from profile dirs.** Skills should only exist in `~/.claude/skills/`. Remove `~/.claude-personal/skills/` and `~/.claude-work/skills/` directories. Verify skills are discoverable from both profiles.

2e. **Remove duplicate hooks/commands from profile dirs.** These now live only in `~/.claude/`. Remove any copies from `.claude-personal/` and `.claude-work/` in the dotfiles repo.

### Phase 3: Fish alias update

3a. **Update fish config.** Add `alias claude='ccp'` and change `claude-work`/`claude-personal` to aliases for `ccw`/`ccp`. Use `command claude` in the base aliases to prevent recursion.

3b. **Update dotfiles repo.** The fish config.fish lives in `macbook/.config/fish/config.fish`.

### Phase 4: Cleanup

4a. **Remove the `/request-refactor` skill from `~/.claude-personal/skills/` and `~/.claude-work/skills/`.** It should only exist in `~/.claude/skills/`.

4b. **Remove stale `.claude-backup-*` directories** if any remain.

4c. **Run `kivtz sync` for real.** With user approval, after all dry-run verification passes.

4d. **Verify end state.** All symlinks correct, fish aliases work, git abbreviations load, statusline displays, both Claude profiles functional with shared skills.

## Decision Document

- **Per-machine over platform groups.** Duplication of ~18 stable files is acceptable to eliminate all merge/resolution complexity. Each machine dir is a complete, self-contained mirror of what that machine's `$HOME` should contain (for managed files).
- **No package abstraction.** The machine directory IS the stow unit. No subdirectories-as-packages. File paths mirror `$HOME` directly.
- **Machine identification via config.toml.** `machine = "macbook"` in config.toml maps to the directory name. Hostname auto-detection as fallback.
- **Claude shared base architecture.** `~/.claude/` is the shared base (skills, hooks, commands, shared CLAUDE.md, shared settings.json). Profile dirs (`~/.claude-work/`, `~/.claude-personal/`) are thin overlays with only auth + differing settings. This leverages the (buggy but useful) behavior where Claude Code reads from both `~/.claude/` and `$CLAUDE_CONFIG_DIR/`.
- **Skills managed by `npx skills` in `~/.claude/skills/` only.** Not tracked in the dotfiles repo (added to `.gitignore`/`.syncignore`). Both profiles see them via the hardcoded `~/.claude/skills/` path.
- **`blocklist.json` and `fish_variables` not symlinked.** These are auto-managed by their respective tools and will be added to `.syncignore`.
- **`Brewfile` and `macos.sh` are setup scripts, not dotfiles.** Move to `scripts/` or `setup/` at repo root, not inside a machine directory.
- **Dry-run is mandatory before any real apply.** `kivtz sync --dry-run` must show the complete plan before `kivtz sync` applies changes.

## Testing Decisions

- **Good tests verify observable behavior**: given a repo structure and a target dir, does the plan produce the correct symlink entries? Given a dry-run flag, does sync print but not apply?
- **Modules to test:**
  - `stow.Plan` — already well-tested, no changes needed
  - `cli.resolveMachineDir` — new function, test: config value, hostname fallback, missing dir error
  - `cli.runSync` with `--dry-run` — test that entries are printed, not applied
  - `cli.planAll` replacement — test single machine dir produces correct entries
  - `drift.Detect` — verify it works with flat machine dir (no package subdirs)
- **Prior art:** existing `sync_test.go`, `stow_test.go`, `drift_test.go` all use `t.TempDir()` for filesystem isolation. Follow the same pattern.

## Out of Scope

- **GitHub remote for kivtzeynekuda.** The private repo needs to be created/restored under `airbugg`. This is a manual GitHub operation, not part of the refactor.
- **GitHub auth for `airbugg` account.** The keychain credential for the `airbugg` GitHub account was lost. Needs `gh auth login` — separate from this refactor.
- **`homeserver/` and `windows-pc/` machine dirs.** Only `macbook/` will be created now. Other machines can be added later when needed.
- **Merging `feat/flatten-and-clean` to main.** The dotfiles repo branch management is separate.
- **kivtz `init` and `add` commands.** These need updating for the machine-based model but are not blocking the sync flow.
- **Testing the Claude settings.json merge behavior.** We know both locations are read. The exact merge semantics (union, last-wins, etc.) for hooks and permissions need empirical testing but won't block the restructuring.

## Further Notes

- The `settings.json` merge behavior between `~/.claude/settings.json` and `$CLAUDE_CONFIG_DIR/settings.json` is undocumented and may change in future Claude Code versions. If Anthropic ships native multi-profile support (requested in issues #20131, #24963, #30031), the entire profile architecture may need revisiting. The shared-base approach is the most forward-compatible design given current constraints.
- The per-machine model means adding a new machine requires copying ~40 files. A future `kivtz add-machine --from macbook` command could automate this, but is out of scope for now.
- Skills installed via `npx @anthropic-ai/claude-code` go to `~/.claude/skills/`. The `npx skills` tool from Vercel is a separate mechanism — verify it also installs to `~/.claude/skills/` and not to `$CLAUDE_CONFIG_DIR/skills/`.
