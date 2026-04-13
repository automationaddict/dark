package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/power"
)

type powerActionRequest struct {
	Profile   string `json:"profile,omitempty"`
	Governor  string `json:"governor,omitempty"`
	EPP       string `json:"epp,omitempty"`
	IdleKind  string `json:"idle_kind,omitempty"`
	IdleSec   int    `json:"idle_sec,omitempty"`
	ButtonKey string `json:"button_key,omitempty"`
	ButtonVal string `json:"button_val,omitempty"`
}

type powerActionResponse struct {
	Snapshot power.Snapshot `json:"snapshot"`
	Error    string         `json:"error,omitempty"`
}

func wirePower(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectPowerSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(power.ReadSnapshot())
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectPowerSnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(powerActionRequest) powerActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req powerActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := powerActionResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				_ = m.Respond(data)
				return
			}
			resp := handler(req)
			data, _ := json.Marshal(resp)
			_ = m.Respond(data)
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	register(bus.SubjectPowerProfileCmd, func(req powerActionRequest) powerActionResponse {
		if req.Profile == "" {
			return powerActionResponse{Error: "missing profile"}
		}
		if err := power.SetProfile(req.Profile); err != nil {
			return powerActionResponse{Error: err.Error()}
		}
		return powerActionResponse{Snapshot: power.ReadSnapshot()}
	})

	register(bus.SubjectPowerGovernorCmd, func(req powerActionRequest) powerActionResponse {
		if req.Governor == "" {
			return powerActionResponse{Error: "missing governor"}
		}
		if err := power.SetGovernor(req.Governor); err != nil {
			return powerActionResponse{Error: err.Error()}
		}
		return powerActionResponse{Snapshot: power.ReadSnapshot()}
	})

	register(bus.SubjectPowerEPPCmd, func(req powerActionRequest) powerActionResponse {
		if req.EPP == "" {
			return powerActionResponse{Error: "missing epp"}
		}
		if err := power.SetEPP(req.EPP); err != nil {
			return powerActionResponse{Error: err.Error()}
		}
		return powerActionResponse{Snapshot: power.ReadSnapshot()}
	})

	register(bus.SubjectPowerIdleCmd, func(req powerActionRequest) powerActionResponse {
		if req.IdleKind == "" {
			return powerActionResponse{Error: "missing idle_kind"}
		}
		if req.IdleSec < 0 {
			return powerActionResponse{Error: "idle_sec must be non-negative"}
		}
		if err := power.SetIdleTimeout(req.IdleKind, req.IdleSec); err != nil {
			return powerActionResponse{Error: err.Error()}
		}
		return powerActionResponse{Snapshot: power.ReadSnapshot()}
	})

	register(bus.SubjectPowerButtonCmd, func(req powerActionRequest) powerActionResponse {
		if req.ButtonKey == "" || req.ButtonVal == "" {
			return powerActionResponse{Error: "missing button_key or button_val"}
		}
		if err := power.SetSystemButton(req.ButtonKey, req.ButtonVal); err != nil {
			return powerActionResponse{Error: err.Error()}
		}
		return powerActionResponse{Snapshot: power.ReadSnapshot()}
	})

	publish := func() {
		data, err := json.Marshal(power.ReadSnapshot())
		if err != nil {
			dn.Error("Power", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectPowerSnapshot, data); err != nil {
			dn.Error("Power", "publish failed: "+err.Error())
		}
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			publish()
		}
	}()

	return publish
}
