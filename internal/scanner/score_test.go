package scanner_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScore_RecentSmallConfigScoresHigherThanStaleLargeDir(t *testing.T) {
	root := t.TempDir()

	// Recent small known config (fish)
	fishDir := filepath.Join(root, ".config", "fish")
	writeFile(t, filepath.Join(fishDir, "config.fish"), "# fish config")

	// Stale large unknown directory — old mod time, more files, bigger
	bigDir := filepath.Join(root, ".config", "bigstuff")
	require.NoError(t, os.MkdirAll(bigDir, 0o755))
	for i := range 50 {
		writeFile(t, filepath.Join(bigDir, fmt.Sprintf("file%d.dat", i)), "data content here padding")
	}
	// Set mod time to 90 days ago
	oldTime := time.Now().Add(-90 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(bigDir, oldTime, oldTime))

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	var fishEntry, bigEntry scanner.Entry
	for _, e := range entries {
		switch e.Name {
		case "fish":
			fishEntry = e
		case "bigstuff":
			bigEntry = e
		}
	}

	fishScore := scanner.Score(fishEntry)
	bigScore := scanner.Score(bigEntry)
	assert.Greater(t, fishScore, bigScore, "recent small known config should score higher than stale large dir")
}

func TestScore_KnownPatternBoost(t *testing.T) {
	root := t.TempDir()

	// Known config name
	writeFile(t, filepath.Join(root, ".config", "nvim", "init.lua"), "-- nvim")
	// Unknown config name
	writeFile(t, filepath.Join(root, ".config", "randomtool", "config.toml"), "key = val")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	var nvimEntry, randomEntry scanner.Entry
	for _, e := range entries {
		switch e.Name {
		case "nvim":
			nvimEntry = e
		case "randomtool":
			randomEntry = e
		}
	}

	nvimScore := scanner.Score(nvimEntry)
	randomScore := scanner.Score(randomEntry)
	assert.Greater(t, nvimScore, randomScore, "known config pattern should score higher")
}

func TestScore_DeepNestingPenalty(t *testing.T) {
	root := t.TempDir()

	// Shallow dir (depth 1)
	writeFile(t, filepath.Join(root, ".config", "shallow", "config.toml"), "ok")

	// Deep dir (depth 7 — well over 5)
	writeFile(t, filepath.Join(root, ".config", "deep", "a", "b", "c", "d", "e", "f", "file.toml"), "ok")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	var shallowEntry, deepEntry scanner.Entry
	for _, e := range entries {
		switch e.Name {
		case "shallow":
			shallowEntry = e
		case "deep":
			deepEntry = e
		}
	}

	shallowScore := scanner.Score(shallowEntry)
	deepScore := scanner.Score(deepEntry)
	assert.Greater(t, shallowScore, deepScore, "deeply nested dir should score lower")
}

func TestPreSelected_ReturnsAboveThresholdSortedByScore(t *testing.T) {
	root := t.TempDir()

	// High score: recent, small, known pattern
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")
	// Medium score: recent, small, unknown
	writeFile(t, filepath.Join(root, ".config", "randomtool", "config.toml"), "key = val")
	// Low score: stale unknown
	staleDir := filepath.Join(root, ".config", "oldstuff")
	writeFile(t, filepath.Join(staleDir, "old.conf"), "old")
	oldTime := time.Now().Add(-90 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(staleDir, oldTime, oldTime))

	entries, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Use a threshold that excludes the stale entry
	fishScore := 0
	for _, e := range entries {
		if e.Name == "fish" {
			fishScore = scanner.Score(e)
		}
	}
	require.Greater(t, fishScore, 0)

	// Threshold of 5: should include fish (known+recent+small+fewfiles=8) but not low scorers
	selected := scanner.PreSelected(entries, 5)
	require.NotEmpty(t, selected)

	// First entry should be fish (highest score)
	assert.Equal(t, "fish", selected[0].Name, "highest scoring entry should be first")

	// All selected should have score >= threshold
	for _, e := range selected {
		assert.GreaterOrEqual(t, scanner.Score(e), 5, "all selected entries should meet threshold")
	}
}

func TestPreSelected_SortedDescending(t *testing.T) {
	root := t.TempDir()

	// Create entries with different scores
	writeFile(t, filepath.Join(root, ".config", "fish", "config.fish"), "# fish")
	writeFile(t, filepath.Join(root, ".config", "nvim", "init.lua"), "-- nvim")
	writeFile(t, filepath.Join(root, ".config", "unknown1", "a.conf"), "a")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	selected := scanner.PreSelected(entries, 0) // threshold 0 includes all
	require.Len(t, selected, len(entries))

	// Verify descending order
	for i := 1; i < len(selected); i++ {
		prev := scanner.Score(selected[i-1])
		curr := scanner.Score(selected[i])
		assert.GreaterOrEqual(t, prev, curr, "entries should be sorted by score descending")
	}
}

func TestScore_KnownPatternsIncludeAllExpected(t *testing.T) {
	expected := []string{"fish", "git", "ghostty", "nvim", "vim", "tmux", "alacritty", "kitty", "starship", "zsh", "bash"}
	for _, name := range expected {
		assert.Contains(t, scanner.KnownPatterns, name, "%s should be in KnownPatterns", name)
	}
}

func TestScore_KnownPatternMatchesTraditionalDotfiles(t *testing.T) {
	root := t.TempDir()

	// .gitconfig should match "git" pattern
	writeFile(t, filepath.Join(root, ".gitconfig"), "[user]\nname = test")
	// .bashrc should match "bash" pattern
	writeFile(t, filepath.Join(root, ".bashrc"), "# bash")

	entries, err := scanner.Scan(root)
	require.NoError(t, err)

	for _, e := range entries {
		s := scanner.Score(e)
		// Both should get known pattern boost (+2) plus recent (+3) plus small (+2) plus few files (+1) = 8
		assert.Equal(t, 8, s, "%s should score 8 (known pattern + recent + small + few files)", e.Name)
	}
}
