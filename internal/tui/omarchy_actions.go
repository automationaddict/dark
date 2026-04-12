package tui

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/weblink"
)

func (m *Model) triggerWebLinkAdd() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 {
		return nil
	}
	m.dialog = NewDialog("Add web link", []DialogFieldSpec{
		{Key: "name", Label: "Name"},
		{Key: "url", Label: "URL"},
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		url := result["url"]
		if name == "" || url == "" {
			return nil
		}
		return func() tea.Msg {
			if err := weblink.Install(name, url); err != nil {
				return nil
			}
			apps, _ := weblink.ListWebApps()
			return WebLinksMsg(apps)
		}
	})
	return nil
}

func (m *Model) triggerWebLinkRemove() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.ContentFocused {
		return nil
	}
	app, ok := m.state.SelectedWebLink()
	if !ok {
		return nil
	}
	name := app.Name
	m.dialog = NewDialog("Remove "+name+"?", nil, func(_ DialogResult) tea.Cmd {
		return func() tea.Msg {
			weblink.Remove(name)
			apps, _ := weblink.ListWebApps()
			return WebLinksMsg(apps)
		}
	})
	return nil
}

func (m *Model) triggerWebLinkEdit() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.ContentFocused {
		return nil
	}
	app, ok := m.state.SelectedWebLink()
	if !ok {
		return nil
	}
	oldName := app.Name
	m.dialog = NewDialog("Edit "+oldName, []DialogFieldSpec{
		{Key: "name", Label: "Name", Value: app.Name},
		{Key: "url", Label: "URL", Value: app.URL},
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		url := result["url"]
		if name == "" || url == "" {
			return nil
		}
		return func() tea.Msg {
			weblink.Remove(oldName)
			weblink.Install(name, url)
			apps, _ := weblink.ListWebApps()
			return WebLinksMsg(apps)
		}
	})
	return nil
}

func (m *Model) triggerWebLinkOpen() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.ContentFocused {
		return nil
	}
	app, ok := m.state.SelectedWebLink()
	if !ok {
		return nil
	}
	url := app.URL
	return func() tea.Msg {
		launchWebApp(url)
		return nil
	}
}

func launchWebApp(url string) {
	cmd := exec.Command("omarchy-launch-webapp", url)
	cmd.Start()
}
