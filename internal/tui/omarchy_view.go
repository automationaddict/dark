package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

func renderOmarchyTab(s *core.State, width, height int) string {
	secs := core.OmarchySections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderSidebarGeneric(s, entries, s.OmarchySidebarIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	var content string
	switch s.ActiveOmarchySection().ID {
	case "weblinks":
		content = renderWebLinks(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented yet."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderWebLinks(s *core.State, width, height int) string {
	if !s.WebLinksLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading web links…"))
	}

	if len(s.WebLinks) == 0 {
		return renderContentPane(width, height,
			placeholderStyle.Render("No web apps installed."))
	}

	innerWidth := width - 6
	if innerWidth < 30 {
		innerWidth = 30
	}

	title := contentTitle.Render("Web Links")

	numW := 5
	nameW := 20
	urlW := innerWidth - numW - nameW - 4
	if urlW < 20 {
		urlW = 20
	}
	tableW := numW + nameW + urlW + 4

	bf := lipgloss.NewStyle().Foreground(colorBorder)
	sep := bf.Render("│")

	topBorder := bf.Render("┌" + strings.Repeat("─", numW) + "┬" +
		strings.Repeat("─", nameW) + "┬" +
		strings.Repeat("─", urlW) + "┐")

	headerDivider := bf.Render("├" + strings.Repeat("─", numW) + "┼" +
		strings.Repeat("─", nameW) + "┼" +
		strings.Repeat("─", urlW) + "┤")

	bottomBorder := bf.Render("└" + strings.Repeat("─", numW) + "┴" +
		strings.Repeat("─", nameW) + "┴" +
		strings.Repeat("─", urlW) + "┘")

	_ = tableW

	header := sep +
		tableHeaderStyle.Render(fmt.Sprintf(" %-*s", numW-1, "#")) + sep +
		tableHeaderStyle.Render(fmt.Sprintf(" %-*s", nameW-1, "Name")) + sep +
		tableHeaderStyle.Render(fmt.Sprintf(" %-*s", urlW-1, "URL")) + sep

	selectedCell := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	var rows []string
	for i, app := range s.WebLinks {
		name := truncateStr(app.Name, nameW-2)
		url := truncateStr(app.URL, urlW-2)
		num := fmt.Sprintf("%d", i+1)

		cell := tableCellStyle
		if i == s.OmarchyFocusIdx && s.ContentFocused {
			cell = selectedCell
		}

		row := sep +
			cell.Render(fmt.Sprintf(" %-*s", numW-1, num)) + sep +
			cell.Render(fmt.Sprintf(" %-*s", nameW-1, name)) + sep +
			cell.Render(fmt.Sprintf(" %-*s", urlW-1, url)) + sep
		rows = append(rows, row)
	}

	hint := lipgloss.NewStyle().Foreground(colorDim).Render(
		"enter open · a add · e edit · d delete")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		topBorder,
		header,
		headerDivider,
		strings.Join(rows, "\n"),
		bottomBorder,
		"", hint)

	return renderContentPane(width, height, body)
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
