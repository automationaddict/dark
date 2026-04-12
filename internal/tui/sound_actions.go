package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/audio"
)

// AudioActions is the set of asynchronous commands the TUI can dispatch
// at darkd to drive the audio service.
type AudioActions struct {
	SetSinkVolume         func(index uint32, pct int) tea.Cmd
	SetSinkMute           func(index uint32, mute bool) tea.Cmd
	SetSourceVolume       func(index uint32, pct int) tea.Cmd
	SetSourceMute         func(index uint32, mute bool) tea.Cmd
	SetDefaultSink        func(name string) tea.Cmd
	SetDefaultSource      func(name string) tea.Cmd
	SetCardProfile        func(cardIndex uint32, profile string) tea.Cmd
	SetSinkPort           func(sinkIndex uint32, port string) tea.Cmd
	SetSourcePort         func(sourceIndex uint32, port string) tea.Cmd
	SetSinkInputVolume    func(index uint32, pct int) tea.Cmd
	SetSinkInputMute      func(index uint32, mute bool) tea.Cmd
	MoveSinkInput         func(streamIndex, sinkIndex uint32) tea.Cmd
	KillSinkInput         func(streamIndex uint32) tea.Cmd
	SetSourceOutputVolume func(index uint32, pct int) tea.Cmd
	SetSourceOutputMute   func(index uint32, mute bool) tea.Cmd
	MoveSourceOutput      func(streamIndex, sourceIndex uint32) tea.Cmd
	KillSourceOutput      func(streamIndex uint32) tea.Cmd
	SuspendSink           func(index uint32, suspend bool) tea.Cmd
	SuspendSource         func(index uint32, suspend bool) tea.Cmd
}

// AudioMsg is dispatched whenever darkd publishes an audio snapshot.
type AudioMsg audio.Snapshot

// AudioLevelsMsg is dispatched at ~20 Hz with the current peak meter
// readings for every sink and source. Lightweight payload — just two
// small maps — so handing it through bubble tea on every tick is fine.
type AudioLevelsMsg audio.Levels

// AudioActionResultMsg is dispatched when an audio action command
// completes. On success, the reply's updated snapshot replaces the
// cached one; on failure, the error is shown inline.
type AudioActionResultMsg struct {
	Snapshot audio.Snapshot
	Err      string
}

func (m *Model) inSoundContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "sound"
}

// triggerAudioVolumeDelta adjusts the volume of the currently selected
// row by delta percentage points (typically ±5). Routes to sink,
// source, sink input, or source output based on which sub-table has
// focus.
func (m *Model) triggerAudioVolumeDelta(delta int) tea.Cmd {
	if !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}

	switch m.state.AudioFocus {
	case core.AudioFocusPlayApps, core.AudioFocusRecordApps:
		stream, isPlay, ok := m.state.SelectedAudioStream()
		if !ok {
			return nil
		}
		target := clampVolume(stream.Volume + delta)
		m.state.AudioBusy = true
		m.state.AudioActionError = ""
		if isPlay {
			if m.audio.SetSinkInputVolume == nil {
				return nil
			}
			return m.audio.SetSinkInputVolume(stream.Index, target)
		}
		if m.audio.SetSourceOutputVolume == nil {
			return nil
		}
		return m.audio.SetSourceOutputVolume(stream.Index, target)
	}

	dev, isSink, ok := m.state.SelectedAudioDevice()
	if !ok {
		return nil
	}
	target := clampVolume(dev.Volume + delta)
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	if isSink {
		if m.audio.SetSinkVolume == nil {
			return nil
		}
		return m.audio.SetSinkVolume(dev.Index, target)
	}
	if m.audio.SetSourceVolume == nil {
		return nil
	}
	return m.audio.SetSourceVolume(dev.Index, target)
}

func clampVolume(v int) int {
	if v < 0 {
		return 0
	}
	if v > 150 {
		return 150
	}
	return v
}

