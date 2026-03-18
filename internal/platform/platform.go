// Package platform detects the current operating environment and resolves
// which dotfile groups should be applied.
package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OS identifies the operating environment.
type OS int

const (
	Darwin OS = iota
	Linux
	WSL
)

func (o OS) String() string {
	return [...]string{"darwin", "linux", "wsl"}[o]
}

// Info contains detected platform information.
type Info struct {
	OS       OS
	Arch     string
	Hostname string
	HomeDir  string
}

// Groups returns the dotfile group directories applicable to this platform.
func (i Info) Groups() []string {
	switch i.OS {
	case Darwin:
		return []string{"common", "macos"}
	case Linux:
		return []string{"common", "linux"}
	case WSL:
		return []string{"common", "linux", "wsl"}
	default:
		return []string{"common"}
	}
}

// Detect identifies the current platform.
func Detect() (Info, error) {
	info := Info{Arch: runtime.GOARCH}

	var err error
	if info.HomeDir, err = os.UserHomeDir(); err != nil {
		return info, err
	}
	if info.Hostname, err = os.Hostname(); err != nil {
		return info, err
	}

	switch runtime.GOOS {
	case "darwin":
		info.OS = Darwin
	case "linux":
		if isWSL() {
			info.OS = WSL
		} else {
			info.OS = Linux
		}
	}

	return info, nil
}

// HasCommand checks if a command is available on PATH.
func HasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}
