package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/help"
)

func renderHelpPanel(s *core.State, height int) string {
	doc := s.HelpDoc
	if doc == nil {
		return helpPanelStyle.Width(s.HelpWidth).Height(height).Render(
			helpPanelErrorStyle.Render("no help available"),
		)
	}

	innerWidth := s.HelpWidth - 4 // padding(0,2) on panel style
	if innerWidth < 1 {
		innerWidth = 1
	}

	header := renderHelpHeader(doc, innerWidth)
	toc := renderHelpTOC(doc, s.HelpScroll, innerWidth)
	footer := renderHelpFooter(s, innerWidth)

	usedByChrome := lipgloss.Height(header) + lipgloss.Height(toc) + lipgloss.Height(footer) + 2
	bodyHeight := height - usedByChrome
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	body := renderHelpBody(doc, s.HelpScroll, s.HelpMatches, innerWidth, bodyHeight)

	stack := lipgloss.JoinVertical(lipgloss.Left,
		header,
		helpDividerStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth)),
		toc,
		helpDividerStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth)),
		body,
		footer,
	)

	return helpPanelStyle.Width(s.HelpWidth).Height(height).Render(stack)
}

func renderHelpHeader(doc *help.Document, width int) string {
	return helpTitleStyle.Width(width).Render("Help · " + doc.Title)
}

// renderHelpTOC shows a compact one-line indicator of where the scroll
// position lands in the document's heading tree. Full section lists
// would dominate a drawer that's only ~30 rows tall for long docs, so
// the header is kept to a single line and navigation between sections
// happens via [ / ] (prev/next heading) and / (search).
func renderHelpTOC(doc *help.Document, scroll, width int) string {
	if len(doc.TOC) == 0 {
		return helpTocMutedStyle.Width(width).Render("(no sections)")
	}

	current := 0
	for i, e := range doc.TOC {
		if e.Line <= scroll {
			current = i
		} else {
			break
		}
	}

	entry := doc.TOC[current]
	position := fmt.Sprintf("%d/%d", current+1, len(doc.TOC))
	pos := helpTocItemStyle.Render(position)
	title := helpTocItemActiveStyle.Render(entry.Title)

	combined := pos + "  " + title
	// Truncate if the title plus the counter would overflow.
	if lipgloss.Width(combined) > width {
		combined = truncate.String(combined, uint(width))
	}
	return helpTocItemStyle.Width(width).Render(combined)
}

func renderHelpBody(doc *help.Document, scroll int, matches []int, width, height int) string {
	if height <= 0 {
		return ""
	}
	start := scroll
	end := start + height
	if end > len(doc.Lines) {
		end = len(doc.Lines)
	}
	if start > end {
		start = end
	}

	matchSet := make(map[int]struct{}, len(matches))
	for _, m := range matches {
		matchSet[m] = struct{}{}
	}

	bodyBg := help.BodyBgEscape()
	emptyRow := bodyBg + strings.Repeat(" ", width) + "\x1b[0m"

	rows := make([]string, 0, height)
	for i := start; i < end; i++ {
		line := truncate.String(doc.Lines[i], uint(width))
		if _, ok := matchSet[i]; ok {
			line = helpMatchStyle.Width(width).Render(stripForHighlight(line))
		}
		rows = append(rows, padBodyLine(line, width, bodyBg))
	}
	for len(rows) < height {
		rows = append(rows, emptyRow)
	}
	return strings.Join(rows, "\n")
}

// padBodyLine fills a body line out to the target width so the darker body
// background extends cleanly to the right edge of the panel's inner area.
// lipgloss.Width reports the visible-cell count regardless of embedded ANSI.
func padBodyLine(line string, width int, bodyBg string) string {
	w := lipgloss.Width(line)
	if w >= width {
		return line
	}
	return line + bodyBg + strings.Repeat(" ", width-w) + "\x1b[0m"
}

func renderHelpFooter(s *core.State, width int) string {
	if s.HelpSearchMode {
		prompt := "/ " + s.HelpSearchQuery + "_"
		return helpSearchStyle.Width(width).Render(truncate.String(prompt, uint(width)))
	}

	var text string
	switch {
	case len(s.HelpMatches) > 0:
		text = fmt.Sprintf("match %d/%d  · n/N next · / new search · ? close",
			s.HelpMatchIdx+1, len(s.HelpMatches))
	default:
		text = "j/k scroll · [/] section · / search · ± resize · ? close"
	}
	return helpFooterStyle.Width(width).Render(truncate.String(text, uint(width)))
}

// stripForHighlight removes ANSI so the match line can be re-styled cleanly.
func stripForHighlight(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
