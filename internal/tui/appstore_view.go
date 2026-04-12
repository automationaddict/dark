package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appstore"
)

// renderAppstore is the top-level entry for the F2 tab. Layout is a
// vertical stack: search bar on top, sidebar + main pane in the
// middle, and a one-line status footer on the bottom. The main pane
// is either the results list or the detail panel depending on
// AppstoreDetailOpen.
func renderAppstore(s *core.State, width, height int) string {
	if width < 60 {
		return contentStyle.Width(width).Height(height).Render(
			placeholderStyle.Render("Terminal too narrow for the App Store — widen to at least 60 columns."))
	}
	if !s.AppstoreLoaded {
		return contentStyle.Width(width).Height(height).Render(
			placeholderStyle.Render("Loading package catalog from darkd…"))
	}
	if s.Appstore.Backend == appstore.BackendNone {
		msg := placeholderStyle.Render(
			"No package backend detected.\nInstall pacman to populate the App Store.")
		return contentStyle.Width(width).Height(height).Render(msg)
	}

	searchBar := renderAppstoreSearchBar(s, width-2)
	status := renderAppstoreStatus(s, width-2)

	bodyH := height - lipgloss.Height(searchBar) - lipgloss.Height(status) - 1
	if bodyH < 6 {
		bodyH = 6
	}

	sidebarW := 22
	mainW := width - sidebarW - 2
	if mainW < 30 {
		mainW = 30
	}

	sidebar := renderAppstoreSidebar(s, sidebarW, bodyH)
	var main string
	if s.AppstoreDetailOpen {
		main = renderAppstoreDetailPane(s, mainW, bodyH)
	} else {
		main = renderAppstoreResults(s, mainW, bodyH)
	}

	middle := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	stack := lipgloss.JoinVertical(lipgloss.Left, searchBar, middle, status)

	return contentStyle.Width(width).Height(height).Render(stack)
}

// renderAppstoreStatus is the one-line footer under the main pane. It
// shows hint keys when there is no active message, and falls back to
// the current StatusMsg or a truncation / rate-limit warning when
// either is set. Always painted with the dim style so it reads as
// ambient chrome.
func renderAppstoreStatus(s *core.State, width int) string {
	var parts []string
	if s.AppstoreStatusMsg != "" {
		parts = append(parts, statusErrorStyle.Render(s.AppstoreStatusMsg))
	} else if s.AppstoreBusy {
		parts = append(parts, statusBusyStyle.Render("working…"))
	} else {
		hints := "↑↓ nav  /  search  enter open  R refresh  A toggle AUR  ?  help"
		parts = append(parts, statusBarStyle.Render(hints))
	}
	if res := s.AppstoreResults; res.Truncated {
		parts = append(parts,
			statusBarStyle.Render("  (results truncated)"))
	}
	if s.Appstore.AURLimit.Active {
		parts = append(parts,
			statusErrorStyle.Render("  AUR limited"))
	}
	line := strings.Join(parts, "")
	return lipgloss.NewStyle().Width(width).Render(line)
}

// handleAppstoreKey routes a key event to the correct handler based on
// which region of the App Store owns focus. Returns handled=true when
// the key was consumed so the outer handler skips its fallthrough. Key
// bindings that should always work (F-keys, ?, ctrl+c, ctrl+r) return
// handled=false so the global handler runs.
func (m Model) handleAppstoreKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	key := msg.String()

	// Global passthroughs — these never belong to the app store.
	switch key {
	case "ctrl+c", "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12", "?", "ctrl+r":
		return false, m, nil
	}

	// Search input mode swallows every other key so the user can type
	// a query that includes letters we'd otherwise treat as shortcuts.
	if m.state.AppstoreSearchActive {
		model, cmd := m.handleAppstoreSearchKey(msg)
		return true, model, cmd
	}

	switch key {
	case "esc":
		switch m.state.AppstoreFocus {
		case core.AppstoreFocusDetail:
			m.state.CloseAppstoreDetail()
		case core.AppstoreFocusResults:
			m.state.FocusAppstoreSidebar()
		default:
			return false, m, nil // let the global handler consider quit
		}
		return true, m, nil

	case "q":
		return false, m, nil // global quit

	case "/":
		m.state.OpenAppstoreSearch()
		return true, m, nil

	case "j", "down":
		m.moveAppstoreSelection(1)
		return true, m, nil

	case "k", "up":
		m.moveAppstoreSelection(-1)
		return true, m, nil

	case "enter":
		model, cmd := m.activateAppstoreFocus()
		return true, model, cmd

	case "R":
		if m.appstore.Refresh == nil {
			return true, m, nil
		}
		m.state.MarkAppstoreBusy()
		return true, m, m.appstore.Refresh()

	case "A":
		m.state.AppstoreIncludeAUR = !m.state.AppstoreIncludeAUR
		return true, m, nil
	}

	return true, m, nil
}

