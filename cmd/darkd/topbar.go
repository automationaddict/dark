package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	topbarsvc "github.com/automationaddict/dark/internal/services/topbar"
)

// wireTopBar installs the top bar command handlers on nc and returns
// a publish closure the main loop invokes for initial publish and on
// SIGHUP. Every mutation re-reads the snapshot and publishes it so
// the TUI's cached state stays fresh.
func wireTopBar(nc *nats.Conn) func() {
	if _, err := nc.Subscribe(bus.SubjectTopBarSnapshotCmd, func(m *nats.Msg) {
		snap := topbarsvc.ReadSnapshot()
		data, _ := json.Marshal(snap)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectTopBarSnapshotCmd, "error", err)
		os.Exit(1)
	}

	// Shared action handler: decode a minimal request, run the
	// mutation, read the new snapshot, reply, and publish. The per-
	// subject closures below just plug in their own mutation fn.
	handleAction := func(subject string, mutate func(m *nats.Msg) error) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var resp topBarResponse
			if err := mutate(m); err != nil {
				resp.Error = err.Error()
			}
			// Give waybar a beat to settle after process moves so
			// the snapshot reflects the new state rather than racing.
			time.Sleep(150 * time.Millisecond)
			resp.Snapshot = topbarsvc.ReadSnapshot()
			data, _ := json.Marshal(resp)
			respond(m, data)
			snapData, _ := json.Marshal(resp.Snapshot)
			publish(nc, bus.SubjectTopBarSnapshot, snapData)
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	handleAction(bus.SubjectTopBarSetRunningCmd, func(m *nats.Msg) error {
		var req struct {
			Running bool `json:"running"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return topbarsvc.SetRunning(req.Running)
	})

	handleAction(bus.SubjectTopBarRestartCmd, func(m *nats.Msg) error {
		return topbarsvc.Restart()
	})

	handleAction(bus.SubjectTopBarResetCmd, func(m *nats.Msg) error {
		return topbarsvc.Reset()
	})

	handleAction(bus.SubjectTopBarSetPositionCmd, func(m *nats.Msg) error {
		var req struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		if err := topbarsvc.SetPosition(req.Value); err != nil {
			return err
		}
		return topbarsvc.Restart()
	})

	handleAction(bus.SubjectTopBarSetLayerCmd, func(m *nats.Msg) error {
		var req struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		if err := topbarsvc.SetLayer(req.Value); err != nil {
			return err
		}
		return topbarsvc.Restart()
	})

	handleAction(bus.SubjectTopBarSetHeightCmd, func(m *nats.Msg) error {
		var req struct {
			Value int `json:"value"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		if err := topbarsvc.SetHeight(req.Value); err != nil {
			return err
		}
		return topbarsvc.Restart()
	})

	handleAction(bus.SubjectTopBarSetSpacingCmd, func(m *nats.Msg) error {
		var req struct {
			Value int `json:"value"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		if err := topbarsvc.SetSpacing(req.Value); err != nil {
			return err
		}
		return topbarsvc.Restart()
	})

	handleAction(bus.SubjectTopBarSetConfigCmd, func(m *nats.Msg) error {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return topbarsvc.SetConfig(req.Content)
	})

	handleAction(bus.SubjectTopBarSetStyleCmd, func(m *nats.Msg) error {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return topbarsvc.SetStyle(req.Content)
	})

	return func() {
		snap := topbarsvc.ReadSnapshot()
		data, err := json.Marshal(snap)
		if err != nil {
			slog.Warn("topbar: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectTopBarSnapshot, data); err != nil {
			slog.Warn("topbar: publish snapshot", "err", err)
		}
	}
}

// topBarResponse is the reply shape for every top bar command
// except the bare snapshot read. Error is non-empty on failure.
type topBarResponse struct {
	Snapshot topbarsvc.Snapshot `json:"snapshot"`
	Error    string             `json:"error,omitempty"`
}
