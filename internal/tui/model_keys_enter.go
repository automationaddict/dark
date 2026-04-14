package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appstore"
)

// handleEnterKey handles enter/return in every context: sidebar focus,
// content drill-in, and section-specific actions.
func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if !m.state.ContentFocused {
		return m.handleEnterSidebar()
	}

	switch {
	case m.state.ActiveTab == core.TabF3:
		return m, m.triggerOmarchyEnter()
	case m.state.ActiveTab == core.TabF2:
		return m.handleEnterAppstore()
	case m.state.ActiveTab == core.TabSettings:
		return m.handleEnterSettings()
	}
	return m, nil
}

// handleEnterSidebar focuses content for the active tab/section.
func (m Model) handleEnterSidebar() (tea.Model, tea.Cmd) {
	if m.state.ActiveTab == core.TabF3 {
		m.state.ContentFocused = true
		m.state.OmarchyLinksFocused = false
		m.state.KeybindTableFocused = false
		return m, nil
	}
	if m.state.ActiveTab == core.TabF2 {
		if m.state.F2OnUpdates() {
			return m, nil
		}
		m.state.ContentFocused = true
		m.state.AppstoreFocus = core.AppstoreFocusResults
		if !m.state.AppstoreResultsLoaded || m.state.AppstoreResults.Query.Category != m.categoryID() {
			return m, m.loadAppstoreCategoryCmd()
		}
		return m, nil
	}
	m.state.FocusContent()
	return m, nil
}

// handleEnterAppstore opens the detail view for the highlighted package.
func (m Model) handleEnterAppstore() (tea.Model, tea.Cmd) {
	if m.state.AppstoreFocus == core.AppstoreFocusResults && !m.state.AppstoreDetailOpen {
		pkg, ok := m.state.SelectedAppstorePackage()
		if ok && m.appstore.Detail != nil {
			m.state.MarkAppstoreBusy()
			return m, m.appstore.Detail(appstore.DetailRequest{
				Name:   pkg.Name,
				Origin: pkg.Origin,
			})
		}
	}
	return m, nil
}

// handleEnterSettings dispatches enter within a settings section.
func (m Model) handleEnterSettings() (tea.Model, tea.Cmd) {
	switch m.state.ActiveSection().ID {
	case "wifi":
		if !m.state.WifiContentFocused {
			m.state.WifiContentFocused = true
			sec := m.state.ActiveWifiSection()
			if sec.ID == "networks" {
				if adapter, ok := m.state.SelectedAdapter(); ok {
					m.state.WifiNetworkSelected = 0
					for i, n := range adapter.Networks {
						if n.Connected {
							m.state.WifiNetworkSelected = i
							break
						}
					}
				}
			}
			return m, nil
		}
	case "bluetooth":
		if !m.state.BluetoothContentFocused {
			m.state.BluetoothContentFocused = true
			sec := m.state.ActiveBluetoothSection()
			if sec.ID == "devices" {
				if sel := m.state.BluetoothSelected; sel < len(m.state.Bluetooth.Adapters) {
					adapter := m.state.Bluetooth.Adapters[sel]
					m.state.BluetoothDevSelected = 0
					for i, d := range adapter.Devices {
						if d.Connected {
							m.state.BluetoothDevSelected = i
							break
						}
					}
				}
			}
			return m, nil
		}
		if m.state.ActiveBluetoothSection().ID == "devices" && !m.state.BluetoothDeviceInfoOpen {
			m.state.OpenBluetoothDeviceInfo()
		}
	case "network":
		if !m.state.NetworkContentFocused {
			sec := m.state.ActiveNetworkSection()
			if sec.ID == "dns" {
				return m, nil
			}
			m.state.NetworkContentFocused = true
			return m, nil
		}
	case "display":
		if !m.state.DisplayContentFocused {
			sec := m.state.ActiveDisplaySection()
			if sec.ID == "monitors" {
				m.state.DisplayContentFocused = true
			}
			return m, nil
		}
	case "sound":
		if !m.state.AudioContentFocused {
			m.state.AudioContentFocused = true
			m.state.SyncAudioFocus()
			return m, nil
		}
		sec := m.state.ActiveAudioSection()
		if (sec.ID == "sinks" || sec.ID == "sources") && !m.state.AudioDeviceInfoOpen {
			m.state.OpenAudioDeviceInfo()
		}
	case "users":
		if !m.state.UsersContentFocused {
			m.state.UsersContentFocused = true
			return m, nil
		}
	case "power":
		switch m.state.ActivePowerSection().ID {
		case "profile":
			return m, m.triggerPowerProfileCycle()
		case "cpu":
			return m, m.triggerPowerGovernorCycle()
		case "buttons":
			m.triggerPowerButtonsDialog()
			return m, nil
		case "idle":
			m.triggerPowerIdleDialog()
			return m, nil
		}
	}
	return m, nil
}
