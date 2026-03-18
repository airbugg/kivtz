package tui

import (
	"strings"

	"github.com/airbugg/kivtz/internal/scanner"
)

// Category represents a group of related config entries.
type Category int

const (
	Shell Category = iota
	Development
	Terminal
	Other
)

func (c Category) String() string {
	switch c {
	case Shell:
		return "Shell"
	case Development:
		return "Development"
	case Terminal:
		return "Terminal"
	default:
		return "Other"
	}
}

// CategoryOrder returns the display order for categories.
func CategoryOrder() []Category {
	return []Category{Shell, Development, Terminal, Other}
}

var categoryMap = map[string]Category{
	"fish":      Shell,
	"zsh":       Shell,
	"bash":      Shell,
	"git":       Development,
	"nvim":      Development,
	"vim":       Development,
	"ghostty":   Terminal,
	"alacritty": Terminal,
	"kitty":     Terminal,
	"tmux":      Terminal,
	"starship":  Terminal,
}

// Categorize groups scanner entries by category.
func Categorize(entries []scanner.Entry) map[Category][]scanner.Entry {
	groups := make(map[Category][]scanner.Entry)
	for _, e := range entries {
		cat := classifyEntry(e.Name)
		groups[cat] = append(groups[cat], e)
	}
	return groups
}

func classifyEntry(name string) Category {
	clean := name
	if len(clean) > 0 && clean[0] == '.' {
		clean = clean[1:]
	}
	for _, suffix := range []string{"config", "rc"} {
		if len(clean) > len(suffix) && strings.HasSuffix(clean, suffix) {
			trimmed := clean[:len(clean)-len(suffix)]
			if cat, ok := categoryMap[trimmed]; ok {
				return cat
			}
		}
	}
	if cat, ok := categoryMap[clean]; ok {
		return cat
	}
	return Other
}
