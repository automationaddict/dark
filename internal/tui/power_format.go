package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/services/power"
)

func profileIcon(name string) string {
	switch name {
	case "performance":
		return "󰓅"
	case "balanced":
		return "󰾅"
	case "power-saver":
		return "󰌪"
	default:
		return "󰂄"
	}
}

func formatTimeout(sec int) string {
	if sec <= 0 {
		return "disabled"
	}
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	m := sec / 60
	s := sec % 60
	if s == 0 {
		return fmt.Sprintf("%dmin", m)
	}
	return fmt.Sprintf("%dmin %ds", m, s)
}

func renderPowerProfile(p power.Snapshot, total int) string {
	return renderPowerProfileFocused(p, total, colorBorder)
}

func renderPowerProfileFocused(p power.Snapshot, total int, border lipgloss.Color) string {
	if len(p.Profiles) == 0 {
		return ""
	}

	var parts []string
	for _, prof := range p.Profiles {
		icon := profileIcon(prof)
		if prof == p.Profile {
			parts = append(parts, lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("▸ "+icon+"  "+prof))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(colorDim).Render("  "+icon+"  "+prof))
		}
	}

	hint := lipgloss.NewStyle().Foreground(colorDim).Render("  p to cycle profiles")

	return groupBoxSections("Power Profile", []string{
		strings.Join(parts, "\n"),
		hint,
	}, total, border)
}

func renderPowerCPU(p power.Snapshot, total int) string {
	return renderPowerCPUFocused(p, total, colorBorder)
}

func renderPowerCPUFocused(p power.Snapshot, total int, border lipgloss.Color) string {
	if len(p.CPUs) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	lines = append(lines, label.Render("Cores")+value.Render(fmt.Sprintf("%d", len(p.CPUs))))

	if p.PState != "" {
		lines = append(lines, label.Render("Driver")+value.Render(p.PState))
	}

	lines = append(lines, label.Render("Governor")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(p.Governor))

	if p.EPP != "" {
		lines = append(lines, label.Render("Energy Pref")+
			lipgloss.NewStyle().Foreground(colorAccent).Render(p.EPP))
	}

	var minFreq, maxFreq, avgFreq int
	for i, c := range p.CPUs {
		if i == 0 || c.CurFreq < minFreq {
			minFreq = c.CurFreq
		}
		if c.CurFreq > maxFreq {
			maxFreq = c.CurFreq
		}
		avgFreq += c.CurFreq
	}
	avgFreq /= len(p.CPUs)

	lines = append(lines, label.Render("Frequency")+
		value.Render(fmt.Sprintf("%.0f – %.0f MHz (avg %.0f)",
			float64(minFreq)/1000, float64(maxFreq)/1000, float64(avgFreq)/1000)))

	if len(p.CPUs) > 0 {
		lines = append(lines, label.Render("Range")+
			value.Render(fmt.Sprintf("%.0f – %.0f MHz",
				float64(p.CPUs[0].MinFreq)/1000, float64(p.CPUs[0].MaxFreq)/1000)))
	}

	return groupBoxSections("CPU", []string{strings.Join(lines, "\n")}, total, border)
}

