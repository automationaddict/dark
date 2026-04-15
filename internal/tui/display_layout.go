package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/display"
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
// nudges it in a direction. Instead of pixel nudging, it jumps to the
// next snap point in the given direction — the nearest edge of another
// monitor that is strictly further along the nudge axis. This makes
// arrow keys feel like "swap sides" rather than "inch by 10 pixels".
// dx/dy are in logical directions (-1, 0, +1).
func snapPosition(monitors []display.Monitor, selectedIdx int, dx, dy int) (int, int) {
	sel := monitors[selectedIdx]
	newX, newY := sel.X, sel.Y

	if dx != 0 {
		newX = findNextSnapX(monitors, selectedIdx, dx)
	}
	if dy != 0 {
		newY = findNextSnapY(monitors, selectedIdx, dy)
	}
	return newX, newY
}

// findNextSnapX finds the next horizontal snap position for the selected
// monitor in direction dx (-1 = left, +1 = right). It collects all
// meaningful X positions from other monitors (left edge, right edge)
// and picks the closest one that is strictly in the nudge direction.
func findNextSnapX(monitors []display.Monitor, selIdx, dx int) int {
	sel := monitors[selIdx]
	curX := sel.X

	var candidates []int
	for i, m := range monitors {
		if i == selIdx {
			continue
		}
		// Place selected monitor's left edge at other monitor's right edge
		candidates = append(candidates, m.X+m.Width)
		// Place selected monitor's right edge at other monitor's left edge
		candidates = append(candidates, m.X-sel.Width)
	}
	// Also consider origin
	candidates = append(candidates, 0)

	best := curX
	bestDist := -1
	for _, cx := range candidates {
		if dx > 0 && cx <= curX {
			continue
		}
		if dx < 0 && cx >= curX {
			continue
		}
		dist := abs(cx - curX)
		if bestDist < 0 || dist < bestDist {
			best = cx
			bestDist = dist
		}
	}
	return best
}

// findNextSnapY finds the next vertical snap position.
func findNextSnapY(monitors []display.Monitor, selIdx, dy int) int {
	sel := monitors[selIdx]
	curY := sel.Y

	var candidates []int
	for i, m := range monitors {
		if i == selIdx {
			continue
		}
		candidates = append(candidates, m.Y+m.Height)
		candidates = append(candidates, m.Y-sel.Height)
	}
	candidates = append(candidates, 0)

	best := curY
	bestDist := -1
	for _, cy := range candidates {
		if dy > 0 && cy <= curY {
			continue
		}
		if dy < 0 && cy >= curY {
			continue
		}
		dist := abs(cy - curY)
		if bestDist < 0 || dist < bestDist {
			best = cy
			bestDist = dist
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
