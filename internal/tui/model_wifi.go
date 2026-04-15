package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/wifi"
)

// triggerWifiScan issues a scan on the currently highlighted adapter.
// Returns nil if the key should be a no-op (wrong section, nothing to
// scan, already scanning, or no scan function wired in).
func (m *Model) triggerWifiScan() tea.Cmd {
	if !m.inWifiContent() || m.state.WifiScanning {
		return nil
	}
	if m.wifi.Scan == nil {
		return m.notifyUnavailable("Wi-Fi")
	}
	adapter, ok := m.state.SelectedAdapter()
	if !ok || adapter.Name == "" {
		return nil
	}
	m.state.WifiScanning = true
	m.state.WifiScanError = ""
	return m.wifi.Scan(adapter.Name)
}

// triggerWifiConnect asks the daemon to associate with an SSID. For
// networks that need credentials and don't have a saved profile, this
// opens a password dialog first and defers the actual connect command
// until the user submits.
func (m *Model) triggerWifiConnect() tea.Cmd {
	if !m.inWifiDetails() || m.state.WifiBusy {
		return nil
	}
	if m.wifi.Connect == nil {
		return m.notifyUnavailable("Wi-Fi")
	}
	adapter, ok := m.state.SelectedAdapter()
	if !ok || adapter.Name == "" {
		return nil
	}

	var ssid string
	var known bool
	var openNet bool
	switch m.state.WifiFocus {
	case core.WifiFocusKnown:
		kn, ok := m.state.SelectedKnownNetwork()
		if !ok || kn.SSID == "" {
			return nil
		}
		ssid = kn.SSID
		known = true // saved profile means iwd has the creds
	default:
		n, ok := m.state.SelectedNetwork()
		if !ok || n.SSID == "" {
			return nil
		}
		ssid = n.SSID
		known = n.Known
		openNet = n.Security == "open"
	}

	// Known profiles and open networks connect directly without a prompt.
	if known || openNet {
		m.state.WifiBusy = true
		m.state.WifiActionError = ""
		m.notifyInfo("Wi-Fi", "Connecting to "+ssid+"…")
		return m.wifi.Connect(adapter.Name, ssid, "")
	}

	// Unknown credentialed network — pop the password dialog.
	m.openPassphraseDialog(adapter.Name, ssid)
	return nil
}

// triggerWifiAPToggle starts an access point on the selected adapter
// when none is running, or stops the current one when it is. Starting
// opens a dialog for SSID + passphrase; stopping is one keystroke.
// Skipped when the selected adapter's hardware doesn't list "ap" in
// its SupportedModes — pressing p on a station-only card is a no-op.
func (m *Model) triggerWifiAPToggle() tea.Cmd {
	if m.wifi.StartAP == nil || m.wifi.StopAP == nil {
		return nil
	}
	if !m.inWifiContent() || m.state.WifiBusy {
		return nil
	}
	adapter, ok := m.state.SelectedAdapter()
	if !ok || adapter.Name == "" {
		return nil
	}
	if !supportsAPMode(adapter) {
		m.state.WifiActionError = "this adapter does not support AP mode"
		return nil
	}
	if adapter.APActive {
		m.state.WifiBusy = true
		m.state.WifiActionError = ""
		return m.wifi.StopAP(adapter.Name)
	}
	m.openAPStartDialog(adapter.Name)
	return nil
}

// openAPStartDialog pops a dialog for hotspot SSID + passphrase and
// dispatches StartAP on submit.
func (m *Model) openAPStartDialog(adapter string) {
	wifi := m.wifi
	state := m.state
	m.dialog = NewDialog("Start access point",
		[]DialogFieldSpec{
			{Key: "ssid", Label: "SSID", Kind: DialogFieldText},
			{Key: "passphrase", Label: "Passphrase (8+ characters)", Kind: DialogFieldPassword},
		},
		func(result DialogResult) tea.Cmd {
			ssid := strings.TrimSpace(result["ssid"])
			if ssid == "" {
				return nil
			}
			state.WifiBusy = true
			state.WifiActionError = ""
			return wifi.StartAP(adapter, ssid, result["passphrase"])
		},
	)
}

// supportsAPMode reports whether the iwd adapter carrying this device
// lists "ap" in its SupportedModes. Hardware that doesn't support AP
// mode gets its Access Point box hidden from the view.
func supportsAPMode(a wifi.Adapter) bool {
	for _, m := range a.SupportedModes {
		if m == "ap" {
			return true
		}
	}
	return false
}

// triggerWifiConnectHidden pops a dialog for SSID + passphrase, then
// dispatches Station.ConnectHiddenNetwork via the bus.
func (m *Model) triggerWifiConnectHidden() tea.Cmd {
	if m.wifi.ConnectHidden == nil || !m.inWifiContent() || m.state.WifiBusy {
		return nil
	}
	adapter, ok := m.state.SelectedAdapter()
	if !ok || adapter.Name == "" {
		return nil
	}
	m.openHiddenNetworkDialog(adapter.Name)
	return nil
}

