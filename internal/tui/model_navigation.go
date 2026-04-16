package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
)

// moveSelection routes vertical-arrow input to whichever region currently
// owns focus. When the sidebar is focused, it walks between settings
// sections; when the content pane is focused, it moves the inner widget's
// selection (currently only the wifi adapter row).
func (m *Model) moveSelection(delta int) {
	switch m.state.ActiveTab {
	case core.TabF2:
		if m.state.F2OnUpdates() {
			m.state.MoveUpdateSection(delta)
			return
		}
		if m.state.ContentFocused && m.state.AppstoreDetailOpen {
			m.state.ScrollAppstoreDetail(delta)
		} else if m.state.ContentFocused {
			m.state.MoveAppstoreResult(delta)
		} else {
			prev := m.state.F2SidebarIdx
			m.state.MoveF2Sidebar(delta)
			if prev != m.state.F2SidebarIdx {
				m.publishEvent("on_f2_category", map[string]interface{}{
					"id": m.currentF2CategoryID(),
				})
			}
		}
		return
	case core.TabF3:
		if m.state.ContentFocused {
			m.state.MoveOmarchyFocus(delta)
		} else {
			prev := m.state.ActiveOmarchySection().ID
			m.state.MoveOmarchySidebar(delta)
			curr := m.state.ActiveOmarchySection().ID
			if prev != curr {
				m.publishEvent("on_f3_section", map[string]interface{}{"name": curr})
			}
		}
		return
	case core.TabF4:
		if !m.state.ContentFocused {
			// Outer F4 sidebar — only SSH for now, no-op.
			return
		}
		if m.state.SSHContentFocused {
			m.state.MoveSSHInner(delta)
		} else {
			m.state.MoveSSHSubsection(delta)
		}
		return
	case core.TabF5:
		if m.state.ScriptingContentFocused {
			m.state.MoveScriptingInner(delta)
		} else {
			m.state.MoveScriptingOuter(delta)
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
			if m.state.DisplayContentFocused {
				m.state.MoveDisplaySelection(delta)
			} else {
				m.state.MoveDisplaySection(delta)
			}
		case "sound":
			if m.state.AudioContentFocused {
				m.state.MoveAudioSelection(delta)
			} else {
				m.state.MoveAudioSection(delta)
				m.state.SyncAudioFocus()
			}
		case "network":
			if m.state.NetworkContentFocused {
				if m.state.NetworkRoutesOpen {
					m.state.MoveNetworkRouteSelection(delta)
				} else {
					sec := m.state.ActiveNetworkSection()
					switch sec.ID {
					case "interfaces":
						m.state.MoveNetworkSelection(delta)
					}
				}
			} else {
				m.state.MoveNetworkSection(delta)
			}
		case "users":
			if m.state.UsersContentFocused {
				m.state.MoveUsersIdx(delta)
			} else {
				m.state.MoveUsersSection(delta)
			}
		case "power":
			m.state.MovePowerSection(delta)
		case "input":
			m.state.MoveInputSection(delta)
		case "appearance":
			if m.state.AppearanceContentFocused &&
				m.state.ActiveAppearanceSection().ID == "theme" &&
				m.state.AppearanceThemeFocus == core.AppearanceFocusBackgrounds {
				m.state.MoveAppearanceBackground(delta)
			} else {
				m.state.MoveAppearanceSection(delta)
			}
		case "notifications":
			m.state.MoveNotifySection(delta)
		case "privacy":
			m.state.MovePrivacySection(delta)
		case "datetime":
			m.state.MoveDateTimeSection(delta)
		case "about":
			m.state.MoveAboutSection(delta)
		case "workspaces":
			if m.state.WorkspacesContentFocused {
				m.state.MoveWorkspaceSelection(delta)
			} else {
				m.state.MoveWorkspacesSection(delta)
			}
		}
		return
	}
	prev := m.state.ActiveSection().ID
	m.state.MoveSettingsFocus(delta)
	curr := m.state.ActiveSection().ID
	if prev != curr {
		m.publishEvent("on_f1_section", map[string]interface{}{"name": curr})
	}
}

// currentF2CategoryID returns the ID of the category currently
// highlighted in the F2 sidebar, or an empty string when the index
// is past the end (e.g. the Updates row).
func (m *Model) currentF2CategoryID() string {
	idx := m.state.F2SidebarIdx
	cats := m.state.Appstore.Categories
	if idx < 0 || idx >= len(cats) {
		return ""
	}
	return cats[idx].ID
}

func (m Model) rebuildCmd() tea.Cmd {
	bin := m.binPath
	return func() tea.Msg {
		return rebuildDoneMsg(core.Rebuild(bin))
	}
}
