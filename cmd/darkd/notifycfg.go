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

type notifyCfgRequest struct {
	Anchor   string `json:"anchor,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
	AppName  string `json:"app_name,omitempty"`
	Hide     *bool  `json:"hide,omitempty"`
	Criteria string `json:"criteria,omitempty"`
}

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

	simple := func(subject string, action func() error) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var resp notifyCfgResponse
			if err := action(); err != nil {
				resp.Error = err.Error()
			}
			resp.Snapshot = notifycfg.ReadSnapshot()
			data, _ := json.Marshal(resp)
			_ = m.Respond(data)
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	simple(bus.SubjectNotifyDNDCmd, notifycfg.ToggleDND)
	simple(bus.SubjectNotifyDismissCmd, notifycfg.DismissAll)

	register := func(subject string, handler func(notifyCfgRequest) notifyCfgResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req notifyCfgRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := notifyCfgResponse{Error: "malformed request: " + err.Error()}
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

	register(bus.SubjectNotifyAnchorCmd, func(req notifyCfgRequest) notifyCfgResponse {
		if req.Anchor == "" {
			return notifyCfgResponse{Error: "missing anchor"}
		}
		if err := notifycfg.SetAnchor(req.Anchor); err != nil {
			return notifyCfgResponse{Error: err.Error()}
		}
		return notifyCfgResponse{Snapshot: notifycfg.ReadSnapshot()}
	})

	register(bus.SubjectNotifyTimeoutCmd, func(req notifyCfgRequest) notifyCfgResponse {
		if err := notifycfg.SetTimeout(req.Timeout); err != nil {
			return notifyCfgResponse{Error: err.Error()}
		}
		return notifyCfgResponse{Snapshot: notifycfg.ReadSnapshot()}
	})

	register(bus.SubjectNotifyAddRuleCmd, func(req notifyCfgRequest) notifyCfgResponse {
		if req.AppName == "" {
			return notifyCfgResponse{Error: "missing app name"}
		}
		hide := req.Hide != nil && *req.Hide
		if err := notifycfg.AddAppRule(req.AppName, hide); err != nil {
			return notifyCfgResponse{Error: err.Error()}
		}
		return notifyCfgResponse{Snapshot: notifycfg.ReadSnapshot()}
	})

	register(bus.SubjectNotifyRemoveRuleCmd, func(req notifyCfgRequest) notifyCfgResponse {
		if req.Criteria == "" {
			return notifyCfgResponse{Error: "missing criteria"}
		}
		if err := notifycfg.RemoveAppRule(req.Criteria); err != nil {
			return notifyCfgResponse{Error: err.Error()}
		}
		return notifyCfgResponse{Snapshot: notifycfg.ReadSnapshot()}
	})

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
