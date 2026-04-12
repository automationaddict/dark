package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/display"
)

func renderDisplay(s *core.State, width, height int) string {
	if !s.DisplayLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading display info…"),
		)
	}
	if len(s.Display.Monitors) == 0 {
		title := contentTitle.Render("Displays")
		body := placeholderStyle.Render("No monitors detected.")
		return renderContentPane(width, height,
			lipgloss.JoinVertical(lipgloss.Left, title, body),
		)
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	focused := s.ContentFocused

	var blocks []string

	if s.DisplayLayoutOpen {
		blocks = append(blocks, renderDisplayLayout(s, innerWidth, height-4))
		body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
		return renderContentPane(width, height, body)
	}

	if s.DisplayInfoOpen {
		mon, ok := s.SelectedMonitor()
		if ok {
			blocks = append(blocks, renderDisplayInfoBox(mon, innerWidth))
		} else {
			s.DisplayInfoOpen = false
		}
	}

	if !s.DisplayInfoOpen {
		monBox := renderDisplayMonitorBox(s, innerWidth, focused)
		blocks = append(blocks, monBox)

		if s.Display.HasBacklight || s.Display.HasKbdLight || s.DisplayLoaded {
			extras := renderDisplayExtras(s, innerWidth)
			if extras != "" {
				blocks = append(blocks, extras)
			}
		}

		if s.DisplayBusy {
			blocks = append(blocks, "", statusBusyStyle.Render("working…"))
		}

		if s.DisplayActionError != "" {
			errStyle := lipgloss.NewStyle().Foreground(colorRed)
			blocks = append(blocks, "", errStyle.Render("  error: "+s.DisplayActionError))
		}

		if focused {
			blocks = append(blocks, renderDisplayHints())
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

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

func renderDisplayInfoBox(mon display.Monitor, total int) string {
	innerWidth := total - 4
	if innerWidth < 24 {
		innerWidth = 24
	}

	desc := mon.Description
	if desc == "" {
		desc = mon.Name
	}
	title := desc + " (" + mon.Name + ")"

	rows := [][2]string{
		{"Name", mon.Name},
		{"Description", orDash(mon.Description)},
	}
	if mon.Make != "" {
		rows = append(rows, [2]string{"Make", mon.Make})
	}
	if mon.Model != "" {
		rows = append(rows, [2]string{"Model", mon.Model})
	}
	if mon.Serial != "" {
		rows = append(rows, [2]string{"Serial", mon.Serial})
	}

	rows = append(rows,
		[2]string{"Resolution", mon.Resolution()},
		[2]string{"Refresh Rate", mon.RefreshRateHz()},
		[2]string{"Scale", fmt.Sprintf("%.2f", mon.Scale)},
		[2]string{"Transform", mon.TransformLabel()},
		[2]string{"Position", fmt.Sprintf("%d, %d", mon.X, mon.Y)},
	)

	if mon.PhysicalWidth > 0 {
		wIn, hIn := mon.PhysicalSizeInches()
		diag := mon.DiagonalInches()
		rows = append(rows,
			[2]string{"Physical Size", fmt.Sprintf("%.1f\" × %.1f\" (%.1f\" diagonal)", wIn, hIn, diag)},
			[2]string{"DPI", fmt.Sprintf("%d", mon.DPI())},
		)
	}

	dpms := "On"
	if !mon.DpmsStatus {
		dpms = "Off (DPMS)"
	}
	rows = append(rows, [2]string{"Display Power", dpms})

	vrr := "Off"
	if mon.Vrr {
		vrr = "On"
	}
	rows = append(rows, [2]string{"VRR", vrr})

	status := "Enabled"
	if mon.Disabled {
		status = "Disabled"
	}
	rows = append(rows, [2]string{"Status", status})

	if mon.MirrorOf != "" && mon.MirrorOf != "none" {
		rows = append(rows, [2]string{"Mirror Of", mon.MirrorOf})
	}
	if mon.ActiveWorkspace.Name != "" {
		rows = append(rows, [2]string{"Workspace", mon.ActiveWorkspace.Name})
	}

	labelWidth := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > labelWidth {
			labelWidth = w
		}
	}
	propLines := make([]string, 0, len(rows))
	for _, r := range rows {
		label := detailLabelStyle.Width(labelWidth + 2).Render(r[0])
		value := detailValueStyle.Render(r[1])
		propLines = append(propLines, label+value)
	}

	sections := []string{strings.Join(propLines, "\n")}

	if len(mon.AvailableModes) > 0 {
		var modeLines []string
		current := fmt.Sprintf("%dx%d@%.2fHz", mon.Width, mon.Height, mon.RefreshRate)
		modesPerRow := 3
		for i := 0; i < len(mon.AvailableModes); i += modesPerRow {
			end := i + modesPerRow
			if end > len(mon.AvailableModes) {
				end = len(mon.AvailableModes)
			}
			chunk := mon.AvailableModes[i:end]
			rendered := make([]string, len(chunk))
			for j, mode := range chunk {
				if mode == current {
					rendered[j] = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(mode)
				} else {
					rendered[j] = lipgloss.NewStyle().Foreground(colorDim).Render(mode)
				}
			}
			modeLines = append(modeLines, "  "+strings.Join(rendered, "  "))
		}
		sections = append(sections, strings.Join(modeLines, "\n"))
	}

	sections = append(sections, placeholderStyle.Render(
		"m mode · r rotate · +/- scale · w dpms · e enable · v vrr · esc back"))

	return groupBoxSections(title, sections, total, colorAccent)
}

func renderDisplayExtras(s *core.State, total int) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var lines []string

	if s.Display.HasBacklight {
		bar := brightnessBar(s.Display.Brightness)
		lines = append(lines, fmt.Sprintf("  ☀ Brightness  %s  %d%%", bar, s.Display.Brightness))
	}
	if s.Display.HasKbdLight {
		bar := brightnessBar(s.Display.KbdBrightness)
		lines = append(lines, fmt.Sprintf("  ⌨ Keyboard    %s  %d%%", bar, s.Display.KbdBrightness))
	}

	if s.NightLightActive {
		nlInfo := fmt.Sprintf("  🌙 Night Light  %s  %s",
			accent.Render(fmt.Sprintf("%dK", s.NightLightTemp)),
			accent.Render("(active)"))
		if s.NightLightGamma != 0 && s.NightLightGamma != 100 {
			nlInfo += fmt.Sprintf("  gamma %s", accent.Render(fmt.Sprintf("%d%%", s.NightLightGamma)))
		}
		lines = append(lines, nlInfo)
	} else {
		lines = append(lines, "  🌙 Night Light  "+dim.Render("off"))
	}

	if s.NightLightGamma != 0 && s.NightLightGamma != 100 && !s.NightLightActive {
		lines = append(lines, fmt.Sprintf("  γ  Gamma  %s", accent.Render(fmt.Sprintf("%d%%", s.NightLightGamma))))
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func brightnessBar(pct int) string {
	const width = 10
	filled := pct * width / 100
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func renderDisplayHints() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var hints []string
	hints = append(hints, accent.Render("enter")+" info")
	hints = append(hints, accent.Render("l")+" layout")
	hints = append(hints, accent.Render("i")+" identify")
	hints = append(hints, accent.Render("m")+" mode")
	hints = append(hints, accent.Render("w")+" dpms")
	hints = append(hints, accent.Render("e")+" enable/disable")
	hints = append(hints, accent.Render("r")+" rotate")
	hints = append(hints, accent.Render("+/-")+" scale")
	hints = append(hints, accent.Render("v")+" vrr")
	hints = append(hints, accent.Render("p")+" position")
	hints = append(hints, accent.Render("s")+" scale")
	hints = append(hints, accent.Render("[/]")+" brightness")
	hints = append(hints, accent.Render("{/}")+" kbd light")
	hints = append(hints, accent.Render("n")+" night light")
	hints = append(hints, accent.Render("g/G")+" gamma")
	hints = append(hints, accent.Render("S")+" save profile")
	hints = append(hints, accent.Render("P")+" load profile")
	hints = append(hints, accent.Render("X")+" delete profile")

	return dim.Render("  " + strings.Join(hints, "  "))
}
