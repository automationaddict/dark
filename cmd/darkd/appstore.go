package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	appstoresvc "github.com/johnnelson/dark/internal/services/appstore"
)

// appstoreLogger returns a child of the global slog default with the
// "service" field set to "appstore". The log level is controlled by
// DARK_LOG_LEVEL (the same env var as the rest of the daemon) — the
// old DARK_APPSTORE_LOG override is no longer needed since everything
// shares one handler now.
func appstoreLogger() *slog.Logger {
	return slog.Default().With("service", "appstore")
}

// wireAppstore registers the five NATS handlers for the appstore
// domain and returns the publisher closure the main loop's slow
// ticker calls. The handler shape mirrors wireBluetooth: each
// request/reply subscription decodes its request, delegates to the
// service, encodes a response, and (for mutating calls, which in
// phase 1 means Refresh only) republishes the refreshed snapshot on
// the catalog event subject.
func wireAppstore(nc *nats.Conn, svc *appstoresvc.Service, logger *slog.Logger, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectAppstoreCatalogCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(svc.Snapshot())
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectAppstoreCatalogCmd, "error", err); os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreSearchCmd, func(m *nats.Msg) {
		var req appstoresvc.SearchQuery
		if err := json.Unmarshal(m.Data, &req); err != nil {
			resp := appstoreSearchResponse{Error: "malformed search request: " + err.Error()}
			data, _ := json.Marshal(resp)
			_ = m.Respond(data)
			return
		}
		res, err := svc.Search(req)
		resp := appstoreSearchResponse{Result: res}
		if err != nil {
			resp.Error = err.Error()
		}
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectAppstoreSearchCmd, "error", err); os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreDetailCmd, func(m *nats.Msg) {
		var req appstoresvc.DetailRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			resp := appstoreDetailResponse{Error: "malformed detail request: " + err.Error()}
			data, _ := json.Marshal(resp)
			_ = m.Respond(data)
			return
		}
		detail, err := svc.Detail(req)
		resp := appstoreDetailResponse{Detail: detail}
		if err != nil {
			resp.Error = err.Error()
		}
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectAppstoreDetailCmd, "error", err); os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreRefreshCmd, func(m *nats.Msg) {
		var resp appstoreRefreshResponse
		if err := svc.Refresh(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = svc.Snapshot()
		data, _ := json.Marshal(resp)
		if err := m.Respond(data); err != nil {
			dn.Error("App Store", "failed to send response: "+err.Error())
		}
		if resp.Error == "" {
			snapData, _ := json.Marshal(resp.Snapshot)
			if err := nc.Publish(bus.SubjectAppstoreCatalog, snapData); err != nil {
				dn.Error("App Store", "failed to publish catalog: "+err.Error())
			}
		}
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectAppstoreRefreshCmd, "error", err); os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreInstallCmd, func(m *nats.Msg) {
		var req appstoresvc.InstallRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			logger.Warn("appstore: malformed install request", "err", err)
		}
		out, err := svc.Install(req)
		resp := appstoreActionResponse{Output: out}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Snapshot = svc.Snapshot()
			snapData, _ := json.Marshal(resp.Snapshot)
			_ = nc.Publish(bus.SubjectAppstoreCatalog, snapData)
		}
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe appstore install cmd: %v", err)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreRemoveCmd, func(m *nats.Msg) {
		var req struct {
			Names []string `json:"names"`
		}
		if err := json.Unmarshal(m.Data, &req); err != nil {
			logger.Warn("appstore: malformed remove request", "err", err)
		}
		out, err := svc.Remove(req.Names)
		resp := appstoreActionResponse{Output: out}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Snapshot = svc.Snapshot()
			snapData, _ := json.Marshal(resp.Snapshot)
			_ = nc.Publish(bus.SubjectAppstoreCatalog, snapData)
		}
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe appstore remove cmd: %v", err)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreUpgradeCmd, func(m *nats.Msg) {
		out, err := svc.Upgrade()
		resp := appstoreActionResponse{Output: out}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Snapshot = svc.Snapshot()
			snapData, _ := json.Marshal(resp.Snapshot)
			_ = nc.Publish(bus.SubjectAppstoreCatalog, snapData)
		}
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe appstore upgrade cmd: %v", err)
	}

	return func() {
		data, err := json.Marshal(svc.Snapshot())
		if err != nil {
			logger.Warn("appstore: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectAppstoreCatalog, data); err != nil {
			logger.Warn("appstore: publish snapshot", "err", err)
		}
	}
}

// appstoreSearchResponse wraps a SearchResult with an optional error
// string so TUI clients can surface a failed search cleanly without
// NATS-level error plumbing.
type appstoreSearchResponse struct {
	Result appstoresvc.SearchResult `json:"result"`
	Error  string                   `json:"error,omitempty"`
}

type appstoreDetailResponse struct {
	Detail appstoresvc.Detail `json:"detail"`
	Error  string             `json:"error,omitempty"`
}

type appstoreRefreshResponse struct {
	Snapshot appstoresvc.Snapshot `json:"snapshot"`
	Error    string               `json:"error,omitempty"`
}

type appstoreActionResponse struct {
	Snapshot appstoresvc.Snapshot `json:"snapshot,omitempty"`
	Output   string               `json:"output,omitempty"`
	Error    string               `json:"error,omitempty"`
}
