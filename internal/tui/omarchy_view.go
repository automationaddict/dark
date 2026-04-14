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
	case "links":
		content = renderLinksSection(s, contentWidth, height)
	case "keybindings":
		content = renderKeybindings(s, contentWidth, height)
	case "limine":
		content = renderLimineSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented yet."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderLinksSection(s *core.State, width, height int) string {
	secs := core.LinksSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.OmarchyLinksFocused
	sidebar := renderInnerSidebarFocused(s, entries, s.OmarchyLinksIdx, height, sidebarFocused)
	innerWidth := width - lipgloss.Width(sidebar)

	var content string
	switch s.ActiveLinksSection().ID {
	case "weblinks":
		content = renderWebLinks(s, innerWidth, height)
	case "tuilinks":
		content = renderTUILinks(s, innerWidth, height)
	case "helplinks":
		content = renderHelpLinks(s, innerWidth, height)
	default:
		content = renderContentPane(innerWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderWebLinks(s *core.State, width, height int) string {
	if !s.LinksLoaded {
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

	selectedCell := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	var data [][]string
	for i, app := range s.WebLinks {
		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			app.Name,
			app.URL,
		})
	}

	table := renderTable(
		[]string{"#", "Name", "URL"},
		[]int{numW, nameW, urlW},
		data,
		s.WebLinkIdx, s.OmarchyLinksFocused, selectedCell,
	)

	hint := lipgloss.NewStyle().Foreground(colorDim).Render(
		"enter open · a add · e edit · d delete")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "", table, "", hint)

	return renderContentPane(width, height, body)
}

func renderTUILinks(s *core.State, width, height int) string {
	if !s.LinksLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading TUI links…"))
	}

	if len(s.TUILinks) == 0 {
		return renderContentPane(width, height,
			placeholderStyle.Render("No TUI apps installed."))
	}

	innerWidth := width - 6
	if innerWidth < 30 {
		innerWidth = 30
	}

	title := contentTitle.Render("TUI Links")

	numW := 5
	nameW := 20
	styleW := 8
	cmdW := innerWidth - numW - nameW - styleW - 5
	if cmdW < 20 {
		cmdW = 20
	}

	selectedCell := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	var data [][]string
	for i, app := range s.TUILinks {
		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			app.Name,
			app.Command,
			app.Style,
		})
	}

	table := renderTable(
		[]string{"#", "Name", "Command", "Style"},
		[]int{numW, nameW, cmdW, styleW},
		data,
		s.TUILinkIdx, s.OmarchyLinksFocused, selectedCell,
	)

	hint := lipgloss.NewStyle().Foreground(colorDim).Render(
		"enter launch · a add · e edit · d delete")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "", table, "", hint)

	return renderContentPane(width, height, body)
}

func renderHelpLinks(s *core.State, width, height int) string {
	if !s.LinksLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading help links…"))
	}

	if len(s.HelpLinks) == 0 {
		return renderContentPane(width, height,
			placeholderStyle.Render("No help links. Press a to add one."))
	}

	innerWidth := width - 6
	if innerWidth < 30 {
		innerWidth = 30
	}

	title := contentTitle.Render("Help Links")

	numW := 5
	nameW := 20
	urlW := innerWidth - numW - nameW - 4
	if urlW < 20 {
		urlW = 20
	}

	selectedCell := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	var data [][]string
	for i, link := range s.HelpLinks {
		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			link.Name,
			link.URL,
		})
	}

	table := renderTable(
		[]string{"#", "Name", "URL"},
		[]int{numW, nameW, urlW},
		data,
		s.HelpLinkIdx, s.OmarchyLinksFocused, selectedCell,
	)

	hint := lipgloss.NewStyle().Foreground(colorDim).Render(
		"enter open · a add · e edit · d delete")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "", table, "", hint)

	return renderContentPane(width, height, body)
}

func renderTable(headers []string, colWidths []int, data [][]string, selectedIdx int, focused bool, selectedStyle lipgloss.Style) string {
	bf := lipgloss.NewStyle().Foreground(colorBorder)
	sep := bf.Render("│")

	buildBorder := func(left, mid, right, fill string) string {
		var b strings.Builder
		b.WriteString(left)
		for i, w := range colWidths {
			b.WriteString(strings.Repeat(fill, w))
			if i < len(colWidths)-1 {
				b.WriteString(mid)
			}
		}
		b.WriteString(right)
		return bf.Render(b.String())
	}

	topBorder := buildBorder("┌", "┬", "┐", "─")
	headerDivider := buildBorder("├", "┼", "┤", "─")
	bottomBorder := buildBorder("└", "┴", "┘", "─")

	var header strings.Builder
	header.WriteString(sep)
	for i, h := range headers {
		header.WriteString(tableHeaderStyle.Render(padCell(h, colWidths[i])))
		header.WriteString(sep)
	}

	var rows []string
	for ri, row := range data {
		cell := tableCellStyle
		if ri == selectedIdx && focused {
			cell = selectedStyle
		}

		var line strings.Builder
		line.WriteString(sep)
		for ci, val := range row {
			w := colWidths[ci]
			truncated := truncateStr(val, w-2)
			line.WriteString(cell.Render(padCell(truncated, w)))
			line.WriteString(sep)
		}
		rows = append(rows, line.String())
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		topBorder,
		header.String(),
		headerDivider,
		strings.Join(rows, "\n"),
		bottomBorder)
}

func padCell(s string, width int) string {
	visW := lipgloss.Width(s)
	content := " " + s
	pad := width - visW - 1
	if pad > 0 {
		content += strings.Repeat(" ", pad)
	}
	return content
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
