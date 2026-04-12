package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	appstoresvc "github.com/johnnelson/dark/internal/services/appstore"
	"github.com/johnnelson/dark/internal/tui"
)

// newAppstoreActions builds the closures that send appstore command
// requests over NATS and return bubble tea messages with the reply.
// Each closure captures nc so the caller can use the returned value
// without threading it through the Model.
func newAppstoreActions(nc *nats.Conn) tui.AppstoreActions {
	return tui.AppstoreActions{
		Search: func(q appstoresvc.SearchQuery) tea.Cmd {
			return func() tea.Msg {
				return appstoreSearchRequest(nc, q)
			}
		},
		Detail: func(req appstoresvc.DetailRequest) tea.Cmd {
			return func() tea.Msg {
				return appstoreDetailRequest(nc, req)
			}
		},
		Refresh: func() tea.Cmd {
			return func() tea.Msg {
				return appstoreRefreshRequest(nc)
			}
		},
		Install: func(req appstoresvc.InstallRequest) tea.Cmd {
			return func() tea.Msg {
				return appstoreInstallRequest(nc, req)
			}
		},
		Remove: func(names []string) tea.Cmd {
			return func() tea.Msg {
				return appstoreRemoveRequest(nc, names)
			}
		},
		Upgrade: func() tea.Cmd {
			return func() tea.Msg {
				return appstoreUpgradeRequest(nc)
			}
		},
	}
}

// appstoreSearchRequest serializes a SearchQuery, issues the NATS
// request, and decodes the reply into the TUI-facing message type.
// Errors from the transport layer and errors reported by the daemon
// are folded into the Err field on the message.
func appstoreSearchRequest(nc *nats.Conn, q appstoresvc.SearchQuery) tui.AppstoreSearchResultMsg {
	payload, _ := json.Marshal(q)
	reply, err := nc.Request(bus.SubjectAppstoreSearchCmd, payload, core.TimeoutSlow)
	if err != nil {
		return tui.AppstoreSearchResultMsg{Err: err.Error()}
	}
	var resp struct {
		Result appstoresvc.SearchResult `json:"result"`
		Error  string                   `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.AppstoreSearchResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.AppstoreSearchResultMsg{Result: resp.Result, Err: resp.Error}
	}
	return tui.AppstoreSearchResultMsg{Result: resp.Result}
}

func appstoreDetailRequest(nc *nats.Conn, req appstoresvc.DetailRequest) tui.AppstoreDetailResultMsg {
	payload, _ := json.Marshal(req)
	reply, err := nc.Request(bus.SubjectAppstoreDetailCmd, payload, core.TimeoutSlow)
	if err != nil {
		return tui.AppstoreDetailResultMsg{Err: err.Error()}
	}
	var resp struct {
		Detail appstoresvc.Detail `json:"detail"`
		Error  string             `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.AppstoreDetailResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.AppstoreDetailResultMsg{Detail: resp.Detail, Err: resp.Error}
	}
	return tui.AppstoreDetailResultMsg{Detail: resp.Detail}
}

func appstoreRefreshRequest(nc *nats.Conn) tui.AppstoreRefreshResultMsg {
	reply, err := nc.Request(bus.SubjectAppstoreRefreshCmd, nil, core.TimeoutRefresh)
	if err != nil {
		return tui.AppstoreRefreshResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot appstoresvc.Snapshot `json:"snapshot"`
		Error    string               `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.AppstoreRefreshResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.AppstoreRefreshResultMsg{Snapshot: resp.Snapshot, Err: resp.Error}
	}
	return tui.AppstoreRefreshResultMsg{Snapshot: resp.Snapshot}
}

func appstoreInstallRequest(nc *nats.Conn, req appstoresvc.InstallRequest) tui.AppstoreActionResultMsg {
	payload, _ := json.Marshal(req)
	reply, err := nc.Request(bus.SubjectAppstoreInstallCmd, payload, core.TimeoutPkexec)
	if err != nil {
		return tui.AppstoreActionResultMsg{Err: err.Error()}
	}
	return decodeAppstoreActionReply(reply.Data)
}

func appstoreRemoveRequest(nc *nats.Conn, names []string) tui.AppstoreActionResultMsg {
	payload, _ := json.Marshal(map[string][]string{"names": names})
	reply, err := nc.Request(bus.SubjectAppstoreRemoveCmd, payload, core.TimeoutPkexec)
	if err != nil {
		return tui.AppstoreActionResultMsg{Err: err.Error()}
	}
	return decodeAppstoreActionReply(reply.Data)
}

func appstoreUpgradeRequest(nc *nats.Conn) tui.AppstoreActionResultMsg {
	reply, err := nc.Request(bus.SubjectAppstoreUpgradeCmd, nil, 5*core.TimeoutPkexec)
	if err != nil {
		return tui.AppstoreActionResultMsg{Err: err.Error()}
	}
	return decodeAppstoreActionReply(reply.Data)
}

func decodeAppstoreActionReply(data []byte) tui.AppstoreActionResultMsg {
	var resp struct {
		Snapshot appstoresvc.Snapshot `json:"snapshot,omitempty"`
		Output   string               `json:"output,omitempty"`
		Error    string               `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.AppstoreActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.AppstoreActionResultMsg{Output: resp.Output, Err: resp.Error}
	}
	return tui.AppstoreActionResultMsg{Snapshot: resp.Snapshot, Output: resp.Output}
}

// requestInitialAppstore fetches a catalog snapshot up front so the
// F2 tab has data on the first frame. Errors are swallowed because
// the periodic publish on a 60s ticker will backfill and the TUI
// handles an unloaded appstore state gracefully.
func requestInitialAppstore(nc *nats.Conn) (appstoresvc.Snapshot, bool) {
	reply, err := nc.Request(bus.SubjectAppstoreCatalogCmd, nil, core.TimeoutFast)
	if err != nil {
		return appstoresvc.Snapshot{}, false
	}
	var snap appstoresvc.Snapshot
	if err := json.Unmarshal(reply.Data, &snap); err != nil {
		return appstoresvc.Snapshot{}, false
	}
	return snap, true
}
