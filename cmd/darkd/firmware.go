package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	fwsvc "github.com/johnnelson/dark/internal/services/firmware"
)

func wireFirmware(nc *nats.Conn, svc *fwsvc.Service, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectFirmwareSnapshotCmd, func(m *nats.Msg) {
		var snap fwsvc.Snapshot
		if svc != nil {
			snap = svc.Snapshot()
		}
		data, _ := json.Marshal(snap)
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectFirmwareSnapshotCmd, "error", err)
		os.Exit(1)
	}

	return func() {
		var snap fwsvc.Snapshot
		if svc != nil {
			snap = svc.Snapshot()
		}
		data, err := json.Marshal(snap)
		if err != nil {
			slog.Warn("firmware: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectFirmwareSnapshot, data); err != nil {
			slog.Warn("firmware: publish snapshot", "err", err)
		}
	}
}
