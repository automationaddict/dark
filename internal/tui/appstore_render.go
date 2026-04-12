package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appstore"
)

// renderAppstoreSearchBar draws the single-line search input at the
// top of the F2 tab. When the user is actively typing the cursor
// block is visible and the border switches to accent blue; otherwise
// the current query renders dimmed so the user can see what's
// filtering the results without mistaking it for input focus.
func renderAppstoreSearchBar(s *core.State, width int) string {
	border := colorBorder
	if s.AppstoreSearchActive {
		border = colorAccent
	}
	inputText := s.AppstoreSearchInput
	if s.AppstoreSearchActive {
		inputText += "▎"
	} else if inputText == "" {
		inputText = placeholderStyle.Render("(press / to search pacman and the AUR)")
	}

	right := appstoreAURBadge(s.AppstoreIncludeAUR)
	rightW := lipgloss.Width(right)
	leftW := width - rightW - 4
	if leftW < 10 {
		leftW = 10
	}

	left := lipgloss.NewStyle().Width(leftW).Render(inputText)
	body := left + "  " + right
	return groupBoxSections("Search", []string{body}, width, border)
}

// appstoreAURBadge renders a small right-aligned badge for whether
// AUR results are enabled. Accent when on, dim when off. Keeping it
// on the search row mirrors Cosmic-style "include sources" chips.
func appstoreAURBadge(includeAUR bool) string {
	if includeAUR {
		return lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("[aur: on]")
	}
	return lipgloss.NewStyle().Foreground(colorDim).Render("[aur: off]")
}

// renderAppstoreSidebar renders the categories box on the left of the
// main body. Disabled categories (the "real" named groups like
// Development, Graphics, etc. that we haven't populated yet) are
// dimmed and italicized so they read as "coming soon" without needing
// a separate explainer. The selection marker only appears on the
// currently highlighted row when the sidebar has focus.
func renderAppstoreSidebar(s *core.State, width, height int) string {
	border := colorBorder
	if s.AppstoreFocus == core.AppstoreFocusSidebar {
		border = colorAccent
	}
	cats := s.Appstore.Categories
	rows := make([]string, 0, len(cats))

	for i, cat := range cats {
		line := appstoreCategoryLine(cat, i == s.AppstoreCategoryIdx && s.AppstoreFocus == core.AppstoreFocusSidebar)
		rows = append(rows, line)
	}

	// Pad to a minimum height so the sidebar doesn't visually collapse
	// when there are fewer categories than body rows.
	for len(rows) < height-2 {
		rows = append(rows, "")
	}

	body := strings.Join(rows, "\n")
	return groupBoxSections("Categories", []string{body}, width, border)
}

// appstoreCategoryLine formats one category row: a selection caret,
// the title, and the count in parens for enabled categories. Disabled
// categories are italic + dim and omit the count since "(0)" would be
// misleading for placeholder rows.
func appstoreCategoryLine(cat appstore.Category, selected bool) string {
	marker := "  "
	if selected {
		marker = tableSelectionMarker.Render("▸ ")
	}
	if !cat.Enabled {
		label := lipgloss.NewStyle().Foreground(colorDim).Italic(true).Render(cat.Title)
		return marker + label
	}
	title := cat.Title
	if selected {
		title = tableCellSelected.Render(title)
	} else {
		title = tableCellStyle.Render(title)
	}
	if cat.Count > 0 {
		count := lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf("  (%d)", cat.Count))
		return marker + title + count
	}
	return marker + title
}

