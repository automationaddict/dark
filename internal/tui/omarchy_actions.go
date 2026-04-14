package tui

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/links"
)

// --- Dispatchers ---

func (m *Model) triggerOmarchyEnter() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "links":
		if !m.state.OmarchyLinksFocused {
			m.state.OmarchyLinksFocused = true
			return nil
		}
		switch m.state.ActiveLinksSection().ID {
		case "weblinks":
			return m.triggerWebLinkOpen()
		case "tuilinks":
			return m.triggerTUILinkLaunch()
		case "helplinks":
			return m.triggerHelpLinkOpen()
		}
	case "keybindings":
		if !m.state.KeybindTableFocused {
			m.state.KeybindTableFocused = true
			return nil
		}
		return m.triggerKeybindEdit()
	}
	return nil
}

func (m *Model) triggerOmarchyAdd() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "links":
		switch m.state.ActiveLinksSection().ID {
		case "weblinks":
			return m.triggerWebLinkAdd()
		case "tuilinks":
			return m.triggerTUILinkAdd()
		case "helplinks":
			return m.triggerHelpLinkAdd()
		}
	case "keybindings":
		return m.triggerKeybindAdd()
	}
	return nil
}

func (m *Model) triggerOmarchyEdit() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "links":
		switch m.state.ActiveLinksSection().ID {
		case "weblinks":
			return m.triggerWebLinkEdit()
		case "tuilinks":
			return m.triggerTUILinkEdit()
		case "helplinks":
			return m.triggerHelpLinkEdit()
		}
	case "keybindings":
		return m.triggerKeybindEdit()
	}
	return nil
}

func (m *Model) triggerOmarchyDelete() tea.Cmd {
	switch m.state.ActiveOmarchySection().ID {
	case "links":
		switch m.state.ActiveLinksSection().ID {
		case "weblinks":
			return m.triggerWebLinkRemove()
		case "tuilinks":
			return m.triggerTUILinkRemove()
		case "helplinks":
			return m.triggerHelpLinkRemove()
		}
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
			if err := links.AddWebLink(name, url); err != nil {
				return nil
			}
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerWebLinkRemove() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
		return nil
	}
	app, ok := m.state.SelectedWebLink()
	if !ok {
		return nil
	}
	name := app.Name
	m.dialog = NewDialog("Remove "+name+"?", nil, func(_ DialogResult) tea.Cmd {
		return func() tea.Msg {
			links.RemoveWebLink(name)
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerWebLinkEdit() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
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
			links.RemoveWebLink(oldName)
			links.AddWebLink(name, url)
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerWebLinkOpen() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
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

// --- Help Links ---

func (m *Model) triggerHelpLinkOpen() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
		return nil
	}
	link, ok := m.state.SelectedHelpLink()
	if !ok {
		return nil
	}
	url := link.URL
	return func() tea.Msg {
		launchWebApp(url)
		return nil
	}
}

func (m *Model) triggerHelpLinkAdd() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 {
		return nil
	}
	m.dialog = NewDialog("Add help link", []DialogFieldSpec{
		{Key: "name", Label: "Name"},
		{Key: "url", Label: "URL"},
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		url := result["url"]
		if name == "" || url == "" {
			return nil
		}
		return func() tea.Msg {
			if err := links.AddHelpLink(name, url); err != nil {
				return nil
			}
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerHelpLinkRemove() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
		return nil
	}
	link, ok := m.state.SelectedHelpLink()
	if !ok {
		return nil
	}
	name := link.Name
	m.dialog = NewDialog("Remove "+name+"?", nil, func(_ DialogResult) tea.Cmd {
		return func() tea.Msg {
			links.RemoveHelpLink(name)
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerHelpLinkEdit() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
		return nil
	}
	link, ok := m.state.SelectedHelpLink()
	if !ok {
		return nil
	}
	oldName := link.Name
	m.dialog = NewDialog("Edit "+oldName, []DialogFieldSpec{
		{Key: "name", Label: "Name", Value: link.Name},
		{Key: "url", Label: "URL", Value: link.URL},
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		url := result["url"]
		if name == "" || url == "" {
			return nil
		}
		return func() tea.Msg {
			links.RemoveHelpLink(oldName)
			links.AddHelpLink(name, url)
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
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
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		command := result["command"]
		style := result["style"]
		if name == "" || command == "" {
			return nil
		}
		return func() tea.Msg {
			links.AddTUILink(name, command, style)
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerTUILinkRemove() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
		return nil
	}
	app, ok := m.state.SelectedTUILink()
	if !ok {
		return nil
	}
	name := app.Name
	m.dialog = NewDialog("Remove "+name+"?", nil, func(_ DialogResult) tea.Cmd {
		return func() tea.Msg {
			links.RemoveTUILink(name)
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerTUILinkEdit() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
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
			links.RemoveTUILink(oldName)
			links.AddTUILink(name, command, style)
			lf, _ := links.Load()
			return LinksMsg(lf)
		}
	})
	return nil
}

func (m *Model) triggerTUILinkLaunch() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.OmarchyLinksFocused {
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
