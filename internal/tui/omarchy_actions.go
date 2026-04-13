package tui

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/tuilink"
	"github.com/johnnelson/dark/internal/services/weblink"
)

// --- Dispatchers ---

func (m *Model) triggerOmarchyEnter() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "weblinks":
		return m.triggerWebLinkOpen()
	case "tuilinks":
		return m.triggerTUILinkLaunch()
	case "keybindings":
		return m.triggerKeybindEdit()
	}
	return nil
}

func (m *Model) triggerOmarchyAdd() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "weblinks":
		return m.triggerWebLinkAdd()
	case "tuilinks":
		return m.triggerTUILinkAdd()
	case "keybindings":
		return m.triggerKeybindAdd()
	}
	return nil
}

func (m *Model) triggerOmarchyEdit() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "weblinks":
		return m.triggerWebLinkEdit()
	case "tuilinks":
		return m.triggerTUILinkEdit()
	case "keybindings":
		return m.triggerKeybindEdit()
	}
	return nil
}

func (m *Model) triggerOmarchyDelete() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "weblinks":
		return m.triggerWebLinkRemove()
	case "tuilinks":
		return m.triggerTUILinkRemove()
	case "keybindings":
		return m.triggerKeybindRemove()
	}
	return nil
}

// --- Web Links ---

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

// --- TUI Links ---

func (m *Model) triggerTUILinkAdd() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 {
		return nil
	}
	m.dialog = NewDialog("Add TUI link", []DialogFieldSpec{
		{Key: "name", Label: "Name"},
		{Key: "command", Label: "Command"},
		{Key: "style", Label: "Style (float/tile)", Value: "float"},
		{Key: "icon", Label: "Icon URL"},
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		command := result["command"]
		style := result["style"]
		iconURL := result["icon"]
		if name == "" || command == "" {
			return nil
		}
		return func() tea.Msg {
			tuilink.Install(name, command, style, iconURL)
			apps, _ := tuilink.ListTUIApps()
			return TUILinksMsg(apps)
		}
	})
	return nil
}

func (m *Model) triggerTUILinkRemove() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.ContentFocused {
		return nil
	}
	app, ok := m.state.SelectedTUILink()
	if !ok {
		return nil
	}
	name := app.Name
	m.dialog = NewDialog("Remove "+name+"?", nil, func(_ DialogResult) tea.Cmd {
		return func() tea.Msg {
			tuilink.Remove(name)
			apps, _ := tuilink.ListTUIApps()
			return TUILinksMsg(apps)
		}
	})
	return nil
}

func (m *Model) triggerTUILinkEdit() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.ContentFocused {
		return nil
	}
	app, ok := m.state.SelectedTUILink()
	if !ok {
		return nil
	}
	oldName := app.Name
	m.dialog = NewDialog("Edit "+oldName, []DialogFieldSpec{
		{Key: "name", Label: "Name", Value: app.Name},
		{Key: "command", Label: "Command", Value: app.Command},
		{Key: "style", Label: "Style (float/tile)", Value: app.Style},
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		command := result["command"]
		style := result["style"]
		if name == "" || command == "" {
			return nil
		}
		return func() tea.Msg {
			tuilink.Remove(oldName)
			tuilink.Install(name, command, style, "")
			apps, _ := tuilink.ListTUIApps()
			return TUILinksMsg(apps)
		}
	})
	return nil
}

func (m *Model) triggerTUILinkLaunch() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.ContentFocused {
		return nil
	}
	app, ok := m.state.SelectedTUILink()
	if !ok {
		return nil
	}
	command := app.Command
	style := app.Style
	return func() tea.Msg {
		launchTUIApp(command, style)
		return nil
	}
}

func launchTUIApp(command, style string) {
	appClass := "TUI." + style
	cmd := exec.Command("xdg-terminal-exec", "--app-id="+appClass, "-e", command)
	cmd.Start()
}
