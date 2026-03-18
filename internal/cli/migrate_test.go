package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/airbugg/kivtz/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command string tests: verify command module produces equivalent commands ---

func TestCommandModule_GitStatusPorcelain(t *testing.T) {
	cmd := command.New("git", "status", "--porcelain").Dir("/tmp/repo")
	assert.Equal(t, "git status --porcelain", cmd.String())
}

func TestCommandModule_GitRevParseHead(t *testing.T) {
	cmd := command.New("git", "rev-parse", "HEAD").Dir("/tmp/repo")
	assert.Equal(t, "git rev-parse HEAD", cmd.String())
}

func TestCommandModule_GitPullFFOnly(t *testing.T) {
	cmd := command.New("git", "pull", "--ff-only").Dir("/tmp/repo")
	assert.Equal(t, "git pull --ff-only", cmd.String())
}

func TestCommandModule_GitAddAll(t *testing.T) {
	cmd := command.New("git", "add", "--all").Dir("/tmp/repo")
	assert.Equal(t, "git add --all", cmd.String())
}

func TestCommandModule_GitCommit(t *testing.T) {
	cmd := command.New("git", "commit", "-m", "update fish").Dir("/tmp/repo")
	assert.Equal(t, "git commit -m update fish", cmd.String())
}

func TestCommandModule_GitPush(t *testing.T) {
	cmd := command.New("git", "push").Dir("/tmp/repo")
	assert.Equal(t, "git push", cmd.String())
}

// --- Integration tests: git helpers work with real repos ---

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runTestGit(t, dir, "init")
	runTestGit(t, dir, "config", "user.email", "test@test.com")
	runTestGit(t, dir, "config", "user.name", "Test")
	return dir
}

func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := command.New("git", args...).Dir(dir).Run()
	require.NoError(t, err)
	return out
}

func TestGitRepoStatus_CleanRepo(t *testing.T) {
	dir := initTestRepo(t)

	// Create initial commit so repo is not empty
	writeTestFile(t, filepath.Join(dir, "README"), "hello")
	runTestGit(t, dir, "add", "--all")
	runTestGit(t, dir, "commit", "-m", "init")

	status, err := gitRepoStatus(dir)
	require.NoError(t, err)
	assert.True(t, status.clean)
	assert.Equal(t, 0, status.changed)
}

func TestGitRepoStatus_DirtyRepo(t *testing.T) {
	dir := initTestRepo(t)

	// Create initial commit
	writeTestFile(t, filepath.Join(dir, "README"), "hello")
	runTestGit(t, dir, "add", "--all")
	runTestGit(t, dir, "commit", "-m", "init")

	// Create untracked file
	writeTestFile(t, filepath.Join(dir, "new.txt"), "new")

	status, err := gitRepoStatus(dir)
	require.NoError(t, err)
	assert.False(t, status.clean)
	assert.Equal(t, 1, status.changed)
}

func TestGitCommitAndPush_CommitsChanges(t *testing.T) {
	dir := initTestRepo(t)

	// Initial commit
	writeTestFile(t, filepath.Join(dir, "README"), "hello")
	runTestGit(t, dir, "add", "--all")
	runTestGit(t, dir, "commit", "-m", "init")

	// Make a change
	writeTestFile(t, filepath.Join(dir, "config.fish"), "# fish config")

	err := gitCommitAndPush(dir, "update fish", false)
	require.NoError(t, err)

	// Verify commit happened
	status, err := gitRepoStatus(dir)
	require.NoError(t, err)
	assert.True(t, status.clean)
}

func TestGenerateCommitMessage_SinglePackage(t *testing.T) {
	dir := initTestRepo(t)

	writeTestFile(t, filepath.Join(dir, "README"), "hello")
	runTestGit(t, dir, "add", "--all")
	runTestGit(t, dir, "commit", "-m", "init")

	// Change in fish/ package
	writeTestFile(t, filepath.Join(dir, "fish", "config.fish"), "# fish")

	msg := generateCommitMessage(dir)
	assert.Equal(t, "update fish", msg)
}

func TestGenerateCommitMessage_NoChanges(t *testing.T) {
	dir := initTestRepo(t)

	writeTestFile(t, filepath.Join(dir, "README"), "hello")
	runTestGit(t, dir, "add", "--all")
	runTestGit(t, dir, "commit", "-m", "init")

	msg := generateCommitMessage(dir)
	assert.Equal(t, "update configs", msg)
}

func TestGitRepoStatus_NotAGitRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo

	_, err := gitRepoStatus(dir)
	assert.Error(t, err)
}

func TestGitPull_NoRemote(t *testing.T) {
	dir := initTestRepo(t)

	writeTestFile(t, filepath.Join(dir, "README"), "hello")
	runTestGit(t, dir, "add", "--all")
	runTestGit(t, dir, "commit", "-m", "init")

	// Pull with no remote should error
	_, err := gitPull(dir)
	assert.Error(t, err)
}

func TestGitPull_WithRemote_NothingToFetch(t *testing.T) {
	// Create a bare "remote"
	remote := t.TempDir()
	runTestGit(t, remote, "init", "--bare")

	// Clone it
	local := filepath.Join(t.TempDir(), "local")
	_, err := command.New("git", "clone", remote, local).Run()
	require.NoError(t, err)

	// Configure user for commits
	runTestGit(t, local, "config", "user.email", "test@test.com")
	runTestGit(t, local, "config", "user.name", "Test")

	// Create initial commit and push
	writeTestFile(t, filepath.Join(local, "README"), "hello")
	runTestGit(t, local, "add", "--all")
	runTestGit(t, local, "commit", "-m", "init")
	runTestGit(t, local, "push")

	// Pull should succeed and return false (no new commits)
	pulled, err := gitPull(local)
	require.NoError(t, err)
	assert.False(t, pulled, "nothing new to pull")
}

func TestGitCommitAndPush_WithRemote_Pushes(t *testing.T) {
	// Create a bare "remote"
	remote := t.TempDir()
	runTestGit(t, remote, "init", "--bare")

	// Clone it
	local := filepath.Join(t.TempDir(), "local")
	_, err := command.New("git", "clone", remote, local).Run()
	require.NoError(t, err)

	runTestGit(t, local, "config", "user.email", "test@test.com")
	runTestGit(t, local, "config", "user.name", "Test")

	// Initial commit and push
	writeTestFile(t, filepath.Join(local, "README"), "hello")
	runTestGit(t, local, "add", "--all")
	runTestGit(t, local, "commit", "-m", "init")
	runTestGit(t, local, "push")

	// Make a change and commit+push via the function
	os.WriteFile(filepath.Join(local, "new.txt"), []byte("data"), 0o644)
	err = gitCommitAndPush(local, "add new", true)
	require.NoError(t, err)

	// Verify push happened: clone again and check file exists
	verify := filepath.Join(t.TempDir(), "verify")
	_, err = command.New("git", "clone", remote, verify).Run()
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(verify, "new.txt"))
	assert.NoError(t, err, "pushed file should exist in remote clone")
}
