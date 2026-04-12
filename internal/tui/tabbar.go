package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

func renderTabBar(s *core.State, width int) string {
	tabs := core.AllTabs()

	activeStyle := lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(colorDim).
		Padding(0, 1)

	bf := lipgloss.NewStyle().Foreground(colorBorder)

	type cell struct {
		label  string
		w      int
		active bool
	}

	cells := make([]cell, len(tabs))
	for i, t := range tabs {
		raw := t.Key + " " + t.Title
		if t.ID == s.ActiveTab {
			r := activeStyle.Render(raw)
			cells[i] = cell{r, lipgloss.Width(r), true}
		} else {
			r := inactiveStyle.Render(raw)
			cells[i] = cell{r, lipgloss.Width(r), false}
		}
	}

	top := bf.Render(strings.Repeat("─", width))

	var mid strings.Builder
	for _, c := range cells {
		mid.WriteString(c.label)
	}

	return top + "\n" + mid.String()
}