// handleAppstoreSearchKey runs while the search input has focus. It
// swallows every key; commit and cancel return to one of the other
// focus regions depending on whether results exist.
func (m Model) handleAppstoreSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.CloseAppstoreSearch()
		return m, nil
	case "enter":
		return m, m.submitAppstoreSearch()
	case "backspace":
		m.state.BackspaceAppstoreSearch()
		return m, nil
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

// moveAppstoreSelection walks the cursor within whichever focus region
// currently owns j/k. Sidebar navigation does not fire a search — the
// user commits with enter so rapid scrolling doesn't flood the daemon
// with requests.
func (m *Model) moveAppstoreSelection(delta int) {
	switch m.state.AppstoreFocus {
	case core.AppstoreFocusSidebar:
		m.state.MoveAppstoreCategory(delta)
	case core.AppstoreFocusResults:
		m.state.MoveAppstoreResult(delta)
	}
}

// activateAppstoreFocus is what enter does from each focus region.
// Sidebar → focus the results list (loading if needed). Results →
// fetch detail for the highlighted package. Search was handled in the
// input-mode branch already.
func (m Model) activateAppstoreFocus() (tea.Model, tea.Cmd) {
	switch m.state.AppstoreFocus {
	case core.AppstoreFocusSidebar:
		if !m.state.AppstoreResultsLoaded || m.state.AppstoreResults.Query.Category != m.categoryID() {
			cmd := m.loadAppstoreCategoryCmd()
			m.state.FocusAppstoreResults()
			return m, cmd
		}
		m.state.FocusAppstoreResults()
		return m, nil
	case core.AppstoreFocusResults:
		pkg, ok := m.state.SelectedAppstorePackage()
		if !ok || m.appstore.Detail == nil {
			return m, nil
		}
		m.state.MarkAppstoreBusy()
		return m, m.appstore.Detail(appstore.DetailRequest{
			Name:   pkg.Name,
			Origin: pkg.Origin,
		})
	}
	return m, nil
}

// loadAppstoreCategory fires a Search for the newly-selected sidebar
// category. Called on every cursor movement so the results pane tracks
// the sidebar in real time.
func (m *Model) loadAppstoreCategory() {
	// The command itself is dispatched via loadAppstoreCategoryCmd; this
	// wrapper exists so the sidebar nav can mark busy before the key
	// handler returns the cmd.
}

// loadAppstoreCategoryCmd builds the tea.Cmd that loads the currently
// selected sidebar category. Used by enter and by submitAppstoreSearch
// when the user clears the query.
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
		Limit:      150,
	}
	m.state.MarkAppstoreBusy()
	return m.appstore.Search(q)
}

// submitAppstoreSearch dispatches a text search with the current
// input. Empty input returns the user to the sidebar view for the
// currently selected category.
func (m *Model) submitAppstoreSearch() tea.Cmd {
	if m.appstore.Search == nil {
		return nil
	}
	text := strings.TrimSpace(m.state.AppstoreSearchInput)
	m.state.AppstoreSearchActive = false
	if text == "" {
		m.state.FocusAppstoreSidebar()
		return m.loadAppstoreCategoryCmd()
	}
	q := appstore.SearchQuery{
		Text:       text,
		IncludeAUR: m.state.AppstoreIncludeAUR,
		Limit:      200,
	}
	m.state.MarkAppstoreBusy()
	m.state.AppstoreFocus = core.AppstoreFocusResults
	return m.appstore.Search(q)
}

// categoryID returns the id of the currently highlighted sidebar
// category, or "" when the catalog isn't loaded yet.
func (m Model) categoryID() string {
	if cat, ok := m.state.SelectedAppstoreCategory(); ok {
		return cat.ID
	}
	return ""
}
