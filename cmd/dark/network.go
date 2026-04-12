package main

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	netsvc "github.com/johnnelson/dark/internal/services/network"
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
