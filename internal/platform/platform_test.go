package platform_test

import (
	"runtime"
	"testing"

	"github.com/airbugg/kivtz/internal/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	info, err := platform.Detect()
	require.NoError(t, err)

	assert.Equal(t, runtime.GOARCH, info.Arch)
	assert.NotEmpty(t, info.HomeDir)
	assert.NotEmpty(t, info.Hostname)
}

func TestGroups_Darwin(t *testing.T) {
	assert.Equal(t, []string{"common", "macos"}, platform.Info{OS: platform.Darwin}.Groups())
}

func TestGroups_Linux(t *testing.T) {
	assert.Equal(t, []string{"common", "linux"}, platform.Info{OS: platform.Linux}.Groups())
}

func TestGroups_WSL(t *testing.T) {
	assert.Equal(t, []string{"common", "linux", "wsl"}, platform.Info{OS: platform.WSL}.Groups())
}