// openPassphraseDialog opens a one-field dialog prompting for the PSK
// of ssid. On submit it dispatches the connect command with the typed
// passphrase.
func (m *Model) openPassphraseDialog(adapter, ssid string) {
	title := "Connect to " + ssid
	wifi := m.wifi
	state := m.state
	notifier := m.notifier
	m.dialog = NewDialog(title,
		[]DialogFieldSpec{
			{Key: "passphrase", Label: "Passphrase", Kind: DialogFieldPassword},
		},
		func(result DialogResult) tea.Cmd {
			state.WifiBusy = true
			state.WifiActionError = ""
			sendNotifyInfo(notifier, "Wi-Fi", "Connecting to "+ssid+"…")
			return wifi.Connect(adapter, ssid, result["passphrase"])
		},
	)
}

// openHiddenNetworkDialog opens a two-field dialog: SSID and passphrase.
// The passphrase may be empty for open hidden networks, which iwd still
// handles via ConnectHiddenNetwork.
func (m *Model) openHiddenNetworkDialog(adapter string) {
	wifi := m.wifi
	state := m.state
	m.dialog = NewDialog("Connect to hidden network",
		[]DialogFieldSpec{
			{Key: "ssid", Label: "SSID", Kind: DialogFieldText},
			{Key: "passphrase", Label: "Passphrase (leave blank for open)", Kind: DialogFieldPassword},
		},
		func(result DialogResult) tea.Cmd {
			ssid := strings.TrimSpace(result["ssid"])
			if ssid == "" {
				return nil
			}
			state.WifiBusy = true
			state.WifiActionError = ""
			return wifi.ConnectHidden(adapter, ssid, result["passphrase"])
		},
	)
}

// triggerWifiDisconnect drops the adapter's current association. Doesn't
// need a network selection — acts on whatever the adapter is connected
// to right now.
func (m *Model) triggerWifiDisconnect() tea.Cmd {
	if !m.inWifiContent() || m.state.WifiBusy {
		return nil
	}
	if m.wifi.Disconnect == nil {
		return m.notifyUnavailable("Wi-Fi")
	}
	adapter, ok := m.state.SelectedAdapter()
	if !ok || adapter.Name == "" {
		return nil
	}
	m.state.WifiBusy = true
	m.state.WifiActionError = ""
	return m.wifi.Disconnect(adapter.Name)
}

// triggerWifiForget removes the saved profile for the highlighted Known
// Network row. Only fires when the Known sub-table is focused — forget
// is a concept that belongs to the saved-profile list, not the ambient
// scan results.
func (m *Model) triggerWifiForget() tea.Cmd {
	if m.wifi.Forget == nil || !m.inWifiDetails() || m.state.WifiBusy {
		return nil
	}
	if m.state.WifiFocus != core.WifiFocusKnown {
		return nil
	}
	adapter, ok := m.state.SelectedAdapter()
	if !ok || adapter.Name == "" {
		return nil
	}
	kn, ok := m.state.SelectedKnownNetwork()
	if !ok || kn.SSID == "" {
		return nil
	}
	m.state.WifiBusy = true
	m.state.WifiActionError = ""
	return m.wifi.Forget(adapter.Name, kn.SSID)
}

// triggerWifiAutoconnectToggle flips AutoConnect on the highlighted
// Known Network row. Only meaningful when the Known sub-table is focused.
func (m *Model) triggerWifiAutoconnectToggle() tea.Cmd {
	if m.wifi.SetAutoConnect == nil || !m.inWifiDetails() || m.state.WifiBusy {
		return nil
	}
	if m.state.WifiFocus != core.WifiFocusKnown {
		return nil
	}
	kn, ok := m.state.SelectedKnownNetwork()
	if !ok || kn.SSID == "" {
		return nil
	}
	m.state.WifiBusy = true
	m.state.WifiActionError = ""
	return m.wifi.SetAutoConnect(kn.SSID, !kn.AutoConnect)
}

// triggerWifiPowerToggle flips the radio on or off. Works whether or not
// the content region is focused — the user presses 'w' on the Wi-Fi
// section, that's intent enough.
func (m *Model) triggerWifiPowerToggle() tea.Cmd {
	if m.wifi.SetPower == nil || m.state.ActiveTab != core.TabSettings {
		return nil
	}
	if m.state.ActiveSection().ID != "wifi" || m.state.WifiBusy {
		return nil
	}
	adapter, ok := m.state.SelectedAdapter()
	if !ok || adapter.Name == "" {
		return nil
	}
	m.state.WifiBusy = true
	m.state.WifiActionError = ""
	return m.wifi.SetPower(adapter.Name, !adapter.Powered)
}

func (m *Model) inWifiContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "wifi"
}

func (m *Model) inWifiDetails() bool {
	return m.inWifiContent() && m.state.WifiContentFocused
}
