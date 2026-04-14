package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/keybind"
)

type keybindRequest struct {
	Mods       string `json:"mods,omitempty"`
	Key        string `json:"key,omitempty"`
	Desc       string `json:"desc,omitempty"`
	Dispatcher string `json:"dispatcher,omitempty"`
	Args       string `json:"args,omitempty"`
	Source     string `json:"source,omitempty"`
	Category   string `json:"category,omitempty"`
	BindType   string `json:"bind_type,omitempty"`

	// For update: the original binding fields.
	OldMods       string `json:"old_mods,omitempty"`
	OldKey        string `json:"old_key,omitempty"`
	OldDesc       string `json:"old_desc,omitempty"`
	OldDispatcher string `json:"old_dispatcher,omitempty"`
	OldArgs       string `json:"old_args,omitempty"`
	OldSource     string `json:"old_source,omitempty"`
	OldCategory   string `json:"old_category,omitempty"`
	OldBindType   string `json:"old_bind_type,omitempty"`
}

type keybindResponse struct {
	Snapshot keybind.Snapshot `json:"snapshot"`
	Error    string          `json:"error,omitempty"`
}

func wireKeybind(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectKeybindSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(keybind.ReadSnapshot())
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectKeybindSnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(keybindRequest) keybindResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req keybindRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := keybindResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				respond(m, data)
				return
			}
			resp := handler(req)
			data, _ := json.Marshal(resp)
			respond(m, data)
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	register(bus.SubjectKeybindAddCmd, func(req keybindRequest) keybindResponse {
		b := bindingFromRequest(req)
		if err := keybind.AddBinding(b); err != nil {
			return keybindResponse{Error: err.Error()}
		}
		return keybindResponse{Snapshot: keybind.ReadSnapshot()}
	})

	register(bus.SubjectKeybindUpdateCmd, func(req keybindRequest) keybindResponse {
		old := keybind.Binding{
			Mods:       req.OldMods,
			Key:        req.OldKey,
			Desc:       req.OldDesc,
			Dispatcher: req.OldDispatcher,
			Args:       req.OldArgs,
			Source:     keybind.Source(req.OldSource),
			Category:   req.OldCategory,
			BindType:   req.OldBindType,
		}
		new := bindingFromRequest(req)
		if err := keybind.UpdateBinding(old, new); err != nil {
			return keybindResponse{Error: err.Error()}
		}
		return keybindResponse{Snapshot: keybind.ReadSnapshot()}
	})

	register(bus.SubjectKeybindRemoveCmd, func(req keybindRequest) keybindResponse {
		b := bindingFromRequest(req)
		if err := keybind.RemoveBinding(b); err != nil {
			return keybindResponse{Error: err.Error()}
		}
		return keybindResponse{Snapshot: keybind.ReadSnapshot()}
	})

	publish := func() {
		data, err := json.Marshal(keybind.ReadSnapshot())
		if err != nil {
			dn.Error("Keybindings", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectKeybindSnapshot, data); err != nil {
			dn.Error("Keybindings", "publish failed: "+err.Error())
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

func bindingFromRequest(req keybindRequest) keybind.Binding {
	return keybind.Binding{
		Mods:       req.Mods,
		Key:        req.Key,
		Desc:       req.Desc,
		Dispatcher: req.Dispatcher,
		Args:       req.Args,
		Source:     keybind.Source(req.Source),
		Category:   req.Category,
		BindType:   req.BindType,
	}
}
