package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

func renderStatusBar(s *core.State, width int) string {
	// The "rebuilding…" indicator stays inline because it's transient
	// state, but build failures fire desktop notifications and don't
	// take over the status bar anymore.
	if s.Rebuilding {
		return statusBusyStyle.Width(width).Render("rebuilding…")
	}

	left := "? help · ctrl+r rebuild · +/- resize sidebar · F1–F12 tabs · q quit"
	right := connectionIndicator(s.BusConnected)
	return statusBarStyle.Width(width).Render(layoutStatusRow(left, right, width))
}

// layoutStatusRow places the left-aligned hint text and the right-aligned
// connection indicator on a single line that fits inside the status bar's
// inner content area. statusBarStyle uses Padding(0, 1) so the usable inner
// width is total - 2.
func layoutStatusRow(left, right string, totalWidth int) string {
	inner := totalWidth - 2 // account for the status bar's horizontal padding
	if inner < 0 {
		inner = 0
	}
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	if leftW+rightW+1 > inner {
		// Not enough room for both — drop the hint, keep the indicator.
		return strings.Repeat(" ", max0(inner-rightW)) + right
	}
	gap := inner - leftW - rightW
	return left + strings.Repeat(" ", gap) + right
}

func connectionIndicator(connected bool) string {
	if connected {
		return statusOnlineStyle.Render("● connected")
	}
	return statusOfflineStyle.Render("● disconnected")
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
