package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/notifycfg"
)

type notifyCfgResponse struct {
	Snapshot notifycfg.Snapshot `json:"snapshot"`
	Error    string             `json:"error,omitempty"`
}

func wireNotifyCfg(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectNotifySnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(notifycfg.ReadSnapshot())
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectNotifySnapshotCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectNotifyDNDCmd, func(m *nats.Msg) {
		var resp notifyCfgResponse
		if err := notifycfg.ToggleDND(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = notifycfg.ReadSnapshot()
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectNotifyDNDCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectNotifyDismissCmd, func(m *nats.Msg) {
		var resp notifyCfgResponse
		if err := notifycfg.DismissAll(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = notifycfg.ReadSnapshot()
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectNotifyDismissCmd, "error", err)
		os.Exit(1)
	}

	publish := func() {
		data, err := json.Marshal(notifycfg.ReadSnapshot())
		if err != nil {
			dn.Error("Notifications", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectNotifySnapshot, data); err != nil {
			dn.Error("Notifications", "publish failed: "+err.Error())
		}
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			publish()
		}
	}()

	return publish
}
