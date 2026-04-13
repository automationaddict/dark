package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/services/audio"
)

// triggerAudioStreamMove cycles the highlighted stream to the next
// available device of the matching kind. For a sink input that means
// the next sink in the snapshot's list; for a source output, the next
// source. Wraps around. No-op when there's only one target available.
func (m *Model) triggerAudioStreamMove() tea.Cmd {
	if !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}
	stream, isPlay, ok := m.state.SelectedAudioStream()
	if !ok {
		return nil
	}
	if isPlay {
		if m.audio.MoveSinkInput == nil {
			return nil
		}
		next, ok := nextDeviceIndex(m.state.Audio.Sinks, stream.DeviceIndex)
		if !ok {
			m.state.AudioActionError = "no other sinks available to move to"
			return nil
		}
		m.state.AudioBusy = true
		m.state.AudioActionError = ""
		return m.audio.MoveSinkInput(stream.Index, next)
	}
	if m.audio.MoveSourceOutput == nil {
		return nil
	}
	next, ok := nextDeviceIndex(m.state.Audio.Sources, stream.DeviceIndex)
	if !ok {
		m.state.AudioActionError = "no other sources available to move to"
		return nil
	}
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	return m.audio.MoveSourceOutput(stream.Index, next)
}

// triggerAudioSuspendToggle flips the suspend state of the highlighted
// device. Suspend tells PulseAudio to release the underlying hardware
// so another client can grab it; resume re-acquires it. Operates on
// whichever device kind currently has focus. Stream sub-tables are
// ignored — suspend is a device-level concept.
func (m *Model) triggerAudioSuspendToggle() tea.Cmd {
	if !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}
	dev, isSink, ok := m.state.SelectedAudioDevice()
	if !ok {
		return nil
	}
	suspend := dev.State != "suspended"
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	if isSink {
		if m.audio.SuspendSink == nil {
			return nil
		}
		return m.audio.SuspendSink(dev.Index, suspend)
	}
	if m.audio.SuspendSource == nil {
		return nil
	}
	return m.audio.SuspendSource(dev.Index, suspend)
}

// triggerAudioKillStream disconnects a stuck per-app stream. Only
// active when focus is on an apps sub-table. Routes to KillSinkInput
// or KillSourceOutput based on which apps box has focus.
func (m *Model) triggerAudioKillStream() tea.Cmd {
	if !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}
	stream, isPlay, ok := m.state.SelectedAudioStream()
	if !ok {
		return nil
	}
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	if isPlay {
		if m.audio.KillSinkInput == nil {
			return nil
		}
		return m.audio.KillSinkInput(stream.Index)
	}
	if m.audio.KillSourceOutput == nil {
		return nil
	}
	return m.audio.KillSourceOutput(stream.Index)
}

// nextDeviceIndex returns the index of the next device after `current`
// in the list, wrapping at the end. Returns ok=false when the list has
// fewer than two entries (no alternative to move to).
func nextDeviceIndex(devs []audio.Device, current uint32) (uint32, bool) {
	if len(devs) < 2 {
		return 0, false
	}
	at := -1
	for i, d := range devs {
		if d.Index == current {
			at = i
			break
		}
	}
	if at < 0 {
		return devs[0].Index, true
	}
	return devs[(at+1)%len(devs)].Index, true
}

// triggerAudioSetDefault marks the selected device as the default.
func (m *Model) triggerAudioSetDefault() tea.Cmd {
	if !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}
	dev, isSink, ok := m.state.SelectedAudioDevice()
	if !ok || dev.Name == "" {
		return nil
	}
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	if isSink {
		if m.audio.SetDefaultSink == nil {
			return nil
		}
		return m.audio.SetDefaultSink(dev.Name)
	}
	if m.audio.SetDefaultSource == nil {
		return nil
	}
	return m.audio.SetDefaultSource(dev.Name)
}
