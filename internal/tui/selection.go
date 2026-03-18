package tui

import "github.com/airbugg/kivtz/internal/scanner"

// Selection tracks which entries are selected in the multi-select UI.
type Selection struct {
	items    []scanner.Entry
	selected map[int]bool
}

// NewSelection creates a Selection with the given entries.
// preSelected entries start checked.
func NewSelection(items []scanner.Entry, preSelected []scanner.Entry) *Selection {
	sel := &Selection{
		items:    items,
		selected: make(map[int]bool),
	}

	preSet := make(map[string]bool, len(preSelected))
	for _, e := range preSelected {
		preSet[e.Path] = true
	}
	for i, e := range items {
		if preSet[e.Path] {
			sel.selected[i] = true
		}
	}
	return sel
}

// Toggle flips the selection state of the entry at index i.
func (s *Selection) Toggle(i int) {
	if i < 0 || i >= len(s.items) {
		return
	}
	if s.selected[i] {
		delete(s.selected, i)
	} else {
		s.selected[i] = true
	}
}

// IsSelected returns whether the entry at index i is selected.
func (s *Selection) IsSelected(i int) bool {
	return s.selected[i]
}

// Selected returns the currently selected entries in their original order.
func (s *Selection) Selected() []scanner.Entry {
	var result []scanner.Entry
	for i, e := range s.items {
		if s.selected[i] {
			result = append(result, e)
		}
	}
	return result
}

// Items returns all entries.
func (s *Selection) Items() []scanner.Entry {
	return s.items
}

// Len returns the total number of entries.
func (s *Selection) Len() int {
	return len(s.items)
}
