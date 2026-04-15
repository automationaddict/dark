package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	wssvc "github.com/johnnelson/dark/internal/services/workspaces"
)

// wireWorkspaces installs handlers for every workspace command
// subject and returns a publish closure the main loop fires on
// the workspace tick. Every mutation re-reads the snapshot and
// publishes so the TUI's cached state catches up without waiting
// for the next tick.
func wireWorkspaces(nc *nats.Conn) func() {
	if _, err := nc.Subscribe(bus.SubjectWorkspacesSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(wssvc.ReadSnapshot())
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectWorkspacesSnapshotCmd, "error", err)
		os.Exit(1)
	}

	// handleAction wraps the decode + mutate + publish dance so
	// every mutating subject shares the shape. Each per-subject
	// closure only writes the mutate function.
	handleAction := func(subject string, mutate func(m *nats.Msg) error) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var resp workspacesResponse
			if err := mutate(m); err != nil {
				resp.Error = err.Error()
			}
			// Hyprland needs a beat between a keyword write and
			// the follow-up getoption read or the snapshot can
			// race the updated value.
			time.Sleep(75 * time.Millisecond)
			resp.Snapshot = wssvc.ReadSnapshot()
			data, _ := json.Marshal(resp)
			respond(m, data)
			snapData, _ := json.Marshal(resp.Snapshot)
			publish(nc, bus.SubjectWorkspacesSnapshot, snapData)
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	handleAction(bus.SubjectWorkspacesSwitchCmd, func(m *nats.Msg) error {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SwitchWorkspace(req.ID)
	})

	handleAction(bus.SubjectWorkspacesRenameCmd, func(m *nats.Msg) error {
		var req struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.RenameWorkspace(req.ID, req.Name)
	})

	handleAction(bus.SubjectWorkspacesMoveToMonitorCmd, func(m *nats.Msg) error {
		var req struct {
			ID      int    `json:"id"`
			Monitor string `json:"monitor"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.MoveWorkspaceToMonitor(req.ID, req.Monitor)
	})

	handleAction(bus.SubjectWorkspacesSetLayoutCmd, func(m *nats.Msg) error {
		var req struct {
			ID     int    `json:"id"`
			Layout string `json:"layout"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SetWorkspaceLayout(req.ID, req.Layout)
	})

	handleAction(bus.SubjectWorkspacesSetDefaultLayoutCmd, func(m *nats.Msg) error {
		var req struct {
			Layout string `json:"layout"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SetDefaultLayout(req.Layout)
	})

	handleAction(bus.SubjectWorkspacesSetDwindleOptionCmd, func(m *nats.Msg) error {
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SetDwindleOption(req.Key, req.Value)
	})

	handleAction(bus.SubjectWorkspacesSetMasterOptionCmd, func(m *nats.Msg) error {
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SetMasterOption(req.Key, req.Value)
	})

	handleAction(bus.SubjectWorkspacesSetCursorWarpCmd, func(m *nats.Msg) error {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SetCursorWarp(req.Enabled)
	})

	handleAction(bus.SubjectWorkspacesSetAnimationsCmd, func(m *nats.Msg) error {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SetAnimationsEnabled(req.Enabled)
	})

	handleAction(bus.SubjectWorkspacesSetHideSpecialCmd, func(m *nats.Msg) error {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			return err
		}
		return wssvc.SetHideSpecialOnChange(req.Enabled)
	})

	return func() {
		snap := wssvc.ReadSnapshot()
		data, err := json.Marshal(snap)
		if err != nil {
			slog.Warn("workspaces: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectWorkspacesSnapshot, data); err != nil {
			slog.Warn("workspaces: publish snapshot", "err", err)
		}
	}
}

// workspacesResponse is the shared reply shape for every workspace
// action command. Error carries any failure; Snapshot carries the
// post-mutation state so the TUI can update its cached copy.
type workspacesResponse struct {
	Snapshot wssvc.Snapshot `json:"snapshot"`
	Error    string         `json:"error,omitempty"`
}
