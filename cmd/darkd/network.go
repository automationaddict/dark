package main

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	netsvc "github.com/johnnelson/dark/internal/services/network"
)

// wireNetwork registers the network reply handler and returns a
// publisher closure the daemon's main ticker uses for periodic
// snapshot pushes. Tier 1 is read-only so there are no action
// handlers — just snapshot delivery.
func wireNetwork(nc *nats.Conn, svc *netsvc.Service) func() {
	if _, err := nc.Subscribe(bus.SubjectNetworkSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotNetwork(svc))
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe network snapshot cmd: %v", err)
	}

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

func snapshotNetwork(svc *netsvc.Service) netsvc.Snapshot {
	if svc != nil {
		return svc.Snapshot()
	}
	return netsvc.Detect()
}
