# kivtz

Cross-platform dotfiles manager. Discovers, adopts, and syncs config files across macOS, Linux, and WSL using symlinks.

## Install

```sh
curl -fsSL airbugg.com/kivtz | sh
```

Or with Go:

```sh
go install github.com/airbugg/kivtz/cmd/kivtz@latest
```

## Usage

```sh
kivtz                # status dashboard
kivtz init           # discover configs on this machine and adopt them
kivtz init <url>     # clone a dotfiles repo and set up this machine
kivtz sync           # pull, apply, detect drift, push — the one daily command
kivtz add <path>     # adopt a single config into the dotfiles repo
kivtz doctor         # health check
kivtz self-update    # download and install the latest release
```

### Init

Without a URL, `init` scans your home directory for config files, scores them by relevance, and presents an interactive picker. Selected configs are moved into the dotfiles repo and symlinked back.

```sh
kivtz init              # interactive discovery + TUI picker
kivtz init --yes        # accept pre-selected configs without prompts
kivtz init --list       # list discovered configs (non-interactive)
kivtz init --json       # output discovered configs as JSON
```

Running `init` again re-scans and offers configs not yet adopted.

### Sync

Orchestrates a full update cycle:

1. `git pull --ff-only`
2. Apply pending symlinks (conflicts shown as diffs)
3. Detect drift — symlinks replaced by files, new untracked files
4. `git add && commit && push` if dirty

Respects `.syncignore` for paths that shouldn't trigger drift alerts.

### Add

```sh
kivtz add ~/.config/fish    # adopts fish config
kivtz add ~/.gitconfig      # adopts gitconfig
```

Moves the file/directory into the dotfiles repo and creates a symlink at the original location.

## Convention

kivtz manages flat dotfiles repos. Each package lives at the root:

```
fish/
  .config/fish/config.fish    → ~/.config/fish/config.fish
git/
  .gitconfig                  → ~/.gitconfig
nvim/
  .config/nvim/init.lua       → ~/.config/nvim/init.lua
```

Each machine selects which packages to manage via the `packages` field in its local config. No platform groups — use each tool's native config modularity (fish `conf.d`, git `include`, etc.) for platform-specific behavior.

## Config

Stored at `~/.config/kivtz/config.toml`:

```toml
dotfiles_dir = "/home/user/.dotfiles"
repo_url = "https://github.com/user/dotfiles"
platform = "darwin"
hostname = "macbook.local"
packages = ["fish", "git", "nvim"]
```

The `packages` field tracks which configs are adopted on this machine, enabling selective stow during sync.
