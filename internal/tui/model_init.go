package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/links"
	"github.com/automationaddict/dark/internal/services/notify"
)

func New(state *core.State, binPath string, wifi WifiActions, bluetooth BluetoothActions, audio AudioActions, network NetworkActions, displayAct DisplayActions, powerAct PowerActions, inputAct InputActions, dateTimeAct DateTimeActions, notifyCfgAct NotifyConfigActions, notifier *notify.Notifier, appstore AppstoreActions, keybindAct KeybindActions, usersAct UsersActions, privacyAct PrivacyActions, appearanceAct AppearanceActions, updateAct UpdateActions, limineAct LimineActions, screensaverAct ScreensaverActions, topbarAct TopBarActions, workspacesAct WorkspacesActions) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{
		state:       state,
		binPath:     binPath,
		wifi:        wifi,
		bluetooth:   bluetooth,
		audio:       audio,
		network:     network,
		display:     displayAct,
		power:       powerAct,
		input:       inputAct,
		dateTime:    dateTimeAct,
		notifyCfg:   notifyCfgAct,
		notifier:    notifier,
		appstore:    appstore,
		keybind:     keybindAct,
		limine:      limineAct,
		users:       usersAct,
		privacy:     privacyAct,
		appearance:  appearanceAct,
		update:      updateAct,
		screensaver: screensaverAct,
		topbar:      topbarAct,
		workspaces:  workspacesAct,
		spinner:     sp,
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

// notifyInfo fires a low-urgency desktop notification for user-visible
// progress on long-running operations (Wi-Fi connect, Bluetooth pair,
// AppStore refresh). Section works the same as notifyError. These
// notifications are important for when the user has moved away from
// the TUI while an operation is in flight.
func (m *Model) notifyInfo(section, message string) {
	if m.notifier == nil || message == "" {
		return
	}
	m.notifier.Send(notify.Message{
		Summary: "dark · " + section,
		Body:    message,
		Urgency: notify.UrgencyLow,
		Icon:    "dialog-information",
	})
}

// sendNotifyError is a standalone variant of notifyError intended for
// use inside dialog callback closures where the Model receiver isn't
// easily accessible. Captures the notifier by value so a nil notifier
// is safely handled.
func sendNotifyError(n *notify.Notifier, section, message string) {
	if n == nil || message == "" {
		return
	}
	n.Send(notify.Message{
		Summary: "dark · " + section,
		Body:    message,
		Urgency: notify.UrgencyCritical,
		Icon:    "dialog-error",
	})
}

// sendNotifyInfo is the dialog-closure companion for notifyInfo.
func sendNotifyInfo(n *notify.Notifier, section, message string) {
	if n == nil || message == "" {
		return
	}
	n.Send(notify.Message{
		Summary: "dark · " + section,
		Body:    message,
		Urgency: notify.UrgencyLow,
		Icon:    "dialog-information",
	})
}

// notifyUnavailable fires a desktop notification when the user tries an
// action whose backend function is not wired in (e.g. the daemon was
// started without the corresponding service). Returns a no-op tea.Cmd
// so the caller can return it to prevent key fallthrough.
func (m *Model) notifyUnavailable(section string) tea.Cmd {
	m.notifyError(section, "service unavailable — is the daemon running?")
	return func() tea.Msg { return nil }
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadLinksCmd())
}

func loadLinksCmd() tea.Cmd {
	return func() tea.Msg {
		lf, _ := links.Load()
		return LinksMsg(lf)
	}
}
