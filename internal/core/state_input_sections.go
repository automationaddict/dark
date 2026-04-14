package core

// InputSection describes one sub-section visible in the Input Devices
// inner sidebar.
type InputSection struct {
	ID    string
	Icon  string
	Label string
}

func InputSections() []InputSection {
	return []InputSection{
		{"keyboard", "󰌌", "Keyboard"},
		{"mouse", "󰍽", "Mouse"},
		{"touchpad", "󰟸", "Touchpad"},
		{"other", "󰗊", "Other"},
	}
}

func (s *State) ActiveInputSection() InputSection {
	secs := InputSections()
	if s.InputSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.InputSectionIdx]
}

func (s *State) MoveInputSection(delta int) {
	n := len(InputSections())
	if n == 0 {
		return
	}
	s.InputSectionIdx = (s.InputSectionIdx + delta + n) % n
}
