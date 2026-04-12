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

	var top, mid strings.Builder

	for i, c := range cells {
		if i == 0 {
			if c.active {
				top.WriteString(bf.Render("┐"))
			} else {
				top.WriteString(bf.Render("┌"))
			}
			mid.WriteString(bf.Render("│"))
		} else {
			prev := cells[i-1]
			switch {
			case prev.active && c.active:
				top.WriteString(bf.Render("│"))
			case prev.active:
				top.WriteString(bf.Render("┌"))
			case c.active:
				top.WriteString(bf.Render("┐"))
			default:
				top.WriteString(bf.Render("┬"))
			}
			mid.WriteString(bf.Render("│"))
		}

		if c.active {
			top.WriteString(strings.Repeat(" ", c.w))
		} else {
			top.WriteString(bf.Render(strings.Repeat("─", c.w)))
		}
		mid.WriteString(c.label)
	}

	last := cells[len(cells)-1]
	if last.active {
		top.WriteString(bf.Render("┌"))
	} else {
		top.WriteString(bf.Render("┬"))
	}
	mid.WriteString(bf.Render("│"))

	topW := lipgloss.Width(top.String())
	if topW < width {
		top.WriteString(bf.Render(strings.Repeat("─", width-topW)))
	}

	return top.String() + "\n" + mid.String()
}
