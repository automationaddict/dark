package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
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
		if m.inTopBarContent() {
			m.triggerTopBarHeightDialog()
			return m, nil
		}
		if cmd := m.triggerWorkspaceHideSpecialToggle(); cmd != nil {
			return m, cmd
		}
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
			switch m.state.ActiveUpdateSection().ID {
			case "dark":
				return m, m.triggerDarkUpdateApply()
			default:
				return m, m.triggerOmarchyUpdate()
			}
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
	case "C":
		if cmd := m.triggerTopBarEditStyle(); cmd != nil {
			return m, cmd
		}
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
		if cmd := m.triggerWorkspaceMasterStatusCycle(); cmd != nil {
			return m, cmd
		}
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
		if cmd := m.triggerWorkspaceLayoutCycle(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerWorkspaceDefaultLayoutCycle(); cmd != nil {
			return m, cmd
		}
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

