package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/truncate"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/help"
	"github.com/johnnelson/dark/internal/services/appstore"
	"github.com/johnnelson/dark/internal/services/audio"
	"github.com/johnnelson/dark/internal/services/bluetooth"
	"github.com/johnnelson/dark/internal/services/display"
	"github.com/johnnelson/dark/internal/services/datetime"
	inputsvc "github.com/johnnelson/dark/internal/services/input"
	"github.com/johnnelson/dark/internal/services/notifycfg"
	"github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/services/notify"
	"github.com/johnnelson/dark/internal/services/power"
	"github.com/johnnelson/dark/internal/services/sysinfo"
	"github.com/johnnelson/dark/internal/services/tuilink"
	"github.com/johnnelson/dark/internal/services/weblink"
	"github.com/johnnelson/dark/internal/services/wifi"
)

// WifiActions is the set of asynchronous commands the TUI can dispatch
// at darkd to drive the wifi service. Each returns a tea.Cmd that, when
// run, sends a NATS request and posts the result back into the program.
type WifiActions struct {
	Scan           func(adapter string) tea.Cmd
	Connect        func(adapter, ssid, passphrase string) tea.Cmd
	ConnectHidden  func(adapter, ssid, passphrase string) tea.Cmd
	Disconnect     func(adapter string) tea.Cmd
	Forget         func(adapter, ssid string) tea.Cmd
	SetPower       func(adapter string, powered bool) tea.Cmd
	SetAutoConnect func(ssid string, enabled bool) tea.Cmd
	StartAP        func(adapter, ssid, passphrase string) tea.Cmd
	StopAP         func(adapter string) tea.Cmd
}

// WifiScanResultMsg is dispatched when a scan command completes.
type WifiScanResultMsg struct {
	Snapshot wifi.Snapshot
	Err      string
}

// WifiActionResultMsg is dispatched when a connect/disconnect/forget
// command completes. The TUI clears the busy indicator and, on success,
// replaces the cached wifi snapshot with the reply's updated one.
type WifiActionResultMsg struct {
	Snapshot wifi.Snapshot
	Err      string
}

type Model struct {
	state     *core.State
	binPath   string
	wifi      WifiActions
	bluetooth BluetoothActions
	audio     AudioActions
	network   NetworkActions
	display   DisplayActions
	power     PowerActions
	input     InputActions
	dateTime  DateTimeActions
	notifyCfg NotifyConfigActions
	notifier  *notify.Notifier
	appstore  AppstoreActions
	dialog    *Dialog
	spinner   spinner.Model
	width     int
	height    int
}

type rebuildDoneMsg core.RebuildResult

// SysInfoMsg is dispatched into the bubble tea program from the bus
// subscriber goroutine whenever darkd publishes a new system snapshot.
type SysInfoMsg sysinfo.SystemInfo

// WifiMsg is dispatched whenever darkd publishes a wifi adapter snapshot.
type WifiMsg wifi.Snapshot

// WebLinksMsg carries the list of installed web apps, loaded from .desktop files.
type WebLinksMsg []weblink.WebApp

// TUILinksMsg carries the list of installed TUI apps, loaded from .desktop files.
type TUILinksMsg []tuilink.TUIApp

// BusStatusMsg flips the connected/disconnected indicator. Sent from the
// NATS connection handlers when the link to darkd goes down or comes back.
type BusStatusMsg bool

func New(state *core.State, binPath string, wifi WifiActions, bluetooth BluetoothActions, audio AudioActions, network NetworkActions, displayAct DisplayActions, powerAct PowerActions, inputAct InputActions, dateTimeAct DateTimeActions, notifyCfgAct NotifyConfigActions, notifier *notify.Notifier, appstore AppstoreActions) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{
		state:     state,
		binPath:   binPath,
		wifi:      wifi,
		bluetooth: bluetooth,
		audio:     audio,
		network:   network,
		display:   displayAct,
		power:     powerAct,
		input:     inputAct,
		dateTime:  dateTimeAct,
		notifyCfg: notifyCfgAct,
		notifier:  notifier,
		appstore:  appstore,
		spinner:   sp,
	}
}

// notifyError fires a critical desktop notification for an action
// failure. Section is the user-facing label (e.g. "Wi-Fi", "Network")
// that becomes part of the summary so the user can tell at a glance
// which part of dark is reporting. No-op when notifications are
// disabled (no notifier wired in) or the message is empty.
func (m *Model) notifyError(section, message string) {
	if m.notifier == nil || message == "" {
		return
	}
	m.notifier.Send(notify.Message{
		Summary: "dark · " + section,
		Body:    message,
		Urgency: notify.UrgencyCritical,
		Icon:    "dialog-error",
	})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadWebLinksCmd(), loadTUILinksCmd())
}

