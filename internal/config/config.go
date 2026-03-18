// Package config manages the ~/.config/kivtz/config.toml file,
// which stores machine-specific preferences like the dotfiles repo path.
package config

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config stores machine-specific kivtz preferences.
type Config struct {
	DotfilesDir string   `toml:"dotfiles_dir"`
	RepoURL     string   `toml:"repo_url,omitempty"`
	Platform    string   `toml:"platform"`
	Hostname    string   `toml:"hostname"`
	Packages    []string `toml:"packages,omitempty"`
}

// DefaultPath returns the XDG-compliant config file path.
func DefaultPath(homeDir string) string {
	return filepath.Join(homeDir, ".config", "kivtz", "config.toml")
}

// Load reads config from path. Returns zero-value Config if file doesn't exist.
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg.Packages = []string{}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	_, err = toml.Decode(string(data), &cfg)
	if cfg.Packages == nil {
		cfg.Packages = []string{}
	}
	return cfg, err
}

// Save writes config to path, creating parent directories as needed.
func Save(cfg Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
