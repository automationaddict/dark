package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	darkupdatesvc "github.com/automationaddict/dark/internal/services/darkupdate"
)

// wireDarkUpdate installs handlers for the self-update subjects
// and returns a publish closure the main loop fires at startup.
// The darkupdate client is stateful (it caches CheckLatest
// results across commands), so we instantiate it once here and
// close over it in every handler.
func wireDarkUpdate(nc *nats.Conn) func() {
	client := darkupdatesvc.NewClient()

	if _, err := nc.Subscribe(bus.SubjectDarkUpdateSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(client.Snapshot())
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDarkUpdateSnapshotCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectDarkUpdateCheckCmd, func(m *nats.Msg) {
		snap := client.CheckLatest()
		data, _ := json.Marshal(snap)
		respond(m, data)
		// Publish the new snapshot so the TUI's cached copy
		// updates immediately rather than waiting for the
		// reply path to land.
		publish(nc, bus.SubjectDarkUpdateSnapshot, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDarkUpdateCheckCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectDarkUpdateApplyCmd, func(m *nats.Msg) {
		var req struct {
			Tag string `json:"tag"`
		}
		// Empty body is fine — Apply falls back to the cached
		// Latest tag when no explicit one is supplied. This
		// matches the common case (TUI: user pressed "apply"
		// after a CheckLatest).
		_ = json.Unmarshal(m.Data, &req)

		var resp darkUpdateResponse
		if err := client.Apply(req.Tag); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = client.Snapshot()
		data, _ := json.Marshal(resp)
		respond(m, data)
		snapData, _ := json.Marshal(resp.Snapshot)
		publish(nc, bus.SubjectDarkUpdateSnapshot, snapData)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDarkUpdateApplyCmd, "error", err)
		os.Exit(1)
	}

	return func() {
		data, err := json.Marshal(client.Snapshot())
		if err != nil {
			slog.Warn("darkupdate: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectDarkUpdateSnapshot, data); err != nil {
			slog.Warn("darkupdate: publish snapshot", "err", err)
		}
	}
}

// darkUpdateResponse is the shared reply shape for Apply.
type darkUpdateResponse struct {
	Snapshot darkupdatesvc.Snapshot `json:"snapshot"`
	Error    string                 `json:"error,omitempty"`
}
