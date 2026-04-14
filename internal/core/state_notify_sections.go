package core

type NotifySection struct {
	ID    string
	Icon  string
	Label string
}

func NotifySections() []NotifySection {
	return []NotifySection{
		{"daemon", "󰂚", "Daemon"},
		{"appearance", "󰏘", "Appearance"},
		{"behavior", "󰂞", "Behavior"},
		{"rules", "󰥔", "Rules"},
	}
}

func (s *State) ActiveNotifySection() NotifySection {
	secs := NotifySections()
	if s.NotifySectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.NotifySectionIdx]
}

func (s *State) MoveNotifySection(delta int) {
	n := len(NotifySections())
	if n == 0 {
		return
	}
	s.NotifySectionIdx = (s.NotifySectionIdx + delta + n) % n
}