func renderPowerThermals(p power.Snapshot, total int) string {
	if len(p.Thermals) == 0 {
		return ""
	}

	lw := 18
	// Find max label width for alignment
	for _, t := range p.Thermals {
		if w := lipgloss.Width(t.Label); w+2 > lw {
			lw = w + 2
		}
	}
	label := detailLabelStyle.Width(lw)

	var lines []string
	for _, t := range p.Thermals {
		tempStyle := detailValueStyle
		if t.Temp >= 80 {
			tempStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
		} else if t.Temp >= 60 {
			tempStyle = lipgloss.NewStyle().Foreground(colorGold)
		}
		lines = append(lines, label.Render(t.Label)+
			tempStyle.Render(fmt.Sprintf("%.1f°C", t.Temp)))
	}

	return groupBoxSections("Thermals", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPowerGPU(p power.Snapshot, total int) string {
	if p.GPU.Temp == 0 && p.GPU.PowerW == 0 && p.GPU.ClockMHz == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	if p.GPU.Temp > 0 {
		tempStyle := value
		if p.GPU.Temp >= 80 {
			tempStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
		} else if p.GPU.Temp >= 60 {
			tempStyle = lipgloss.NewStyle().Foreground(colorGold)
		}
		lines = append(lines, label.Render("Temperature")+
			tempStyle.Render(fmt.Sprintf("%.1f°C", p.GPU.Temp)))
	}

	if p.GPU.PowerW > 0 {
		lines = append(lines, label.Render("Power")+
			value.Render(fmt.Sprintf("%.1f W", p.GPU.PowerW)))
	}

	if p.GPU.ClockMHz > 0 {
		lines = append(lines, label.Render("Clock")+
			value.Render(fmt.Sprintf("%d MHz", p.GPU.ClockMHz)))
	}

	return groupBoxSections("GPU", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPowerFans(p power.Snapshot, total int) string {
	if len(p.Fans) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	for _, f := range p.Fans {
		rpmStr := fmt.Sprintf("%d RPM", f.RPM)
		if f.RPM == 0 {
			rpmStr = lipgloss.NewStyle().Foreground(colorDim).Render("stopped")
		}
		lines = append(lines, label.Render(f.Label)+value.Render(rpmStr))
	}

	return groupBoxSections("Fans", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPowerPeripherals(p power.Snapshot, total int) string {
	if len(p.Peripherals) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)

	var lines []string
	for _, dev := range p.Peripherals {
		name := dev.Model
		if name == "" {
			name = dev.Name
		}
		chargeStr := fmt.Sprintf("%d%%", dev.Charge)
		if dev.Status != "" {
			chargeStr += " (" + dev.Status + ")"
		}
		lines = append(lines, label.Render(name)+detailValueStyle.Render(chargeStr))
	}

	return groupBoxSections("Peripherals", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPowerButtons(p power.Snapshot, total int) string {
	return renderPowerButtonsFocused(p, total, colorBorder)
}

func renderPowerButtonsFocused(p power.Snapshot, total int, border lipgloss.Color) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	btn := p.Buttons
	var lines []string

	lines = append(lines, label.Render("Power Button")+value.Render(btn.PowerKeyAction))
	lines = append(lines, label.Render("Lid Close")+value.Render(btn.LidSwitch))
	lines = append(lines, label.Render("Lid + AC")+value.Render(btn.LidSwitchPower))
	lines = append(lines, label.Render("Lid + Dock")+value.Render(btn.LidSwitchDocked))

	pctStatus := lipgloss.NewStyle().Foreground(colorDim).Render("hidden")
	if btn.ShowBatteryPct {
		pctStatus = lipgloss.NewStyle().Foreground(colorGreen).Render("visible")
	}
	lines = append(lines, label.Render("Battery %")+pctStatus)

	return groupBoxSections("System Buttons", []string{strings.Join(lines, "\n")}, total, border)
}

func renderPowerIdle(p power.Snapshot, total int) string {
	return renderPowerIdleFocused(p, total, colorBorder)
}

func renderPowerIdleFocused(p power.Snapshot, total int, border lipgloss.Color) string {
	idle := p.Idle

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("running")
	if !idle.Running {
		status = lipgloss.NewStyle().Foreground(colorRed).Render("stopped")
	}
	lines = append(lines, label.Render("Idle Daemon")+status)

	if idle.KbdBacklightSec > 0 {
		lines = append(lines, label.Render("Kbd Light Off")+
			value.Render(formatTimeout(idle.KbdBacklightSec)))
	}

	if idle.ScreensaverSec > 0 {
		lines = append(lines, label.Render("Screensaver")+
			value.Render(formatTimeout(idle.ScreensaverSec)))
	}

	if idle.LockSec > 0 {
		lines = append(lines, label.Render("Screen Lock")+
			value.Render(formatTimeout(idle.LockSec)))
	}

	if idle.DPMSOffSec > 0 {
		lines = append(lines, label.Render("Screen Off")+
			value.Render(formatTimeout(idle.DPMSOffSec)))
	}

	if !idle.Running && idle.ScreensaverSec == 0 && idle.LockSec == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorDim).Italic(true).
			Render("  hypridle not configured"))
	}

	return groupBoxSections("Screen & Idle", []string{strings.Join(lines, "\n")}, total, border)
}

func renderPowerSleep(p power.Snapshot, total int) string {
	if len(p.SleepStates) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	lines = append(lines, label.Render("States")+
		value.Render(strings.Join(p.SleepStates, ", ")))
	if p.MemSleep != "" {
		lines = append(lines, label.Render("Active")+
			lipgloss.NewStyle().Foreground(colorAccent).Render(p.MemSleep))
	}

	return groupBoxSections("Sleep", []string{strings.Join(lines, "\n")}, total, colorBorder)
}
