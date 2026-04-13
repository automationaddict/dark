package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/bluetooth"
)

// BluetoothActions is the set of asynchronous commands the TUI can
// dispatch at darkd to drive the bluetooth service. Mirrors WifiActions.
type BluetoothActions struct {
	SetPowered             func(adapter string, powered bool) tea.Cmd
	SetDiscoverable        func(adapter string, discoverable bool) tea.Cmd
	SetDiscoverableTimeout func(adapter string, seconds uint32) tea.Cmd
	SetPairable            func(adapter string, pairable bool) tea.Cmd
	SetAlias               func(adapter, alias string) tea.Cmd
	SetDiscoveryFilter     func(adapter string, filter bluetooth.DiscoveryFilter) tea.Cmd
	StartDiscovery         func(adapter string) tea.Cmd
	StopDiscovery          func(adapter string) tea.Cmd
	Connect                func(device string) tea.Cmd
	Disconnect             func(device string) tea.Cmd
	Pair                   func(device, pin string) tea.Cmd
	CancelPair             func(device string) tea.Cmd
	Remove                 func(adapter, device string) tea.Cmd
	SetTrusted             func(device string, trusted bool) tea.Cmd
	SetBlocked             func(device string, blocked bool) tea.Cmd
}

// BluetoothMsg is dispatched whenever darkd publishes a bluetooth snapshot.
type BluetoothMsg bluetooth.Snapshot

// BluetoothActionResultMsg is dispatched when a bluetooth action command
// completes. On success, the reply's updated snapshot replaces the
// cached one; on failure, the error is shown inline.
type BluetoothActionResultMsg struct {
	Snapshot bluetooth.Snapshot
	Err      string
}

func (m *Model) inBluetoothContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "bluetooth"
}

func (m *Model) inBluetoothDetails() bool {
	return m.inBluetoothContent() && m.state.BluetoothContentFocused
}

