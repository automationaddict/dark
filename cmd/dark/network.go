package main

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	netsvc "github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/tui"
)

// requestInitialNetwork fetches a network snapshot up front so the
// Network section has data on the first frame.
func requestInitialNetwork(nc *nats.Conn) (netsvc.Snapshot, bool) {
	reply, err := nc.Request(bus.SubjectNetworkSnapshotCmd, nil, 1*time.Second)
	if err != nil {
		return netsvc.Snapshot{}, false
	}
	var snap netsvc.Snapshot
	if err := json.Unmarshal(reply.Data, &snap); err != nil {
		return netsvc.Snapshot{}, false
	}
	return snap, true
}

// newNetworkActions builds the closures the TUI uses to dispatch
// network commands over NATS. Mirrors the pattern in audio.go and
// bluetooth.go.
func newNetworkActions(nc *nats.Conn) tui.NetworkActions {
	return tui.NetworkActions{
		Reconfigure: func(iface string) tea.Cmd {
			return func() tea.Msg {
				return networkReconfigureRequest(nc, iface)
			}
		},
		ConfigureIPv4: func(iface string, cfg netsvc.IPv4Config) tea.Cmd {
			return func() tea.Msg {
				return networkConfigureIPv4Request(nc, iface, cfg)
			}
		},
		ResetInterface: func(iface string) tea.Cmd {
			return func() tea.Msg {
				return networkResetRequest(nc, iface)
			}
		},
	}
}

func networkResetRequest(nc *nats.Conn, iface string) tui.NetworkActionResultMsg {
	payload, _ := json.Marshal(map[string]string{"interface": iface})
	// Long timeout — the helper still goes through pkexec which can
	// sit waiting for the user to type their password.
	reply, err := nc.Request(bus.SubjectNetworkResetCmd, payload, 120*time.Second)
	if err != nil {
		return tui.NetworkActionResultMsg{Err: err.Error()}
	}
	return decodeNetworkReply(reply.Data)
}

func networkReconfigureRequest(nc *nats.Conn, iface string) tui.NetworkActionResultMsg {
	payload, _ := json.Marshal(map[string]string{"interface": iface})
	reply, err := nc.Request(bus.SubjectNetworkReconfigureCmd, payload, 15*time.Second)
	if err != nil {
		return tui.NetworkActionResultMsg{Err: err.Error()}
	}
	return decodeNetworkReply(reply.Data)
}

func networkConfigureIPv4Request(nc *nats.Conn, iface string, cfg netsvc.IPv4Config) tui.NetworkActionResultMsg {
	payload, _ := json.Marshal(map[string]any{
		"interface": iface,
		"ipv4":      cfg,
	})
	// Long timeout — pkexec might be sitting on a polkit dialog waiting
	// for the user to type their password.
	reply, err := nc.Request(bus.SubjectNetworkConfigureIPv4Cmd, payload, 120*time.Second)
	if err != nil {
		return tui.NetworkActionResultMsg{Err: err.Error()}
	}
	return decodeNetworkReply(reply.Data)
}

func decodeNetworkReply(data []byte) tui.NetworkActionResultMsg {
	var resp struct {
		Snapshot netsvc.Snapshot `json:"snapshot"`
		Error    string          `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.NetworkActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.NetworkActionResultMsg{Err: resp.Error}
	}
	return tui.NetworkActionResultMsg{Snapshot: resp.Snapshot}
}
