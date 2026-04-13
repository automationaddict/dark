package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
)

// moveSelection routes vertical-arrow input to whichever region currently
// owns focus. When the sidebar is focused, it walks between settings
// sections; when the content pane is focused, it moves the inner widget's
// selection (currently only the wifi adapter row).
func (m *Model) moveSelection(delta int) {
	switch m.state.ActiveTab {
	case core.TabF2:
		if m.state.ContentFocused && m.state.AppstoreDetailOpen {
			m.state.ScrollAppstoreDetail(delta)
		} else if m.state.ContentFocused {
			m.state.MoveAppstoreResult(delta)
		} else {
			m.state.MoveAppstoreCategory(delta)
		}
		return
	case core.TabF3:
		if m.state.ContentFocused {
			m.state.MoveOmarchyFocus(delta)
		} else {
			m.state.MoveOmarchySidebar(delta)
		}
		return
	case core.TabSettings:
	default:
		return
	}
	if m.state.ContentFocused {
		switch m.state.ActiveSection().ID {
		case "wifi":
			if m.state.WifiContentFocused {
				sec := m.state.ActiveWifiSection()
				switch sec.ID {
				case "adapters":
					m.state.MoveWifiSelection(delta)
				case "networks":
					m.state.MoveWifiNetworkSelection(delta)
				case "known":
					m.state.MoveWifiKnownSelection(delta)
				}
			} else {
				m.state.MoveWifiSection(delta)
			}
		case "bluetooth":
			if m.state.BluetoothContentFocused {
				sec := m.state.ActiveBluetoothSection()
				switch sec.ID {
				case "adapters":
					m.state.MoveBluetoothSelection(delta)
				case "devices":
					m.state.MoveBluetoothDeviceSelection(delta)
				}
			} else {
				m.state.MoveBluetoothSection(delta)
			}
		case "display":
			m.state.MoveDisplaySelection(delta)
		case "sound":
			m.state.MoveAudioSelection(delta)
		case "network":
			if m.state.NetworkRoutesOpen {
				m.state.MoveNetworkRouteSelection(delta)
			} else {
				m.state.MoveNetworkSelection(delta)
			}
		case "users":
			m.state.MoveUsersIdx(delta)
		case "power":
			m.state.MovePowerSection(delta)
		case "input", "notifications", "datetime", "privacy", "appearance":
			m.state.ScrollContent(delta)
		}
		return
	}
	m.state.MoveSettingsFocus(delta)
}

func (m Model) rebuildCmd() tea.Cmd {
	bin := m.binPath
	return func() tea.Msg {
		return rebuildDoneMsg(core.Rebuild(bin))
	}
}
