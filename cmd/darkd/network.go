package main

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	netsvc "github.com/johnnelson/dark/internal/services/network"
)

type networkActionRequest struct {
	Interface string             `json:"interface,omitempty"`
	IPv4      *netsvc.IPv4Config `json:"ipv4,omitempty"`
}

type networkActionResponse struct {
	Snapshot netsvc.Snapshot `json:"snapshot"`
	Error    string          `json:"error,omitempty"`
}

// wireNetwork registers the network NATS handlers and returns a
// publisher closure the daemon's main ticker uses for periodic
// snapshot pushes.
func wireNetwork(nc *nats.Conn, svc *netsvc.Service, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectNetworkSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotNetwork(svc))
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe network snapshot cmd: %v", err)
	}

	register := func(subject string, handler func(*netsvc.Service, networkActionRequest) networkActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req networkActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := networkActionResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				_ = m.Respond(data)
				return
			}
			resp := handler(svc, req)
			data, _ := json.Marshal(resp)
			if err := m.Respond(data); err != nil {
				dn.Error("Network", "failed to send response: "+err.Error())
			}
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				if err := nc.Publish(bus.SubjectNetworkSnapshot, snapData); err != nil {
					dn.Error("Network", "failed to publish snapshot: "+err.Error())
				}
			}
		}); err != nil {
			log.Fatalf("subscribe %s: %v", subject, err)
		}
	}

	register(bus.SubjectNetworkReconfigureCmd, handleNetworkReconfigure)
	register(bus.SubjectNetworkConfigureIPv4Cmd, handleNetworkConfigureIPv4)
	register(bus.SubjectNetworkResetCmd, handleNetworkReset)

	return func() {
		data, err := json.Marshal(snapshotNetwork(svc))
		if err != nil {
			log.Printf("marshal network: %v", err)
			return
		}
		if err := nc.Publish(bus.SubjectNetworkSnapshot, data); err != nil {
			log.Printf("publish network: %v", err)
		}
	}
}

func handleNetworkReconfigure(svc *netsvc.Service, req networkActionRequest) networkActionResponse {
	if svc == nil {
		return networkActionResponse{Error: "network service unavailable"}
	}
	if req.Interface == "" {
		return networkActionResponse{Error: "missing interface name"}
	}
	if err := svc.Reconfigure(req.Interface); err != nil {
		return networkActionResponse{Error: err.Error()}
	}
	return networkActionResponse{Snapshot: svc.Snapshot()}
}

func handleNetworkConfigureIPv4(svc *netsvc.Service, req networkActionRequest) networkActionResponse {
	if svc == nil {
		return networkActionResponse{Error: "network service unavailable"}
	}
	if req.Interface == "" {
		return networkActionResponse{Error: "missing interface name"}
	}
	if req.IPv4 == nil {
		return networkActionResponse{Error: "missing ipv4 config"}
	}
	if err := svc.ConfigureIPv4(req.Interface, *req.IPv4); err != nil {
		return networkActionResponse{Error: err.Error()}
	}
	return networkActionResponse{Snapshot: svc.Snapshot()}
}

func handleNetworkReset(svc *netsvc.Service, req networkActionRequest) networkActionResponse {
	if svc == nil {
		return networkActionResponse{Error: "network service unavailable"}
	}
	if req.Interface == "" {
		return networkActionResponse{Error: "missing interface name"}
	}
	if err := svc.ResetInterface(req.Interface); err != nil {
		return networkActionResponse{Error: err.Error()}
	}
	return networkActionResponse{Snapshot: svc.Snapshot()}
}

func snapshotNetwork(svc *netsvc.Service) netsvc.Snapshot {
	if svc != nil {
		return svc.Snapshot()
	}
	return netsvc.Detect()
}
