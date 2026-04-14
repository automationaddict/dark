package core

type PrivacySection struct {
	ID    string
	Icon  string
	Label string
}

func PrivacySections() []PrivacySection {
	return []PrivacySection{
		{"screenlock", "󰌾", "Screen Lock"},
		{"network", "󰒄", "Network"},
		{"data", "󰗮", "Data"},
		{"location", "󰍎", "Location"},
	}
}

func (s *State) ActivePrivacySection() PrivacySection {
	secs := PrivacySections()
	if s.PrivacySectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.PrivacySectionIdx]
}

func (s *State) MovePrivacySection(delta int) {
	n := len(PrivacySections())
	if n == 0 {
		return
	}
	s.PrivacySectionIdx = (s.PrivacySectionIdx + delta + n) % n
}