func loadWebLinksCmd() tea.Cmd {
	return func() tea.Msg {
		apps, _ := weblink.ListWebApps()
		return WebLinksMsg(apps)
	}
}

func loadTUILinksCmd() tea.Cmd {
	return func() tea.Msg {
		apps, _ := tuilink.ListTUIApps()
		return TUILinksMsg(apps)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case SysInfoMsg:
		m.state.SetSysInfo(sysinfo.SystemInfo(msg))
		return m, nil

	case WebLinksMsg:
		m.state.SetWebLinks([]weblink.WebApp(msg))
		return m, nil

	case TUILinksMsg:
		m.state.SetTUILinks([]tuilink.TUIApp(msg))
		return m, nil

	case WifiMsg:
		m.state.SetWifi(wifi.Snapshot(msg))
		return m, nil

	case WifiScanResultMsg:
		m.state.WifiScanning = false
		if msg.Err != "" {
			m.state.WifiScanError = msg.Err
			m.notifyError("Wi-Fi", msg.Err)
			return m, nil
		}
		m.state.WifiScanError = ""
		m.state.SetWifi(msg.Snapshot)
		return m, nil

	case WifiActionResultMsg:
		m.state.WifiBusy = false
		if msg.Err != "" {
			m.state.WifiActionError = msg.Err
			m.notifyError("Wi-Fi", msg.Err)
			return m, nil
		}
		m.state.WifiActionError = ""
		m.state.SetWifi(msg.Snapshot)
		return m, nil

	case BluetoothMsg:
		m.state.SetBluetooth(bluetooth.Snapshot(msg))
		return m, nil

	case BluetoothActionResultMsg:
		m.state.BluetoothBusy = false
		if msg.Err != "" {
			m.state.BluetoothActionError = msg.Err
			m.notifyError("Bluetooth", msg.Err)
			return m, nil
		}
		m.state.BluetoothActionError = ""
		m.state.SetBluetooth(msg.Snapshot)
		return m, nil

	case DisplayMsg:
		m.state.SetDisplay(display.Snapshot(msg))
		return m, nil

	case DisplayActionResultMsg:
		m.state.DisplayBusy = false
		if msg.Err != "" {
			m.state.DisplayActionError = msg.Err
			m.notifyError("Displays", msg.Err)
			return m, nil
		}
		m.state.DisplayActionError = ""
		m.state.SetDisplay(msg.Snapshot)
		return m, nil

	case DateTimeMsg:
		m.state.SetDateTime(datetime.Snapshot(msg))
		return m, nil

	case DateTimeActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Date & Time", msg.Err)
			return m, nil
		}
		m.state.SetDateTime(msg.Snapshot)
		return m, nil

	case NotifyCfgMsg:
		m.state.SetNotify(notifycfg.Snapshot(msg))
		return m, nil

	case NotifyCfgActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Notifications", msg.Err)
			return m, nil
		}
		m.state.SetNotify(msg.Snapshot)
		return m, nil

	case InputMsg:
		m.state.SetInputDevices(inputsvc.Snapshot(msg))
		return m, nil

	case InputActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Input", msg.Err)
			return m, nil
		}
		m.state.SetInputDevices(msg.Snapshot)
		return m, nil

	case PowerMsg:
		m.state.SetPower(power.Snapshot(msg))
		return m, nil

	case PowerActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Power", msg.Err)
			return m, nil
		}
		m.state.SetPower(msg.Snapshot)
		return m, nil

	case AudioMsg:
		m.state.SetAudio(audio.Snapshot(msg))
		return m, nil

	case AudioLevelsMsg:
		m.state.SetAudioLevels(audio.Levels(msg))
		return m, nil

	case NetworkMsg:
		m.state.SetNetwork(network.Snapshot(msg))
		return m, nil

	case NetworkActionResultMsg:
		m.state.NetworkBusy = false
		if msg.Err != "" {
			m.state.NetworkActionError = msg.Err
			m.notifyError("Network", msg.Err)
			return m, nil
		}
		m.state.NetworkActionError = ""
		m.state.SetNetwork(msg.Snapshot)
		return m, nil

	case AudioActionResultMsg:
		m.state.AudioBusy = false
		if msg.Err != "" {
			m.state.AudioActionError = msg.Err
			m.notifyError("Sound", msg.Err)
			return m, nil
		}
		m.state.AudioActionError = ""
		m.state.SetAudio(msg.Snapshot)
		return m, nil

	case AppstoreMsg:
		m.state.SetAppstore(appstore.Snapshot(msg))
		return m, nil

	case AppstoreSearchResultMsg:
		if msg.Err != "" {
			m.state.SetAppstoreError(msg.Err)
			m.notifyError("App Store", msg.Err)
			return m, nil
		}
		m.state.SetAppstoreResults(msg.Result)
		return m, nil

	case AppstoreDetailResultMsg:
		if msg.Err != "" {
			m.state.SetAppstoreError(msg.Err)
			m.notifyError("App Store", msg.Err)
			return m, nil
		}
		m.state.SetAppstoreDetail(msg.Detail)
		return m, nil

	case AppstoreRefreshResultMsg:
		m.state.AppstoreBusy = false
		if msg.Err != "" {
			m.state.SetAppstoreError(msg.Err)
			m.notifyError("App Store", msg.Err)
			return m, nil
		}
		m.state.AppstoreStatusMsg = ""
		m.state.SetAppstore(msg.Snapshot)
		return m, nil

	case AppstoreActionResultMsg:
		m.state.AppstoreBusy = false
		if msg.Err != "" {
			m.state.SetAppstoreError(msg.Err)
			return m, nil
		}
		m.state.AppstoreStatusMsg = ""
		m.state.SetAppstore(msg.Snapshot)
		return m, nil

	case BusStatusMsg:
		m.state.SetBusConnected(bool(msg))
		return m, nil

	case rebuildDoneMsg:
		m.state.Rebuilding = false
		if msg.Ok {
			m.state.BuildError = ""
			m.state.RestartRequested = true
			return m, tea.Quit
		}
		m.state.BuildError = msg.Output
		m.notifyError("Rebuild", msg.Output)
		return m, nil

	case tea.KeyMsg:
		// An open dialog captures every key. The dialog's own Update
		// handles esc/enter and decides when to close itself; any
		// tea.Cmd returned here (typically a bus request spawned by
		// the submit callback) is passed straight through to bubble
		// tea so the async result lands back in this Update loop.
		if m.dialog != nil {
			cmd := m.dialog.Update(msg)
			if m.dialog.Closed() {
				stopPreview()
				m.dialog = nil
			}
			return m, cmd
		}
		if m.state.HelpOpen {
			return m.handleHelpKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

// triggerWifiScan issues a scan on the currently highlighted adapter.
// Returns nil if the key should be a no-op (wrong section, nothing to
// scan, already scanning, or no scan function wired in).
func (m *Model) triggerWifiScan() tea.Cmd {
	if m.wifi.Scan == nil || !m.inWifiContent() || m.state.WifiScanning {
		return nil
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
	if m.wifi.Connect == nil || !m.inWifiDetails() || m.state.WifiBusy {
		return nil
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
	m.dialog = NewDialog(title,
		[]DialogFieldSpec{
			{Key: "passphrase", Label: "Passphrase", Kind: DialogFieldPassword},
		},
		func(result DialogResult) tea.Cmd {
			state.WifiBusy = true
			state.WifiActionError = ""
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
	if m.wifi.Disconnect == nil || !m.inWifiContent() || m.state.WifiBusy {
		return nil
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
	return m.inWifiContent() && m.state.WifiDetailsOpen
}

// moveSelection routes vertical-arrow input to whichever region currently
// owns focus. When the sidebar is focused, it walks between settings
// sections; when the content pane is focused, it moves the inner widget's
// selection (currently only the wifi adapter row).
func (m *Model) moveSelection(delta int) {
	switch m.state.ActiveTab {
	case core.TabF2:
		if m.state.ContentFocused && m.state.AppstoreDetailOpen {
			m.state.ScrollAppstoreDetail(delta)
		} else if m.state.ContentFocused {
			m.state.MoveAppstoreResult(delta)
		} else {
			m.state.MoveAppstoreCategory(delta)
		}
		return
	case core.TabF3:
		if m.state.ContentFocused {
			m.state.MoveOmarchyFocus(delta)
		} else {
			m.state.MoveOmarchySidebar(delta)
		}
		return
	case core.TabSettings:
	default:
		return
	}
	if m.state.ContentFocused {
		switch m.state.ActiveSection().ID {
		case "wifi":
			switch m.state.WifiFocus {
			case core.WifiFocusAdapters:
				m.state.MoveWifiSelection(delta)
			case core.WifiFocusKnown:
				m.state.MoveWifiKnownSelection(delta)
			default:
				m.state.MoveWifiNetworkSelection(delta)
			}
		case "bluetooth":
			switch m.state.BluetoothFocus {
			case core.BluetoothFocusAdapters:
				m.state.MoveBluetoothSelection(delta)
			default:
				m.state.MoveBluetoothDeviceSelection(delta)
			}
		case "display":
			m.state.MoveDisplaySelection(delta)
		case "sound":
			m.state.MoveAudioSelection(delta)
		case "network":
			if m.state.NetworkRoutesOpen {
				m.state.MoveNetworkRouteSelection(delta)
			} else {
				m.state.MoveNetworkSelection(delta)
			}
		case "power", "input", "notifications", "datetime":
			m.state.ScrollContent(delta)
		}
		return
	}
	m.state.MoveSettingsFocus(delta)
}

func (m Model) rebuildCmd() tea.Cmd {
	bin := m.binPath
	return func() tea.Msg {
		return rebuildDoneMsg(core.Rebuild(bin))
	}
}

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
			case "bluetooth":
				if !m.state.BluetoothDeviceInfoOpen {
					m.state.OpenBluetoothDeviceInfo()
				}
			case "sound":
				if !m.state.AudioDeviceInfoOpen {
					m.state.OpenAudioDeviceInfo()
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
		if m.inWifiContent() {
			m.state.CycleWifiFocus()
		}
		if m.inBluetoothContent() && !m.state.BluetoothDeviceInfoOpen {
			m.state.CycleBluetoothFocus()
		}
		if m.inSoundContent() {
			m.state.CycleAudioFocus()
		}
	case "s":
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
		if cmd := m.triggerWifiConnect(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothConnect(); cmd != nil {
			return m, cmd
		}
	case "d":
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
		if m.inDateTimeContent() {
			m.triggerDTTimezoneDialog()
			return m, nil
		}
	case "f":
		if m.inDateTimeContent() {
			if cmd := m.triggerDTClockFormatToggle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerWifiForget(); cmd != nil {
			return m, cmd
		}
	case "w":
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
	case "o":
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
		if cmd := m.triggerDisplayCycleTransform(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerBluetoothRename(); cmd != nil {
			return m, cmd
		}
		if cmd := m.triggerNetworkReconfigure(); cmd != nil {
			return m, cmd
		}
	case "b":
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
		if m.inNotifyContent() {
			if cmd := m.triggerNotifySoundDisable(); cmd != nil {
				return m, cmd
			}
		}
	case "l":
		if m.inNotifyContent() {
			if cmd := m.triggerNotifyLayerToggle(); cmd != nil {
				return m, cmd
			}
		}
	case "x":
		if m.inNotifyContent() {
			m.triggerNotifyRemoveRuleDialog()
			return m, nil
		}
		if cmd := m.triggerBluetoothCancelPair(); cmd != nil {
			return m, cmd
		}
	case "R":
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
		if cmd := m.triggerAudioSuspendToggle(); cmd != nil {
			return m, cmd
		}
	case "K":
		if cmd := m.triggerAudioKillStream(); cmd != nil {
			return m, cmd
		}
	case "i":
		if cmd := m.triggerDisplayIdentify(); cmd != nil {
			return m, cmd
		}
		if m.state.ActiveTab == core.TabF2 {
			return m, m.triggerAppstoreInstall()
		}
	case "X":
		if m.inDisplayContent() {
			m.triggerDisplayDeleteProfile()
			return m, nil
		}
		if m.state.ActiveTab == core.TabF2 {
			return m, m.triggerAppstoreRemove()
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
	case "A":
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
		if m.inPowerContent() {
			if cmd := m.triggerPowerGovernorCycle(); cmd != nil {
				return m, cmd
			}
		}
		if cmd := m.triggerDisplayGammaDelta(-5); cmd != nil {
			return m, cmd
		}
	case "G":
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

func (m Model) View() string {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 40
	}

	tabBar := renderTabBar(m.state, width)
	statusBar := renderStatusBar(m.state, width)
	bodyHeight := height - lipgloss.Height(tabBar) - lipgloss.Height(statusBar)

	var body string
	switch m.state.ActiveTab {
	case core.TabSettings:
		body = renderSettings(m.state, width, bodyHeight)
	case core.TabF2:
		body = renderAppStoreTab(m.state, width, bodyHeight, m.spinner.View())
	case core.TabF3:
		body = renderOmarchyTab(m.state, width, bodyHeight)
	default:
		body = renderEmpty(m.state, width, bodyHeight)
	}

	base := appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, body, statusBar, tabBar))

	if m.state.HelpOpen {
		chromeHeight := lipgloss.Height(statusBar) + lipgloss.Height(tabBar)
		panelHeight := height - chromeHeight
		if panelHeight < 3 {
			panelHeight = 3
		}
		panel := renderHelpPanel(m.state, panelHeight)
		panel = help.ReapplyPanelBackground(panel)
		base = overlayRight(base, panel, width, m.state.HelpWidth)
	}

	if m.dialog != nil {
		return overlayCenter(base, m.dialog.View(), width, height)
	}
	return base
}

// overlayRight composes the help panel onto the right portion of the base
// view. Each base line is ANSI-truncated to (totalWidth - panelWidth) visible
// columns then concatenated with the corresponding panel line.
func overlayRight(base, panel string, totalWidth, panelWidth int) string {
	if panelWidth <= 0 || panelWidth >= totalWidth {
		return panel
	}
	keep := totalWidth - panelWidth
	baseLines := strings.Split(base, "\n")
	panelLines := strings.Split(panel, "\n")

	n := len(baseLines)
	if len(panelLines) > n {
		n = len(panelLines)
	}

	out := make([]string, n)
	for i := 0; i < n; i++ {
		// Rows below the panel pass through untouched so the status bar and
		// tab bar remain visible across the full terminal width.
		if i >= len(panelLines) {
			if i < len(baseLines) {
				out[i] = baseLines[i]
			}
			continue
		}
		var left string
		if i < len(baseLines) {
			left = truncate.String(baseLines[i], uint(keep))
		}
		leftW := lipgloss.Width(left)
		if leftW < keep {
			left += strings.Repeat(" ", keep-leftW)
		}
		out[i] = left + panelLines[i]
	}
	return strings.Join(out, "\n")
}

// overlayCenter composes an overlay (typically a dialog box) on top of
// the base view, centered horizontally and vertically. Each row the
// overlay occupies is rebuilt as base[:left] + overlay + base[left+oW:]
// with ANSI escapes preserved on both sides, so the sidebar and other
// content to the left and right of the dialog stay visible. Rows above
// and below the overlay pass through the base untouched.
func overlayCenter(base, overlay string, totalWidth, totalHeight int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	oH := len(overlayLines)
	oW := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > oW {
			oW = w
		}
	}
	if oW == 0 || oH == 0 {
		return base
	}

	top := (totalHeight - oH) / 2
	if top < 0 {
		top = 0
	}
	left := (totalWidth - oW) / 2
	if left < 0 {
		left = 0
	}

	out := make([]string, len(baseLines))
	copy(out, baseLines)

	for i, oLine := range overlayLines {
		y := top + i
		if y >= len(out) {
			break
		}
		baseLine := ""
		if y < len(baseLines) {
			baseLine = baseLines[y]
		}

		// Left slice: first `left` visible cells of base, padded with
		// spaces if the base line is shorter than the dialog column.
		leftPart := truncate.String(baseLine, uint(left))
		if padNeeded := left - lipgloss.Width(leftPart); padNeeded > 0 {
			leftPart += strings.Repeat(" ", padNeeded)
		}

		// Right slice: drop the first (left + oW) visible cells from the
		// base line, keep the rest. A leading reset prevents ANSI state
		// leaking out of the dialog onto the resumed base content.
		rightPart := ansiSkipCells(baseLine, left+oW)
		if rightPart != "" {
			rightPart = "\x1b[0m" + rightPart
		}

		out[y] = leftPart + "\x1b[0m" + oLine + rightPart
	}

	return strings.Join(out, "\n")
}

// ansiSkipCells returns the tail of s after the first `skip` visible
// cells, preserving every ANSI escape sequence that appears in the
// skipped prefix so the tail inherits the correct styling state.
// Visible width is measured by runewidth, the same library lipgloss
// uses, so this agrees with lipgloss.Width.
func ansiSkipCells(s string, skip int) string {
	if skip <= 0 {
		return s
	}
	total := lipgloss.Width(s)
	if skip >= total {
		return ""
	}

	var b strings.Builder
	visible := 0
	i := 0
	for i < len(s) {
		// CSI escape sequence: ESC [ ... <final>
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			// Always emit style escapes so the tail carries color state.
			b.WriteString(s[i:j])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if size == 0 {
			i++
			continue
		}
		w := runewidth.RuneWidth(r)
		if visible >= skip {
			b.WriteRune(r)
		}
		visible += w
		i += size
	}
	return b.String()
}
