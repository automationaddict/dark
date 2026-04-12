package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/display"
)

// renderDisplayLayout draws a scaled bird's-eye view of the monitor
// arrangement. Each monitor is a rectangle whose proportions reflect
// the real pixel dimensions and positions. The selected monitor is
// highlighted and can be nudged with arrow keys.
func renderDisplayLayout(s *core.State, totalWidth, totalHeight int) string {
	monitors := s.Display.Monitors
	if len(monitors) == 0 {
		return groupBoxSections("Arrangement", []string{
			placeholderStyle.Render("No monitors to arrange."),
		}, totalWidth, colorAccent)
	}

	canvasW := totalWidth - 8
	canvasH := totalHeight - 10
	if canvasW < 30 {
		canvasW = 30
	}
	if canvasH < 10 {
		canvasH = 10
	}

	grid := renderMonitorGrid(monitors, s.DisplayMonitorIdx, canvasW, canvasH)

	var sections []string
	sections = append(sections, grid)

	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	legend := dim.Render("  " +
		accent.Render("←→↑↓") + " move  " +
		accent.Render("j/k") + " select  " +
		accent.Render("i") + " identify  " +
		accent.Render("esc") + " back")
	sections = append(sections, legend)

	return groupBoxSections("Arrangement", sections, totalWidth, colorAccent)
}