// triggerAudioMuteToggle flips mute on the selected row, routing
// through the stream path when focus is on an apps sub-table.
func (m *Model) triggerAudioMuteToggle() tea.Cmd {
	if !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}

	switch m.state.AudioFocus {
	case core.AudioFocusPlayApps, core.AudioFocusRecordApps:
		stream, isPlay, ok := m.state.SelectedAudioStream()
		if !ok {
			return nil
		}
		m.state.AudioBusy = true
		m.state.AudioActionError = ""
		if isPlay {
			if m.audio.SetSinkInputMute == nil {
				return nil
			}
			return m.audio.SetSinkInputMute(stream.Index, !stream.Mute)
		}
		if m.audio.SetSourceOutputMute == nil {
			return nil
		}
		return m.audio.SetSourceOutputMute(stream.Index, !stream.Mute)
	}

	dev, isSink, ok := m.state.SelectedAudioDevice()
	if !ok {
		return nil
	}
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	if isSink {
		if m.audio.SetSinkMute == nil {
			return nil
		}
		return m.audio.SetSinkMute(dev.Index, !dev.Mute)
	}
	if m.audio.SetSourceMute == nil {
		return nil
	}
	return m.audio.SetSourceMute(dev.Index, !dev.Mute)
}

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

// triggerAudioCycleProfile advances the active profile on the card
// backing the currently selected device. Cycles through available
// profiles only — unavailable ones (like bluetooth HFP when the
// remote end has it disabled) are skipped. No-op when the device
// isn't card-backed (virtual sinks like null sinks have CardIndex
// equal to PulseAudio's "undefined" sentinel).
func (m *Model) triggerAudioCycleProfile() tea.Cmd {
	if m.audio.SetCardProfile == nil || !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}
	dev, _, ok := m.state.SelectedAudioDevice()
	if !ok {
		return nil
	}
	card, ok := m.state.Audio.CardByIndex(dev.CardIndex)
	if !ok || len(card.Profiles) == 0 {
		m.state.AudioActionError = "selected device has no card-backed profiles"
		return nil
	}
	next := nextAvailableProfile(card)
	if next == "" {
		m.state.AudioActionError = "no other available profiles on this card"
		return nil
	}
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	return m.audio.SetCardProfile(card.Index, next)
}

// nextAvailableProfile returns the name of the next profile after the
// active one in the card's profile list, skipping any profile reported
// as unavailable. Wraps around the list. Returns "" when there is no
// alternative profile available.
func nextAvailableProfile(card audio.Card) string {
	avail := []string{}
	current := -1
	for _, p := range card.Profiles {
		// Available 0 = unknown (treat as available), 1 = no, 2 = yes.
		if p.Available == 1 {
			continue
		}
		if p.Name == card.ActiveProfile {
			current = len(avail)
		}
		avail = append(avail, p.Name)
	}
	if len(avail) < 2 {
		return ""
	}
	if current < 0 {
		return avail[0]
	}
	return avail[(current+1)%len(avail)]
}

// triggerAudioCyclePort advances the active port on the currently
// selected sink/source. Like profile cycling, skips unavailable ports
// (e.g. headphones jack with nothing plugged in).
func (m *Model) triggerAudioCyclePort() tea.Cmd {
	if !m.inSoundContent() || m.state.AudioBusy {
		return nil
	}
	dev, isSink, ok := m.state.SelectedAudioDevice()
	if !ok || len(dev.Ports) == 0 {
		m.state.AudioActionError = "selected device has no switchable ports"
		return nil
	}
	next := nextAvailablePort(dev)
	if next == "" {
		m.state.AudioActionError = "no other available ports on this device"
		return nil
	}
	m.state.AudioBusy = true
	m.state.AudioActionError = ""
	if isSink {
		if m.audio.SetSinkPort == nil {
			return nil
		}
		return m.audio.SetSinkPort(dev.Index, next)
	}
	if m.audio.SetSourcePort == nil {
		return nil
	}
	return m.audio.SetSourcePort(dev.Index, next)
}

func nextAvailablePort(dev audio.Device) string {
	avail := []string{}
	current := -1
	for _, p := range dev.Ports {
		if p.Available == 1 {
			continue
		}
		if p.Name == dev.ActivePort {
			current = len(avail)
		}
		avail = append(avail, p.Name)
	}
	if len(avail) < 2 {
		return ""
	}
	if current < 0 {
		return avail[0]
	}
	return avail[(current+1)%len(avail)]
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
