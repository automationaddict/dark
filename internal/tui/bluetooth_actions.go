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
			return m.notifyUnavailable("Bluetooth")
		}
		m.state.BluetoothBusy = true
		m.state.BluetoothActionError = ""
		return m.bluetooth.StopDiscovery(adapter.Path)
	}
	if m.bluetooth.StartDiscovery == nil {
		return m.notifyUnavailable("Bluetooth")
	}
	m.state.BluetoothBusy = true
	m.state.BluetoothActionError = ""
	return m.bluetooth.StartDiscovery(adapter.Path)
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
	notify := m.notifier
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
				msg := "invalid timeout: " + err.Error()
				state.BluetoothActionError = msg
				sendNotifyError(notify, "Bluetooth", msg)
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
	notify := m.notifier
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
				msg := "transport must be auto, bredr, or le"
				state.BluetoothActionError = msg
				sendNotifyError(notify, "Bluetooth", msg)
				return nil
			}
			var rssiVal int16
			if raw := strings.TrimSpace(result["rssi"]); raw != "" {
				n, err := strconv.ParseInt(raw, 10, 16)
				if err != nil {
					msg := "invalid rssi: " + err.Error()
					state.BluetoothActionError = msg
					sendNotifyError(notify, "Bluetooth", msg)
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

