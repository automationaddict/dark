package core

type AboutSection struct {
	ID    string
	Icon  string
	Label string
}

func AboutSections() []AboutSection {
	return []AboutSection{
		{"system", "󰌢", "System"},
		{"hardware", "󰍛", "Hardware"},
		{"session", "󰆥", "Session"},
		{"omarchy", "󰣇", "Omarchy"},
		{"dark", "󰊠", "dark"},
	}
}

func (s *State) ActiveAboutSection() AboutSection {
	secs := AboutSections()
	if s.AboutSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.AboutSectionIdx]
}

func (s *State) MoveAboutSection(delta int) {
	n := len(AboutSections())
	if n == 0 {
		return
	}
	s.AboutSectionIdx = (s.AboutSectionIdx + delta + n) % n
}
