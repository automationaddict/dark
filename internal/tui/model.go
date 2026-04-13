package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appearance"
	"github.com/johnnelson/dark/internal/services/appstore"
	"github.com/johnnelson/dark/internal/services/audio"
	"github.com/johnnelson/dark/internal/services/bluetooth"
	"github.com/johnnelson/dark/internal/services/display"
	"github.com/johnnelson/dark/internal/services/datetime"
	inputsvc "github.com/johnnelson/dark/internal/services/input"
	"github.com/johnnelson/dark/internal/services/keybind"
	"github.com/johnnelson/dark/internal/services/notifycfg"
	"github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/services/notify"
	"github.com/johnnelson/dark/internal/services/power"
	privacysvc "github.com/johnnelson/dark/internal/services/privacy"
	"github.com/johnnelson/dark/internal/services/sysinfo"
	"github.com/johnnelson/dark/internal/services/tuilink"
	userssvc "github.com/johnnelson/dark/internal/services/users"
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
	keybind   KeybindActions
	users     UsersActions
	privacy    PrivacyActions
	appearance AppearanceActions
	dialog     *Dialog
	spinner    spinner.Model
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

	case KeybindMsg:
		m.state.SetKeybindings(keybind.Snapshot(msg))
		return m, nil

	case KeybindActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Keybindings", msg.Err)
			return m, nil
		}
		m.state.SetKeybindings(msg.Snapshot)
		return m, nil

	case keybindConflictMsg:
		return m, m.handleKeybindConflict(msg)

	case UsersMsg:
		m.state.SetUsers(userssvc.Snapshot(msg))
		return m, nil

	case UsersActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Users", msg.Err)
			return m, nil
		}
		m.state.SetUsers(msg.Snapshot)
		return m, nil

	case UsersElevatedMsg:
		m.handleUsersElevated(msg)
		return m, nil

	case PrivacyMsg:
		m.state.SetPrivacy(privacysvc.Snapshot(msg))
		return m, nil

	case PrivacyActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Privacy", msg.Err)
			return m, nil
		}
		m.state.SetPrivacy(msg.Snapshot)
		return m, nil

	case AppearanceMsg:
		m.state.SetAppearance(appearance.Snapshot(msg))
		return m, nil

	case AppearanceActionResultMsg:
		if msg.Err != "" {
			m.notifyError("Appearance", msg.Err)
			return m, nil
		}
		m.state.SetAppearance(msg.Snapshot)
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
		if m.state.DisplayLayoutOpen {
			return m.handleDisplayLayoutKey(msg)
		}
		if m.state.HelpOpen {
			return m.handleHelpKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}
