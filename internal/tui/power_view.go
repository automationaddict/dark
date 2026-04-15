package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/power"
)

func renderPower(s *core.State, width, height int) string {
	if !s.PowerLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading power info…"))
	}

	secs := core.PowerSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderInnerSidebar(s, entries, s.PowerSectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActivePowerSection()
	var content string
	switch sec.ID {
	case "overview":
		content = renderPowerOverview(s, contentWidth, height)
	case "profile":
		content = renderPowerProfileSection(s, contentWidth, height)
	case "cpu":
		content = renderPowerCPUSection(s, contentWidth, height)
	case "thermal":
		content = renderPowerThermalSection(s, contentWidth, height)
	case "buttons":
		content = renderPowerButtonsSection(s, contentWidth, height)
	case "idle":
		content = renderPowerIdleSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderPowerOverview(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	var blocks []string
	if bat := renderPowerBattery(s.Power, innerWidth); bat != "" {
		blocks = append(blocks, bat)
	}
	if gpu := renderPowerGPU(s.Power, innerWidth); gpu != "" {
		blocks = append(blocks, gpu)
	}
	if fans := renderPowerFans(s.Power, innerWidth); fans != "" {
		blocks = append(blocks, fans)
	}
	if periph := renderPowerPeripherals(s.Power, innerWidth); periph != "" {
		blocks = append(blocks, periph)
	}
	if sleep := renderPowerSleep(s.Power, innerWidth); sleep != "" {
		blocks = append(blocks, sleep)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, placeholderStyle.Render("No power data available."))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderPowerProfileSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	var blocks []string
	blocks = append(blocks, renderPowerProfile(s.Power, innerWidth))

	hint := lipgloss.NewStyle().Foreground(colorDim).Render("  p cycle profiles")
	blocks = append(blocks, hint)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPowerCPUSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	var blocks []string
	blocks = append(blocks, renderPowerCPU(s.Power, innerWidth))

	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	hint := dim.Render("  ") +
		accent.Render("g") + dim.Render(" governor  ") +
		accent.Render("E") + dim.Render(" epp")
	blocks = append(blocks, hint)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPowerThermalSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	var blocks []string
	if thermal := renderPowerThermals(s.Power, innerWidth); thermal != "" {
		blocks = append(blocks, thermal)
	}
	if gpu := renderPowerGPU(s.Power, innerWidth); gpu != "" {
		blocks = append(blocks, gpu)
	}
	if fans := renderPowerFans(s.Power, innerWidth); fans != "" {
		blocks = append(blocks, fans)
	}
	if len(blocks) == 0 {
		blocks = append(blocks, placeholderStyle.Render("No thermal data available."))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderPowerButtonsSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	var blocks []string
	blocks = append(blocks, renderPowerButtons(s.Power, innerWidth))

	hint := lipgloss.NewStyle().Foreground(colorDim).Render("  b edit system buttons")
	blocks = append(blocks, hint)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPowerIdleSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	var blocks []string
	blocks = append(blocks, renderPowerIdle(s.Power, innerWidth))

	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	hint := dim.Render("  ") +
		accent.Render("i") + dim.Render(" edit idle timers  ") +
		accent.Render("l") + dim.Render(" toggle idle daemon")
	blocks = append(blocks, hint)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPowerBattery(p power.Snapshot, total int) string {
	if len(p.Batteries) == 0 && len(p.ACAdapters) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	for _, ac := range p.ACAdapters {
		status := "Offline"
		if ac.Online {
			status = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("Online")
		}
		lines = append(lines, label.Render("AC Adapter")+value.Render(status))
	}

	for _, bat := range p.Batteries {
		if len(lines) > 0 {
			lines = append(lines, "")
		}

		statusStyle := detailValueStyle
		switch bat.Status {
		case "Charging":
			statusStyle = lipgloss.NewStyle().Foreground(colorGreen)
		case "Discharging":
			statusStyle = lipgloss.NewStyle().Foreground(colorGold)
		case "Full":
			statusStyle = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
		}
		lines = append(lines, label.Render("Status")+statusStyle.Render(bat.Status))

		barWidth := total - lw - 12
		if barWidth < 10 {
			barWidth = 10
		}
		filled := bat.Capacity * barWidth / 100
		if filled > barWidth {
			filled = barWidth
		}
		barStyle := audioBarFilledStyle
		if bat.Capacity <= 20 {
			barStyle = lipgloss.NewStyle().Foreground(colorRed)
		} else if bat.Capacity <= 40 {
			barStyle = lipgloss.NewStyle().Foreground(colorGold)
		}
		bar := barStyle.Render(strings.Repeat("─", filled)) +
			audioBarEmptyStyle.Render(strings.Repeat("┄", barWidth-filled))
		lines = append(lines, label.Render("Charge")+
			placeholderStyle.Render(fmt.Sprintf("%3d%% ", bat.Capacity))+bar)

		lines = append(lines, label.Render("Energy")+
			value.Render(fmt.Sprintf("%.1f / %.1f Wh", bat.EnergyNow, bat.EnergyFull)))

		if bat.EnergyFullDesign > 0 {
			lines = append(lines, label.Render("Health")+
				value.Render(fmt.Sprintf("%d%% (%.1f / %.1f Wh design)", bat.Health(), bat.EnergyFull, bat.EnergyFullDesign)))
		}

		if bat.Voltage > 0 {
			lines = append(lines, label.Render("Voltage")+
				value.Render(fmt.Sprintf("%.2f V", bat.Voltage)))
		}

		if bat.PowerNow > 0 {
			lines = append(lines, label.Render("Power Draw")+
				value.Render(fmt.Sprintf("%.1f W", bat.PowerNow)))
		}

		if bat.Technology != "" {
			lines = append(lines, label.Render("Technology")+value.Render(bat.Technology))
		}

		if bat.CycleCount > 0 {
			lines = append(lines, label.Render("Cycle Count")+
				value.Render(fmt.Sprintf("%d", bat.CycleCount)))
		}
	}

	return groupBoxSections("Battery", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

