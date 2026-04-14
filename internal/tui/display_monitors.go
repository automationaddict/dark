package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/display"
)


func renderDisplayMonitorBox(s *core.State, total int, focused bool) string {
	border := colorBorder
	if focused {
		border = colorAccent
	}

	type col struct {
		header string
		cell   func(display.Monitor) string
	}
	cols := []col{
		{"Name", func(m display.Monitor) string {
			if m.Description != "" {
				return m.Description
			}
			return m.Name
		}},
		{"Resolution", func(m display.Monitor) string {
			return m.Resolution() + " @ " + m.RefreshRateHz()
		}},
		{"Scale", func(m display.Monitor) string {
			return fmt.Sprintf("%.2f", m.Scale)
		}},
		{"Rotation", func(m display.Monitor) string {
			return m.TransformLabel()
		}},
		{"Position", func(m display.Monitor) string {
			return fmt.Sprintf("%d,%d", m.X, m.Y)
		}},
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, mon := range s.Display.Monitors {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(mon)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines := []string{"   " + strings.Join(headerCells, gap)}

	for i, mon := range s.Display.Monitors {
		isSel := focused && i == s.DisplayMonitorIdx
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(mon)
			var style lipgloss.Style
			switch {
			case isSel:
				style = tableCellSelected
			case mon.Focused:
				style = tableCellAccent
			case mon.Disabled:
				style = placeholderStyle
			default:
				style = tableCellStyle
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}

		marker := displayRowMarker(isSel, mon)
		row := marker + strings.Join(cells, gap)
		lines = append(lines, row)

		tags := renderDisplayTags(mon)
		if tags != "" {
			lines = append(lines, "   "+tags)
		}
	}

	return groupBoxSections("Monitors", []string{strings.Join(lines, "\n")}, total, border)
}

func displayRowMarker(selected bool, mon display.Monitor) string {
	if selected {
		return "▸ "
	}
	status := "● "
	if mon.Disabled {
		status = "○ "
	} else if !mon.DpmsStatus {
		status = "◌ "
	}
	var statusColor lipgloss.Color
	if mon.Disabled {
		statusColor = colorDim
	} else if !mon.DpmsStatus {
		statusColor = colorGold
	} else {
		statusColor = colorGreen
	}
	return lipgloss.NewStyle().Foreground(statusColor).Render(status)
}

func renderDisplayTags(mon display.Monitor) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	var tags []string

	if mon.Focused {
		tags = append(tags, lipgloss.NewStyle().Foreground(colorAccent).Render("focused"))
	}
	if mon.Vrr {
		tags = append(tags, "VRR")
	}
	if !mon.DpmsStatus {
		tags = append(tags, lipgloss.NewStyle().Foreground(colorGold).Render("DPMS off"))
	}
	if mon.Disabled {
		tags = append(tags, lipgloss.NewStyle().Foreground(colorRed).Render("disabled"))
	}
	if mon.MirrorOf != "" && mon.MirrorOf != "none" {
		tags = append(tags, "mirror:"+mon.MirrorOf)
	}
	if mon.ActiveWorkspace.Name != "" {
		tags = append(tags, "ws:"+mon.ActiveWorkspace.Name)
	}
	if len(tags) == 0 {
		return ""
	}
	return dim.Render("(" + strings.Join(tags, " · ") + ")")
}

func renderDisplayLayoutFull(s *core.State, width, height, innerWidth int) string {
	canvasW := innerWidth - 8
	canvasH := height - 12
	if canvasW < 40 {
		canvasW = 40
	}
	if canvasH < 10 {
		canvasH = 10
	}

	grid := renderMonitorGrid(s.Display.Monitors, s.DisplayMonitorIdx, canvasW, canvasH)

	mon, ok := s.SelectedMonitor()
	var info string
	if ok {
		info = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(mon.Description) +
			lipgloss.NewStyle().Foreground(colorDim).Render(
				fmt.Sprintf("  %dx%d at (%d, %d)", mon.Width, mon.Height, mon.X, mon.Y))
	}

	hint := lipgloss.NewStyle().Foreground(colorDim).
		Render("  ←/→/↑/↓ nudge · j/k select monitor · i identify · esc close")

	var blocks []string
	blocks = append(blocks, groupBoxSections("Arrange Monitors", []string{grid}, innerWidth, colorAccent))
	if info != "" {
		blocks = append(blocks, info)
	}
	blocks = append(blocks, hint)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDisplayLayoutCompact(s *core.State, total int) string {
	canvasW := total - 8
	canvasH := 8
	if canvasW < 30 {
		canvasW = 30
	}
	grid := renderMonitorGrid(s.Display.Monitors, s.DisplayMonitorIdx, canvasW, canvasH)
	return groupBoxSections("Layout", []string{grid}, total, colorBorder)
}
