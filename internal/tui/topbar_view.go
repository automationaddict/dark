package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
)

// renderAppearanceTopBarSection is the Appearance → Top Bar content
// pane. It shows four blocks in order: Status (running state +
// scalar settings), Modules (three rows for left/center/right),
// Files (the two on-disk paths), and a hint line. Actions and the
// Omarchy defaults availability are made visible so the reset path
// is discoverable.
func renderAppearanceTopBarSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	focused := s.ContentFocused
	borderColor := colorBorder
	if focused {
		borderColor = colorAccent
	}

	var blocks []string
	blocks = append(blocks, renderTopBarStatus(s, innerWidth, borderColor))
	blocks = append(blocks, renderTopBarModules(s, innerWidth))
	blocks = append(blocks, renderTopBarFiles(s, innerWidth))
	blocks = append(blocks, renderTopBarHint(s))
	if s.TopBarActionError != "" {
		blocks = append(blocks, statusOfflineStyle.Render("  "+s.TopBarActionError))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderTopBarStatus(s *core.State, total int, border lipgloss.Color) string {
	label := detailLabelStyle.Width(14)
	value := detailValueStyle

	stateText := boolIndicator(s.TopBar.Running)
	if s.TopBarBusy {
		stateText = statusBusyStyle.Render("working…")
	}

	lines := []string{
		label.Render("Daemon") + value.Render(stateText),
		label.Render("Position") + value.Render(orDash(s.TopBar.Position)),
		label.Render("Layer") + value.Render(orDash(s.TopBar.Layer)),
		label.Render("Height") + value.Render(pxValue(s.TopBar.Height)),
		label.Render("Spacing") + value.Render(pxValue(s.TopBar.Spacing)),
	}
	return groupBoxSections("Status", lines, total, border)
}

func renderTopBarModules(s *core.State, total int) string {
	label := detailLabelStyle.Width(14)
	value := detailValueStyle

	lines := []string{
		label.Render("Left") + value.Render(joinModules(s.TopBar.ModulesLeft)),
		label.Render("Center") + value.Render(joinModules(s.TopBar.ModulesCenter)),
		label.Render("Right") + value.Render(joinModules(s.TopBar.ModulesRight)),
	}
	return groupBoxSections("Modules", lines, total, colorBorder)
}

func renderTopBarFiles(s *core.State, total int) string {
	label := detailLabelStyle.Width(14)
	value := detailValueStyle

	lines := []string{
		label.Render("Config") + value.Render(orDash(s.TopBar.ConfigPath)),
		label.Render("Style") + value.Render(orDash(s.TopBar.StylePath)),
	}
	if s.TopBar.DefaultsAvailable {
		lines = append(lines, label.Render("Defaults")+
			lipgloss.NewStyle().Foreground(colorGreen).Render("available"))
	} else {
		lines = append(lines, label.Render("Defaults")+
			lipgloss.NewStyle().Foreground(colorGold).Render("not found"))
	}
	return groupBoxSections("Files", lines, total, colorBorder)
}

func renderTopBarHint(s *core.State) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var parts []string
	parts = append(parts, accent.Render("t")+" toggle")
	parts = append(parts, accent.Render("r")+" restart")
	if s.TopBar.DefaultsAvailable {
		parts = append(parts, accent.Render("R")+" reset")
	}
	parts = append(parts, accent.Render("p")+" position")
	parts = append(parts, accent.Render("l")+" layer")
	parts = append(parts, accent.Render("h")+" height")
	parts = append(parts, accent.Render("s")+" spacing")
	parts = append(parts, accent.Render("c")+" edit config")
	parts = append(parts, accent.Render("C")+" edit style")
	return dim.Render("  " + strings.Join(parts, "  "))
}

func joinModules(modules []string) string {
	if len(modules) == 0 {
		return "—"
	}
	return strings.Join(modules, ", ")
}

func pxValue(n int) string {
	if n <= 0 {
		return "—"
	}
	return fmt.Sprintf("%dpx", n)
}
