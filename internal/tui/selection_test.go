package tui

import (
	"testing"
	"time"

	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
)

func makeEntry(name string) scanner.Entry {
	return scanner.Entry{
		Name:      name,
		Path:      "/home/user/.config/" + name,
		Size:      1024,
		ModTime:   time.Now(),
		FileCount: 3,
		IsDir:     true,
	}
}

func TestNewSelection_PreSelectedAreChecked(t *testing.T) {
	all := []scanner.Entry{makeEntry("fish"), makeEntry("git"), makeEntry("myapp")}
	preSelected := []scanner.Entry{makeEntry("fish"), makeEntry("git")}

	sel := NewSelection(all, preSelected)

	assert.True(t, sel.IsSelected(0))
	assert.True(t, sel.IsSelected(1))
	assert.False(t, sel.IsSelected(2))
}

func TestSelection_Toggle(t *testing.T) {
	all := []scanner.Entry{makeEntry("fish"), makeEntry("git")}
	sel := NewSelection(all, nil)

	assert.False(t, sel.IsSelected(0))
	sel.Toggle(0)
	assert.True(t, sel.IsSelected(0))
	sel.Toggle(0)
	assert.False(t, sel.IsSelected(0))
}

func TestSelection_Selected(t *testing.T) {
	all := []scanner.Entry{makeEntry("fish"), makeEntry("git"), makeEntry("myapp")}
	preSelected := []scanner.Entry{makeEntry("fish")}

	sel := NewSelection(all, preSelected)
	sel.Toggle(2) // also select myapp

	selected := sel.Selected()
	assert.Len(t, selected, 2)
	assert.Equal(t, "fish", selected[0].Name)
	assert.Equal(t, "myapp", selected[1].Name)
}

func TestSelection_Items(t *testing.T) {
	all := []scanner.Entry{makeEntry("fish"), makeEntry("git")}
	sel := NewSelection(all, nil)
	assert.Equal(t, all, sel.Items())
}

func TestSelection_Len(t *testing.T) {
	all := []scanner.Entry{makeEntry("fish"), makeEntry("git"), makeEntry("myapp")}
	sel := NewSelection(all, nil)
	assert.Equal(t, 3, sel.Len())
}

func TestSelection_ToggleOutOfBounds(t *testing.T) {
	all := []scanner.Entry{makeEntry("fish")}
	sel := NewSelection(all, nil)
	sel.Toggle(-1) // should not panic
	sel.Toggle(5)  // should not panic
}