// renderMonitorGrid builds the scaled ASCII grid. Returns a string of
// exactly canvasH lines, each canvasW columns wide.
func renderMonitorGrid(monitors []display.Monitor, selected, canvasW, canvasH int) string {
	// Find the bounding box of all monitors in pixel space.
	minX, minY := monitors[0].X, monitors[0].Y
	maxX, maxY := monitors[0].X+monitors[0].Width, monitors[0].Y+monitors[0].Height
	for _, m := range monitors[1:] {
		if m.X < minX {
			minX = m.X
		}
		if m.Y < minY {
			minY = m.Y
		}
		right := m.X + m.Width
		bottom := m.Y + m.Height
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	totalPxW := maxX - minX
	totalPxH := maxY - minY
	if totalPxW <= 0 {
		totalPxW = 1
	}
	if totalPxH <= 0 {
		totalPxH = 1
	}

	// Scale factor: fit the bounding box into the canvas with some
	// padding. Use the tighter axis so nothing overflows.
	padW := canvasW - 4
	padH := canvasH - 2
	if padW < 10 {
		padW = 10
	}
	if padH < 5 {
		padH = 5
	}

	scaleX := float64(padW) / float64(totalPxW)
	scaleY := float64(padH) / float64(totalPxH)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	type rect struct {
		x, y, w, h int
		label       string
		selected    bool
		idx         int
	}

	rects := make([]rect, len(monitors))
	for i, m := range monitors {
		r := rect{
			x:        int(float64(m.X-minX) * scale),
			y:        int(float64(m.Y-minY) * scale),
			w:        int(float64(m.Width) * scale),
			h:        int(float64(m.Height) * scale),
			selected: i == selected,
			idx:      i,
		}
		if r.w < 8 {
			r.w = 8
		}
		if r.h < 3 {
			r.h = 3
		}
		name := m.Name
		if m.Description != "" {
			name = m.Description
		}
		r.label = fmt.Sprintf(" %d: %s ", i+1, name)
		if len(r.label) > r.w-2 {
			r.label = fmt.Sprintf(" %d: %s ", i+1, m.Name)
		}
		if len(r.label) > r.w-2 {
			r.label = fmt.Sprintf(" %d ", i+1)
		}
		rects[i] = r
	}

	// Paint onto a character grid.
	gridW := canvasW
	gridH := canvasH
	grid := make([][]rune, gridH)
	color := make([][]int, gridH) // -1 = bg, monitor idx otherwise
	for y := 0; y < gridH; y++ {
		grid[y] = make([]rune, gridW)
		color[y] = make([]int, gridW)
		for x := 0; x < gridW; x++ {
			grid[y][x] = ' '
			color[y][x] = -1
		}
	}

	for _, r := range rects {
		paintRect(grid, color, r.x+2, r.y+1, r.w, r.h, r.idx, r.label)
	}

	// Render to styled string.
	selectedStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(colorText)
	dimStyle := lipgloss.NewStyle().Foreground(colorDim)
	selBorderStyle := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string
	for y := 0; y < gridH; y++ {
		var line strings.Builder
		for x := 0; x < gridW; x++ {
			ch := string(grid[y][x])
			ci := color[y][x]
			if ci < 0 {
				line.WriteString(dimStyle.Render(ch))
			} else if rects[ci].selected {
				isBorder := grid[y][x] == '┌' || grid[y][x] == '┐' ||
					grid[y][x] == '└' || grid[y][x] == '┘' ||
					grid[y][x] == '─' || grid[y][x] == '│'
				if isBorder {
					line.WriteString(selBorderStyle.Render(ch))
				} else {
					line.WriteString(selectedStyle.Render(ch))
				}
			} else {
				isBorder := grid[y][x] == '┌' || grid[y][x] == '┐' ||
					grid[y][x] == '└' || grid[y][x] == '┘' ||
					grid[y][x] == '─' || grid[y][x] == '│'
				if isBorder {
					line.WriteString(dimStyle.Render(ch))
				} else {
					line.WriteString(normalStyle.Render(ch))
				}
			}
		}
		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

func paintRect(grid [][]rune, color [][]int, ox, oy, w, h, idx int, label string) {
	gridH := len(grid)
	if gridH == 0 {
		return
	}
	gridW := len(grid[0])

	set := func(x, y int, ch rune) {
		if x >= 0 && x < gridW && y >= 0 && y < gridH {
			grid[y][x] = ch
			color[y][x] = idx
		}
	}

	// Top border
	set(ox, oy, '┌')
	for x := 1; x < w-1; x++ {
		set(ox+x, oy, '─')
	}
	set(ox+w-1, oy, '┐')

	// Bottom border
	set(ox, oy+h-1, '└')
	for x := 1; x < w-1; x++ {
		set(ox+x, oy+h-1, '─')
	}
	set(ox+w-1, oy+h-1, '┘')

	// Side borders and fill
	for y := 1; y < h-1; y++ {
		set(ox, oy+y, '│')
		for x := 1; x < w-1; x++ {
			set(ox+x, oy+y, ' ')
		}
		set(ox+w-1, oy+y, '│')
	}

	// Label centered vertically and horizontally
	labelY := oy + h/2
	labelX := ox + (w-len([]rune(label)))/2
	if labelX < ox+1 {
		labelX = ox + 1
	}
	for i, ch := range label {
		set(labelX+i, labelY, ch)
	}

	// Resolution on the line below the label
	if h > 3 {
		resY := labelY + 1
		if resY < oy+h-1 {
			// We don't have the monitor here, so skip resolution line
			// (it's in the label already via the number)
		}
	}
}

// snapPosition calculates the new position for a monitor when the user
// nudges it in a direction. Snaps to the edge of adjacent monitors or
// to the origin. dx/dy are in logical directions (-1, 0, +1).
func snapPosition(monitors []display.Monitor, selectedIdx int, dx, dy int) (int, int) {
	sel := monitors[selectedIdx]
	nudgePx := 10

	newX := sel.X + dx*nudgePx
	newY := sel.Y + dy*nudgePx

	// Snap to edges of other monitors when close.
	const snapThreshold = 30
	for i, m := range monitors {
		if i == selectedIdx {
			continue
		}
		// Snap right edge of sel to left edge of m
		if dx > 0 {
			target := m.X - sel.Width
			if abs(newX-target) < snapThreshold {
				newX = target
			}
			target = m.X + m.Width
			if abs(newX-target) < snapThreshold {
				newX = target
			}
		}
		// Snap left edge of sel to right edge of m
		if dx < 0 {
			target := m.X + m.Width
			if abs(newX-target) < snapThreshold {
				newX = target
			}
			target = m.X - sel.Width
			if abs(newX-target) < snapThreshold {
				newX = target
			}
		}
		// Snap bottom edge of sel to top edge of m
		if dy > 0 {
			target := m.Y - sel.Height
			if abs(newY-target) < snapThreshold {
				newY = target
			}
			target = m.Y + m.Height
			if abs(newY-target) < snapThreshold {
				newY = target
			}
		}
		// Snap top edge of sel to bottom edge of m
		if dy < 0 {
			target := m.Y + m.Height
			if abs(newY-target) < snapThreshold {
				newY = target
			}
			target = m.Y - sel.Height
			if abs(newY-target) < snapThreshold {
				newY = target
			}
		}
		// Align tops/bottoms when moving horizontally
		if dx != 0 && dy == 0 {
			if abs(newY-m.Y) < snapThreshold {
				newY = m.Y
			}
			if abs((newY+sel.Height)-(m.Y+m.Height)) < snapThreshold {
				newY = m.Y + m.Height - sel.Height
			}
		}
		// Align lefts/rights when moving vertically
		if dy != 0 && dx == 0 {
			if abs(newX-m.X) < snapThreshold {
				newX = m.X
			}
			if abs((newX+sel.Width)-(m.X+m.Width)) < snapThreshold {
				newX = m.X + m.Width - sel.Width
			}
		}
	}

	return newX, newY
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
