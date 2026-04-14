package core

type DateTimeSection struct {
	ID    string
	Icon  string
	Label string
}

func DateTimeSections() []DateTimeSection {
	return []DateTimeSection{
		{"time", "󰥔", "Time"},
		{"sync", "󰔟", "Sync"},
		{"hardware", "󰻂", "Hardware"},
	}
}

func (s *State) ActiveDateTimeSection() DateTimeSection {
	secs := DateTimeSections()
	if s.DateTimeSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.DateTimeSectionIdx]
}

func (s *State) MoveDateTimeSection(delta int) {
	n := len(DateTimeSections())
	if n == 0 {
		return
	}
	s.DateTimeSectionIdx = (s.DateTimeSectionIdx + delta + n) % n
}
