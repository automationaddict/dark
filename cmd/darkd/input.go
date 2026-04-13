package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/input"
)

type inputActionRequest struct {
	Rate    int     `json:"rate,omitempty"`
	Delay   int     `json:"delay,omitempty"`
	Sens    float64 `json:"sens,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	Factor  float64 `json:"factor,omitempty"`
	Layout  string  `json:"layout,omitempty"`
	Profile string  `json:"profile,omitempty"`
}

type inputActionResponse struct {
	Snapshot input.Snapshot `json:"snapshot"`
	Error    string         `json:"error,omitempty"`
}

func wireInput(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectInputSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(input.ReadSnapshot())
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectInputSnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(inputActionRequest) inputActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req inputActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := inputActionResponse{Error: "malformed request: " + err.Error()}
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

	register(bus.SubjectInputRepeatRateCmd, func(req inputActionRequest) inputActionResponse {
		if err := input.SetRepeatRate(req.Rate); err != nil {
			return inputActionResponse{Error: err.Error()}
		}
		return inputActionResponse{Snapshot: input.ReadSnapshot()}
	})

	register(bus.SubjectInputRepeatDelayCmd, func(req inputActionRequest) inputActionResponse {
		if err := input.SetRepeatDelay(req.Delay); err != nil {
			return inputActionResponse{Error: err.Error()}
		}
		return inputActionResponse{Snapshot: input.ReadSnapshot()}
	})

	register(bus.SubjectInputSensitivityCmd, func(req inputActionRequest) inputActionResponse {
		if err := input.SetSensitivity(req.Sens); err != nil {
			return inputActionResponse{Error: err.Error()}
		}
		return inputActionResponse{Snapshot: input.ReadSnapshot()}
	})

	register(bus.SubjectInputNatScrollCmd, func(req inputActionRequest) inputActionResponse {
		enabled := req.Enabled != nil && *req.Enabled
		if err := input.SetNaturalScroll(enabled); err != nil {
			return inputActionResponse{Error: err.Error()}
		}
		return inputActionResponse{Snapshot: input.ReadSnapshot()}
	})

	register(bus.SubjectInputScrollFactorCmd, func(req inputActionRequest) inputActionResponse {
		if err := input.SetScrollFactor(req.Factor); err != nil {
			return inputActionResponse{Error: err.Error()}
		}
		return inputActionResponse{Snapshot: input.ReadSnapshot()}
	})

	register(bus.SubjectInputKBLayoutCmd, func(req inputActionRequest) inputActionResponse {
		if req.Layout == "" {
			return inputActionResponse{Error: "missing layout"}
		}
		if err := input.SetKBLayout(req.Layout); err != nil {
			return inputActionResponse{Error: err.Error()}
		}
		return inputActionResponse{Snapshot: input.ReadSnapshot()}
	})

	register(bus.SubjectInputAccelProfileCmd, func(req inputActionRequest) inputActionResponse {
		if err := input.SetAccelProfile(req.Profile); err != nil {
			return inputActionResponse{Error: err.Error()}
		}
		return inputActionResponse{Snapshot: input.ReadSnapshot()}
	})

	registerBool := func(subject string, setter func(bool) error) {
		register(subject, func(req inputActionRequest) inputActionResponse {
			enabled := req.Enabled != nil && *req.Enabled
			if err := setter(enabled); err != nil {
				return inputActionResponse{Error: err.Error()}
			}
			return inputActionResponse{Snapshot: input.ReadSnapshot()}
		})
	}

	registerBool(bus.SubjectInputForceNoAccelCmd, input.SetForceNoAccel)
	registerBool(bus.SubjectInputLeftHandedCmd, input.SetLeftHanded)
	registerBool(bus.SubjectInputDisableTypingCmd, input.SetDisableWhileTyping)
	registerBool(bus.SubjectInputTapToClickCmd, input.SetTapToClick)
	registerBool(bus.SubjectInputTapAndDragCmd, input.SetTapAndDrag)
	registerBool(bus.SubjectInputDragLockCmd, input.SetDragLock)
	registerBool(bus.SubjectInputMiddleBtnCmd, input.SetMiddleButtonEmu)
	registerBool(bus.SubjectInputClickfingerCmd, input.SetClickfingerBehavior)

	publish := func() {
		data, err := json.Marshal(input.ReadSnapshot())
		if err != nil {
			dn.Error("Input", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectInputSnapshot, data); err != nil {
			dn.Error("Input", "publish failed: "+err.Error())
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
