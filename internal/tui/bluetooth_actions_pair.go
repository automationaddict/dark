package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
)

func (m *Model) triggerBluetoothConnect() tea.Cmd {
	if !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
	}
	if m.bluetooth.Connect == nil {
		return m.notifyUnavailable("Bluetooth")
	}
	dev, ok := m.state.SelectedBluetoothDevice()
	if !ok || dev.Path == "" {
		return nil
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.Connect(dev.Path)
}

func (m *Model) triggerBluetoothDisconnect() tea.Cmd {
	if !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
	}
	if m.bluetooth.Disconnect == nil {
		return m.notifyUnavailable("Bluetooth")
	}
	dev, ok := m.state.SelectedBluetoothDevice()
	if !ok || dev.Path == "" {
		return nil
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.Disconnect(dev.Path)
}

// triggerBluetoothPair sends a pair command for the highlighted device.
// When BlueZ has flagged the device as LegacyPairing (pre-SSP), a PIN
// dialog pops first so the user can supply the code the device expects
// (most old hardware uses "0000" or "1234"). Modern devices go straight
// through the numeric-comparison auto-confirm path the agent handles.
func (m *Model) triggerBluetoothPair() tea.Cmd {
	if m.bluetooth.Pair == nil || !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
	}
	dev, ok := m.state.SelectedBluetoothDevice()
	if !ok || dev.Path == "" {
		return nil
	}

	if dev.LegacyPairing {
		m.openLegacyPairDialog(dev.Path, dev.DisplayName())
		return nil
	}

	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	m.notifyInfo("Bluetooth", "Pairing with "+dev.DisplayName()+"…")
	return m.bluetooth.Pair(dev.Path, "")
}

func (m *Model) openLegacyPairDialog(devicePath, deviceName string) {
	actions := m.bluetooth
	state := m.state
	notifier := m.notifier
	title := "Pair " + deviceName + " (legacy PIN)"
	if deviceName == "" {
		title = "Pair device (legacy PIN)"
	}
	m.dialog = NewDialog(title,
		[]DialogFieldSpec{
			{Key: "pin", Label: "PIN (usually 0000 or 1234)", Kind: DialogFieldText},
		},
		func(result DialogResult) tea.Cmd {
			pin := strings.TrimSpace(result["pin"])
			if pin == "" {
				return nil
			}
			state.BluetoothBusy = true
			state.BluetoothActionError = ""
			label := deviceName
			if label == "" {
				label = "device"
			}
			sendNotifyInfo(notifier, "Bluetooth", "Pairing with "+label+"…")
			return actions.Pair(devicePath, pin)
		},
	)
}

// triggerBluetoothCancelPair aborts an in-flight Device1.Pair for the
// highlighted device. Only makes sense while a pair is pending, which
// maps to BluetoothBusy — we allow the keystroke then and let BlueZ
// reject with DoesNotExist if nothing is actually pending.
func (m *Model) triggerBluetoothCancelPair() tea.Cmd {
	if m.bluetooth.CancelPair == nil || !m.inBluetoothDetails() {
		return nil
	}
	dev, ok := m.state.SelectedBluetoothDevice()
	if !ok || dev.Path == "" {
		return nil
	}
	m.state.BluetoothActionError = ""
	return m.bluetooth.CancelPair(dev.Path)
}

func (m *Model) triggerBluetoothTrustToggle() tea.Cmd {
	if m.bluetooth.SetTrusted == nil || !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
	}
	dev, ok := m.state.SelectedBluetoothDevice()
	if !ok || dev.Path == "" {
		return nil
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.SetTrusted(dev.Path, !dev.Trusted)
}

// triggerBluetoothBlockToggle flips Blocked on the highlighted device.
// A blocked device cannot be connected to or paired with until unblocked.
func (m *Model) triggerBluetoothBlockToggle() tea.Cmd {
	if m.bluetooth.SetBlocked == nil || !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
	}
	dev, ok := m.state.SelectedBluetoothDevice()
	if !ok || dev.Path == "" {
		return nil
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.SetBlocked(dev.Path, !dev.Blocked)
}

// triggerBluetoothRemove is "unpair": delete the bond and remove the
// Device1 object from the adapter.
func (m *Model) triggerBluetoothRemove() tea.Cmd {
	if m.bluetooth.Remove == nil || !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
	}
	adapter, ok := m.state.SelectedBluetoothAdapter()
	if !ok || adapter.Path == "" {
		return nil
	}
	dev, ok := m.state.SelectedBluetoothDevice()
	if !ok || dev.Path == "" {
		return nil
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.Remove(adapter.Path, dev.Path)
}

// triggerBluetoothDiscoverableToggle flips Discoverable on the selected
// adapter. Works from sidebar focus too, same as the power toggle.
func (m *Model) triggerBluetoothDiscoverableToggle() tea.Cmd {
	if m.bluetooth.SetDiscoverable == nil || m.state.ActiveTab != core.TabSettings {
		return nil
	}
	if m.state.ActiveSection().ID != "bluetooth" || m.state.BluetoothBusy {
		return nil
	}
	adapter, ok := m.state.SelectedBluetoothAdapter()
	if !ok || adapter.Path == "" {
		return nil
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.SetDiscoverable(adapter.Path, !adapter.Discoverable)
}
