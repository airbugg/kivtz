package tui

import (
	"fmt"
	"strings"

	"github.com/airbugg/kivtz/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	categoryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dimStyle      = lipgloss.NewStyle().Faint(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

// Item is a flat list item combining category group info with an entry and its index in the selection.
type Item struct {
	Entry      scanner.Entry
	Category   Category
	SelectIdx  int // index into Selection
	IsHeader   bool
	HeaderName string
}

// Model is the bubbletea model for the multi-select discovery interface.
type Model struct {
	items     []Item
	sel       *Selection
	cursor    int
	confirmed bool
	quitted   bool
}

// NewModel creates a multi-select model with entries grouped by category.
// preSelected entries start checked.
func NewModel(entries []scanner.Entry, preSelected []scanner.Entry) Model {
	sel := NewSelection(entries, preSelected)
	groups := Categorize(entries)

	var items []Item
	for _, cat := range CategoryOrder() {
		catEntries := groups[cat]
		if len(catEntries) == 0 {
			continue
		}
		items = append(items, Item{IsHeader: true, HeaderName: cat.String(), Category: cat})
		for _, e := range catEntries {
			idx := findIndex(entries, e)
			items = append(items, Item{Entry: e, Category: cat, SelectIdx: idx})
		}
	}

	// Start cursor on first selectable item
	cursor := 0
	for i, item := range items {
		if !item.IsHeader {
			cursor = i
			break
		}
	}

	return Model{items: items, sel: sel, cursor: cursor}
}

func findIndex(entries []scanner.Entry, target scanner.Entry) int {
	for i, e := range entries {
		if e.Path == target.Path {
			return i
		}
	}
	return -1
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitted = true
			return m, tea.Quit
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case " ":
			item := m.items[m.cursor]
			if !item.IsHeader {
				m.sel.Toggle(item.SelectIdx)
			}
		}
	}
	return m, nil
}

func (m *Model) moveCursor(dir int) {
	for {
		m.cursor += dir
		if m.cursor < 0 {
			m.cursor = 0
			return
		}
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
			return
		}
		if !m.items[m.cursor].IsHeader {
			return
		}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select configs to manage"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("space: toggle  enter: confirm  q: quit"))
	b.WriteString("\n\n")

	for i, item := range m.items {
		if item.IsHeader {
			b.WriteString(categoryStyle.Render(item.HeaderName))
			b.WriteString("\n")
			continue
		}

		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		check := "[ ]"
		if m.sel.IsSelected(item.SelectIdx) {
			check = selectedStyle.Render("[x]")
		}

		info := formatEntryInfo(item.Entry)
		name := item.Entry.Name
		if i == m.cursor {
			name = cursorStyle.Render(name)
		}

		fmt.Fprintf(&b, "%s%s %s  %s\n", cursor, check, name, dimStyle.Render(info))
	}

	return b.String()
}

func formatEntryInfo(e scanner.Entry) string {
	size := formatSize(e.Size)
	if e.IsDir {
		return fmt.Sprintf("%d files, %s", e.FileCount, size)
	}
	return size
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Selected returns the entries the user selected, or nil if they quit without confirming.
func (m Model) Selected() []scanner.Entry {
	if m.quitted {
		return nil
	}
	return m.sel.Selected()
}

// Confirmed returns true if the user pressed Enter to confirm.
func (m Model) Confirmed() bool {
	return m.confirmed
}

// RunSelector runs the multi-select TUI and returns the selected entries.
// Returns nil if the user quits without confirming.
func RunSelector(entries []scanner.Entry, preSelected []scanner.Entry) ([]scanner.Entry, error) {
	model := NewModel(entries, preSelected)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}
	m := finalModel.(Model)
	if !m.Confirmed() {
		return nil, nil
	}
	return m.Selected(), nil
}
