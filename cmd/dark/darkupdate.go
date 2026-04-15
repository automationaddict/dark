package main

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	darkupdatesvc "github.com/automationaddict/dark/internal/services/darkupdate"
	"github.com/automationaddict/dark/internal/tui"
)

// darkUpdateCheckTimeout is short — the check is just a GitHub
// API GET and shouldn't ever take multiple seconds in practice.
const darkUpdateCheckTimeout = 20 * time.Second

// darkUpdateApplyTimeout has to accommodate the full download +
// verify + extract + install pipeline. Even a ~20MB tarball on a
// slow connection should comfortably land within 3 minutes.
const darkUpdateApplyTimeout = 3 * time.Minute

func newDarkUpdateActions(nc *nats.Conn) tui.DarkUpdateActions {
	return tui.DarkUpdateActions{
		Check: func() tea.Cmd {
			return func() tea.Msg {
				return darkUpdateRequest(nc, bus.SubjectDarkUpdateCheckCmd, nil, darkUpdateCheckTimeout, "check")
			}
		},
		Apply: func() tea.Cmd {
			return func() tea.Msg {
				return darkUpdateRequest(nc, bus.SubjectDarkUpdateApplyCmd, nil, darkUpdateApplyTimeout, "apply")
			}
		},
	}
}

func darkUpdateRequest(nc *nats.Conn, subject string, payload any, timeout time.Duration, action string) tui.DarkUpdateActionResultMsg {
	var data []byte
	if payload != nil {
		data, _ = json.Marshal(payload)
	}
	reply, err := nc.Request(subject, data, timeout)
	if err != nil {
		return tui.DarkUpdateActionResultMsg{Err: err.Error(), Action: action}
	}
	var resp struct {
		Snapshot darkupdatesvc.Snapshot `json:"snapshot"`
		Error    string                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.DarkUpdateActionResultMsg{Err: err.Error(), Action: action}
	}
	if resp.Error != "" {
		return tui.DarkUpdateActionResultMsg{Snapshot: resp.Snapshot, Err: resp.Error, Action: action}
	}
	return tui.DarkUpdateActionResultMsg{Snapshot: resp.Snapshot, Action: action}
}
