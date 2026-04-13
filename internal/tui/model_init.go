package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/notify"
	"github.com/johnnelson/dark/internal/services/tuilink"
	"github.com/johnnelson/dark/internal/services/weblink"
)

func New(state *core.State, binPath string, wifi WifiActions, bluetooth BluetoothActions, audio AudioActions, network NetworkActions, displayAct DisplayActions, powerAct PowerActions, inputAct InputActions, dateTimeAct DateTimeActions, notifyCfgAct NotifyConfigActions, notifier *notify.Notifier, appstore AppstoreActions, keybindAct KeybindActions, usersAct UsersActions, privacyAct PrivacyActions, appearanceAct AppearanceActions) Model {
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
		keybind:   keybindAct,
		users:     usersAct,
		privacy:    privacyAct,
		appearance: appearanceAct,
		spinner:    sp,
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
