package core

// DisplaySection describes one sub-section visible in the Display
// inner sidebar. Follows the same pattern as NetworkSection, etc.
type DisplaySection struct {
	ID    string
	Icon  string
	Label string
}

func DisplaySections() []DisplaySection {
	return []DisplaySection{
		{"monitors", "󰍹", "Monitors"},
		{"controls", "󰃟", "Controls"},
		{"gpu", "󰢮", "GPU"},
		{"layout", "󰕮", "Layout"},
		{"profiles", "󰆓", "Profiles"},
	}
}

func (s *State) ActiveDisplaySection() DisplaySection {
	secs := DisplaySections()
	if s.DisplaySectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.DisplaySectionIdx]
}

func (s *State) MoveDisplaySection(delta int) {
	n := len(DisplaySections())
	if n == 0 {
		return
	}
	s.DisplaySectionIdx = (s.DisplaySectionIdx + delta + n) % n
}
