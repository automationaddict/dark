package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

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
