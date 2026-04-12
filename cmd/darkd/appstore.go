package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	appstoresvc "github.com/johnnelson/dark/internal/services/appstore"
)

// appstoreLogger builds the package-local slog logger the appstore
// service and its NATS handlers use. Logging for the rest of the daemon
// still flows through the global log.Printf; a separate migration task
// will consolidate them later. Keeping slog local here means the
// appstore code is unaffected by that migration when it lands.
func appstoreLogger() *slog.Logger {
	level := slog.LevelInfo
	if v := os.Getenv("DARK_APPSTORE_LOG"); v != "" {
		switch v {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})).With("component", "appstore")
}

// wireAppstore registers the five NATS handlers for the appstore
// domain and returns the publisher closure the main loop's slow
// ticker calls. The handler shape mirrors wireBluetooth: each
// request/reply subscription decodes its request, delegates to the
// service, encodes a response, and (for mutating calls, which in
// phase 1 means Refresh only) republishes the refreshed snapshot on
// the catalog event subject.
func wireAppstore(nc *nats.Conn, svc *appstoresvc.Service, logger *slog.Logger) func() {
	if _, err := nc.Subscribe(bus.SubjectAppstoreCatalogCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(svc.Snapshot())
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe appstore catalog cmd: %v", err)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreSearchCmd, func(m *nats.Msg) {
		var req appstoresvc.SearchQuery
		if err := json.Unmarshal(m.Data, &req); err != nil {
			logger.Warn("appstore: malformed search request", "err", err)
		}
		res, err := svc.Search(req)
		resp := appstoreSearchResponse{Result: res}
		if err != nil {
			resp.Error = err.Error()
		}
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe appstore search cmd: %v", err)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreDetailCmd, func(m *nats.Msg) {
		var req appstoresvc.DetailRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			logger.Warn("appstore: malformed detail request", "err", err)
		}
		detail, err := svc.Detail(req)
		resp := appstoreDetailResponse{Detail: detail}
		if err != nil {
			resp.Error = err.Error()
		}
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe appstore detail cmd: %v", err)
	}

	if _, err := nc.Subscribe(bus.SubjectAppstoreRefreshCmd, func(m *nats.Msg) {
		var resp appstoreRefreshResponse
		if err := svc.Refresh(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = svc.Snapshot()
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
		if resp.Error == "" {
			snapData, _ := json.Marshal(resp.Snapshot)
			_ = nc.Publish(bus.SubjectAppstoreCatalog, snapData)
		}
	}); err != nil {
		log.Fatalf("subscribe appstore refresh cmd: %v", err)
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
