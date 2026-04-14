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

	// Layout mode is full-screen — bypasses the inner sidebar.
	if s.DisplayLayoutOpen {
		innerWidth := width - 6
		if innerWidth < 46 {
			innerWidth = 46
		}
		return renderDisplayLayoutFull(s, width, height, innerWidth)
	}

	secs := core.DisplaySections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.DisplayContentFocused
	sidebar := renderInnerSidebarFocused(s, entries, s.DisplaySectionIdx, height, sidebarFocused)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveDisplaySection()
	var content string
	switch sec.ID {
	case "monitors":
		content = renderDisplayMonitorsSection(s, contentWidth, height)
	case "controls":
		content = renderDisplayControlsSection(s, contentWidth, height)
	case "gpu":
		content = renderDisplayGPUSection(s, contentWidth, height)
	case "layout":
		content = renderDisplayLayoutSection(s, contentWidth, height)
	case "profiles":
		content = renderDisplayProfilesSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Monitors section ────────────────────────────────────────────────

func renderDisplayMonitorsSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	focused := s.DisplayContentFocused

	monBox := renderDisplayMonitorBox(s, innerWidth, focused)
	var blocks []string
	blocks = append(blocks, monBox)

	if detail := renderDisplaySelectedDetail(s, innerWidth); detail != "" {
		blocks = append(blocks, detail)
	}

	if s.DisplayBusy {
		blocks = append(blocks, "", statusBusyStyle.Render("working…"))
	}

	blocks = append(blocks, renderDisplayMonitorsHint(s, focused))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDisplayMonitorsHint(s *core.State, focused bool) string {
	if !focused {
		return statusBarStyle.Render("enter to select")
	}
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var hints []string
	hints = append(hints, accent.Render("m")+" mode")
	hints = append(hints, accent.Render("w")+" dpms")
	hints = append(hints, accent.Render("e")+" enable/disable")
	hints = append(hints, accent.Render("r")+" rotate")
	hints = append(hints, accent.Render("+/-")+" scale")
	hints = append(hints, accent.Render("s")+" scale")
	hints = append(hints, accent.Render("v")+" vrr")
	hints = append(hints, accent.Render("R")+" mirror")
	hints = append(hints, accent.Render("p")+" position")
	hints = append(hints, accent.Render("i")+" identify")
	hints = append(hints, accent.Render("esc"))

	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Controls section ────────────────────────────────────────────────

func renderDisplayControlsSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	hasControls := s.Display.HasBacklight || s.Display.HasKbdLight || s.NightLightActive || s.NightLightGamma != 0

	if !hasControls && !s.DisplayLoaded {
		blocks = append(blocks, placeholderStyle.Render("No display controls available."))
	} else {
		extras := renderDisplayExtras(s, innerWidth)
		if extras != "" {
			blocks = append(blocks, extras)
		} else {
			blocks = append(blocks, placeholderStyle.Render("No brightness/backlight controls detected."))
		}
	}

	blocks = append(blocks, renderDisplayControlsHint(s))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDisplayControlsHint(s *core.State) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var hints []string
	if s.Display.HasBacklight {
		hints = append(hints, accent.Render("[/]")+" brightness")
	}
	if s.Display.HasKbdLight {
		hints = append(hints, accent.Render("{/}")+" kbd light")
	}
	hints = append(hints, accent.Render("n")+" night light")
	hints = append(hints, accent.Render("N")+" temperature")
	hints = append(hints, accent.Render("g/G")+" gamma")
	hints = append(hints, accent.Render("esc"))

	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── GPU section ────────────────────────────────────────────────────

func renderDisplayGPUSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	gpu := s.Display.GPU
	label := detailLabelStyle.Width(14)
	value := detailValueStyle

	// GPU list
	var gpuLines []string
	if len(gpu.GPUs) == 0 {
		gpuLines = append(gpuLines, placeholderStyle.Render("No GPUs detected"))
	} else {
		for i, g := range gpu.GPUs {
			gpuLines = append(gpuLines,
				label.Render(fmt.Sprintf("GPU %d", i))+value.Render(g))
		}
	}
	gpuBox := groupBoxSections("Graphics", []string{
		strings.Join(gpuLines, "\n"),
	}, innerWidth, colorBorder)

	// Hybrid GPU status
	var hybridSection string
	if !gpu.HybridSupported {
		hybridSection = groupBoxSections("Hybrid GPU", []string{
			placeholderStyle.Render("Not supported — single GPU detected"),
		}, innerWidth, colorBorder)
	} else {
		mode := gpu.Mode
		if mode == "" {
			mode = "Unknown"
		}
		var modeStyle string
		switch mode {
		case "Hybrid":
			modeStyle = statusOnlineStyle.Render(mode)
		case "Integrated":
			modeStyle = statusBusyStyle.Render(mode)
		default:
			modeStyle = value.Render(mode)
		}
		var lines []string
		lines = append(lines, label.Render("Mode")+modeStyle)

		dim := lipgloss.NewStyle().Foreground(colorDim)
		accent := lipgloss.NewStyle().Foreground(colorAccent)
		hint := dim.Render(accent.Render("g") + " toggle hybrid GPU")
		lines = append(lines, "")
		lines = append(lines, hint)

		hybridSection = groupBoxSections("Hybrid GPU", []string{
			strings.Join(lines, "\n"),
		}, innerWidth, colorAccent)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, gpuBox, "", hybridSection)
	return renderContentPane(width, height, body)
}

// ── Layout section ──────────────────────────────────────────────────

func renderDisplayLayoutSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	layout := renderDisplayLayoutCompact(s, innerWidth)
	blocks = append(blocks, layout)

	blocks = append(blocks, renderDisplayLayoutHint(s))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDisplayLayoutHint(s *core.State) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var hints []string
	if len(s.Display.Monitors) > 1 {
		hints = append(hints, accent.Render("a")+" arrange")
	}
	hints = append(hints, accent.Render("i")+" identify")
	hints = append(hints, accent.Render("esc"))

	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Profiles section ────────────────────────────────────────────────

func renderDisplayProfilesSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	if len(s.Display.Profiles) == 0 {
		blocks = append(blocks, placeholderStyle.Render("No saved display profiles."))
	} else {
		var lines []string
		for _, p := range s.Display.Profiles {
			lines = append(lines, "  "+detailValueStyle.Render(p))
		}
		box := groupBoxSections("Saved Profiles",
			[]string{strings.Join(lines, "\n")}, innerWidth, colorBorder)
		blocks = append(blocks, box)
	}

	blocks = append(blocks, renderDisplayProfilesHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDisplayProfilesHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var hints []string
	hints = append(hints, accent.Render("S")+" save")
	hints = append(hints, accent.Render("P")+" apply")
	hints = append(hints, accent.Render("X")+" delete")
	hints = append(hints, accent.Render("esc"))

	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Shared rendering helpers ────────────────────────────────────────

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

func renderDisplaySelectedDetail(s *core.State, total int) string {
	mon, ok := s.SelectedMonitor()
	if !ok {
		return ""
	}

	accent := lipgloss.NewStyle().Foreground(colorAccent)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	lw := 16
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	currentMode := fmt.Sprintf("%dx%d @ %s", mon.Width, mon.Height, mon.RefreshRateHz())
	modeLine := label.Render("Mode") + value.Render(currentMode)
	if s.DisplayContentFocused && len(mon.AvailableModes) > 0 {
		modeLine += dim.Render(fmt.Sprintf("  (%d available · ", len(mon.AvailableModes))) +
			accent.Render("m") + dim.Render(")")
	}
	lines = append(lines, modeLine)

	scaleLine := label.Render("Scale") + value.Render(fmt.Sprintf("%.2f", mon.Scale))
	if s.DisplayContentFocused {
		scaleLine += dim.Render("  (") + accent.Render("s") + dim.Render(" select · ") +
			accent.Render("+/-") + dim.Render(" step)")
	}
	lines = append(lines, scaleLine)

	lines = append(lines, label.Render("Rotation")+value.Render(mon.TransformLabel()))
	lines = append(lines, label.Render("Position")+value.Render(fmt.Sprintf("%d, %d", mon.X, mon.Y)))

	displayMode := "Extend"
	if mon.Disabled {
		displayMode = "Disabled"
	} else if mon.MirrorOf != "" && mon.MirrorOf != "none" {
		displayMode = "Mirror → " + mon.MirrorOf
	}
	dispModeLine := label.Render("Display Mode") + value.Render(displayMode)
	if s.DisplayContentFocused && len(s.Display.Monitors) > 1 {
		dispModeLine += dim.Render("  (") + accent.Render("R") + dim.Render(" mirror · ") +
			accent.Render("e") + dim.Render(" toggle)")
	}
	lines = append(lines, dispModeLine)

	dpms := "On"
	if !mon.DpmsStatus {
		dpms = "Off"
	}
	lines = append(lines, label.Render("Display Power")+value.Render(dpms))

	vrr := "Off"
	if mon.Vrr {
		vrr = "On"
	}
	lines = append(lines, label.Render("VRR")+value.Render(vrr))

	if mon.Make != "" || mon.Model != "" {
		hw := ""
		if mon.Make != "" {
			hw = mon.Make
		}
		if mon.Model != "" {
			if hw != "" {
				hw += " "
			}
			hw += mon.Model
		}
		lines = append(lines, label.Render("Hardware")+value.Render(hw))
	}

	if mon.PhysicalWidth > 0 {
		diag := mon.DiagonalInches()
		lines = append(lines, label.Render("Size")+value.Render(fmt.Sprintf("%.1f\" diagonal, %d DPI", diag, mon.DPI())))
	}

	desc := mon.Description
	if desc == "" {
		desc = mon.Name
	}

	return groupBoxSections(desc, []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDisplayExtras(s *core.State, total int) string {
	const indent = 3
	const labelWidth = 16
	barWidth := total - indent - labelWidth - 7
	if barWidth < 10 {
		barWidth = 10
	}

	var lines []string

	if s.Display.HasBacklight {
		lines = append(lines, renderDisplaySlider("☀ Brightness", fmt.Sprintf("%3d%%", s.Display.Brightness), s.Display.Brightness, barWidth, indent, labelWidth, false))
	}
	if s.Display.HasKbdLight {
		lines = append(lines, renderDisplaySlider("⌨ Keyboard", fmt.Sprintf("%3d%%", s.Display.KbdBrightness), s.Display.KbdBrightness, barWidth, indent, labelWidth, false))
	}

	if s.NightLightActive {
		pct := s.NightLightTemp * 100 / 6500
		lines = append(lines, renderDisplaySlider("🌙 Night Light", fmt.Sprintf("%dK", s.NightLightTemp), pct, barWidth, indent, labelWidth, false))
	} else {
		lines = append(lines, renderDisplaySlider("🌙 Night Light", "off", 0, barWidth, indent, labelWidth, true))
	}

	if len(lines) == 0 {
		return ""
	}
	return groupBoxSections("Controls", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDisplaySlider(name, label string, pct, barWidth, indent, labelWidth int, muted bool) string {
	if pct < 0 {
		pct = 0
	}
	filled := pct * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}

	filledStyle := audioBarFilledStyle
	if muted {
		filledStyle = audioBarMutedStyle
	}

	pad := strings.Repeat(" ", indent)
	nameStr := lipgloss.NewStyle().Foreground(colorText).Width(labelWidth).Render(name)
	labelStr := placeholderStyle.Render(fmt.Sprintf("%5s ", label))
	filledPart := filledStyle.Render(strings.Repeat("─", filled))
	emptyPart := audioBarEmptyStyle.Render(strings.Repeat("┄", barWidth-filled))

	return pad + nameStr + labelStr + filledPart + emptyPart
}