// renderAppstoreResults is the right-hand pane when the user is
// browsing: one card per package with name, version, description,
// size, and install-state. The currently selected row has an accent
// caret and accent-colored name; everything else renders with the
// standard cell style.
func renderAppstoreResults(s *core.State, width, height int) string {
	border := colorBorder
	if s.AppstoreFocus == core.AppstoreFocusResults {
		border = colorAccent
	}

	title := "Results"
	if cat, ok := s.SelectedAppstoreCategory(); ok && s.AppstoreResults.Query.Text == "" {
		title = cat.Title
	} else if s.AppstoreResults.Query.Text != "" {
		title = "Search: " + s.AppstoreResults.Query.Text
	}

	if !s.AppstoreResultsLoaded {
		body := placeholderStyle.Render(
			"Press Enter on a category or / to search.")
		return groupBoxSections(title, []string{body}, width, border)
	}

	pkgs := s.AppstoreResults.Packages
	if len(pkgs) == 0 {
		body := placeholderStyle.Render("No packages found.")
		return groupBoxSections(title, []string{body}, width, border)
	}

	innerW := width - 4
	rows := make([]string, 0, len(pkgs))
	maxRows := height - 2
	if maxRows < 1 {
		maxRows = 1
	}

	// Simple windowing: keep the selected row on screen. We don't try
	// to animate or preserve a center offset — the user's focus is
	// what matters, everything else follows it.
	start := 0
	if s.AppstoreResultIdx >= maxRows {
		start = s.AppstoreResultIdx - maxRows + 1
	}
	end := start + maxRows
	if end > len(pkgs) {
		end = len(pkgs)
	}

	for i := start; i < end; i++ {
		rows = append(rows, renderAppstoreResultRow(pkgs[i], i == s.AppstoreResultIdx, innerW))
	}
	// Pad to full height so the box doesn't shrink.
	for len(rows) < maxRows {
		rows = append(rows, "")
	}

	body := strings.Join(rows, "\n")
	return groupBoxSections(title, []string{body}, width, border)
}

// renderAppstoreResultRow formats one line in the results list.
// Layout: caret, name, version, size, origin badge, description
// trailing to fill. The name column is fixed-width so the eye can
// scan down it without zigzagging.
func renderAppstoreResultRow(p appstore.Package, selected bool, width int) string {
	marker := "  "
	if selected {
		marker = tableSelectionMarker.Render("▸ ")
	}

	nameCol := 24
	sizeCol := 10
	badgeCol := 10

	name := p.Name
	if selected {
		name = tableCellSelected.Render(fitWidth(name, nameCol))
	} else if p.Installed {
		name = tableCellAccent.Render(fitWidth(name, nameCol))
	} else {
		name = tableCellStyle.Render(fitWidth(name, nameCol))
	}

	size := lipgloss.NewStyle().Foreground(colorDim).Render(
		fitWidth(appstore.HumanSize(p.InstalledSize), sizeCol))

	badge := appstoreOriginBadge(p, badgeCol)

	descW := width - lipgloss.Width(marker) - nameCol - sizeCol - badgeCol - 3
	if descW < 10 {
		descW = 10
	}
	desc := lipgloss.NewStyle().Foreground(colorDim).Render(
		appstore.TruncateDesc(p.Description, descW))

	return marker + name + " " + size + " " + badge + " " + desc
}

// appstoreOriginBadge renders the small right-side tag that shows
// where a result came from and whether it's already installed.
// Matches the badge style used on the search bar for consistency.
func appstoreOriginBadge(p appstore.Package, width int) string {
	var text string
	var style lipgloss.Style
	switch {
	case p.Installed:
		text = "installed"
		style = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	case p.Origin == appstore.OriginAUR:
		text = "AUR"
		style = lipgloss.NewStyle().Foreground(colorGold)
	default:
		text = p.Repo
		if text == "" {
			text = "pacman"
		}
		style = lipgloss.NewStyle().Foreground(colorDim)
	}
	return style.Render(fitWidth(text, width))
}

// fitWidth pads or truncates s to exactly w display columns. Used for
// column alignment in the results list and sidebar.
func fitWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	width := lipgloss.Width(s)
	if width == w {
		return s
	}
	if width > w {
		runes := []rune(s)
		for len(runes) > 0 && lipgloss.Width(string(runes)) > w {
			runes = runes[:len(runes)-1]
		}
		return string(runes)
	}
	return s + strings.Repeat(" ", w-width)
}
