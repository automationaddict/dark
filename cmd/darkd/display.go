package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	displaysvc "github.com/automationaddict/dark/internal/services/display"
)

type displayActionRequest struct {
	Name        string  `json:"name"`
	Width       int     `json:"width,omitempty"`
	Height      int     `json:"height,omitempty"`
	RefreshRate float64 `json:"refresh_rate,omitempty"`
	Scale       float64 `json:"scale,omitempty"`
	Transform   int     `json:"transform,omitempty"`
	X           int     `json:"x,omitempty"`
	Y           int     `json:"y,omitempty"`
	On          *bool   `json:"on,omitempty"`
	Mode        int     `json:"mode,omitempty"`
	MirrorOf    string  `json:"mirror_of,omitempty"`
	Pct         int     `json:"pct,omitempty"`
	Enable      *bool   `json:"enable,omitempty"`
	Temperature int     `json:"temperature,omitempty"`
	Gamma       int     `json:"gamma,omitempty"`
	Profile     string  `json:"profile,omitempty"`
	GPUMode     string  `json:"gpu_mode,omitempty"`
}

type displayActionResponse struct {
	Snapshot displaysvc.Snapshot `json:"snapshot"`
	Error    string              `json:"error,omitempty"`
}

func wireDisplay(nc *nats.Conn, svc *displaysvc.Service, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectDisplayMonitorsCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(svc.Snapshot())
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDisplayMonitorsCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(*displaysvc.Service, displayActionRequest) displayActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req displayActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := displayActionResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				respond(m, data)
				return
			}
			resp := handler(svc, req)
			data, _ := json.Marshal(resp)
			if err := m.Respond(data); err != nil {
				dn.Error("Displays", "failed to send response: "+err.Error())
			}
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				if err := nc.Publish(bus.SubjectDisplayMonitors, snapData); err != nil {
					dn.Error("Displays", "failed to publish snapshot: "+err.Error())
				}
			}
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	register(bus.SubjectDisplayResolutionCmd, handleDisplayResolution)
	register(bus.SubjectDisplayScaleCmd, handleDisplayScale)
	register(bus.SubjectDisplayTransformCmd, handleDisplayTransform)
	register(bus.SubjectDisplayPositionCmd, handleDisplayPosition)
	register(bus.SubjectDisplayDpmsCmd, handleDisplayDpms)
	register(bus.SubjectDisplayVrrCmd, handleDisplayVrr)
	register(bus.SubjectDisplayMirrorCmd, handleDisplayMirror)
	register(bus.SubjectDisplayToggleCmd, handleDisplayToggle)
	register(bus.SubjectDisplayBrightnessCmd, handleDisplayBrightness)
	register(bus.SubjectDisplayKbdBrightnessCmd, handleDisplayKbdBrightness)
	register(bus.SubjectDisplayNightLightCmd, handleDisplayNightLight)
	register(bus.SubjectDisplayGammaCmd, handleDisplayGamma)
	register(bus.SubjectDisplaySaveProfileCmd, handleDisplaySaveProfile)
	register(bus.SubjectDisplayApplyProfileCmd, handleDisplayApplyProfile)
	register(bus.SubjectDisplayDeleteProfileCmd, handleDisplayDeleteProfile)

	// GPU mode toggle uses dark-helper for privileged config changes.
	helperPath := findHelperPath()
	if _, err := nc.Subscribe(bus.SubjectDisplayGPUModeCmd, func(m *nats.Msg) {
		var req displayActionRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			resp := displayActionResponse{Error: "malformed request"}
			data, _ := json.Marshal(resp)
			respond(m, data)
			return
		}
		var resp displayActionResponse
		cmd := exec.Command("pkexec", helperPath, "gpu-hybrid", req.GPUMode)
		if out, err := cmd.CombinedOutput(); err != nil {
			resp.Error = strings.TrimSpace(string(out))
			if resp.Error == "" {
				resp.Error = err.Error()
			}
		}
		resp.Snapshot = svc.Snapshot()
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDisplayGPUModeCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectDisplayIdentifyCmd, func(m *nats.Msg) {
		var resp displayActionResponse
		if err := svc.Identify(); err != nil {
			resp.Error = err.Error()
		}
		resp.Snapshot = svc.Snapshot()
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectDisplayIdentifyCmd, "error", err)
		os.Exit(1)
	}

	publish := func() {
		data, err := json.Marshal(svc.Snapshot())
		if err != nil {
			dn.Error("Displays", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectDisplayMonitors, data); err != nil {
			dn.Error("Displays", "publish failed: "+err.Error())
		}
	}

	if events := svc.Events(); events != nil {
		go func() {
			debounce := core.DisplayEventDebounce
			var timer *time.Timer
			for range events {
				if timer == nil {
					timer = time.AfterFunc(debounce, publish)
				} else {
					timer.Reset(debounce)
				}
			}
		}()
	}

	return publish
}

