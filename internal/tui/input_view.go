package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/input"
)

func renderInputDevices(s *core.State, width, height int) string {
	if !s.InputDevicesLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading input devices…"))
	}

	secs := core.InputSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderInnerSidebar(s, entries, s.InputSectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveInputSection()
	var content string
	switch sec.ID {
	case "keyboard":
		content = renderInputKeyboardSection(s, contentWidth, height)
	case "mouse":
		content = renderInputMouseSection(s, contentWidth, height)
	case "touchpad":
		content = renderInputTouchpadSection(s, contentWidth, height)
	case "other":
		content = renderInputOtherSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Keyboard section ────────────────────────────────────────────────

func renderInputKeyboardSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	blocks = append(blocks, renderInputKeyboardSettings(s.InputDevices, innerWidth))

	if kb := renderInputKeyboards(s.InputDevices, innerWidth); kb != "" {
		blocks = append(blocks, kb)
	}

	blocks = append(blocks, renderInputKeyboardHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderInputKeyboardHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("L")+" layout")
	hints = append(hints, accent.Render("+/-")+" repeat rate")
	hints = append(hints, accent.Render("[/]")+" repeat delay")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Mouse section ───────────────────────────────────────────────────

func renderInputMouseSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	blocks = append(blocks, renderInputMouseSettings(s.InputDevices, innerWidth))

	if m := renderInputMice(s.InputDevices, innerWidth); m != "" {
		blocks = append(blocks, m)
	}

	blocks = append(blocks, renderInputMouseHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderInputMouseHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("s/S")+" sensitivity")
	hints = append(hints, accent.Render("a")+" accel profile")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Touchpad section ────────────────────────────────────────────────

func renderInputTouchpadSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	if tp := renderInputTouchpadSettings(s.InputDevices, innerWidth); tp != "" {
		blocks = append(blocks, tp)
	} else {
		blocks = append(blocks, placeholderStyle.Render("No touchpad detected."))
	}

	if tp := renderInputTouchpads(s.InputDevices, innerWidth); tp != "" {
		blocks = append(blocks, tp)
	}

	blocks = append(blocks, renderInputTouchpadHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderInputTouchpadHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("n")+" natural scroll")
	hints = append(hints, accent.Render("t")+" tap to click")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Other section ───────────────────────────────────────────────────

func renderInputOtherSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	if leds := renderInputLEDs(s.InputDevices, innerWidth); leds != "" {
		blocks = append(blocks, leds)
	}

	if o := renderInputOthers(s.InputDevices, innerWidth); o != "" {
		blocks = append(blocks, o)
	}

	if len(blocks) == 0 {
		blocks = append(blocks, placeholderStyle.Render("No other input devices detected."))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// ── Shared rendering helpers ────────────────────────────────────────

func renderInputKeyboardSettings(snap input.Snapshot, total int) string {
	cfg := snap.Config
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	lines = append(lines, label.Render("Layout")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(cfg.KBLayout)+
		dim.Render("  (L to change)"))

	if cfg.KBVariant != "" {
		lines = append(lines, label.Render("Variant")+value.Render(cfg.KBVariant))
	}
	if cfg.KBModel != "" {
		lines = append(lines, label.Render("Model")+value.Render(cfg.KBModel))
	}
	if cfg.KBOptions != "" {
		lines = append(lines, label.Render("Options")+value.Render(cfg.KBOptions))
	}

	lines = append(lines, label.Render("Repeat Rate")+
		value.Render(fmt.Sprintf("%d keys/sec", cfg.RepeatRate))+
		dim.Render("  (+/-)"))

	lines = append(lines, label.Render("Repeat Delay")+
		value.Render(fmt.Sprintf("%d ms", cfg.RepeatDelay))+
		dim.Render("  ([/])"))

	lines = append(lines, label.Render("Numlock Default")+value.Render(onOff(cfg.NumlockDefault)))

	return groupBoxSections("Keyboard Settings", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderInputMouseSettings(snap input.Snapshot, total int) string {
	cfg := snap.Config
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	lines = append(lines, label.Render("Sensitivity")+
		value.Render(fmt.Sprintf("%.2f", cfg.Sensitivity))+
		dim.Render("  (s/S)"))

	accelProfile := cfg.AccelProfile
	if accelProfile == "" {
		accelProfile = "default"
	}
	lines = append(lines, label.Render("Accel Profile")+
		value.Render(accelProfile)+
		dim.Render("  (a)"))

	lines = append(lines, label.Render("Force No Accel")+value.Render(onOff(cfg.ForceNoAccel)))
	lines = append(lines, label.Render("Left Handed")+value.Render(onOff(cfg.LeftHanded)))

	scrollMethod := cfg.ScrollMethod
	if scrollMethod == "" {
		scrollMethod = "default"
	}
	lines = append(lines, label.Render("Scroll Method")+value.Render(scrollMethod))

	followLabels := map[int]string{0: "disabled", 1: "full", 2: "loose", 3: "lazy"}
	follow := followLabels[cfg.FollowMouse]
	if follow == "" {
		follow = fmt.Sprintf("%d", cfg.FollowMouse)
	}
	lines = append(lines, label.Render("Follow Mouse")+value.Render(follow))

	return groupBoxSections("Mouse Settings", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderInputTouchpadSettings(snap input.Snapshot, total int) string {
	if len(snap.Touchpads) == 0 {
		return ""
	}
	cfg := snap.Config
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	lines = append(lines, label.Render("Natural Scroll")+
		value.Render(onOff(cfg.NaturalScroll))+
		dim.Render("  (n)"))

	lines = append(lines, label.Render("Scroll Factor")+
		value.Render(fmt.Sprintf("%.2f", cfg.ScrollFactor)))

	lines = append(lines, label.Render("Tap to Click")+
		value.Render(onOff(cfg.TapToClick))+
		dim.Render("  (t)"))

	lines = append(lines, label.Render("Tap and Drag")+value.Render(onOff(cfg.TapAndDrag)))
	lines = append(lines, label.Render("Drag Lock")+value.Render(onOff(cfg.DragLock)))
	lines = append(lines, label.Render("Disable Typing")+value.Render(onOff(cfg.DisableWhileTyping)))
	lines = append(lines, label.Render("Middle Btn Emu")+value.Render(onOff(cfg.MiddleButtonEmu)))
	lines = append(lines, label.Render("Clickfinger")+value.Render(onOff(cfg.ClickfingerBehavior)))

	return groupBoxSections("Touchpad Settings", []string{strings.Join(lines, "\n")}, total, colorBorder)
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

	lw := 14
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var sections []string
	for _, tp := range snap.Touchpads {
		var lines []string
		lines = append(lines, label.Render("Name")+value.Render(tp.Name))
		lines = append(lines, label.Render("Bus")+value.Render(tp.Bus))
		if tp.VendorID != "0000" || tp.ProductID != "0000" {
			lines = append(lines, label.Render("ID")+
				value.Render(tp.VendorID+":"+tp.ProductID))
		}
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
