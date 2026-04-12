package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// groupBox wraps a single block of content in a rounded border with a
// short title spliced into the top. Content width inside the box is
// (total - 4) columns: two border glyphs plus two gutter spaces.
func groupBox(title, content string, total int) string {
	return groupBoxSections(title, []string{content}, total, colorBorder)
}

// groupBoxSections wraps multiple content blocks in a single rounded
// border, inserting a ├──┤ tee divider between each pair of blocks so
// the eye reads them as related sections of the same container.
// borderColor lets callers highlight the box when it holds focus.
func groupBoxSections(title string, sections []string, total int, borderColor lipgloss.Color) string {
	if total < 6 {
		total = 6
	}
	inner := total - 2
	contentW := inner - 2

	border := lipgloss.NewStyle().Foreground(borderColor)

	var topMid string
	if title != "" {
		label := " " + title + " "
		labelW := lipgloss.Width(label)
		if labelW+1 < inner {
			topMid = "─" + label + strings.Repeat("─", inner-1-labelW)
		} else {
			topMid = strings.Repeat("─", inner)
		}
	} else {
		topMid = strings.Repeat("─", inner)
	}

	var lines []string
	lines = append(lines, border.Render("╭"+topMid+"╮"))

	left := border.Render("│ ")
	right := border.Render(" │")
	tee := border.Render("├" + strings.Repeat("─", inner) + "┤")

	for i, section := range sections {
		if i > 0 {
			lines = append(lines, tee)
		}
		for _, raw := range strings.Split(section, "\n") {
			raw = lipgloss.NewStyle().MaxWidth(contentW).Render(raw)
			pad := contentW - lipgloss.Width(raw)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, left+raw+strings.Repeat(" ", pad)+right)
		}
	}

	lines = append(lines, border.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return strings.Join(lines, "\n")
}
