package tui

import (
	"testing"

	"github.com/airbugg/kivtz/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModel_GroupsEntriesByCategory(t *testing.T) {
	entries := []scanner.Entry{
		makeEntry("fish"),
		makeEntry("git"),
		makeEntry("ghostty"),
		makeEntry("myapp"),
	}
	m := NewModel(entries, nil)

	// Should have headers + entries: Shell header, fish, Development header, git, Terminal header, ghostty, Other header, myapp
	headers := 0
	selectables := 0
	for _, item := range m.items {
		if item.IsHeader {
			headers++
		} else {
			selectables++
		}
	}
	assert.Equal(t, 4, headers)
	assert.Equal(t, 4, selectables)
}

func TestNewModel_PreSelectedStartChecked(t *testing.T) {
	entries := []scanner.Entry{makeEntry("fish"), makeEntry("git")}
	preSelected := []scanner.Entry{makeEntry("fish")}

	m := NewModel(entries, preSelected)

	// Find fish item and verify it's selected
	for _, item := range m.items {
		if !item.IsHeader && item.Entry.Name == "fish" {
			assert.True(t, m.sel.IsSelected(item.SelectIdx))
		}
		if !item.IsHeader && item.Entry.Name == "git" {
			assert.False(t, m.sel.IsSelected(item.SelectIdx))
		}
	}
}

func TestNewModel_CursorStartsOnFirstSelectable(t *testing.T) {
	entries := []scanner.Entry{makeEntry("fish")}
	m := NewModel(entries, nil)

	// Cursor should skip the header
	require.False(t, m.items[m.cursor].IsHeader)
}

func TestModel_SelectedAfterQuit(t *testing.T) {
	entries := []scanner.Entry{makeEntry("fish")}
	m := NewModel(entries, entries)
	m.quitted = true

	assert.Nil(t, m.Selected())
}

func TestModel_SelectedAfterConfirm(t *testing.T) {
	entries := []scanner.Entry{makeEntry("fish"), makeEntry("git")}
	m := NewModel(entries, entries)
	m.confirmed = true

	selected := m.Selected()
	assert.Len(t, selected, 2)
}

func TestModel_EmptyCategories_Skipped(t *testing.T) {
	// Only shell entries — Development, Terminal, Other headers should not appear
	entries := []scanner.Entry{makeEntry("fish"), makeEntry("bash")}
	m := NewModel(entries, nil)

	headerCount := 0
	for _, item := range m.items {
		if item.IsHeader {
			headerCount++
			assert.Equal(t, "Shell", item.HeaderName)
		}
	}
	assert.Equal(t, 1, headerCount)
}

func TestFormatEntryInfo_Directory(t *testing.T) {
	e := scanner.Entry{Name: "fish", Size: 2048, FileCount: 5, IsDir: true}
	assert.Equal(t, "5 files, 2.0 KB", formatEntryInfo(e))
}

func TestFormatEntryInfo_File(t *testing.T) {
	e := scanner.Entry{Name: ".gitconfig", Size: 512, IsDir: false}
	assert.Equal(t, "512 B", formatEntryInfo(e))
}
