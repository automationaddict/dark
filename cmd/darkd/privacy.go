package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/privacy"
)

type privacyRequest struct {
	Field   string `json:"field,omitempty"`
	Seconds int    `json:"seconds,omitempty"`
	Value   string `json:"value,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type privacyResponse struct {
	Snapshot privacy.Snapshot `json:"snapshot"`
	Error    string          `json:"error,omitempty"`
}

func wirePrivacy(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectPrivacySnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(privacy.ReadSnapshot())
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectPrivacySnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(privacyRequest) privacyResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req privacyRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := privacyResponse{Error: "malformed request: " + err.Error()}
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

	register(bus.SubjectPrivacyIdleCmd, func(req privacyRequest) privacyResponse {
		if req.Field == "" {
			return privacyResponse{Error: "missing field"}
		}
		if err := privacy.SetIdleTimeout(req.Field, req.Seconds); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacyDNSTLSCmd, func(req privacyRequest) privacyResponse {
		if req.Value == "" {
			return privacyResponse{Error: "missing value"}
		}
		if err := privacy.SetDNSOverTLS(req.Value); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacyDNSSECCmd, func(req privacyRequest) privacyResponse {
		if req.Value == "" {
			return privacyResponse{Error: "missing value"}
		}
		if err := privacy.SetDNSSEC(req.Value); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacyFirewallCmd, func(req privacyRequest) privacyResponse {
		enabled := req.Enabled != nil && *req.Enabled
		if err := privacy.SetFirewall(enabled); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacySSHCmd, func(req privacyRequest) privacyResponse {
		enabled := req.Enabled != nil && *req.Enabled
		if err := privacy.SetSSH(enabled); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacyLocationCmd, func(req privacyRequest) privacyResponse {
		enabled := req.Enabled != nil && *req.Enabled
		if err := privacy.SetLocation(enabled); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacyMACCmd, func(req privacyRequest) privacyResponse {
		if req.Value == "" {
			return privacyResponse{Error: "missing value"}
		}
		if err := privacy.SetMACRandomizationElevated(req.Value); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacyIndexerCmd, func(req privacyRequest) privacyResponse {
		enabled := req.Enabled != nil && *req.Enabled
		if err := privacy.SetIndexer(enabled); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	register(bus.SubjectPrivacyCoredumpCmd, func(req privacyRequest) privacyResponse {
		if req.Value == "" {
			return privacyResponse{Error: "missing value"}
		}
		if err := privacy.SetCoredumpStorage(req.Value); err != nil {
			return privacyResponse{Error: err.Error()}
		}
		return privacyResponse{Snapshot: privacy.ReadSnapshot()}
	})

	if _, err := nc.Subscribe(bus.SubjectPrivacyClearCmd, func(m *nats.Msg) {
		var resp privacyResponse
		if err := privacy.ClearRecentFiles(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = privacy.ReadSnapshot()
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectPrivacyClearCmd, "error", err)
		os.Exit(1)
	}

	publish := func() {
		data, err := json.Marshal(privacy.ReadSnapshot())
		if err != nil {
			dn.Error("Privacy", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectPrivacySnapshot, data); err != nil {
			dn.Error("Privacy", "publish failed: "+err.Error())
		}
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			publish()
		}
	}()

	return publish
}
