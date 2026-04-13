package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appstore"
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

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When the appstore search input is active, swallow all keys
	// so typed characters don't trigger action shortcuts.
	if m.state.ActiveTab == core.TabF2 && m.state.AppstoreSearchActive {
		return m.handleAppstoreSearchInput(msg)
	}
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		return m, tea.Quit
	case "esc":
		// Deepest drill-ins close first, then content focus, then quit.
		if m.state.NetworkRoutesOpen {
			m.state.CloseNetworkRoutes()
			return m, nil
		}
		if m.state.BluetoothDeviceInfoOpen {
			m.state.CloseBluetoothDeviceInfo()
			return m, nil
		}
		if m.state.AudioDeviceInfoOpen {
			m.state.CloseAudioDeviceInfo()
			return m, nil
		}
		if m.state.AppstoreDetailOpen {
			m.state.CloseAppstoreDetail()
			return m, nil
		}
		if m.state.WifiContentFocused {
			m.state.WifiContentFocused = false
			return m, nil
		}
		if m.state.BluetoothContentFocused {
			m.state.BluetoothContentFocused = false
			return m, nil
		}
		if m.state.KeybindTableFocused {
			m.state.KeybindTableFocused = false
			return m, nil
		}
		if m.state.ContentFocused {
			m.state.FocusSidebar()
			return m, nil
		}
		return m, tea.Quit
	case "enter":
		if !m.state.ContentFocused {
			if m.state.ActiveTab == core.TabF3 {
				m.state.ContentFocused = true
				return m, nil
			}
			if m.state.ActiveTab == core.TabF2 {
				m.state.ContentFocused = true
				m.state.AppstoreFocus = core.AppstoreFocusResults
				// Load the category if results aren't already showing it.
				if !m.state.AppstoreResultsLoaded || m.state.AppstoreResults.Query.Category != m.categoryID() {
					return m, m.loadAppstoreCategoryCmd()
				}
				return m, nil
			}
			m.state.FocusContent()
			return m, nil
		}
		switch {
		case m.state.ActiveTab == core.TabF3:
			return m, m.triggerOmarchyEnter()
		case m.state.ActiveTab == core.TabF2:
			// Enter in results → open detail for highlighted package.
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
		case m.state.ActiveTab == core.TabSettings:
			switch m.state.ActiveSection().ID {
			case "wifi":
				if !m.state.WifiContentFocused {
					m.state.WifiContentFocused = true
					// Initialize selection to the connected network.
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
					// Initialize selection to the connected device.
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
			case "sound":
				if !m.state.AudioDeviceInfoOpen {
					m.state.OpenAudioDeviceInfo()
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
		}
		return m, nil
	case "?":
		m.state.OpenHelp()
		return m, nil
	case "ctrl+r":
		if m.state.Rebuilding {
			return m, nil
		}
		m.state.Rebuilding = true
		m.state.BuildError = ""
		return m, m.rebuildCmd()
	case "f1":
		m.state.SelectTab(core.TabSettings)
	case "f2":
		m.state.SelectTab(core.TabF2)
	case "f3":
		m.state.SelectTab(core.TabF3)
	case "f4":
		m.state.SelectTab(core.TabF4)
	case "f5":
		m.state.SelectTab(core.TabF5)
	case "f6":
		m.state.SelectTab(core.TabF6)
	case "f7":
		m.state.SelectTab(core.TabF7)
	case "f8":
		m.state.SelectTab(core.TabF8)
	case "f9":
		m.state.SelectTab(core.TabF9)
	case "f10":
		m.state.SelectTab(core.TabF10)
	case "f11":
		m.state.SelectTab(core.TabF11)
	case "f12":
		m.state.SelectTab(core.TabF12)
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "tab":
		if m.inSoundContent() {
			m.state.CycleAudioFocus()
		}
	case "1":
		if m.inPrivacyContent() {
			m.triggerPrivacyScreensaverDialog()
			return m, nil
		}
	case "2":
		if m.inPrivacyContent() {
			m.triggerPrivacyLockDialog()
			return m, nil
		}
	case "3":
		if m.inPrivacyContent() {
			m.triggerPrivacyScreenOffDialog()
			return m, nil
		}
	case "s":
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacySSHToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inUsersContent() {
			m.triggerUserShellChange()
			return m, nil
		}
		if m.inInputContent() {
			if cmd := m.triggerInputSensitivityDelta(0.05); cmd != nil {
				return m, cmd
			}
		}
		if m.inDisplayContent() {
			m.triggerDisplayScaleDialog()
			return m, nil
		}
		if cmd := m.triggerWifiScan(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothScanToggle(); cmd != nil {
			return m, cmd
		}
	case "c":
		if m.inUsersContent() {
			m.triggerUserRename()
			return m, nil
		}
		if cmd := m.triggerWifiConnect(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothConnect(); cmd != nil {
			return m, cmd
		}
	case "d":
		if m.inUsersContent() {
			m.triggerUserRemove()
			return m, nil
		}
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyDNDToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.state.ActiveTab == core.TabF3 {
			return m, m.triggerOmarchyDelete()
		}
		if cmd := m.triggerWifiDisconnect(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothDisconnect(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerNetworkRouteDelete(); cmd != nil {
			return m, cmd
		}
	case "z":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceBlurSize(1); cmd != nil {
				return m, cmd
			}
		}
		if m.inDateTimeContent() {
			m.triggerDTTimezoneDialog()
			return m, nil
		}
	case "f":
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacyFirewallToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inDateTimeContent() {
			if cmd := m.triggerDTClockFormatToggle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerWifiForget(); cmd != nil {
			return m, cmd
		}
	case "w":
		if m.inUsersContent() {
			if cmd := m.triggerUserAdminToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyWidthDelta(20); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayDpmsToggle(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerWifiPowerToggle(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothPowerToggle(); cmd != nil {
			return m, cmd
		}
	case "a":
		if m.inDisplayContent() && len(m.state.Display.Monitors) > 1 {
			m.state.OpenDisplayLayout()
			return m, nil
		}
		if m.inUsersContent() {
			m.triggerUserAdd()
			return m, nil
		}
		if m.inNotifyContent() {
			m.triggerNotifyAddRuleDialog()
			return m, nil
		}
		if m.inInputContent() {
			if cmd := m.triggerInputAccelProfileCycle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerWifiAutoconnectToggle(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothPairableToggle(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerNetworkRouteAdd(); cmd != nil {
			return m, cmd
		}
		if m.state.ActiveTab == core.TabF3 {
			return m, m.triggerOmarchyAdd()
		}
	case "h":
		if cmd := m.triggerWifiConnectHidden(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerNetworkUseDHCP(); cmd != nil {
			return m, cmd
		}
	case "e":
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacyDNSSECCycle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayToggleEnabled(); cmd != nil {
			return m, cmd
		}
		if m.state.ActiveTab == core.TabF3 {
			return m, m.triggerOmarchyEdit()
		}
		if cmd := m.triggerNetworkEditStatic(); cmd != nil {
			return m, cmd
		}
	case "p":
		if m.inUsersContent() {
			if cmd := m.triggerUserPasswordChange(); cmd != nil {
				return m, cmd
			}
			return m, nil
		}
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyAnchorCycle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inPowerContent() {
			if cmd := m.triggerPowerProfileCycle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inDisplayContent() {
			m.triggerDisplayPositionDialog()
			return m, nil
		}
		if cmd := m.triggerWifiAPToggle(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothPair(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerAudioCycleProfile(); cmd != nil {
			return m, cmd
		}
	case "I":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceGapsIn(-1); cmd != nil {
				return m, cmd
			}
		}
	case "o":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceGapsOut(1); cmd != nil {
				return m, cmd
			}
		}
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacyCoredumpCycle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inNotifyContent() {
			m.triggerNotifySoundDialog()
			return m, nil
		}
		if cmd := m.triggerAudioCyclePort(); cmd != nil {
			return m, cmd
		}
	case "u":
		if cmd := m.triggerBluetoothRemove(); cmd != nil {
			return m, cmd
		}
	case "t":
		if m.inAppearanceContent() {
			m.triggerAppearanceThemeDialog()
			return m, nil
		}
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacyDNSTLSCycle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inDateTimeContent() {
			m.triggerDTSetTimeDialog()
			return m, nil
		}
		if m.inInputContent() {
			if cmd := m.triggerInputTapToClickToggle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerBluetoothTrustToggle(); cmd != nil {
			return m, cmd
		}
		// triggerNetworkRoutesOpen returns nil unconditionally — it's
		// a state change, not an async command — so we just call it
		// and let the next View() see the new state.
		m.triggerNetworkRoutesOpen()
	case "y":
		if cmd := m.triggerBluetoothDiscoverableToggle(); cmd != nil {
			return m, cmd
		}
	case "r":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceRounding(1); cmd != nil {
				return m, cmd
			}
		}
		if m.inDateTimeContent() {
			if cmd := m.triggerDTRTCToggle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayCycleTransform(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothRename(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerNetworkReconfigure(); cmd != nil {
			return m, cmd
		}
	case "X":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceBlurPasses(-1); cmd != nil {
				return m, cmd
			}
		}
		if m.inDisplayContent() {
			m.triggerDisplayDeleteProfile()
			return m, nil
		}
		if m.state.ActiveTab == core.TabF2 {
			return m, m.triggerAppstoreRemove()
		}
	case "b":
		if m.inPowerContent() {
			m.triggerPowerButtonsDialog()
			return m, nil
		}
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceBorderCycle(1); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerBluetoothBlockToggle(); cmd != nil {
			return m, cmd
		}
	case "W":
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyWidthDelta(-20); cmd != nil {
				return m, cmd
			}
		}
	case "O":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceGapsOut(-1); cmd != nil {
				return m, cmd
			}
		}
		if m.inNotifyContent() {
			if cmd := m.triggerNotifySoundDisable(); cmd != nil {
				return m, cmd
			}
		}
	case "l":
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacyLocationToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inUsersContent() {
			if cmd := m.triggerUserLockToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyLayerToggle(); cmd != nil {
				return m, cmd
			}
		}
	case "x":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceBlurPasses(1); cmd != nil {
				return m, cmd
			}
		}
		if m.inPrivacyContent() {
			m.triggerPrivacyClearRecent()
			return m, nil
		}
		if m.inNotifyContent() {
			m.triggerNotifyRemoveRuleDialog()
			return m, nil
		}
		if cmd := m.triggerBluetoothCancelPair(); cmd != nil {
			return m, cmd
		}
	case "R":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceRounding(-1); cmd != nil {
				return m, cmd
			}
		}
		if m.inDisplayContent() {
			m.triggerDisplayMirrorDialog()
			return m, nil
		}
		if cmd := m.triggerBluetoothResetAlias(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerNetworkReset(); cmd != nil {
			return m, cmd
		}
		if m.state.ActiveTab == core.TabF2 && m.appstore.Refresh != nil {
			m.state.MarkAppstoreBusy()
			return m, m.appstore.Refresh()
		}
	case "T":
		if cmd := m.triggerBluetoothSetTimeout(); cmd != nil {
			return m, cmd
		}
	case "F":
		if cmd := m.triggerBluetoothScanFilter(); cmd != nil {
			return m, cmd
		}
	case "D":
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyDismissAll(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerAudioSetDefault(); cmd != nil {
			return m, cmd
		}
	case "M":
		if cmd := m.triggerAudioStreamMove(); cmd != nil {
			return m, cmd
		}
	case "Z":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceBlurSize(-1); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerAudioSuspendToggle(); cmd != nil {
			return m, cmd
		}
	case "K":
		if cmd := m.triggerAudioKillStream(); cmd != nil {
			return m, cmd
		}
	case "i":
		if m.inPowerContent() {
			m.triggerPowerIdleDialog()
			return m, nil
		}
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceGapsIn(1); cmd != nil {
				return m, cmd
			}
		}
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacyIndexerToggle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayIdentify(); cmd != nil {
			return m, cmd
		}
		if m.state.ActiveTab == core.TabF2 {
			return m, m.triggerAppstoreInstall()
		}
	case "U":
		if m.state.ActiveTab == core.TabF2 {
			return m, m.triggerAppstoreUpgrade()
		}
	case "/":
		if m.state.ActiveTab == core.TabF2 {
			m.state.OpenAppstoreSearch()
			return m, nil
		}
	case "B":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceBlurToggle(); cmd != nil {
				return m, cmd
			}
		}
	case "A":
		if m.inNetworkContent() {
			if cmd := m.triggerNetworkAirplaneToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceAnimToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.state.ActiveTab == core.TabF2 {
			m.state.AppstoreIncludeAUR = !m.state.AppstoreIncludeAUR
			return m, nil
		}
	case "E":
		if m.inPowerContent() {
			if cmd := m.triggerPowerEPPCycle(); cmd != nil {
				return m, cmd
			}
		}
	case "g":
		if m.inUsersContent() {
			m.triggerUserGroupAdd()
			return m, nil
		}
		if m.inPowerContent() {
			if cmd := m.triggerPowerGovernorCycle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayGammaDelta(-5); cmd != nil {
			return m, cmd
		}
	case "G":
		if m.inUsersContent() {
			m.triggerUserGroupRemove()
			return m, nil
		}
		if cmd := m.triggerDisplayGammaDelta(5); cmd != nil {
			return m, cmd
		}
	case "S":
		if m.inInputContent() {
			if cmd := m.triggerInputSensitivityDelta(-0.05); cmd != nil {
				return m, cmd
			}
		}
		if m.inDisplayContent() {
			m.triggerDisplaySaveProfile()
			return m, nil
		}
	case "P":
		if m.inDisplayContent() {
			m.triggerDisplayApplyProfile()
			return m, nil
		}
	case "v":
		if cmd := m.triggerDisplayVrrToggle(); cmd != nil {
			return m, cmd
		}
	case "L":
		if m.inInputContent() {
			m.triggerInputKBLayoutDialog()
			return m, nil
		}
	case "+", "=":
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyTimeoutDelta(1000); cmd != nil {
				return m, cmd
			}
		}
		if m.inInputContent() {
			if cmd := m.triggerInputRepeatRateDelta(5); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayScaleUp(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerAudioVolumeDelta(core.VolumeStepPercent); cmd != nil {
			return m, cmd
		}
		if m.state.ActiveTab == core.TabSettings {
			m.state.ResizeSidebar(1)
		}
	case "-", "_":
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyTimeoutDelta(-1000); cmd != nil {
				return m, cmd
			}
		}
		if m.inInputContent() {
			if cmd := m.triggerInputRepeatRateDelta(-5); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayScaleDown(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerAudioVolumeDelta(-core.VolumeStepPercent); cmd != nil {
			return m, cmd
		}
		if m.state.ActiveTab == core.TabSettings {
			m.state.ResizeSidebar(-1)
		}
	case "<":
		if cmd := m.triggerAudioBalanceDelta(-5); cmd != nil {
			return m, cmd
		}
	case ">":
		if cmd := m.triggerAudioBalanceDelta(5); cmd != nil {
			return m, cmd
		}
	case "m":
		if m.inPrivacyContent() {
			if cmd := m.triggerPrivacyMACCycle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inDisplayContent() {
			m.triggerDisplayModeDialog()
			return m, nil
		}
		if cmd := m.triggerAudioMuteToggle(); cmd != nil {
			return m, cmd
		}
	case "[":
		if m.inInputContent() {
			if cmd := m.triggerInputRepeatDelayDelta(-50); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayBrightnessDelta(-5); cmd != nil {
			return m, cmd
		}
	case "]":
		if m.inInputContent() {
			if cmd := m.triggerInputRepeatDelayDelta(50); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayBrightnessDelta(5); cmd != nil {
			return m, cmd
		}
	case "{":
		if cmd := m.triggerDisplayKbdBrightnessDelta(-5); cmd != nil {
			return m, cmd
		}
	case "}":
		if cmd := m.triggerDisplayKbdBrightnessDelta(5); cmd != nil {
			return m, cmd
		}
	case "n":
		if m.inDateTimeContent() {
			if cmd := m.triggerDTNTPToggle(); cmd != nil {
				return m, cmd
			}
		}
		if m.inInputContent() {
			if cmd := m.triggerInputNaturalScrollToggle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayNightLightToggle(); cmd != nil {
			return m, cmd
		}
	case "N":
		if m.inDisplayContent() {
			m.triggerDisplayNightLightTempDialog()
			return m, nil
		}
	}
	return m, nil
}
