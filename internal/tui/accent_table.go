package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// accentColumn declares one column of an accentTable. value produces
// the display string for a row, and accent (optional) decides whether
// the cell should use the accent style — used to highlight hot-path
// values like "connected" or "powered on" without forcing the caller
// to precompute styling.
type accentColumn[T any] struct {
	header string
	value  func(T) string
	accent func(T) bool
}

// accentRowMarker builds the left-gutter prefix for a row. Callers
// supply their own so the wifi networks table can show a ★ for the
// connected SSID, the wifi adapters table can show just the selection
// arrow, and so on. selected/focused describe the table-wide state;
// the row itself is passed so markers can react to per-row flags.
type accentRowMarker[T any] func(selected, focused bool, row T) string

// defaultRowMarker is the simple " "/"▸ " marker used by any table
// that doesn't need a status glyph. It dims the arrow when the table
// is unfocused so the row still reads as "selected but not active".
func defaultRowMarker[T any](selected, focused bool, _ T) string {
	switch {
	case selected && focused:
		return tableSelectionMarker.Render("▸ ")
	case selected:
		return tableSelectionMarkerDim.Render("▸ ")
	default:
		return "  "
	}
}

// renderAccentTable renders a selectable, column-aligned table.
//
// cols declares the columns; rows is the data. selected is the index
// of the highlighted row (−1 to disable selection). focused controls
// whether the selection renders as active. rowAccent, if non-nil,
// promotes an entire row to the accent style regardless of column
// accent rules (used for "currently connected" rows). marker
// overrides the left-gutter prefix per row; nil picks the default.
//
// Column widths are sized to fit the widest of {header, every cell}.
// Cells are padded with a two-space gap to match the existing
// wifi/bluetooth/network tables so a drop-in swap is visually
// identical.
func renderAccentTable[T any](
	cols []accentColumn[T],
	rows []T,
	selected int,
	focused bool,
	rowAccent func(T) bool,
	marker accentRowMarker[T],
) string {
	if marker == nil {
		marker = defaultRowMarker[T]
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, r := range rows {
		for i, c := range cols {
			if w := lipgloss.Width(c.value(r)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	// Pad the header to match the row marker width so columns line up
	// no matter how wide the marker is. Sampling the marker with a
	// zero row value is safe because markers always produce the
	// "unselected, unfocused" prefix in that case.
	var zero T
	headerIndent := strings.Repeat(" ", lipgloss.Width(marker(false, false, zero)))
	lines := []string{headerIndent + strings.Join(headerCells, gap)}

	for i, row := range rows {
		isSel := selected >= 0 && i == selected
		rowIsAccent := rowAccent != nil && rowAccent(row)

		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.value(row)
			var style lipgloss.Style
			switch {
			case isSel:
				style = tableCellSelected
			case rowIsAccent:
				style = tableCellAccent
			case c.accent != nil && c.accent(row):
				style = tableCellAccent
			default:
				style = tableCellStyle
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, marker(isSel, focused, row)+strings.Join(cells, gap))
	}
	return strings.Join(lines, "\n")
}
