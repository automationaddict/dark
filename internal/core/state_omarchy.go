package core

import "github.com/johnnelson/dark/internal/services/weblink"

type OmarchySection struct {
	ID    string
	Icon  string
	Label string
}

func OmarchySections() []OmarchySection {
	return []OmarchySection{
		{"weblinks", "󰖟", "Web Links"},
	}
}

func (s *State) SetWebLinks(apps []weblink.WebApp) {
	s.WebLinks = apps
	s.WebLinksLoaded = true
	if s.OmarchyFocusIdx >= len(apps) {
		s.OmarchyFocusIdx = 0
	}
}

func (s *State) MoveOmarchySidebar(delta int) {
	n := len(OmarchySections())
	if n == 0 {
		return
	}
	s.OmarchySidebarIdx = (s.OmarchySidebarIdx + delta + n) % n
}

func (s *State) MoveOmarchyFocus(delta int) {
	n := len(s.WebLinks)
	if n == 0 {
		return
	}
	s.OmarchyFocusIdx = (s.OmarchyFocusIdx + delta + n) % n
}

func (s *State) ActiveOmarchySection() OmarchySection {
	secs := OmarchySections()
	if s.OmarchySidebarIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.OmarchySidebarIdx]
}

func (s *State) SelectedWebLink() (weblink.WebApp, bool) {
	if len(s.WebLinks) == 0 {
		return weblink.WebApp{}, false
	}
	if s.OmarchyFocusIdx >= len(s.WebLinks) {
		s.OmarchyFocusIdx = 0
	}
	return s.WebLinks[s.OmarchyFocusIdx], true
}
