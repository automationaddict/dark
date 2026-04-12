package main

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	btsvc "github.com/johnnelson/dark/internal/services/bluetooth"
	"github.com/johnnelson/dark/internal/tui"
)

// newBluetoothActions builds the closures that send bluetooth command
// requests over NATS and return bubble tea messages with the reply.
func newBluetoothActions(nc *nats.Conn) tui.BluetoothActions {
	return tui.BluetoothActions{
		SetPowered: func(adapter string, powered bool) tea.Cmd {
			return func() tea.Msg {
				return btBoolRequest(nc, bus.SubjectBluetoothPowerCmd, adapter, "", powered)
			}
		},
		StartDiscovery: func(adapter string) tea.Cmd {
			return func() tea.Msg {
				return btPathRequest(nc, bus.SubjectBluetoothDiscoverOnCmd, adapter, "")
			}
		},
		StopDiscovery: func(adapter string) tea.Cmd {
			return func() tea.Msg {
				return btPathRequest(nc, bus.SubjectBluetoothDiscoverOffCmd, adapter, "")
			}
		},
		Connect: func(device string) tea.Cmd {
			return func() tea.Msg {
				return btPathRequest(nc, bus.SubjectBluetoothConnectCmd, "", device)
			}
		},
		Disconnect: func(device string) tea.Cmd {
			return func() tea.Msg {
				return btPathRequest(nc, bus.SubjectBluetoothDisconnectCmd, "", device)
			}
		},
		Pair: func(device, pin string) tea.Cmd {
			return func() tea.Msg {
				return btPairRequest(nc, device, pin)
			}
		},
		CancelPair: func(device string) tea.Cmd {
			return func() tea.Msg {
				return btPathRequest(nc, bus.SubjectBluetoothCancelPairCmd, "", device)
			}
		},
		Remove: func(adapter, device string) tea.Cmd {
			return func() tea.Msg {
				return btPathRequest(nc, bus.SubjectBluetoothRemoveCmd, adapter, device)
			}
		},
		SetTrusted: func(device string, trusted bool) tea.Cmd {
			return func() tea.Msg {
				return btBoolRequest(nc, bus.SubjectBluetoothTrustCmd, "", device, trusted)
			}
		},
		SetDiscoverable: func(adapter string, discoverable bool) tea.Cmd {
			return func() tea.Msg {
				return btBoolRequest(nc, bus.SubjectBluetoothDiscoverableCmd, adapter, "", discoverable)
			}
		},
		SetPairable: func(adapter string, pairable bool) tea.Cmd {
			return func() tea.Msg {
				return btBoolRequest(nc, bus.SubjectBluetoothPairableCmd, adapter, "", pairable)
			}
		},
		SetDiscoverableTimeout: func(adapter string, seconds uint32) tea.Cmd {
			return func() tea.Msg {
				return btTimeoutRequest(nc, adapter, seconds)
			}
		},
		SetDiscoveryFilter: func(adapter string, filter btsvc.DiscoveryFilter) tea.Cmd {
			return func() tea.Msg {
				return btFilterRequest(nc, adapter, filter)
			}
		},
		SetBlocked: func(device string, blocked bool) tea.Cmd {
			return func() tea.Msg {
				return btBoolRequest(nc, bus.SubjectBluetoothBlockCmd, "", device, blocked)
			}
		},
		SetAlias: func(adapter, alias string) tea.Cmd {
			return func() tea.Msg {
				return btAliasRequest(nc, adapter, alias)
			}
		},
	}
}

func btAliasRequest(nc *nats.Conn, adapter, alias string) tui.BluetoothActionResultMsg {
	payload, _ := json.Marshal(map[string]string{"adapter": adapter, "alias": alias})
	reply, err := nc.Request(bus.SubjectBluetoothAliasCmd, payload, 10*time.Second)
	if err != nil {
		return tui.BluetoothActionResultMsg{Err: err.Error()}
	}
	return decodeBluetoothReply(reply.Data)
}

func btTimeoutRequest(nc *nats.Conn, adapter string, seconds uint32) tui.BluetoothActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"adapter": adapter, "seconds": seconds})
	reply, err := nc.Request(bus.SubjectBluetoothDiscoverableTimeoutCmd, payload, 10*time.Second)
	if err != nil {
		return tui.BluetoothActionResultMsg{Err: err.Error()}
	}
	return decodeBluetoothReply(reply.Data)
}

func btFilterRequest(nc *nats.Conn, adapter string, filter btsvc.DiscoveryFilter) tui.BluetoothActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"adapter": adapter, "filter": filter})
	reply, err := nc.Request(bus.SubjectBluetoothDiscoveryFilterCmd, payload, 10*time.Second)
	if err != nil {
		return tui.BluetoothActionResultMsg{Err: err.Error()}
	}
	return decodeBluetoothReply(reply.Data)
}

func btPairRequest(nc *nats.Conn, device, pin string) tui.BluetoothActionResultMsg {
	payload, _ := json.Marshal(map[string]string{"device": device, "pin": pin})
	reply, err := nc.Request(bus.SubjectBluetoothPairCmd, payload, 60*time.Second)
	if err != nil {
		return tui.BluetoothActionResultMsg{Err: err.Error()}
	}
	return decodeBluetoothReply(reply.Data)
}

func btPathRequest(nc *nats.Conn, subject, adapter, device string) tui.BluetoothActionResultMsg {
	payload, _ := json.Marshal(map[string]string{"adapter": adapter, "device": device})
	reply, err := nc.Request(subject, payload, 40*time.Second)
	if err != nil {
		return tui.BluetoothActionResultMsg{Err: err.Error()}
	}
	return decodeBluetoothReply(reply.Data)
}

func btBoolRequest(nc *nats.Conn, subject, adapter, device string, flag bool) tui.BluetoothActionResultMsg {
	payload, _ := json.Marshal(map[string]any{
		"adapter": adapter,
		"device":  device,
		"on":      flag,
	})
	reply, err := nc.Request(subject, payload, 10*time.Second)
	if err != nil {
		return tui.BluetoothActionResultMsg{Err: err.Error()}
	}
	return decodeBluetoothReply(reply.Data)
}

func decodeBluetoothReply(data []byte) tui.BluetoothActionResultMsg {
	var resp struct {
		Snapshot btsvc.Snapshot `json:"snapshot"`
		Error    string         `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.BluetoothActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.BluetoothActionResultMsg{Err: resp.Error}
	}
	return tui.BluetoothActionResultMsg{Snapshot: resp.Snapshot}
}

// requestInitialBluetooth fetches a bluetooth snapshot up front so the
// Bluetooth section has data on the first frame. Errors are swallowed
// since the periodic publish will backfill.
func requestInitialBluetooth(nc *nats.Conn) (btsvc.Snapshot, bool) {
	reply, err := nc.Request(bus.SubjectBluetoothAdaptersCmd, nil, 1*time.Second)
	if err != nil {
		return btsvc.Snapshot{}, false
	}
	var snap btsvc.Snapshot
	if err := json.Unmarshal(reply.Data, &snap); err != nil {
		return btsvc.Snapshot{}, false
	}
	return snap, true
}
