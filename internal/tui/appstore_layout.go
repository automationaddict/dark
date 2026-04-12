package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

// renderAppStoreTab is the top-level F2 renderer. It mirrors
// renderSettings exactly: a narrow sidebar on the left populated
// with the appstore categories, and a wide content area on the
// right showing the search bar, results, and detail for the
// highlighted category. Same styles, same proportions, same
// focus model.
func renderAppStoreTab(s *core.State, width, height int) string {
	sidebar := renderAppStoreSidebarColumn(s, height)
	contentWidth := width - lipgloss.Width(sidebar)
	content := renderAppStoreContent(s, contentWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// renderAppStoreSidebarColumn renders the categories using the exact
// same sidebarStyle / sidebarItem / sidebarItemActive styles that the
// F1 settings sidebar uses. Every item gets Icon + Label in the same
// layout as settings sections. Disabled categories use the regular
// item style but with dim foreground so they visually recede without
// changing size, font, or spacing.
func renderAppStoreSidebarColumn(s *core.State, height int) string {
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)

	dim := lipgloss.NewStyle().Foreground(colorDim)

	var rows []string
	for i, cat := range s.Appstore.Categories {
		line := cat.Icon + "  " + cat.Title
		if i == s.AppstoreCategoryIdx {
			rows = append(rows, active.Render(line))
		} else {
			if !cat.Enabled {
				line = dim.Render(line)
			}
			rows = append(rows, item.Render(line))
		}
	}
	body := strings.Join(rows, "\n")
	return sidebarStyle.Height(height).Render(body)
}

// renderAppStoreContent is the right-hand pane for F2, mirroring
// renderSettingsContent for F1. It stacks a search bar, the results
// or detail panel, and a one-line status footer.
func renderAppStoreContent(s *core.State, width, height int) string {
	if !s.AppstoreLoaded {
		return contentStyle.Width(width).Height(height).Render(
			placeholderStyle.Render("Loading package catalog…"))
	}

	searchBar := renderAppstoreSearchBar(s, width-6)
	status := renderAppstoreStatus(s, width-6)

	bodyH := height - lipgloss.Height(searchBar) - lipgloss.Height(status) - 2
	if bodyH < 6 {
		bodyH = 6
	}

	innerWidth := width - 6
	if innerWidth < 30 {
		innerWidth = 30
	}

	var mainPane string
	if s.AppstoreDetailOpen {
		mainPane = renderAppstoreDetailPane(s, innerWidth, bodyH)
	} else {
		mainPane = renderAppstoreResults(s, innerWidth, bodyH)
	}

	stack := lipgloss.JoinVertical(lipgloss.Left, searchBar, mainPane, status)
	return contentStyle.Width(width).Height(height).Render(stack)
}
