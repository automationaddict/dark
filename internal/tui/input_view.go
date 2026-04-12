package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/input"
)

func renderInputDevices(s *core.State, width, height int) string {
	if !s.InputDevicesLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading input devices…"))
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	if cfg := renderInputConfig(s.InputDevices, innerWidth); cfg != "" {
		blocks = append(blocks, cfg)
	}

	if kb := renderInputKeyboards(s.InputDevices, innerWidth); kb != "" {
		blocks = append(blocks, kb)
	}

	if tp := renderInputTouchpads(s.InputDevices, innerWidth); tp != "" {
		blocks = append(blocks, tp)
	}

	if m := renderInputMice(s.InputDevices, innerWidth); m != "" {
		blocks = append(blocks, m)
	}

	if leds := renderInputLEDs(s.InputDevices, innerWidth); leds != "" {
		blocks = append(blocks, leds)
	}

	if o := renderInputOthers(s.InputDevices, innerWidth); o != "" {
		blocks = append(blocks, o)
	}

	if len(blocks) == 0 {
		blocks = append(blocks, placeholderStyle.Render("No input devices detected."))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderInputConfig(snap input.Snapshot, total int) string {
	cfg := snap.Config
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	lines = append(lines, label.Render("Layout")+
		accent.Render(cfg.KBLayout)+
		dim.Render("  (L to change)"))

	if cfg.KBOptions != "" {
		lines = append(lines, label.Render("Options")+value.Render(cfg.KBOptions))
	}

	lines = append(lines, label.Render("Repeat Rate")+
		value.Render(fmt.Sprintf("%d keys/sec", cfg.RepeatRate))+
		dim.Render("  (+/- to adjust)"))

	lines = append(lines, label.Render("Repeat Delay")+
		value.Render(fmt.Sprintf("%d ms", cfg.RepeatDelay))+
		dim.Render("  ([/] to adjust)"))

	numlock := "off"
	if cfg.NumlockDefault {
		numlock = "on"
	}
	lines = append(lines, label.Render("Numlock Default")+value.Render(numlock))

	sens := fmt.Sprintf("%.2f", cfg.Sensitivity)
	lines = append(lines, label.Render("Sensitivity")+
		value.Render(sens)+
		dim.Render("  (s/S to adjust)"))

	accel := "enabled"
	if cfg.ForceNoAccel {
		accel = "disabled"
	}
	lines = append(lines, label.Render("Acceleration")+value.Render(accel))

	return groupBoxSections("Settings", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderInputKeyboards(snap input.Snapshot, total int) string {
	if len(snap.Keyboards) == 0 {
		return ""
	}

	lw := 14
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var sections []string
	for _, kb := range snap.Keyboards {
		var lines []string
		lines = append(lines, label.Render("Name")+value.Render(kb.Name))
		lines = append(lines, label.Render("Bus")+value.Render(kb.Bus))
		if kb.VendorID != "0000" || kb.ProductID != "0000" {
			lines = append(lines, label.Render("ID")+
				value.Render(kb.VendorID+":"+kb.ProductID))
		}
		if kb.Phys != "" {
			lines = append(lines, label.Render("Phys")+value.Render(kb.Phys))
		}
		if kb.Uniq != "" {
			lines = append(lines, label.Render("Unique")+value.Render(kb.Uniq))
		}
		if kb.Inhibited {
			lines = append(lines, label.Render("Status")+
				lipgloss.NewStyle().Foreground(colorRed).Render("inhibited"))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	return groupBoxSections("Keyboards", sections, total, colorBorder)
}

func renderInputTouchpads(snap input.Snapshot, total int) string {
	if len(snap.Touchpads) == 0 {
		return ""
	}

	lw := 16
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var sections []string
	for _, tp := range snap.Touchpads {
		var lines []string
		lines = append(lines, label.Render("Name")+value.Render(tp.Name))
		lines = append(lines, label.Render("Bus")+value.Render(tp.Bus))
		if tp.VendorID != "0000" || tp.ProductID != "0000" {
			lines = append(lines, label.Render("ID")+
				value.Render(tp.VendorID+":"+tp.ProductID))
		}

		natScroll := "off"
		if snap.Config.NaturalScroll {
			natScroll = accent.Render("on")
		}
		lines = append(lines, label.Render("Natural Scroll")+
			lipgloss.NewStyle().Render(natScroll))

		lines = append(lines, label.Render("Scroll Factor")+
			value.Render(fmt.Sprintf("%.2f", snap.Config.ScrollFactor)))

		if tp.Inhibited {
			lines = append(lines, label.Render("Status")+
				lipgloss.NewStyle().Foreground(colorRed).Render("inhibited"))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	return groupBoxSections("Touchpad", sections, total, colorBorder)
}

func renderInputMice(snap input.Snapshot, total int) string {
	if len(snap.Mice) == 0 {
		return ""
	}

	lw := 14
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var sections []string
	for _, m := range snap.Mice {
		var lines []string
		lines = append(lines, label.Render("Name")+value.Render(m.Name))
		lines = append(lines, label.Render("Bus")+value.Render(m.Bus))
		if m.VendorID != "0000" || m.ProductID != "0000" {
			lines = append(lines, label.Render("ID")+
				value.Render(m.VendorID+":"+m.ProductID))
		}
		if m.Uniq != "" {
			lines = append(lines, label.Render("Address")+value.Render(m.Uniq))
		}
		if m.Inhibited {
			lines = append(lines, label.Render("Status")+
				lipgloss.NewStyle().Foreground(colorRed).Render("inhibited"))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	return groupBoxSections("Mice", sections, total, colorBorder)
}

func renderInputLEDs(snap input.Snapshot, total int) string {
	if len(snap.LEDs) == 0 {
		return ""
	}

	lw := 24
	label := detailLabelStyle.Width(lw)

	var lines []string
	for _, led := range snap.LEDs {
		name := led.Name
		name = strings.ReplaceAll(name, "input2::", "")
		name = strings.ReplaceAll(name, "rgb:", "")

		state := lipgloss.NewStyle().Foreground(colorDim).Render("off")
		if led.Brightness > 0 {
			if led.MaxBrightness > 1 {
				state = lipgloss.NewStyle().Foreground(colorAccent).
					Render(fmt.Sprintf("%d/%d", led.Brightness, led.MaxBrightness))
			} else {
				state = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("on")
			}
		}
		lines = append(lines, label.Render(name)+state)
	}

	return groupBoxSections("LEDs", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderInputOthers(snap input.Snapshot, total int) string {
	if len(snap.Others) == 0 {
		return ""
	}

	lw := 30
	label := detailLabelStyle.Width(lw)

	var lines []string
	for _, d := range snap.Others {
		bus := lipgloss.NewStyle().Foreground(colorDim).Render(" (" + d.Bus + ")")
		lines = append(lines, label.Render(d.Name)+bus)
	}

	return groupBoxSections("Other Devices", []string{strings.Join(lines, "\n")}, total, colorBorder)
}
