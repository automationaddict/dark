package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/appearance"
)

type appearanceActionRequest struct {
	Value   int    `json:"value,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
	Theme   string `json:"theme,omitempty"`
	Font    string `json:"font,omitempty"`
}

type appearanceActionResponse struct {
	Snapshot appearance.Snapshot `json:"snapshot"`
	Error    string              `json:"error,omitempty"`
}

func wireAppearance(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectAppearanceSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(appearance.ReadSnapshot())
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectAppearanceSnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(appearanceActionRequest) appearanceActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req appearanceActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := appearanceActionResponse{Error: "malformed request: " + err.Error()}
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

	register(bus.SubjectAppearanceGapsInCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetGapsIn(req.Value); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceGapsOutCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetGapsOut(req.Value); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceBorderCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetBorderSize(req.Value); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceRoundingCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetRounding(req.Value); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceBlurCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetBlurEnabled(req.Enabled); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceBlurSizeCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetBlurSize(req.Value); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceBlurPassCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetBlurPasses(req.Value); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceAnimCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetAnimEnabled(req.Enabled); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceThemeCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if req.Theme == "" {
			return appearanceActionResponse{Error: "missing theme name"}
		}
		if err := appearance.SetTheme(req.Theme); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceFontCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if req.Font == "" {
			return appearanceActionResponse{Error: "missing font name"}
		}
		if err := appearance.SetFont(req.Font); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	register(bus.SubjectAppearanceFontSizeCmd, func(req appearanceActionRequest) appearanceActionResponse {
		if err := appearance.SetFontSize(req.Value); err != nil {
			return appearanceActionResponse{Error: err.Error()}
		}
		return appearanceActionResponse{Snapshot: appearance.ReadSnapshot()}
	})

	publish := func() {
		data, err := json.Marshal(appearance.ReadSnapshot())
		if err != nil {
			dn.Error("Appearance", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectAppearanceSnapshot, data); err != nil {
			dn.Error("Appearance", "publish failed: "+err.Error())
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
