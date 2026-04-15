package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
)

// This file holds the per-key handleActionX helpers — each one walks
// through the panels that bind a letter to an action, in priority
// order, and returns the first command produced. The dispatch switch
// in model_keys_actions.go routes each raw key to the matching
// handler here, keeping the top-level switch compact.

func (m Model) handleActionS() (tea.Model, tea.Cmd) {
	if m.inTopBarContent() {
		m.triggerTopBarSpacingDialog()
		return m, nil
	}
	if m.state.ActiveTab == core.TabF3 &&
		m.state.ActiveOmarchySection().ID == "limine" &&
		m.state.ActiveLimineSection().ID == "snapshots" {
		return m, m.triggerLimineSync()
	}
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
	if cmd := m.triggerTopBarEditConfig(); cmd != nil {
		return m, cmd
	}
	if cmd := m.triggerWorkspaceCursorWarpToggle(); cmd != nil {
		return m, cmd
	}
	if m.inScreensaverContent() {
		return m, m.triggerScreensaverEditContent()
	}
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
	if cmd := m.triggerWorkspaceDwindleForceSplitCycle(); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerWorkspaceAnimationsToggle(); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerScreensaverToggle(); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerTopBarCyclePosition(); cmd != nil {
		return m, cmd
	}
	if cmd := m.triggerWorkspaceDwindleToggle("preserve_split", m.state.Workspaces.Dwindle.PreserveSplit); cmd != nil {
		return m, cmd
	}
	if cmd := m.triggerScreensaverPreview(); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerTopBarToggle(); cmd != nil {
		return m, cmd
	}
	if cmd := m.triggerWorkspaceDwindleToggle("pseudotile", m.state.Workspaces.Dwindle.Pseudotile); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerTopBarRestart(); cmd != nil {
		return m, cmd
	}
	if cmd := m.triggerWorkspaceRename(); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerTopBarCycleLayer(); cmd != nil {
		return m, cmd
	}
	if cmd := m.triggerPowerIdleToggle(); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerTopBarReset(); cmd != nil {
		return m, cmd
	}
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
	if cmd := m.triggerWorkspaceMoveToMonitor(); cmd != nil {
		return m, cmd
	}
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
