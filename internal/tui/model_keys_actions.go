package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
)

// handleActionKey dispatches context-aware action shortcuts (letter and
// symbol keys). Each key falls through section checks in priority order.
func (m Model) handleActionKey(key string) (tea.Model, tea.Cmd) {
	switch key {
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
		return m.handleActionS()
	case "c":
		return m.handleActionC()
	case "d":
		return m.handleActionD()
	case "z":
		return m.handleActionZ()
	case "f":
		return m.handleActionF()
	case "w":
		return m.handleActionW()
	case "a":
		return m.handleActionA()
	case "h":
		if cmd := m.triggerWifiConnectHidden(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerNetworkUseDHCP(); cmd != nil {
			return m, cmd
		}
	case "e":
		return m.handleActionE()
	case "p":
		return m.handleActionP()
	case "I":
		if m.inAppearanceContent() {
			if cmd := m.triggerAppearanceGapsIn(-1); cmd != nil {
				return m, cmd
			}
		}
	case "o":
		return m.handleActionO()
	case "u":
		if m.state.ActiveTab == core.TabF2 && m.state.F2OnUpdates() {
			return m, m.triggerOmarchyUpdate()
		}
		if cmd := m.triggerBluetoothRemove(); cmd != nil {
			return m, cmd
		}
	case "t":
		return m.handleActionT()
	case "y":
		if cmd := m.triggerBluetoothDiscoverableToggle(); cmd != nil {
			return m, cmd
		}
	case "r":
		return m.handleActionR()
	case "X":
		return m.handleActionShiftX()
	case "b":
		return m.handleActionB()
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
		return m.handleActionL()
	case "x":
		return m.handleActionX()
	case "R":
		return m.handleActionShiftR()
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
		return m.handleActionI()
	case "U":
		if m.state.ActiveTab == core.TabF2 && !m.state.F2OnUpdates() {
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
		return m.handleActionShiftA()
	case "E":
		if m.inPowerContent() {
			if cmd := m.triggerPowerEPPCycle(); cmd != nil {
				return m, cmd
			}
		}
	case "g":
		return m.handleActionG()
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
		return m.handleActionPlus()
	case "-", "_":
		return m.handleActionMinus()
	case "<":
		if cmd := m.triggerAudioBalanceDelta(-5); cmd != nil {
			return m, cmd
		}
	case ">":
		if cmd := m.triggerAudioBalanceDelta(5); cmd != nil {
			return m, cmd
		}
	case "m":
		return m.handleActionM()
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
		return m.handleActionN()
	case "N":
		if m.inDisplayContent() {
			m.triggerDisplayNightLightTempDialog()
			return m, nil
		}
	}
	return m, nil
}

func (m Model) handleActionS() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionC() (tea.Model, tea.Cmd) {
	if m.inF2Updates() {
		return m, m.triggerChannelChange()
	}
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
	return m, nil
}

func (m Model) handleActionD() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionZ() (tea.Model, tea.Cmd) {
	if m.inAppearanceContent() {
		if cmd := m.triggerAppearanceBlurSize(1); cmd != nil {
			return m, cmd
		}
	}
	if m.inDateTimeContent() {
		m.triggerDTTimezoneDialog()
		return m, nil
	}
	return m, nil
}

func (m Model) handleActionF() (tea.Model, tea.Cmd) {
	if m.inAppearanceContent() {
		m.triggerAppearanceFontDialog()
		return m, nil
	}
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
	return m, nil
}

func (m Model) handleActionW() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionA() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionE() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionP() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionO() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionT() (tea.Model, tea.Cmd) {
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
	m.triggerNetworkRoutesOpen()
	return m, nil
}

func (m Model) handleActionR() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionShiftX() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionB() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionL() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionX() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionShiftR() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionI() (tea.Model, tea.Cmd) {
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
		if m.inF2Firmware() {
			return m, m.triggerFwupdInstall()
		}
		return m, m.triggerAppstoreInstall()
	}
	return m, nil
}

func (m Model) handleActionShiftA() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionG() (tea.Model, tea.Cmd) {
	if m.inDisplayContent() && m.state.ActiveDisplaySection().ID == "gpu" {
		return m, m.triggerGPUModeToggle()
	}
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
	return m, nil
}

func (m Model) handleActionPlus() (tea.Model, tea.Cmd) {
	if m.inAppearanceFonts() {
		if cmd := m.triggerAppearanceFontSize(1); cmd != nil {
			return m, cmd
		}
	}
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
	return m, nil
}

func (m Model) handleActionMinus() (tea.Model, tea.Cmd) {
	if m.inAppearanceFonts() {
		if cmd := m.triggerAppearanceFontSize(-1); cmd != nil {
			return m, cmd
		}
	}
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
	return m, nil
}

func (m Model) handleActionM() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleActionN() (tea.Model, tea.Cmd) {
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
	return m, nil
}
