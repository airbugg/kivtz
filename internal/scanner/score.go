package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// KnownPatterns are config names that get a scoring boost.
var KnownPatterns = []string{
	"fish", "git", "ghostty", "nvim", "vim", "tmux",
	"alacritty", "kitty", "starship", "zsh", "bash",
}

// Score computes a relevance score for a scanner Entry.
// Positive signals: recently modified (+3), small size (+2), known config name (+2), few files (+1).
// Penalties: large >1MB (-5), deep nesting >5 levels (-2), many files >100 (-3).
func Score(e Entry) int {
	score := 0

	// Recently modified (within last 30 days)
	if time.Since(e.ModTime) < 30*24*time.Hour {
		score += 3
	}

	// Small size (<50KB)
	if e.Size < 50*1024 {
		score += 2
	}

	// Known config name pattern
	if isKnownPattern(e.Name) {
		score += 2
	}

	// Few files (≤10)
	if e.FileCount <= 10 {
		score += 1
	}

	// Penalty: large >1MB
	if e.Size > MaxSize {
		score -= 5
	}

	// Penalty: deep nesting >5 levels
	if e.IsDir {
		if depth := maxDepth(e.Path); depth > 5 {
			score -= 2
		}
	}

	// Penalty: many files >100
	if e.FileCount > MaxFileCount {
		score -= 3
	}

	return score
}

// maxDepth returns the maximum directory nesting depth under path.
func maxDepth(root string) int {
	max := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		depth := len(filepath.SplitList(rel))
		// Count separators for depth
		depth = 1
		for _, c := range rel {
			if c == filepath.Separator {
				depth++
			}
		}
		if depth > max {
			max = depth
		}
		return nil
	})
	return max
}

// PreSelected returns entries with a score above the given threshold,
// sorted by score descending.
func PreSelected(entries []Entry, threshold int) []Entry {
	type scored struct {
		entry Entry
		score int
	}

	var selected []scored
	for _, e := range entries {
		s := Score(e)
		if s >= threshold {
			selected = append(selected, scored{entry: e, score: s})
		}
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].score > selected[j].score
	})

	result := make([]Entry, len(selected))
	for i, s := range selected {
		result[i] = s.entry
	}
	return result
}

func isKnownPattern(name string) bool {
	// Strip leading dot for traditional dotfiles (e.g., .gitconfig → git)
	clean := name
	if len(clean) > 0 && clean[0] == '.' {
		clean = clean[1:]
	}
	// Strip common suffixes like "config", "rc"
	for _, suffix := range []string{"config", "rc"} {
		if len(clean) > len(suffix) {
			trimmed := clean[:len(clean)-len(suffix)]
			for _, p := range KnownPatterns {
				if trimmed == p {
					return true
				}
			}
		}
	}
	for _, p := range KnownPatterns {
		if clean == p {
			return true
		}
	}
	return false
}
