package core

// AudioSection describes one sub-section visible in the Sound
// inner sidebar. Replaces the Tab-cycling AudioFocus approach.
type AudioSection struct {
	ID    string
	Icon  string
	Label string
}

func AudioSections() []AudioSection {
	return []AudioSection{
		{"sinks", "󰕾", "Output Devices"},
		{"sources", "󰍬", "Input Devices"},
		{"play_apps", "󰐊", "Playing Apps"},
		{"record_apps", "󰻃", "Recording Apps"},
	}
}

func (s *State) ActiveAudioSection() AudioSection {
	secs := AudioSections()
	if s.AudioSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.AudioSectionIdx]
}

func (s *State) MoveAudioSection(delta int) {
	n := len(AudioSections())
	if n == 0 {
		return
	}
	s.AudioSectionIdx = (s.AudioSectionIdx + delta + n) % n
}
