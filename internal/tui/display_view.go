package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
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
