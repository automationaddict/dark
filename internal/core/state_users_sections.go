package core

type UsersSection struct {
	ID    string
	Icon  string
	Label string
}

func UsersSections() []UsersSection {
	return []UsersSection{
		{"identity", "󰀄", "Identity"},
		{"account", "󰟃", "Account"},
		{"security", "󰒃", "Security"},
	}
}

func (s *State) ActiveUsersSection() UsersSection {
	secs := UsersSections()
	if s.UsersSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.UsersSectionIdx]
}

func (s *State) MoveUsersSection(delta int) {
	n := len(UsersSections())
	if n == 0 {
		return
	}
	s.UsersSectionIdx = (s.UsersSectionIdx + delta + n) % n
}
