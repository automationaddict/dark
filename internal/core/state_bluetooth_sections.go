package core

type BluetoothSection struct {
	ID    string
	Icon  string
	Label string
}

func BluetoothSections() []BluetoothSection {
	return []BluetoothSection{
		{"adapters", "󰂯", "Adapters"},
		{"devices", "󰥰", "Devices"},
	}
}

func (s *State) ActiveBluetoothSection() BluetoothSection {
	secs := BluetoothSections()
	if s.BluetoothSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.BluetoothSectionIdx]
}

func (s *State) MoveBluetoothSection(delta int) {
	n := len(BluetoothSections())
	if n == 0 {
		return
	}
	s.BluetoothSectionIdx = (s.BluetoothSectionIdx + delta + n) % n
}
