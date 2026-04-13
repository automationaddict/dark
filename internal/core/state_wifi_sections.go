package core

type WifiSection struct {
	ID    string
	Icon  string
	Label string
}

func WifiSections() []WifiSection {
	return []WifiSection{
		{"adapters", "󰖩", "Adapters"},
		{"networks", "󰀂", "Networks"},
		{"known", "󰄡", "Known Networks"},
		{"ap", "󱚿", "Access Point"},
	}
}

func (s *State) ActiveWifiSection() WifiSection {
	secs := WifiSections()
	if s.WifiSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.WifiSectionIdx]
}

func (s *State) MoveWifiSection(delta int) {
	n := len(WifiSections())
	if n == 0 {
		return
	}
	s.WifiSectionIdx = (s.WifiSectionIdx + delta + n) % n
}
