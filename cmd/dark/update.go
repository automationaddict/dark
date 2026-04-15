package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	updatesvc "github.com/automationaddict/dark/internal/services/update"
	"github.com/automationaddict/dark/internal/tui"
)

func newUpdateActions(nc *nats.Conn) tui.UpdateActions {
	return tui.UpdateActions{
		Run: func() tea.Cmd {
			return func() tea.Msg {
				return updateRunRequest(nc)
			}
		},
		ChangeChannel: func(channel string) tea.Cmd {
			return func() tea.Msg {
				return updateChannelRequest(nc, channel)
			}
		},
	}
}

func updateRunRequest(nc *nats.Conn) tui.UpdateRunResultMsg {
	reply, err := nc.Request(bus.SubjectUpdateRunCmd, nil, 10*core.TimeoutPkexec)
	if err != nil {
		return tui.UpdateRunResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot updatesvc.Snapshot  `json:"snapshot"`
		Result   updatesvc.RunResult `json:"result"`
		Error    string              `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.UpdateRunResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.UpdateRunResultMsg{Result: resp.Result, Err: resp.Error}
	}
	return tui.UpdateRunResultMsg{Snapshot: resp.Snapshot, Result: resp.Result}
}

func updateChannelRequest(nc *nats.Conn, channel string) tui.UpdateChannelResultMsg {
	payload, _ := json.Marshal(map[string]string{"channel": channel})
	reply, err := nc.Request(bus.SubjectUpdateChannelCmd, payload, core.TimeoutPkexec)
	if err != nil {
		return tui.UpdateChannelResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot updatesvc.Snapshot `json:"snapshot"`
		Error    string             `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.UpdateChannelResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.UpdateChannelResultMsg{Err: resp.Error}
	}
	return tui.UpdateChannelResultMsg{Snapshot: resp.Snapshot}
}

func requestInitialUpdate(nc *nats.Conn) (updatesvc.Snapshot, bool) {
	reply, err := nc.Request(bus.SubjectUpdateSnapshotCmd, nil, core.TimeoutFast)
	if err != nil {
		return updatesvc.Snapshot{}, false
	}
	var snap updatesvc.Snapshot
	if err := json.Unmarshal(reply.Data, &snap); err != nil {
		return updatesvc.Snapshot{}, false
	}
	return snap, true
}
