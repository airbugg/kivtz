# kivtz

Cross-platform dotfiles manager. Manages symlinks across macOS, Linux, and WSL.

## Install

```sh
curl -fsSL airbugg.com/kivtz | sh
```

## Usage

```sh
kivtz init [url]     # clone a dotfiles repo and set up this machine
kivtz                # status dashboard
kivtz sync           # pull, apply, detect drift, push — the one daily command
kivtz doctor         # health check
```

## Convention

kivtz manages dotfiles repos with this layout:

```
common/     # stowed on all platforms
macos/      # stowed on macOS only
linux/      # stowed on Linux only
wsl/        # stowed on WSL (layered on top of linux)
```

Each directory contains stow-style packages mirroring your home directory structure.
