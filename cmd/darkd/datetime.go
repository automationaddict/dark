package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/datetime"
)

type dateTimeRequest struct {
	Timezone string `json:"timezone,omitempty"`
	Enabled  *bool  `json:"enabled,omitempty"`
	Time     string `json:"time,omitempty"`
	Local    *bool  `json:"local,omitempty"`
}

type dateTimeResponse struct {
	Snapshot datetime.Snapshot `json:"snapshot"`
	Error    string            `json:"error,omitempty"`
}

func wireDateTime(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectDateTimeSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(datetime.ReadSnapshot())
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDateTimeSnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(dateTimeRequest) dateTimeResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req dateTimeRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := dateTimeResponse{Error: "malformed request: " + err.Error()}
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

	register(bus.SubjectDateTimeTZCmd, func(req dateTimeRequest) dateTimeResponse {
		if req.Timezone == "" {
			return dateTimeResponse{Error: "missing timezone"}
		}
		if err := datetime.SetTimezone(req.Timezone); err != nil {
			return dateTimeResponse{Error: err.Error()}
		}
		return dateTimeResponse{Snapshot: datetime.ReadSnapshot()}
	})

	register(bus.SubjectDateTimeNTPCmd, func(req dateTimeRequest) dateTimeResponse {
		enabled := req.Enabled != nil && *req.Enabled
		if err := datetime.SetNTP(enabled); err != nil {
			return dateTimeResponse{Error: err.Error()}
		}
		return dateTimeResponse{Snapshot: datetime.ReadSnapshot()}
	})

	register(bus.SubjectDateTimeSetTimeCmd, func(req dateTimeRequest) dateTimeResponse {
		if req.Time == "" {
			return dateTimeResponse{Error: "missing time"}
		}
		if err := datetime.SetTime(req.Time); err != nil {
			return dateTimeResponse{Error: err.Error()}
		}
		return dateTimeResponse{Snapshot: datetime.ReadSnapshot()}
	})

	register(bus.SubjectDateTimeRTCCmd, func(req dateTimeRequest) dateTimeResponse {
		local := req.Local != nil && *req.Local
		if err := datetime.SetLocalRTC(local); err != nil {
			return dateTimeResponse{Error: err.Error()}
		}
		return dateTimeResponse{Snapshot: datetime.ReadSnapshot()}
	})

	if _, err := nc.Subscribe(bus.SubjectDateTimeFormatCmd, func(m *nats.Msg) {
		var resp dateTimeResponse
		if err := datetime.ToggleClockFormat(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = datetime.ReadSnapshot()
		data, _ := json.Marshal(resp)
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDateTimeFormatCmd, "error", err)
		os.Exit(1)
	}

	publish := func() {
		data, err := json.Marshal(datetime.ReadSnapshot())
		if err != nil {
			dn.Error("Date & Time", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectDateTimeSnapshot, data); err != nil {
			dn.Error("Date & Time", "publish failed: "+err.Error())
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
