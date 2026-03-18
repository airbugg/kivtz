package tui

import (
	"testing"

	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
)

func TestCategorize_ShellEntries(t *testing.T) {
	entries := []scanner.Entry{
		{Name: "fish"},
		{Name: ".zshrc"},
		{Name: ".bashrc"},
	}
	groups := Categorize(entries)
	assert.Len(t, groups[Shell], 3)
}

func TestCategorize_DevelopmentEntries(t *testing.T) {
	entries := []scanner.Entry{
		{Name: "git"},
		{Name: "nvim"},
		{Name: "vim"},
	}
	groups := Categorize(entries)
	assert.Len(t, groups[Development], 3)
}

func TestCategorize_TerminalEntries(t *testing.T) {
	entries := []scanner.Entry{
		{Name: "ghostty"},
		{Name: "alacritty"},
		{Name: "kitty"},
		{Name: "tmux"},
	}
	groups := Categorize(entries)
	assert.Len(t, groups[Terminal], 4)
}

func TestCategorize_UnknownGoesToOther(t *testing.T) {
	entries := []scanner.Entry{
		{Name: "myapp"},
		{Name: ".weirdrc"},
	}
	groups := Categorize(entries)
	assert.Len(t, groups[Other], 2)
}

func TestCategorize_MixedEntries(t *testing.T) {
	entries := []scanner.Entry{
		{Name: "fish"},
		{Name: "git"},
		{Name: "ghostty"},
		{Name: "myapp"},
	}
	groups := Categorize(entries)
	assert.Len(t, groups[Shell], 1)
	assert.Len(t, groups[Development], 1)
	assert.Len(t, groups[Terminal], 1)
	assert.Len(t, groups[Other], 1)
}

func TestCategoryString(t *testing.T) {
	assert.Equal(t, "Shell", Shell.String())
	assert.Equal(t, "Development", Development.String())
	assert.Equal(t, "Terminal", Terminal.String())
	assert.Equal(t, "Other", Other.String())
}

func TestCategoryOrder(t *testing.T) {
	order := CategoryOrder()
	assert.Equal(t, []Category{Shell, Development, Terminal, Other}, order)
}
