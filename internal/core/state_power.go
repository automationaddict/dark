package core

type PowerFocus string

const (
	PowerFocusProfile PowerFocus = "profile"
	PowerFocusIdle    PowerFocus = "idle"
	PowerFocusButtons PowerFocus = "buttons"
)

type PowerSection struct {
	ID    string
	Icon  string
	Label string
}

func PowerSections() []PowerSection {
	return []PowerSection{
		{"overview", "󰂄", "Overview"},
		{"profile", "󰓅", "Power Profile"},
		{"cpu", "󰘚", "CPU"},
		{"thermal", "󰔏", "Thermal"},
		{"buttons", "󰐥", "System Buttons"},
		{"idle", "󰒲", "Screen & Idle"},
	}
}

func (s *State) ActivePowerSection() PowerSection {
	secs := PowerSections()
	if s.PowerSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.PowerSectionIdx]
}

func (s *State) MovePowerSection(delta int) {
	n := len(PowerSections())
	if n == 0 {
		return
	}
	s.PowerSectionIdx = (s.PowerSectionIdx + delta + n) % n
}
