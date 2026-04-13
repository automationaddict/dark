package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/notifycfg"
	"github.com/johnnelson/dark/internal/tui"
)

func newNotifyCfgActions(nc *nats.Conn) tui.NotifyConfigActions {
	return tui.NotifyConfigActions{
		ToggleDND: func() tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyDNDCmd)
			}
		},
		DismissAll: func() tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyDismissCmd)
			}
		},
	}
}

func notifyCfgRequest(nc *nats.Conn, subject string) tui.NotifyCfgActionResultMsg {
	reply, err := nc.Request(subject, nil, core.TimeoutNormal)
	if err != nil {
		return tui.NotifyCfgActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot notifycfg.Snapshot `json:"snapshot"`
		Error    string            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.NotifyCfgActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.NotifyCfgActionResultMsg{Err: resp.Error}
	}
	return tui.NotifyCfgActionResultMsg{Snapshot: resp.Snapshot}
}
