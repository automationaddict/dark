package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	btsvc "github.com/johnnelson/dark/internal/services/bluetooth"
)

type bluetoothActionRequest struct {
	Adapter string                 `json:"adapter,omitempty"`
	Device  string                 `json:"device,omitempty"`
	On      *bool                  `json:"on,omitempty"`
	Alias   string                 `json:"alias,omitempty"`
	PIN     string                 `json:"pin,omitempty"`
	Seconds uint32                 `json:"seconds,omitempty"`
	Filter  btsvc.DiscoveryFilter  `json:"filter,omitempty"`
}

type bluetoothActionResponse struct {
	Snapshot btsvc.Snapshot `json:"snapshot"`
	Error    string         `json:"error,omitempty"`
}

// wireBluetooth registers all bluetooth NATS handlers on nc. It returns
// a publisher closure the ticker loop uses to push periodic snapshots.
func wireBluetooth(nc *nats.Conn, svc *btsvc.Service, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectBluetoothAdaptersCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotBluetooth(svc))
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectBluetoothAdaptersCmd, "error", err); os.Exit(1)
	}

	register := func(subject string, handler func(*btsvc.Service, bluetoothActionRequest) bluetoothActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req bluetoothActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := bluetoothActionResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				_ = m.Respond(data)
				return
			}
			resp := handler(svc, req)
			data, _ := json.Marshal(resp)
			if err := m.Respond(data); err != nil {
				dn.Error("Bluetooth", "failed to send response: "+err.Error())
			}
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				if err := nc.Publish(bus.SubjectBluetoothAdapters, snapData); err != nil {
					dn.Error("Bluetooth", "failed to publish snapshot: "+err.Error())
				}
			}
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err); os.Exit(1)
		}
	}

	register(bus.SubjectBluetoothPowerCmd, handleBluetoothPower)
	register(bus.SubjectBluetoothDiscoverOnCmd, handleBluetoothDiscoverOn)
	register(bus.SubjectBluetoothDiscoverOffCmd, handleBluetoothDiscoverOff)
	register(bus.SubjectBluetoothConnectCmd, handleBluetoothConnect)
	register(bus.SubjectBluetoothDisconnectCmd, handleBluetoothDisconnect)
	register(bus.SubjectBluetoothPairCmd, handleBluetoothPair)
	register(bus.SubjectBluetoothRemoveCmd, handleBluetoothRemove)
	register(bus.SubjectBluetoothTrustCmd, handleBluetoothTrust)
	register(bus.SubjectBluetoothDiscoverableCmd, handleBluetoothDiscoverable)
	register(bus.SubjectBluetoothAliasCmd, handleBluetoothAlias)
	register(bus.SubjectBluetoothPairableCmd, handleBluetoothPairable)
	register(bus.SubjectBluetoothBlockCmd, handleBluetoothBlock)
	register(bus.SubjectBluetoothCancelPairCmd, handleBluetoothCancelPair)
	register(bus.SubjectBluetoothDiscoverableTimeoutCmd, handleBluetoothDiscoverableTimeout)
	register(bus.SubjectBluetoothDiscoveryFilterCmd, handleBluetoothDiscoveryFilter)

	return func() {
		data, err := json.Marshal(snapshotBluetooth(svc))
		if err != nil {
			dn.Error("Bluetooth", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectBluetoothAdapters, data); err != nil {
			dn.Error("Bluetooth", "publish failed: "+err.Error())
		}
	}
}

func snapshotBluetooth(svc *btsvc.Service) btsvc.Snapshot {
	if svc != nil {
		return svc.Snapshot()
	}
	return btsvc.Detect()
}

func handleBluetoothPower(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if req.On == nil {
		return bluetoothActionResponse{Error: "missing powered flag"}
	}
	if err := svc.SetPowered(req.Adapter, *req.On); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	time.Sleep(150 * time.Millisecond)
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothDiscoverOn(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if err := svc.StartDiscovery(req.Adapter); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothDiscoverOff(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if err := svc.StopDiscovery(req.Adapter); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothConnect(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Device == "" {
		return bluetoothActionResponse{Error: "missing device"}
	}
	if err := svc.Connect(req.Device); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothDisconnect(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Device == "" {
		return bluetoothActionResponse{Error: "missing device"}
	}
	if err := svc.Disconnect(req.Device); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothPair(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Device == "" {
		return bluetoothActionResponse{Error: "missing device"}
	}
	if err := svc.Pair(req.Device, req.PIN); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothCancelPair(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Device == "" {
		return bluetoothActionResponse{Error: "missing device"}
	}
	if err := svc.CancelPairing(req.Device); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothPairable(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if req.On == nil {
		return bluetoothActionResponse{Error: "missing pairable flag"}
	}
	if err := svc.SetPairable(req.Adapter, *req.On); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothDiscoverableTimeout(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if err := svc.SetDiscoverableTimeout(req.Adapter, req.Seconds); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothDiscoveryFilter(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if err := svc.SetDiscoveryFilter(req.Adapter, req.Filter); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothBlock(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Device == "" {
		return bluetoothActionResponse{Error: "missing device"}
	}
	if req.On == nil {
		return bluetoothActionResponse{Error: "missing blocked flag"}
	}
	if err := svc.SetBlocked(req.Device, *req.On); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothRemove(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" || req.Device == "" {
		return bluetoothActionResponse{Error: "missing adapter or device"}
	}
	if err := svc.Remove(req.Adapter, req.Device); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothDiscoverable(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if req.On == nil {
		return bluetoothActionResponse{Error: "missing discoverable flag"}
	}
	if err := svc.SetDiscoverable(req.Adapter, *req.On); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothAlias(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Adapter == "" {
		return bluetoothActionResponse{Error: "missing adapter"}
	}
	if err := svc.SetAlias(req.Adapter, req.Alias); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}

func handleBluetoothTrust(svc *btsvc.Service, req bluetoothActionRequest) bluetoothActionResponse {
	if svc == nil {
		return bluetoothActionResponse{Error: "bluetooth service unavailable"}
	}
	if req.Device == "" {
		return bluetoothActionResponse{Error: "missing device"}
	}
	if req.On == nil {
		return bluetoothActionResponse{Error: "missing trusted flag"}
	}
	if err := svc.SetTrusted(req.Device, *req.On); err != nil {
		return bluetoothActionResponse{Error: err.Error()}
	}
	return bluetoothActionResponse{Snapshot: svc.Snapshot()}
}
