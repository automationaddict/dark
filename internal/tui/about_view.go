package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/sysinfo"
)

type aboutCard struct {
	title  string
	fields [][2]string
}

func renderAbout(s *core.State, width, height int) string {
	info := s.SysInfo

	cards := []aboutCard{
		{"System", [][2]string{
			{"Host", info.Hostname},
			{"OS", info.OSPretty},
			{"Kernel", info.Kernel},
			{"Arch", info.Arch},
			{"Uptime", sysinfo.FormatDuration(info.Uptime)},
		}},
		{"Hardware", [][2]string{
			{"CPU", info.CPUModel},
			{"Cores", fmt.Sprintf("%d", info.CPUCores)},
			{"Memory", formatMem(info.MemUsed, info.MemTotal)},
			{"Swap", formatMem(info.SwapUsed, info.SwapTotal)},
		}},
		{"Session", [][2]string{
			{"User", info.User},
			{"Shell", info.Shell},
			{"Terminal", info.Terminal},
			{"Desktop", info.Desktop},
		}},
		{"dark", [][2]string{
			{"Go", info.GoVersion},
			{"Binary", info.BinaryPath},
			{"Built", info.BinaryMTime.Format("2006-01-02 15:04:05")},
		}},
	}

	innerMax := 0
	innerBodies := make([]string, len(cards))
	for i, c := range cards {
		innerBodies[i] = cardInner(c.title, c.fields)
		if w := lipgloss.Width(innerBodies[i]); w > innerMax {
			innerMax = w
		}
	}

	// cardStyle uses Padding(0, 2), so lipgloss's Width includes 4 columns
	// of horizontal padding. We add that to innerMax so the content area
	// is large enough for the widest measured row without wrapping.
	frameWidth := innerMax + 4

	rendered := make([]string, len(cards))
	for i, inner := range innerBodies {
		rendered[i] = cardStyle.Width(frameWidth).Render(inner)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rendered...)
	return renderContentPane(width, height, body)
}

func cardInner(title string, fields [][2]string) string {
	rows := make([]string, 0, len(fields)+1)
	rows = append(rows, cardTitleStyle.Render(title))
	for _, f := range fields {
		label := fieldLabelStyle.Render(f[0])
		value := fieldValueStyle.Render(displayValue(f[1]))
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, value))
	}
	return strings.Join(rows, "\n")
}

func displayValue(v string) string {
	if strings.TrimSpace(v) == "" {
		return placeholderStyle.Render("—")
	}
	return v
}

func formatMem(used, total uint64) string {
	if total == 0 {
		return "—"
	}
	pct := float64(used) / float64(total) * 100
	return fmt.Sprintf("%s / %s  (%.0f%%)", sysinfo.FormatBytes(used), sysinfo.FormatBytes(total), pct)
}
