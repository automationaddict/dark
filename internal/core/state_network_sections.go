package core

// NetworkSection describes one sub-section visible in the Network
// inner sidebar. Follows the same pattern as WifiSection, BluetoothSection,
// and PowerSection.
type NetworkSection struct {
	ID    string
	Icon  string
	Label string
}

func NetworkSections() []NetworkSection {
	return []NetworkSection{
		{"interfaces", "󰛳", "Interfaces"},
		{"dns", "󰇖", "DNS"},
		{"routes", "󰑪", "Routes"},
	}
}

func (s *State) ActiveNetworkSection() NetworkSection {
	secs := NetworkSections()
	if s.NetworkSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.NetworkSectionIdx]
}

func (s *State) MoveNetworkSection(delta int) {
	n := len(NetworkSections())
	if n == 0 {
		return
	}
	s.NetworkSectionIdx = (s.NetworkSectionIdx + delta + n) % n
}
