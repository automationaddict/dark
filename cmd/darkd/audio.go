package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/automationaddict/dark/internal/core"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	audiosvc "github.com/automationaddict/dark/internal/services/audio"
)

type audioActionRequest struct {
	Index       uint32 `json:"index,omitempty"`
	TargetIndex uint32 `json:"target_index,omitempty"`
	Volume      int    `json:"volume,omitempty"`
	Mute        *bool  `json:"mute,omitempty"`
	Suspend     *bool  `json:"suspend,omitempty"`
	Name        string `json:"name,omitempty"`
	Profile     string `json:"profile,omitempty"`
	Port        string `json:"port,omitempty"`
}

type audioActionResponse struct {
	Snapshot audiosvc.Snapshot `json:"snapshot"`
	Error    string            `json:"error,omitempty"`
}

// wireAudio registers all audio NATS handlers on nc and returns a
// publisher closure the ticker loop uses to push safety-net periodic
// snapshots. Reactive snapshot publishing (driven by Pulse subscription
// events) and the 20 Hz level meter publisher are spawned as their own
// goroutines from here so the main loop in main.go doesn't need to
// know about either pipeline.
func wireAudio(nc *nats.Conn, svc *audiosvc.Service, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectAudioDevicesCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotAudio(svc))
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectAudioDevicesCmd, "error", err); os.Exit(1)
	}

	register := func(subject string, handler func(*audiosvc.Service, audioActionRequest) audioActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req audioActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := audioActionResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				respond(m, data)
				return
			}
			resp := handler(svc, req)
			data, _ := json.Marshal(resp)
			if err := m.Respond(data); err != nil {
				dn.Error("Sound", "failed to send response: "+err.Error())
			}
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				if err := nc.Publish(bus.SubjectAudioDevices, snapData); err != nil {
					dn.Error("Sound", "failed to publish snapshot: "+err.Error())
				}
			}
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err); os.Exit(1)
		}
	}

	register(bus.SubjectAudioSinkVolumeCmd, handleAudioSinkVolume)
	register(bus.SubjectAudioSinkMuteCmd, handleAudioSinkMute)
	register(bus.SubjectAudioSinkBalanceCmd, handleAudioSinkBalance)
	register(bus.SubjectAudioSourceVolumeCmd, handleAudioSourceVolume)
	register(bus.SubjectAudioSourceMuteCmd, handleAudioSourceMute)
	register(bus.SubjectAudioSourceBalanceCmd, handleAudioSourceBalance)
	register(bus.SubjectAudioDefaultSinkCmd, handleAudioDefaultSink)
	register(bus.SubjectAudioDefaultSourceCmd, handleAudioDefaultSource)
	register(bus.SubjectAudioCardProfileCmd, handleAudioCardProfile)
	register(bus.SubjectAudioSinkPortCmd, handleAudioSinkPort)
	register(bus.SubjectAudioSourcePortCmd, handleAudioSourcePort)
	register(bus.SubjectAudioSinkInputVolumeCmd, handleAudioSinkInputVolume)
	register(bus.SubjectAudioSinkInputMuteCmd, handleAudioSinkInputMute)
	register(bus.SubjectAudioSinkInputMoveCmd, handleAudioSinkInputMove)
	register(bus.SubjectAudioSinkInputKillCmd, handleAudioSinkInputKill)
	register(bus.SubjectAudioSourceOutputVolumeCmd, handleAudioSourceOutputVolume)
	register(bus.SubjectAudioSourceOutputMuteCmd, handleAudioSourceOutputMute)
	register(bus.SubjectAudioSourceOutputMoveCmd, handleAudioSourceOutputMove)
	register(bus.SubjectAudioSourceOutputKillCmd, handleAudioSourceOutputKill)
	register(bus.SubjectAudioSuspendSinkCmd, handleAudioSuspendSink)
	register(bus.SubjectAudioSuspendSourceCmd, handleAudioSuspendSource)

	publish := func() {
		data, err := json.Marshal(snapshotAudio(svc))
		if err != nil {
			dn.Error("Sound", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectAudioDevices, data); err != nil {
			dn.Error("Sound", "publish failed: "+err.Error())
		}
	}

	// Reactive snapshot publisher: drains the Pulse subscription event
	// channel and republishes the snapshot whenever the server reports
	// a property change. Coalesces bursts of events into a single
	// republish via a short debounce so a flurry of related changes
	// (which is normal during connect/disconnect) doesn't pin the bus.
	if events := svc.Events(); events != nil {
		go func() {
			debounce := core.AudioEventDebounce
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

	// 20 Hz peak meter publisher: reads the current level snapshot
	// and broadcasts it on dark.audio.levels. Independent of the
	// device snapshot publisher so the meter ticks even when no
	// property changes are happening.
	go func() {
		ticker := time.NewTicker(core.AudioMeterTickRate)
		defer ticker.Stop()
		for range ticker.C {
			data, err := json.Marshal(svc.Levels())
			if err != nil {
				continue
			}
			if err := nc.Publish(bus.SubjectAudioLevels, data); err != nil {
				slog.Warn("nats publish failed", "subject", bus.SubjectAudioLevels, "error", err)
			}
		}
	}()

	return publish
}

func snapshotAudio(svc *audiosvc.Service) audiosvc.Snapshot {
	if svc != nil {
		return svc.Snapshot()
	}
	return audiosvc.Detect()
}

func handleAudioSinkVolume(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.SetSinkVolume(req.Index, req.Volume); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSinkMute(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Mute == nil {
		return audioActionResponse{Error: "missing mute flag"}
	}
	if err := svc.SetSinkMute(req.Index, *req.Mute); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSinkBalance(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.SetSinkBalance(req.Index, req.Volume); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourceBalance(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.SetSourceBalance(req.Index, req.Volume); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourceVolume(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.SetSourceVolume(req.Index, req.Volume); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourceMute(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Mute == nil {
		return audioActionResponse{Error: "missing mute flag"}
	}
	if err := svc.SetSourceMute(req.Index, *req.Mute); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioDefaultSink(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Name == "" {
		return audioActionResponse{Error: "missing sink name"}
	}
	if err := svc.SetDefaultSink(req.Name); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioDefaultSource(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Name == "" {
		return audioActionResponse{Error: "missing source name"}
	}
	if err := svc.SetDefaultSource(req.Name); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioCardProfile(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Profile == "" {
		return audioActionResponse{Error: "missing profile name"}
	}
	if err := svc.SetCardProfile(req.Index, req.Profile); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSinkPort(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Port == "" {
		return audioActionResponse{Error: "missing port name"}
	}
	if err := svc.SetSinkPort(req.Index, req.Port); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourcePort(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Port == "" {
		return audioActionResponse{Error: "missing port name"}
	}
	if err := svc.SetSourcePort(req.Index, req.Port); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSinkInputVolume(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.SetSinkInputVolume(req.Index, req.Volume); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSinkInputMute(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Mute == nil {
		return audioActionResponse{Error: "missing mute flag"}
	}
	if err := svc.SetSinkInputMute(req.Index, *req.Mute); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSinkInputMove(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.MoveSinkInput(req.Index, req.TargetIndex); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourceOutputVolume(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.SetSourceOutputVolume(req.Index, req.Volume); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourceOutputMute(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Mute == nil {
		return audioActionResponse{Error: "missing mute flag"}
	}
	if err := svc.SetSourceOutputMute(req.Index, *req.Mute); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourceOutputMove(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.MoveSourceOutput(req.Index, req.TargetIndex); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSinkInputKill(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.KillSinkInput(req.Index); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSourceOutputKill(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if err := svc.KillSourceOutput(req.Index); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSuspendSink(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Suspend == nil {
		return audioActionResponse{Error: "missing suspend flag"}
	}
	if err := svc.SuspendSink(req.Index, *req.Suspend); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}

func handleAudioSuspendSource(svc *audiosvc.Service, req audioActionRequest) audioActionResponse {
	if svc == nil {
		return audioActionResponse{Error: "audio service unavailable"}
	}
	if req.Suspend == nil {
		return audioActionResponse{Error: "missing suspend flag"}
	}
	if err := svc.SuspendSource(req.Index, *req.Suspend); err != nil {
		return audioActionResponse{Error: err.Error()}
	}
	return audioActionResponse{Snapshot: svc.Snapshot()}
}
