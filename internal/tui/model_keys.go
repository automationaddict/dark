package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/sysinfo"
)

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state.HelpSearchMode {
		return m.handleHelpSearchKey(msg)
	}

	switch msg.String() {
	case "?", "esc":
		m.state.CloseHelp()
	case "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.state.ScrollHelp(1)
	case "k", "up":
		m.state.ScrollHelp(-1)
	case "pgdown", "ctrl+d", " ":
		m.state.ScrollHelp(10)
	case "pgup", "ctrl+u":
		m.state.ScrollHelp(-10)
	case "g", "home":
		m.state.ScrollHelpTo(0)
	case "G", "end":
		if m.state.HelpDoc != nil {
			m.state.ScrollHelpTo(len(m.state.HelpDoc.Lines))
		}
	case "]", "}":
		m.state.JumpHelpSection(1)
	case "[", "{":
		m.state.JumpHelpSection(-1)
	case "+", "=":
		m.state.ResizeHelp(2)
	case "-", "_":
		m.state.ResizeHelp(-2)
	case "/":
		m.state.BeginHelpSearch()
	case "n":
		m.state.NextHelpMatch(1)
	case "N":
		m.state.NextHelpMatch(-1)
	}
	return m, nil
}

func (m Model) handleHelpSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.state.CommitHelpSearch()
	case tea.KeyEsc:
		m.state.CancelHelpSearch()
	case tea.KeyBackspace:
		m.state.BackspaceSearch()
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyRunes, tea.KeySpace:
		for _, r := range msg.Runes {
			m.state.AppendSearchRune(r)
		}
	}
	return m, nil
}

// handleKey is the top-level key dispatcher. Global keys (quit, esc,
// enter, tabs, navigation) are handled here; context-aware action
// shortcuts are delegated to handleActionKey.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state.ActiveTab == core.TabF2 && m.state.AppstoreSearchActive {
		return m.handleAppstoreSearchInput(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		return m.handleEscKey()
	case "enter":
		return m.handleEnterKey()
	case "?":
		m.state.OpenHelp()
		return m, nil
	case "ctrl+r":
		if !sysinfo.IsDevBuild() {
			return m, nil
		}
		if m.state.Rebuilding {
			return m, nil
		}
		m.state.Rebuilding = true
		m.state.BuildError = ""
		return m, m.rebuildCmd()
	case "f1":
		m.switchTab(core.TabSettings, "on_f1")
	case "f2":
		m.switchTab(core.TabF2, "on_f2")
	case "f3":
		m.switchTab(core.TabF3, "on_f3")
	case "f4":
		m.switchTab(core.TabF4, "on_f4")
	case "f5":
		m.switchTab(core.TabF5, "on_f5")
		return m, m.loadScriptingIfNeeded()
	case "f6":
		m.switchTab(core.TabF6, "on_f6")
	case "f7":
		m.switchTab(core.TabF7, "on_f7")
	case "f8":
		m.switchTab(core.TabF8, "on_f8")
	case "f9":
		m.switchTab(core.TabF9, "on_f9")
	case "f10":
		m.switchTab(core.TabF10, "on_f10")
	case "f11":
		m.switchTab(core.TabF11, "on_f11")
	case "f12":
		m.switchTab(core.TabF12, "on_f12")
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "pgup":
		m.handlePageKey(-1)
	case "pgdown":
		m.handlePageKey(1)
	case "tab":
		return m.handleTabKey(1)
	case "shift+tab":
		return m.handleTabKey(-1)
	default:
		return m.handleActionKey(msg.String())
	}
	return m, nil
}

// handleEscKey closes the deepest open drill-in, then content focus,
// then quits.
func (m Model) handleEscKey() (tea.Model, tea.Cmd) {
	switch {
	case m.state.NetworkRoutesOpen:
		m.state.CloseNetworkRoutes()
	case m.state.BluetoothDeviceInfoOpen:
		m.state.CloseBluetoothDeviceInfo()
	case m.state.AudioDeviceInfoOpen:
		m.state.CloseAudioDeviceInfo()
	case m.state.AppstoreDetailOpen:
		m.state.CloseAppstoreDetail()
	case m.state.UsersContentFocused:
		m.state.UsersContentFocused = false
	case m.state.AudioContentFocused:
		m.state.AudioContentFocused = false
	case m.state.DisplayContentFocused:
		m.state.DisplayContentFocused = false
	case m.state.NetworkContentFocused:
		m.state.NetworkContentFocused = false
	case m.state.WifiContentFocused:
		m.state.WifiContentFocused = false
	case m.state.BluetoothContentFocused:
		m.state.BluetoothContentFocused = false
	case m.state.WorkspacesContentFocused:
		m.state.WorkspacesContentFocused = false
	case m.state.AppearanceContentFocused:
		m.state.AppearanceContentFocused = false
	case m.state.OmarchyLinksFocused:
		m.state.OmarchyLinksFocused = false
	case m.state.KeybindTableFocused:
		m.state.KeybindTableFocused = false
	case m.state.ScriptingContentFocused:
		m.state.ScriptingContentFocused = false
	case m.state.ContentFocused:
		m.state.FocusSidebar()
	default:
		return m, tea.Quit
	}
	return m, nil
}

// handleTabKey routes tab / shift-tab to panels that use tab for
// content-level focus cycling. Appearance → Theme is the first
// user of this pattern (toggling between the Theme and
// Backgrounds boxes); additional panels can plug in here without
// having to grow the main handleKey switch.
func (m Model) handleTabKey(delta int) (tea.Model, tea.Cmd) {
	if m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "appearance" &&
		m.state.AppearanceContentFocused &&
		m.state.ActiveAppearanceSection().ID == "theme" {
		m.state.CycleAppearanceThemeFocus(delta)
	}
	return m, nil
}