// persistAndRespond builds a response with the fresh snapshot, then
// best-effort persists the named monitor's config to monitors.conf.
// Persistence failures are logged but don't fail the action — the
// runtime change already took effect via hyprctl.
func persistAndRespond(svc *displaysvc.Service, name string) displayActionResponse {
	snap := svc.Snapshot()
	for _, m := range snap.Monitors {
		if m.Name == name {
			if err := displaysvc.PersistMonitor(m); err != nil {
				slog.Warn("display persist failed", "monitor", name, "error", err)
			}
			break
		}
	}
	return displayActionResponse{Snapshot: snap}
}

func handleDisplayResolution(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if err := svc.SetResolution(req.Name, req.Width, req.Height, req.RefreshRate); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return persistAndRespond(svc, req.Name)
}

func handleDisplayScale(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if err := svc.SetScale(req.Name, req.Scale); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return persistAndRespond(svc, req.Name)
}

func handleDisplayTransform(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if err := svc.SetTransform(req.Name, req.Transform); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return persistAndRespond(svc, req.Name)
}

func handleDisplayPosition(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if err := svc.SetPosition(req.Name, req.X, req.Y); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return persistAndRespond(svc, req.Name)
}

// DPMS is runtime-only — no persistence.
func handleDisplayDpms(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if req.On == nil {
		return displayActionResponse{Error: "missing on flag"}
	}
	if err := svc.SetDpms(req.Name, *req.On); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisplayVrr(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if err := svc.SetVrr(req.Name, req.Mode); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return persistAndRespond(svc, req.Name)
}

func handleDisplayMirror(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if err := svc.SetMirror(req.Name, req.MirrorOf); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return persistAndRespond(svc, req.Name)
}

func handleDisplayToggle(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Name == "" {
		return displayActionResponse{Error: "missing monitor name"}
	}
	if err := svc.ToggleEnabled(req.Name); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return persistAndRespond(svc, req.Name)
}

func handleDisplayBrightness(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if err := svc.SetBrightness(req.Pct); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisplayKbdBrightness(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if err := svc.SetKbdBrightness(req.Pct); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisplayNightLight(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	enable := req.Enable != nil && *req.Enable
	temp := req.Temperature
	if temp == 0 {
		temp = 4500
	}
	gamma := req.Gamma
	if gamma == 0 {
		gamma = 100
	}
	if err := svc.SetNightLight(enable, temp, gamma); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisplayGamma(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if err := svc.SetGamma(req.Pct); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisplaySaveProfile(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Profile == "" {
		return displayActionResponse{Error: "missing profile name"}
	}
	if err := svc.SaveProfile(req.Profile); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisplayApplyProfile(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Profile == "" {
		return displayActionResponse{Error: "missing profile name"}
	}
	if err := svc.ApplyProfile(req.Profile); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisplayDeleteProfile(svc *displaysvc.Service, req displayActionRequest) displayActionResponse {
	if req.Profile == "" {
		return displayActionResponse{Error: "missing profile name"}
	}
	if err := svc.DeleteProfile(req.Profile); err != nil {
		return displayActionResponse{Error: err.Error()}
	}
	return displayActionResponse{Snapshot: svc.Snapshot()}
}
