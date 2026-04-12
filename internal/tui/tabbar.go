package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

func renderTabBar(s *core.State, width int) string {
	tabs := core.AllTabs()
	rendered := make([]string, 0, len(tabs))
	for _, t := range tabs {
		label := t.Key + " " + t.Title
		if t.ID == s.ActiveTab {
			rendered = append(rendered, tabItemActive.Render(label))
		} else {
			rendered = append(rendered, tabItem.Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
	return tabBarStyle.Width(width).Render(row)
}
