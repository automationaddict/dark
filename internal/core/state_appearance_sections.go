package core

// AppearanceSection describes one sub-section visible in the Appearance
// inner sidebar.
type AppearanceSection struct {
	ID    string
	Icon  string
	Label string
}

func AppearanceSections() []AppearanceSection {
	return []AppearanceSection{
		{"theme", "󰸌", "Theme"},
		{"fonts", "󰛖", "Fonts"},
		{"windows", "󱂬", "Windows"},
		{"effects", "󰓆", "Effects"},
		{"cursor", "󰳽", "Cursor"},
	}
}

func (s *State) ActiveAppearanceSection() AppearanceSection {
	secs := AppearanceSections()
	if s.AppearanceSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.AppearanceSectionIdx]
}

func (s *State) MoveAppearanceSection(delta int) {
	n := len(AppearanceSections())
	if n == 0 {
		return
	}
	s.AppearanceSectionIdx = (s.AppearanceSectionIdx + delta + n) % n
}
