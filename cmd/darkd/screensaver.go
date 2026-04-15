package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	screensaversvc "github.com/automationaddict/dark/internal/services/screensaver"
)

// wireScreensaver installs handlers for the screensaver command subjects
// and returns a publish closure the main loop fires on each tick. Every
// mutation dark can perform on the screensaver is a user-file operation,
// so nothing here touches dark-helper.
func wireScreensaver(nc *nats.Conn) func() {
	if _, err := nc.Subscribe(bus.SubjectScreensaverSnapshotCmd, func(m *nats.Msg) {
		snap := screensaversvc.ReadSnapshot()
		data, _ := json.Marshal(snap)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScreensaverSnapshotCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScreensaverSetEnabledCmd, func(m *nats.Msg) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			respondScreensaver(m, "malformed request: "+err.Error())
			return
		}
		var resp screensaverResponse
		if err := screensaversvc.SetEnabled(req.Enabled); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = screensaversvc.ReadSnapshot()
		data, _ := json.Marshal(resp)
		respond(m, data)
		snapData, _ := json.Marshal(resp.Snapshot)
		publish(nc, bus.SubjectScreensaverSnapshot, snapData)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScreensaverSetEnabledCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScreensaverSetContentCmd, func(m *nats.Msg) {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			respondScreensaver(m, "malformed request: "+err.Error())
			return
		}
		var resp screensaverResponse
		if err := screensaversvc.WriteContent(req.Content); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = screensaversvc.ReadSnapshot()
		data, _ := json.Marshal(resp)
		respond(m, data)
		snapData, _ := json.Marshal(resp.Snapshot)
		publish(nc, bus.SubjectScreensaverSnapshot, snapData)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScreensaverSetContentCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScreensaverPreviewCmd, func(m *nats.Msg) {
		// LaunchPreview blocks until the screensaver exits (the user
		// presses a key, loses focus, or the failsafe timeout fires).
		// The caller gets a single reply once that's done, so the TUI
		// can show a "previewing…" modal and then clear it.
		var resp screensaverResponse
		if err := screensaversvc.LaunchPreview(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = screensaversvc.ReadSnapshot()
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScreensaverPreviewCmd, "error", err)
		os.Exit(1)
	}

	return func() {
		snap := screensaversvc.ReadSnapshot()
		data, err := json.Marshal(snap)
		if err != nil {
			slog.Warn("screensaver: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectScreensaverSnapshot, data); err != nil {
			slog.Warn("screensaver: publish snapshot", "err", err)
		}
	}
}

// screensaverResponse is the shared reply shape for every command
// except the plain snapshot fetch.
type screensaverResponse struct {
	Snapshot screensaversvc.Snapshot `json:"snapshot"`
	Error    string                  `json:"error,omitempty"`
}

// respondScreensaver replies with just an error string — used for
// malformed requests where there's no snapshot to attach.
func respondScreensaver(m *nats.Msg, errText string) {
	resp := screensaverResponse{Error: errText}
	data, _ := json.Marshal(resp)
	respond(m, data)
}
