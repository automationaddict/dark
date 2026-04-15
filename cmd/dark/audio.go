package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	audiosvc "github.com/automationaddict/dark/internal/services/audio"
	"github.com/automationaddict/dark/internal/tui"
)

func newAudioActions(nc *nats.Conn) tui.AudioActions {
	return tui.AudioActions{
		SetSinkVolume: func(index uint32, pct int) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioSinkVolumeCmd, index, pct, nil, "")
			}
		},
		SetSinkMute: func(index uint32, mute bool) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioSinkMuteCmd, index, 0, &mute, "")
			}
		},
		SetSinkBalance: func(index uint32, balance int) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioSinkBalanceCmd, index, balance, nil, "")
			}
		},
		SetSourceVolume: func(index uint32, pct int) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioSourceVolumeCmd, index, pct, nil, "")
			}
		},
		SetSourceMute: func(index uint32, mute bool) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioSourceMuteCmd, index, 0, &mute, "")
			}
		},
		SetSourceBalance: func(index uint32, balance int) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioSourceBalanceCmd, index, balance, nil, "")
			}
		},
		SetDefaultSink: func(name string) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioDefaultSinkCmd, 0, 0, nil, name)
			}
		},
		SetDefaultSource: func(name string) tea.Cmd {
			return func() tea.Msg {
				return audioIndexRequest(nc, bus.SubjectAudioDefaultSourceCmd, 0, 0, nil, name)
			}
		},
		SetCardProfile: func(cardIndex uint32, profile string) tea.Cmd {
			return func() tea.Msg {
				return audioProfileRequest(nc, cardIndex, profile)
			}
		},
		SetSinkPort: func(sinkIndex uint32, port string) tea.Cmd {
			return func() tea.Msg {
				return audioPortRequest(nc, bus.SubjectAudioSinkPortCmd, sinkIndex, port)
			}
		},
		SetSourcePort: func(sourceIndex uint32, port string) tea.Cmd {
			return func() tea.Msg {
				return audioPortRequest(nc, bus.SubjectAudioSourcePortCmd, sourceIndex, port)
			}
		},
		SetSinkInputVolume: func(index uint32, pct int) tea.Cmd {
			return func() tea.Msg {
				return audioStreamVolumeRequest(nc, bus.SubjectAudioSinkInputVolumeCmd, index, pct)
			}
		},
		SetSinkInputMute: func(index uint32, mute bool) tea.Cmd {
			return func() tea.Msg {
				return audioStreamMuteRequest(nc, bus.SubjectAudioSinkInputMuteCmd, index, mute)
			}
		},
		MoveSinkInput: func(streamIndex, sinkIndex uint32) tea.Cmd {
			return func() tea.Msg {
				return audioStreamMoveRequest(nc, bus.SubjectAudioSinkInputMoveCmd, streamIndex, sinkIndex)
			}
		},
		SetSourceOutputVolume: func(index uint32, pct int) tea.Cmd {
			return func() tea.Msg {
				return audioStreamVolumeRequest(nc, bus.SubjectAudioSourceOutputVolumeCmd, index, pct)
			}
		},
		SetSourceOutputMute: func(index uint32, mute bool) tea.Cmd {
			return func() tea.Msg {
				return audioStreamMuteRequest(nc, bus.SubjectAudioSourceOutputMuteCmd, index, mute)
			}
		},
		MoveSourceOutput: func(streamIndex, sourceIndex uint32) tea.Cmd {
			return func() tea.Msg {
				return audioStreamMoveRequest(nc, bus.SubjectAudioSourceOutputMoveCmd, streamIndex, sourceIndex)
			}
		},
		KillSinkInput: func(streamIndex uint32) tea.Cmd {
			return func() tea.Msg {
				return audioKillRequest(nc, bus.SubjectAudioSinkInputKillCmd, streamIndex)
			}
		},
		KillSourceOutput: func(streamIndex uint32) tea.Cmd {
			return func() tea.Msg {
				return audioKillRequest(nc, bus.SubjectAudioSourceOutputKillCmd, streamIndex)
			}
		},
		SuspendSink: func(index uint32, suspend bool) tea.Cmd {
			return func() tea.Msg {
				return audioSuspendRequest(nc, bus.SubjectAudioSuspendSinkCmd, index, suspend)
			}
		},
		SuspendSource: func(index uint32, suspend bool) tea.Cmd {
			return func() tea.Msg {
				return audioSuspendRequest(nc, bus.SubjectAudioSuspendSourceCmd, index, suspend)
			}
		},
	}
}

func audioKillRequest(nc *nats.Conn, subject string, index uint32) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"index": index})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	return decodeAudioReply(reply.Data)
}

func audioSuspendRequest(nc *nats.Conn, subject string, index uint32, suspend bool) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"index": index, "suspend": suspend})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	return decodeAudioReply(reply.Data)
}

func audioStreamVolumeRequest(nc *nats.Conn, subject string, index uint32, pct int) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"index": index, "volume": pct})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	return decodeAudioReply(reply.Data)
}

func audioStreamMuteRequest(nc *nats.Conn, subject string, index uint32, mute bool) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"index": index, "mute": mute})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	return decodeAudioReply(reply.Data)
}

func audioStreamMoveRequest(nc *nats.Conn, subject string, streamIndex, targetIndex uint32) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"index": streamIndex, "target_index": targetIndex})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	return decodeAudioReply(reply.Data)
}

func audioProfileRequest(nc *nats.Conn, cardIndex uint32, profile string) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"index": cardIndex, "profile": profile})
	reply, err := nc.Request(bus.SubjectAudioCardProfileCmd, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	return decodeAudioReply(reply.Data)
}

func audioPortRequest(nc *nats.Conn, subject string, deviceIndex uint32, port string) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{"index": deviceIndex, "port": port})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	return decodeAudioReply(reply.Data)
}

func decodeAudioReply(data []byte) tui.AudioActionResultMsg {
	var resp struct {
		Snapshot audiosvc.Snapshot `json:"snapshot"`
		Error    string            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.AudioActionResultMsg{Err: resp.Error}
	}
	return tui.AudioActionResultMsg{Snapshot: resp.Snapshot}
}

func audioIndexRequest(nc *nats.Conn, subject string, index uint32, volume int, mute *bool, name string) tui.AudioActionResultMsg {
	payload, _ := json.Marshal(map[string]any{
		"index":  index,
		"volume": volume,
		"mute":   mute,
		"name":   name,
	})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot audiosvc.Snapshot `json:"snapshot"`
		Error    string            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.AudioActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.AudioActionResultMsg{Err: resp.Error}
	}
	return tui.AudioActionResultMsg{Snapshot: resp.Snapshot}
}

func requestInitialAudio(nc *nats.Conn) (audiosvc.Snapshot, bool) {
	reply, err := nc.Request(bus.SubjectAudioDevicesCmd, nil, core.TimeoutFast)
	if err != nil {
		return audiosvc.Snapshot{}, false
	}
	var snap audiosvc.Snapshot
	if err := json.Unmarshal(reply.Data, &snap); err != nil {
		return audiosvc.Snapshot{}, false
	}
	return snap, true
}
