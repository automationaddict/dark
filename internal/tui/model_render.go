package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/truncate"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/help"
)

func (m Model) View() string {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 40
	}

	tabBar := renderTabBar(m.state, width)
	statusBar := renderStatusBar(m.state, width)
	bodyHeight := height - lipgloss.Height(tabBar) - lipgloss.Height(statusBar)

	var body string
	switch m.state.ActiveTab {
	case core.TabSettings:
		body = renderSettings(m.state, width, bodyHeight)
	case core.TabF2:
		body = renderAppStoreTab(m.state, width, bodyHeight, m.spinner.View())
	case core.TabF3:
		body = renderOmarchyTab(m.state, width, bodyHeight)
	case core.TabF4:
		body = renderSSHTab(m.state, width, bodyHeight)
	case core.TabF5:
		body = renderScriptingTab(m.state, width, bodyHeight)
	default:
		body = renderEmpty(m.state, width, bodyHeight)
	}

	base := appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, body, statusBar, tabBar))

	if m.state.HelpOpen {
		chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(tabBar)
		panelHeight := height - chromeHeight
		if panelHeight < 3 {
			panelHeight = 3
		}
		panel := renderHelpPanel(m.state, panelHeight)
		panel = help.ReapplyPanelBackground(panel)
		base = overlayRight(base, panel, width, m.state.HelpWidth)
	}

	if m.dialog != nil {
		return overlayCenter(base, m.dialog.View(), width, height)
	}
	return base
}

// overlayRight composes the help panel onto the right portion of the base
// view. Each base line is ANSI-truncated to (totalWidth - panelWidth) visible
// columns then concatenated with the corresponding panel line.
func overlayRight(base, panel string, totalWidth, panelWidth int) string {
	if panelWidth <= 0 || panelWidth >= totalWidth {
		return panel
	}
	keep := totalWidth - panelWidth
	baseLines := strings.Split(base, "\n")
	panelLines := strings.Split(panel, "\n")

	n := len(baseLines)
	if len(panelLines) > n {
		n = len(panelLines)
	}

	out := make([]string, n)
	for i := 0; i < n; i++ {
		// Rows below the panel pass through untouched so the status bar and
		// tab bar remain visible across the full terminal width.
		if i >= len(panelLines) {
			if i < len(baseLines) {
				out[i] = baseLines[i]
			}
			continue
		}
		var left string
		if i < len(baseLines) {
			left = truncate.String(baseLines[i], uint(keep))
		}
		leftW := lipgloss.Width(left)
		if leftW < keep {
			left += strings.Repeat(" ", keep-leftW)
		}
		out[i] = left + panelLines[i]
	}
	return strings.Join(out, "\n")
}

// overlayCenter composes an overlay (typically a dialog box) on top of
// the base view, centered horizontally and vertically. Each row the
// overlay occupies is rebuilt as base[:left] + overlay + base[left+oW:]
// with ANSI escapes preserved on both sides, so the sidebar and other
// content to the left and right of the dialog stay visible. Rows above
// and below the overlay pass through the base untouched.
func overlayCenter(base, overlay string, totalWidth, totalHeight int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	oH := len(overlayLines)
	oW := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > oW {
			oW = w
		}
	}
	if oW == 0 || oH == 0 {
		return base
	}

	top := (totalHeight - oH) / 2
	if top < 0 {
		top = 0
	}
	left := (totalWidth - oW) / 2
	if left < 0 {
		left = 0
	}

	out := make([]string, len(baseLines))
	copy(out, baseLines)

	for i, oLine := range overlayLines {
		y := top + i
		if y >= len(out) {
			break
		}
		baseLine := ""
		if y < len(baseLines) {
			baseLine = baseLines[y]
		}

		// Left slice: first `left` visible cells of base, padded with
		// spaces if the base line is shorter than the dialog column.
		leftPart := truncate.String(baseLine, uint(left))
		if padNeeded := left - lipgloss.Width(leftPart); padNeeded > 0 {
			leftPart += strings.Repeat(" ", padNeeded)
		}

		// Right slice: drop the first (left + oW) visible cells from the
		// base line, keep the rest. A leading reset prevents ANSI state
		// leaking out of the dialog onto the resumed base content.
		rightPart := ansiSkipCells(baseLine, left+oW)
		if rightPart != "" {
			rightPart = "\x1b[0m" + rightPart
		}

		out[y] = leftPart + "\x1b[0m" + oLine + rightPart
	}

	return strings.Join(out, "\n")
}

// ansiSkipCells returns the tail of s after the first `skip` visible
// cells, preserving every ANSI escape sequence that appears in the
// skipped prefix so the tail inherits the correct styling state.
// Visible width is measured by runewidth, the same library lipgloss
// uses, so this agrees with lipgloss.Width.
func ansiSkipCells(s string, skip int) string {
	if skip <= 0 {
		return s
	}
	total := lipgloss.Width(s)
	if skip >= total {
		return ""
	}

	var b strings.Builder
	visible := 0
	i := 0
	for i < len(s) {
		// CSI escape sequence: ESC [ ... <final>
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			// Always emit style escapes so the tail carries color state.
			b.WriteString(s[i:j])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if size == 0 {
			i++
			continue
		}
		w := runewidth.RuneWidth(r)
		if visible >= skip {
			b.WriteRune(r)
		}
		visible += w
		i += size
	}
	return b.String()
}
