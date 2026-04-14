package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/services/firmware"
	"github.com/johnnelson/dark/internal/services/update"
)

// UpdateActions holds the NATS-backed commands for system updates.
type UpdateActions struct {
	Run           func() tea.Cmd
	ChangeChannel func(channel string) tea.Cmd
}

// UpdateMsg is dispatched when darkd publishes an update snapshot.
type UpdateMsg update.Snapshot

// UpdateRunResultMsg is the reply from a run-update command.
type UpdateRunResultMsg struct {
	Snapshot update.Snapshot
	Result   update.RunResult
	Err      string
}

// UpdateChannelResultMsg is the reply from a channel-change command.
type UpdateChannelResultMsg struct {
	Snapshot update.Snapshot
	Err      string
}

// FirmwareMsg is dispatched when darkd publishes a firmware snapshot.
type FirmwareMsg firmware.Snapshot
