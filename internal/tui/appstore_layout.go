package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

// renderAppStoreTab is the top-level F2 renderer. It mirrors
// renderSettings exactly: a narrow sidebar on the left populated
// with the appstore categories, and a wide content area on the
// right showing the search bar, results, and detail for the
// highlighted category. Same styles, same proportions, same
// focus model.
func renderAppStoreTab(s *core.State, width, height int, spinnerView string) string {
	// Build sidebar: appstore categories, then Updates at the bottom with separator.
	var entries []sidebarEntry
	for _, cat := range s.Appstore.Categories {
		entries = append(entries, sidebarEntry{Icon: cat.Icon, Label: cat.Title, Enabled: cat.Enabled})
	}
	updatesIdx := len(entries)
	entries = append(entries, sidebarEntry{Icon: "󰚰", Label: "Updates", Enabled: true, Separator: true})

	sidebar := renderSidebarGeneric(s, entries, s.F2SidebarIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	var content string
	if s.F2SidebarIdx == updatesIdx {
		content = renderUpdateContent(s, contentWidth, height)
	} else {
		content = renderAppStoreContent(s, contentWidth, height, spinnerView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// renderAppStoreContent is the right-hand pane for F2. Same pattern
// as renderWifi / renderBluetooth / renderNetwork: build blocks, join
// vertically, wrap once in contentStyle.Height. No intermediate
// height constraints on sub-components — contentStyle clips the whole
// thing to the correct height.
func renderAppStoreContent(s *core.State, width, height int, spinnerView string) string {
	if !s.AppstoreLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading package catalog…"))
	}

	innerWidth := width - 6
	if innerWidth < 30 {
		innerWidth = 30
	}

	searchBar := renderAppstoreSearchBar(s, innerWidth)
	status := renderAppstoreStatus(s, innerWidth, spinnerView)

	// contentStyle has Padding(1, 3) → 2 vertical padding rows.
	// The search bar, two blank-line separators, and the status footer
	// consume fixed vertical space. The results/detail pane gets
	// whatever remains.
	searchH := lipgloss.Height(searchBar)
	statusH := lipgloss.Height(status)
	fixedH := 2 + searchH + 2 + statusH // padding + search + gaps + status
	resultRows := height - fixedH
	if resultRows < 3 {
		resultRows = 3
	}

	var mainPane string
	if s.AppstoreDetailOpen {
		mainPane = renderAppstoreDetailPane(s, innerWidth, resultRows)
	} else {
		mainPane = renderAppstoreResults(s, innerWidth, resultRows)
	}

	blocks := []string{searchBar, "", mainPane, "", status}
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height,body)
}
