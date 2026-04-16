package tui

import tea "github.com/charmbracelet/bubbletea"

// loadSSHIfNeeded fires nothing today — the snapshot arrives via the
// initial request in main.go at TUI startup and via the broadcast
// subscription thereafter. This function exists so F4's key handler
// mirrors F5's shape and future explicit refresh commands can land
// here without another round of plumbing.
func (m *Model) loadSSHIfNeeded() tea.Cmd {
	return nil
}