// triggerBluetoothPowerToggle flips the radio on the selected adapter.
// Works from sidebar focus too — pressing w on the Bluetooth section is
// enough intent, same as wifi.
func (m *Model) triggerBluetoothPowerToggle() tea.Cmd {
	if m.bluetooth.SetPowered == nil || m.state.ActiveTab != core.TabSettings {
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
	return m.bluetooth.SetPowered(adapter.Path, !adapter.Powered)
}

// triggerBluetoothScanToggle toggles discovery on the selected adapter.
func (m *Model) triggerBluetoothScanToggle() tea.Cmd {
	if !m.inBluetoothContent() || m.state.BluetoothBusy {
		return nil
	}
	adapter, ok := m.state.SelectedBluetoothAdapter()
	if !ok || adapter.Path == "" {
		return nil
	}
	if adapter.Discovering {
		if m.bluetooth.StopDiscovery == nil {
			return nil
		}
		m.state.BluetoothBusy = true
		m.state.BluetoothActionError = ""
		return m.bluetooth.StopDiscovery(adapter.Path)
	}
	if m.bluetooth.StartDiscovery == nil {
		return nil
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.StartDiscovery(adapter.Path)
}

func (m *Model) triggerBluetoothConnect() tea.Cmd {
	if m.bluetooth.Connect == nil || !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
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
	if m.bluetooth.Disconnect == nil || !m.inBluetoothDetails() || m.state.BluetoothBusy {
		return nil
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
	return m.bluetooth.Pair(dev.Path, "")
}

func (m *Model) openLegacyPairDialog(devicePath, deviceName string) {
	actions := m.bluetooth
	state := m.state
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

// triggerBluetoothPairableToggle flips Pairable on the selected
// adapter. Separate from Discoverable: Pairable controls whether
// incoming pair requests are accepted, Discoverable controls whether
// the adapter advertises itself.
func (m *Model) triggerBluetoothPairableToggle() tea.Cmd {
	if m.bluetooth.SetPairable == nil || m.state.ActiveTab != core.TabSettings {
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
	return m.bluetooth.SetPairable(adapter.Path, !adapter.Pairable)
}

// triggerBluetoothResetAlias clears the adapter's custom Alias, which
// BlueZ treats as a reset to the system default (usually the hostname).
// One-shot: no dialog, no confirmation — if the user wants a new
// name they can press 'r' to rename.
func (m *Model) triggerBluetoothResetAlias() tea.Cmd {
	if m.bluetooth.SetAlias == nil || m.state.ActiveTab != core.TabSettings {
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
	return m.bluetooth.SetAlias(adapter.Path, "")
}

// triggerBluetoothSetTimeout opens a single-field dialog for the
// discoverable timeout in seconds. Zero is the BlueZ "never times out"
// value.
func (m *Model) triggerBluetoothSetTimeout() tea.Cmd {
	if m.bluetooth.SetDiscoverableTimeout == nil || m.state.ActiveTab != core.TabSettings {
		return nil
	}
	if m.state.ActiveSection().ID != "bluetooth" || m.state.BluetoothBusy {
		return nil
	}
	adapter, ok := m.state.SelectedBluetoothAdapter()
	if !ok || adapter.Path == "" {
		return nil
	}
	actions := m.bluetooth
	state := m.state
	path := adapter.Path
	title := "Discoverable timeout · " + adapter.Name
	m.dialog = NewDialog(title,
		[]DialogFieldSpec{
			{
				Key:   "seconds",
				Label: "Seconds (0 = never time out)",
				Kind:  DialogFieldText,
				Value: fmt.Sprintf("%d", adapter.DiscoverableTimeout),
			},
		},
		func(result DialogResult) tea.Cmd {
			raw := strings.TrimSpace(result["seconds"])
			if raw == "" {
				return nil
			}
			n, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				state.BluetoothActionError = "invalid timeout: " + err.Error()
				return nil
			}
			state.BluetoothBusy = true
			state.BluetoothActionError = ""
			return actions.SetDiscoverableTimeout(path, uint32(n))
		},
	)
	return nil
}

// triggerBluetoothScanFilter opens a three-field dialog for the
// SetDiscoveryFilter parameters dark exposes: transport, RSSI floor,
// and name pattern. Submitting all-empty clears the filter.
func (m *Model) triggerBluetoothScanFilter() tea.Cmd {
	if m.bluetooth.SetDiscoveryFilter == nil || m.state.ActiveTab != core.TabSettings {
		return nil
	}
	if m.state.ActiveSection().ID != "bluetooth" || m.state.BluetoothBusy {
		return nil
	}
	adapter, ok := m.state.SelectedBluetoothAdapter()
	if !ok || adapter.Path == "" {
		return nil
	}
	actions := m.bluetooth
	state := m.state
	path := adapter.Path

	rssi := ""
	if state.BluetoothScanFilter.RSSI != 0 {
		rssi = fmt.Sprintf("%d", state.BluetoothScanFilter.RSSI)
	}

	m.dialog = NewDialog("Scan filter · "+adapter.Name,
		[]DialogFieldSpec{
			{
				Key:   "transport",
				Label: "Transport (auto / bredr / le)",
				Kind:  DialogFieldText,
				Value: state.BluetoothScanFilter.Transport,
			},
			{
				Key:   "rssi",
				Label: "RSSI floor in dBm (blank = no filter)",
				Kind:  DialogFieldText,
				Value: rssi,
			},
			{
				Key:   "pattern",
				Label: "Name pattern (blank = no filter)",
				Kind:  DialogFieldText,
				Value: state.BluetoothScanFilter.Pattern,
			},
		},
		func(result DialogResult) tea.Cmd {
			transport := strings.ToLower(strings.TrimSpace(result["transport"]))
			switch transport {
			case "", "auto", "bredr", "le":
			default:
				state.BluetoothActionError = "transport must be auto, bredr, or le"
				return nil
			}
			var rssiVal int16
			if raw := strings.TrimSpace(result["rssi"]); raw != "" {
				n, err := strconv.ParseInt(raw, 10, 16)
				if err != nil {
					state.BluetoothActionError = "invalid rssi: " + err.Error()
					return nil
				}
				rssiVal = int16(n)
			}
			filter := bluetooth.DiscoveryFilter{
				Transport: transport,
				RSSI:      rssiVal,
				Pattern:   strings.TrimSpace(result["pattern"]),
			}
			state.BluetoothScanFilter = filter
			state.BluetoothBusy = true
			state.BluetoothActionError = ""
			return actions.SetDiscoveryFilter(path, filter)
		},
	)
	return nil
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

// triggerBluetoothRename opens a single-field dialog prefilled with the
// current adapter alias; on submit, dispatches SetAlias.
func (m *Model) triggerBluetoothRename() tea.Cmd {
	if m.bluetooth.SetAlias == nil || m.state.ActiveTab != core.TabSettings {
		return nil
	}
	if m.state.ActiveSection().ID != "bluetooth" || m.state.BluetoothBusy {
		return nil
	}
	adapter, ok := m.state.SelectedBluetoothAdapter()
	if !ok || adapter.Path == "" {
		return nil
	}
	actions := m.bluetooth
	state := m.state
	path := adapter.Path
	m.dialog = NewDialog("Rename "+adapter.Name,
		[]DialogFieldSpec{
			{Key: "alias", Label: "Alias", Kind: DialogFieldText, Value: adapter.Alias},
		},
		func(result DialogResult) tea.Cmd {
			alias := strings.TrimSpace(result["alias"])
			if alias == "" {
				return nil
			}
			state.BluetoothBusy = true
			state.BluetoothActionError = ""
			return actions.SetAlias(path, alias)
		},
	)
	return nil
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
