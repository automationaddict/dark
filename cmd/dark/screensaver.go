package main

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/screensaver"
	"github.com/johnnelson/dark/internal/tui"
)

// screensaverPreviewTimeout is longer than the backend's 60s internal
// failsafe so dark never times out the NATS request before the service
// has a chance to clean up after itself.
const screensaverPreviewTimeout = 90 * time.Second

// screensaverActionTimeout covers the non-preview commands
// (set_enabled / set_content) which are quick file operations.
const screensaverActionTimeout = 10 * time.Second

func newScreensaverActions(nc *nats.Conn) tui.ScreensaverActions {
	return tui.ScreensaverActions{
		SetEnabled: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return screensaverRequest(nc, bus.SubjectScreensaverSetEnabledCmd, map[string]any{
					"enabled": enabled,
				}, screensaverActionTimeout, "set_enabled")
			}
		},
		SetContent: func(content string) tea.Cmd {
			return func() tea.Msg {
				return screensaverRequest(nc, bus.SubjectScreensaverSetContentCmd, map[string]any{
					"content": content,
				}, screensaverActionTimeout, "set_content")
			}
		},
		Preview: func() tea.Cmd {
			return func() tea.Msg {
				return screensaverRequest(nc, bus.SubjectScreensaverPreviewCmd, nil, screensaverPreviewTimeout, "preview")
			}
		},
	}
}

func screensaverRequest(nc *nats.Conn, subject string, payload any, timeout time.Duration, action string) tui.ScreensaverActionResultMsg {
	var data []byte
	if payload != nil {
		data, _ = json.Marshal(payload)
	}
	reply, err := nc.Request(subject, data, timeout)
	if err != nil {
		return tui.ScreensaverActionResultMsg{Err: err.Error(), Action: action}
	}
	return parseScreensaverResponse(reply.Data, action)
}

func parseScreensaverResponse(data []byte, action string) tui.ScreensaverActionResultMsg {
	var resp struct {
		Snapshot screensaver.Snapshot `json:"snapshot"`
		Error    string               `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.ScreensaverActionResultMsg{Err: err.Error(), Action: action}
	}
	if resp.Error != "" {
		return tui.ScreensaverActionResultMsg{Snapshot: resp.Snapshot, Err: resp.Error, Action: action}
	}
	return tui.ScreensaverActionResultMsg{Snapshot: resp.Snapshot, Action: action}
}
