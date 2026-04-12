package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

func renderSettings(s *core.State, width, height int) string {
	sidebar := renderSidebar(s, height)
	contentWidth := width - lipgloss.Width(sidebar)
	content := renderSettingsContent(s, contentWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderSidebar(s *core.State, height int) string {
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)

	var rows []string
	for i, sec := range s.Sections() {
		line := sec.Icon + "  " + sec.Label
		if i == s.SettingsFocus {
			rows = append(rows, active.Render(line))
		} else {
			rows = append(rows, item.Render(line))
		}
	}
	body := strings.Join(rows, "\n")
	return sidebarStyle.Height(height).Render(body)
}

func renderSettingsContent(s *core.State, width, height int) string {
	sec := s.ActiveSection()
	switch sec.ID {
	case "about":
		return renderAbout(s, width, height)
	case "wifi":
		return renderWifi(s, width, height)
	case "bluetooth":
		return renderBluetooth(s, width, height)
	case "sound":
		return renderSound(s, width, height)
	case "network":
		return renderNetwork(s, width, height)
	}
	title := contentTitle.Render(sec.Label)
	body := placeholderStyle.Render("Nothing wired up yet for " + sec.Label + ".")
	return contentStyle.
		Width(width).
		Height(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, body))
}

func renderEmpty(s *core.State, width, height int) string {
	tabs := core.AllTabs()
	var label string
	for _, t := range tabs {
		if t.ID == s.ActiveTab {
			label = t.Key
			break
		}
	}
	msg := placeholderStyle.Render(label + " — empty for now.")
	return contentStyle.Width(width).Height(height).Render(msg)
}
