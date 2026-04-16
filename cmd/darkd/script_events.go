package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/scripting"
)

// registerScriptActions walks the bus command catalog and installs a
// first-class Lua wrapper for every command. Must run before user
// scripts load so their top-level code can reference dark.actions
// without hitting nil lookups.
func registerScriptActions(engine *scripting.Engine) {
	if engine == nil {
		return
	}
	for _, c := range bus.APICommandCatalog() {
		fields := make([]scripting.RegistryField, 0, len(c.Fields))
		for _, f := range c.Fields {
			fields = append(fields, scripting.RegistryField{
				Name:     f.Name,
				Type:     f.Type,
				Required: f.Required,
				Desc:     f.Desc,
			})
		}
		engine.RegisterAction(c.Subject, c.Summary, fields)
	}
}

// wireScriptClientEvents bridges client-side UI/lifecycle publishes
// into the Lua engine's event dispatcher. The dark TUI publishes on
// `dark.client.<event>` whenever something user-visible happens
// (startup, tab switch, section change); this subscription forwards
// each publish to Engine.DispatchEvent so scripts can
// `dark.on("on_f1", ...)` and react without needing to reach into
// the client process.
func wireScriptClientEvents(nc *nats.Conn, engine *scripting.Engine) {
	if engine == nil || nc == nil {
		return
	}
	_, err := nc.Subscribe("dark.client.>", func(m *nats.Msg) {
		event := strings.TrimPrefix(m.Subject, "dark.client.")
		if event == "" {
			return
		}
		var payload map[string]interface{}
		if len(m.Data) > 0 {
			_ = json.Unmarshal(m.Data, &payload)
		}
		engine.DispatchEvent(event, payload)
	})
	if err != nil {
		slog.Error("subscribe failed", "subject", "dark.client.>", "error", err)
		os.Exit(1)
	}
}

// wireScriptEvents plumbs every F1 service's snapshot publish through
// to the Lua engine's event dispatcher. Scripts that registered a
// hook via `dark.on("on_wifi", ...)` see a fresh call with the full
// snapshot on every publish. The subscriptions live on the daemon's
// own NATS connection, so events fire even when no client is
// connected — hooks run as soon as a snapshot is available.
//
// High-frequency subjects (audio level meters, daemon heartbeat)
// are intentionally excluded: dispatching a Lua hook at 20 Hz would
// starve the engine mutex.
func wireScriptEvents(nc *nats.Conn, engine *scripting.Engine) {
	if engine == nil || nc == nil {
		return
	}
	bindings := []struct {
		subject string
		event   string
	}{
		{bus.SubjectSystemInfo, "on_sysinfo"},
		{bus.SubjectWifiAdapters, "on_wifi"},
		{bus.SubjectBluetoothAdapters, "on_bluetooth"},
		{bus.SubjectAudioDevices, "on_audio"},
		{bus.SubjectNetworkSnapshot, "on_network"},
		{bus.SubjectDisplayMonitors, "on_display"},
		{bus.SubjectDateTimeSnapshot, "on_datetime"},
		{bus.SubjectNotifySnapshot, "on_notify"},
		{bus.SubjectInputSnapshot, "on_input"},
		{bus.SubjectPowerSnapshot, "on_power"},
		{bus.SubjectPrivacySnapshot, "on_privacy"},
		{bus.SubjectUsersSnapshot, "on_users"},
		{bus.SubjectAppearanceSnapshot, "on_appearance"},
		{bus.SubjectWorkspacesSnapshot, "on_workspaces"},
		{bus.SubjectSSHSnapshot, "on_ssh"},
	}
	for _, b := range bindings {
		subject, event := b.subject, b.event
		_, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var payload map[string]interface{}
			if err := json.Unmarshal(m.Data, &payload); err != nil {
				return
			}
			engine.DispatchEvent(event, payload)
		})
		if err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}
}
