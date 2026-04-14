package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderDetailRows renders a list of [label, value] pairs as aligned
// columns. Labels are right-padded to the widest label; empty rows
// (both cells blank) become blank lines. Used by every detail panel
// that shows a key/value table (wifi adapter info, bluetooth device
// info, audio card properties, network interface details, etc.).
func renderDetailRows(rows [][2]string) string {
	if len(rows) == 0 {
		return ""
	}
	labelWidth := 0
	for _, r := range rows {
		if r[0] == "" {
			continue
		}
		if w := lipgloss.Width(r[0]); w > labelWidth {
			labelWidth = w
		}
	}
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		if r[0] == "" && r[1] == "" {
			lines = append(lines, "")
			continue
		}
		label := detailLabelStyle.Width(labelWidth + 2).Render(r[0])
		value := detailValueStyle.Render(orDash(r[1]))
		lines = append(lines, label+value)
	}
	return strings.Join(lines, "\n")
}
