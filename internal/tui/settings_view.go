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

// sidebarEntry is one item in a sidebar list. Both F1 (settings
// sections) and F2 (appstore categories) use this so the sidebar
// rendering code is shared — same styles, same width, same function.
type sidebarEntry struct {
	Icon      string
	Label     string
	Enabled   bool
	Separator bool // render a divider line above this entry
}

// renderSidebarGeneric renders a sidebar from a list of entries with
// one entry highlighted. Used by both F1 and F2 so the rendering is
// identical. Disabled entries are dimmed but use the same container
// style so their dimensions match enabled entries exactly.
func renderSidebarGeneric(s *core.State, entries []sidebarEntry, selected int, height int) string {
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var rows []string
	for i, e := range entries {
		if e.Separator {
			sep := lipgloss.NewStyle().Foreground(colorBorder).
				Width(itemWidth).Render(strings.Repeat("─", itemWidth))
			rows = append(rows, sep)
		}
		line := e.Icon + "  " + e.Label
		if i == selected {
			rows = append(rows, active.Render(line))
		} else {
			if !e.Enabled {
				line = dim.Render(line)
			}
			rows = append(rows, item.Render(line))
		}
	}
	body := strings.Join(rows, "\n")
	sidebarFocused := !s.ContentFocused
	return renderSidebarPane(height, body, sidebarFocused)
}

func renderSidebar(s *core.State, height int) string {
	sections := s.Sections()
	entries := make([]sidebarEntry, len(sections))
	for i, sec := range sections {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	return renderSidebarGeneric(s, entries, s.SettingsFocus, height)
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
	case "display":
		return renderDisplay(s, width, height)
	case "sound":
		return renderSound(s, width, height)
	case "network":
		return renderNetwork(s, width, height)
	case "power":
		return renderPower(s, width, height)
	case "input":
		return renderInputDevices(s, width, height)
	case "notifications":
		return renderNotifications(s, width, height)
	case "datetime":
		return renderDateTime(s, width, height)
	case "privacy":
		return renderPrivacy(s, width, height)
	case "users":
		return renderUsers(s, width, height)
	case "appearance":
		return renderAppearance(s, width, height)
	}
	title := contentTitle.Render(sec.Label)
	body := placeholderStyle.Render("Nothing wired up yet for " + sec.Label + ".")
	return renderContentPane(width, height,
		lipgloss.JoinVertical(lipgloss.Left, title, body))
}

func renderEmpty(s *core.State, width, height int) string {
	sidebar := renderSidebarGeneric(s, nil, -1, height)
	contentWidth := width - lipgloss.Width(sidebar)

	tabs := core.AllTabs()
	var label string
	for _, t := range tabs {
		if t.ID == s.ActiveTab {
			label = t.Key
			break
		}
	}
	msg := placeholderStyle.Render(label + " — empty for now.")
	content := renderContentPane(contentWidth, height, msg)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}
