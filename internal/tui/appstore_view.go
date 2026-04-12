package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appstore"
)

// renderAppstoreStatus is the one-line footer under the main pane.
func renderAppstoreStatus(s *core.State, width int) string {
	var parts []string
	if s.AppstoreStatusMsg != "" {
		parts = append(parts, statusErrorStyle.Render(s.AppstoreStatusMsg))
	} else if s.AppstoreBusy {
		parts = append(parts, statusBusyStyle.Render("working…"))
	} else {
		var text string
		switch {
		case s.AppstoreSearchActive:
			text = "type to search · enter submit · esc cancel"
		case s.ContentFocused && s.AppstoreDetailOpen:
			text = "esc back to results"
		case s.ContentFocused:
			text = "j/k nav · enter detail · / search · R refresh · A toggle AUR · esc sidebar"
		default:
			text = "j/k categories · enter browse · / search · R refresh · A toggle AUR"
		}
		parts = append(parts, statusBarStyle.Render(text))
	}
	if res := s.AppstoreResults; res.Truncated {
		parts = append(parts, statusBarStyle.Render("  (results truncated)"))
	}
	if s.Appstore.AURLimit.Active {
		parts = append(parts, statusErrorStyle.Render("  AUR limited"))
	}
	line := strings.Join(parts, "")
	return lipgloss.NewStyle().Width(width).Render(line)
}

// submitAppstoreSearch dispatches a text search with the current
// input. Empty input returns the user to the sidebar category. This
// is called from model.go's key handler when enter is pressed in
// search mode.
func (m *Model) submitAppstoreSearch() (tea.Cmd, bool) {
	if m.appstore.Search == nil {
		return nil, false
	}
	text := strings.TrimSpace(m.state.AppstoreSearchInput)
	m.state.AppstoreSearchActive = false
	if text == "" {
		m.state.FocusAppstoreSidebar()
		return m.loadAppstoreCategoryCmd(), true
	}
	q := appstore.SearchQuery{
		Text:       text,
		IncludeAUR: m.state.AppstoreIncludeAUR,
		Limit:      core.AppstoreSearchLimit,
	}
	m.state.MarkAppstoreBusy()
	m.state.AppstoreFocus = core.AppstoreFocusResults
	m.state.ContentFocused = true
	return m.appstore.Search(q), true
}

// loadAppstoreCategoryCmd builds the tea.Cmd that loads the currently
// selected sidebar category.
func (m Model) loadAppstoreCategoryCmd() tea.Cmd {
	if m.appstore.Search == nil {
		return nil
	}
	cat, ok := m.state.SelectedAppstoreCategory()
	if !ok {
		return nil
	}
	q := appstore.SearchQuery{
		Category:   cat.ID,
		IncludeAUR: m.state.AppstoreIncludeAUR && cat.ID == "aur",
		Limit:      core.AppstoreCategoryLimit,
	}
	m.state.MarkAppstoreBusy()
	return m.appstore.Search(q)
}

// handleAppstoreSearchInput runs while the search input has focus.
// It swallows every key so typed characters don't trigger shortcuts.
func (m Model) handleAppstoreSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.CloseAppstoreSearch()
		return m, nil
	case "enter":
		cmd, _ := m.submitAppstoreSearch()
		return m, cmd
	case "backspace":
		m.state.BackspaceAppstoreSearch()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}
	if msg.Type == tea.KeyRunes {
		for _, r := range msg.Runes {
			m.state.AppendAppstoreSearchRune(r)
		}
		return m, nil
	}
	if msg.Type == tea.KeySpace {
		m.state.AppendAppstoreSearchRune(' ')
		return m, nil
	}
	return m, nil
}

// categoryID returns the id of the currently highlighted sidebar
// category, or "" when the catalog isn't loaded yet.
func (m Model) categoryID() string {
	if cat, ok := m.state.SelectedAppstoreCategory(); ok {
		return cat.ID
	}
	return ""
}
