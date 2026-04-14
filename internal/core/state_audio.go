package core

import "github.com/johnnelson/dark/internal/services/audio"

// AudioFocus identifies which sub-table owns j/k and the action keys
// while the Sound section has content focus. Tab cycles between them
// in order: Sinks → Sources → PlayApps → RecordApps → Sinks.
type AudioFocus string

const (
	AudioFocusSinks      AudioFocus = "sinks"
	AudioFocusSources    AudioFocus = "sources"
	AudioFocusPlayApps   AudioFocus = "play_apps"
	AudioFocusRecordApps AudioFocus = "record_apps"
)

// SetAudio replaces the cached audio snapshot with one received from
// darkd. Selection indices are clamped to the new list sizes so a
// hot-unplugged device or a closed app stream doesn't leave an
// out-of-bounds cursor.
func (s *State) SetAudio(snap audio.Snapshot) {
	s.Audio = snap
	s.AudioLoaded = true
	s.SyncAudioFocus()
	if s.AudioSinkIdx >= len(snap.Sinks) {
		s.AudioSinkIdx = 0
	}
	if s.AudioSourceIdx >= len(snap.Sources) {
		s.AudioSourceIdx = 0
	}
	if s.AudioPlayAppIdx >= len(snap.SinkInputs) {
		s.AudioPlayAppIdx = 0
	}
	if s.AudioRecordAppIdx >= len(snap.SourceOutputs) {
		s.AudioRecordAppIdx = 0
	}
}

// SetAudioLevels updates the cached peak meter readings. Called from
// the bus subscriber on every levels publish (~20 Hz). Levels are
// stored on the state so the view can read them during the next
// frame; bubble tea will redraw on every Msg, which is what we want
// for a live meter.
func (s *State) SetAudioLevels(levels audio.Levels) {
	s.AudioLevels = levels
}

// SinkLevel returns the latest stereo peak pair [left, right] for a
// sink, or zeros when no reading is available.
func (s *State) SinkLevel(index uint32) [2]float32 {
	if s.AudioLevels.Sinks == nil {
		return [2]float32{}
	}
	return s.AudioLevels.Sinks[index]
}

// SourceLevel returns the latest stereo peak pair for a source.
func (s *State) SourceLevel(index uint32) [2]float32 {
	if s.AudioLevels.Sources == nil {
		return [2]float32{}
	}
	return s.AudioLevels.Sources[index]
}

// syncAudioFocus keeps AudioFocus in sync with AudioSectionIdx so
// existing action triggers that check AudioFocus still work.
func (s *State) SyncAudioFocus() {
	sec := s.ActiveAudioSection()
	switch sec.ID {
	case "sinks":
		s.AudioFocus = AudioFocusSinks
	case "sources":
		s.AudioFocus = AudioFocusSources
	case "play_apps":
		s.AudioFocus = AudioFocusPlayApps
	case "record_apps":
		s.AudioFocus = AudioFocusRecordApps
	default:
		s.AudioFocus = AudioFocusSinks
	}
}

// MoveAudioSelection walks the selection within the focused sub-table.
func (s *State) MoveAudioSelection(delta int) {
	switch s.AudioFocus {
	case AudioFocusSources:
		n := len(s.Audio.Sources)
		if n == 0 {
			return
		}
		s.AudioSourceIdx = (s.AudioSourceIdx + delta + n) % n
	case AudioFocusPlayApps:
		n := len(s.Audio.SinkInputs)
		if n == 0 {
			return
		}
		s.AudioPlayAppIdx = (s.AudioPlayAppIdx + delta + n) % n
	case AudioFocusRecordApps:
		n := len(s.Audio.SourceOutputs)
		if n == 0 {
			return
		}
		s.AudioRecordAppIdx = (s.AudioRecordAppIdx + delta + n) % n
	default:
		n := len(s.Audio.Sinks)
		if n == 0 {
			return
		}
		s.AudioSinkIdx = (s.AudioSinkIdx + delta + n) % n
	}
}

// SelectedAudioStream returns the highlighted per-app stream and a
// flag indicating whether it's a sink input (true) or a source output
// (false). Returns ok=false when the focus isn't on an apps sub-table
// or the relevant list is empty.
func (s *State) SelectedAudioStream() (audio.Stream, bool, bool) {
	switch s.AudioFocus {
	case AudioFocusPlayApps:
		if len(s.Audio.SinkInputs) == 0 {
			return audio.Stream{}, true, false
		}
		if s.AudioPlayAppIdx >= len(s.Audio.SinkInputs) {
			s.AudioPlayAppIdx = 0
		}
		return s.Audio.SinkInputs[s.AudioPlayAppIdx], true, true
	case AudioFocusRecordApps:
		if len(s.Audio.SourceOutputs) == 0 {
			return audio.Stream{}, false, false
		}
		if s.AudioRecordAppIdx >= len(s.Audio.SourceOutputs) {
			s.AudioRecordAppIdx = 0
		}
		return s.Audio.SourceOutputs[s.AudioRecordAppIdx], false, true
	}
	return audio.Stream{}, false, false
}

// OpenAudioDeviceInfo drills into a per-device info panel for the
// highlighted device. Only valid when content focus is on Sound and
// at least one device exists.
func (s *State) OpenAudioDeviceInfo() {
	if !s.ContentFocused || s.ActiveSection().ID != "sound" {
		return
	}
	if _, _, ok := s.SelectedAudioDevice(); !ok {
		return
	}
	s.AudioDeviceInfoOpen = true
}

// CloseAudioDeviceInfo backs out of the device info panel.
func (s *State) CloseAudioDeviceInfo() {
	s.AudioDeviceInfoOpen = false
}

// SelectedAudioDevice returns the currently-highlighted device and a
// flag indicating whether it's a sink (true) or a source (false).
func (s *State) SelectedAudioDevice() (audio.Device, bool, bool) {
	if s.AudioFocus == AudioFocusSources {
		if len(s.Audio.Sources) == 0 {
			return audio.Device{}, false, false
		}
		if s.AudioSourceIdx >= len(s.Audio.Sources) {
			s.AudioSourceIdx = 0
		}
		return s.Audio.Sources[s.AudioSourceIdx], false, true
	}
	if len(s.Audio.Sinks) == 0 {
		return audio.Device{}, true, false
	}
	if s.AudioSinkIdx >= len(s.Audio.Sinks) {
		s.AudioSinkIdx = 0
	}
	return s.Audio.Sinks[s.AudioSinkIdx], true, true
}
