package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
)

// SSHActions is the injected set of F4 SSH command closures. Every
// field is a tea.Cmd factory that dispatches against the daemon and
// returns an SSHActionResultMsg carrying the refreshed snapshot.
// Matches the pattern used by WifiActions, BluetoothActions, etc.
type SSHActions struct {
	GenerateKey         func(opts core.SSHGenerateKeyOptions) tea.Cmd
	DeleteKey           func(path string) tea.Cmd
	ChangePassphrase    func(path, oldPass, newPass string) tea.Cmd
	AgentStart          func() tea.Cmd
	AgentStop           func() tea.Cmd
	AgentAdd            func(path, passphrase string, lifetimeSeconds int) tea.Cmd
	AgentRemove         func(fingerprint string) tea.Cmd
	AgentRemoveAll      func() tea.Cmd
	SaveHost            func(entry core.SSHHostEntry) tea.Cmd
	DeleteHost          func(pattern string) tea.Cmd
	RemoveKnownHost     func(hostname string) tea.Cmd
	ScanHost            func(hostname string) tea.Cmd
	AddAuthorizedKey    func(line string) tea.Cmd
	RemoveAuthorizedKey func(fingerprint string) tea.Cmd
	SaveServerConfig    func(edit core.SSHServerConfigEdit) tea.Cmd
	RestoreBackup       func(target string) tea.Cmd
}

// SSHActionResultMsg is the shared response for every SSH command.
// Snapshot is the refreshed state after the mutation (or the
// unchanged pre-mutation state on error). Err is non-empty when
// the operation failed; the TUI surfaces it through notifyError.
type SSHActionResultMsg struct {
	Action   string
	Snapshot core.SSHSnapshot
	Err      string
}

// SSHSnapshotMsg carries the broadcast snapshot publishes from the
// daemon's periodic ticker and post-mutation fan-outs.
type SSHSnapshotMsg core.SSHSnapshot
