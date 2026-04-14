package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	updatesvc "github.com/johnnelson/dark/internal/services/update"
)

func wireUpdate(nc *nats.Conn, dn *daemonNotifier) func() {
	helperPath := findHelperPath()

	if _, err := nc.Subscribe(bus.SubjectUpdateSnapshotCmd, func(m *nats.Msg) {
		snap := updatesvc.Check()
		data, _ := json.Marshal(snap)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectUpdateSnapshotCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectUpdateRunCmd, func(m *nats.Msg) {
		result := updatesvc.Run(helperPath)
		resp := updateRunResponse{Result: result}
		if result.Error != "" {
			resp.Error = result.Error
		}
		// Refresh the snapshot after update
		resp.Snapshot = updatesvc.Check()
		data, _ := json.Marshal(resp)
		respond(m, data)
		// Publish updated snapshot
		snapData, _ := json.Marshal(resp.Snapshot)
		publish(nc, bus.SubjectUpdateSnapshot, snapData)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectUpdateRunCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectUpdateChannelCmd, func(m *nats.Msg) {
		var req struct {
			Channel string `json:"channel"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			resp := updateChannelResponse{Error: "malformed request: " + err.Error()}
			data, _ := json.Marshal(resp)
			respond(m, data)
			return
		}
		err := updatesvc.ChangeChannel(helperPath, req.Channel)
		var resp updateChannelResponse
		if err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = updatesvc.Check()
		data, _ := json.Marshal(resp)
		respond(m, data)
		snapData, _ := json.Marshal(resp.Snapshot)
		publish(nc, bus.SubjectUpdateSnapshot, snapData)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectUpdateChannelCmd, "error", err)
		os.Exit(1)
	}

	return func() {
		snap := updatesvc.Check()
		data, err := json.Marshal(snap)
		if err != nil {
			slog.Warn("update: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectUpdateSnapshot, data); err != nil {
			slog.Warn("update: publish snapshot", "err", err)
		}
	}
}

type updateChannelResponse struct {
	Snapshot updatesvc.Snapshot `json:"snapshot"`
	Error    string             `json:"error,omitempty"`
}

type updateRunResponse struct {
	Snapshot updatesvc.Snapshot  `json:"snapshot"`
	Result   updatesvc.RunResult `json:"result"`
	Error    string              `json:"error,omitempty"`
}

func findHelperPath() string {
	if env := os.Getenv("DARK_HELPER"); env != "" {
		return env
	}
	if exe, err := os.Executable(); err == nil {
		candidate := exe[:len(exe)-len("darkd")] + "dark-helper"
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	for _, p := range []string{"/usr/local/bin/dark-helper", "/usr/bin/dark-helper"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "dark-helper"
}
