package core

import (
	"github.com/johnnelson/dark/internal/services/keybind"
	"github.com/johnnelson/dark/internal/services/links"
)

type OmarchySection struct {
	ID    string
	Icon  string
	Label string
}

func OmarchySections() []OmarchySection {
	return []OmarchySection{
		{"links", "󰖟", "Links"},
		{"keybindings", "󰌌", "Keybindings"},
	}
}

// LinksSection describes a sub-section inside the "Links" omarchy section.
type LinksSection struct {
	ID    string
	Icon  string
	Label string
}

func LinksSections() []LinksSection {
	return []LinksSection{
		{"weblinks", "󰖟", "Web Links"},
		{"tuilinks", "󰆍", "TUI Links"},
		{"helplinks", "󰋗", "Help Links"},
	}
}

func (s *State) SelectedHelpLink() (links.HelpLink, bool) {
	if len(s.HelpLinks) == 0 {
		return links.HelpLink{}, false
	}
	if s.HelpLinkIdx >= len(s.HelpLinks) {
		s.HelpLinkIdx = 0
	}
	return s.HelpLinks[s.HelpLinkIdx], true
}

func (s *State) ActiveLinksSection() LinksSection {
	secs := LinksSections()
	if s.OmarchyLinksIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.OmarchyLinksIdx]
}

func (s *State) MoveLinksSection(delta int) {
	n := len(LinksSections())
	if n == 0 {
		return
	}
	s.OmarchyLinksIdx = (s.OmarchyLinksIdx + delta + n) % n
}

func (s *State) SetLinks(lf links.LinksFile) {
	s.WebLinks = lf.WebLinks
	s.TUILinks = lf.TUILinks
	s.HelpLinks = lf.HelpLinks
	s.LinksLoaded = true
	if s.WebLinkIdx >= len(lf.WebLinks) {
		s.WebLinkIdx = 0
	}
	if s.TUILinkIdx >= len(lf.TUILinks) {
		s.TUILinkIdx = 0
	}
	if s.HelpLinkIdx >= len(lf.HelpLinks) {
		s.HelpLinkIdx = 0
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
	switch s.ActiveOmarchySection().ID {
	case "links":
		if s.OmarchyLinksFocused {
			switch s.ActiveLinksSection().ID {
			case "weblinks":
				n := len(s.WebLinks)
				if n == 0 {
					return
				}
				s.WebLinkIdx = (s.WebLinkIdx + delta + n) % n
			case "tuilinks":
				n := len(s.TUILinks)
				if n == 0 {
					return
				}
				s.TUILinkIdx = (s.TUILinkIdx + delta + n) % n
			case "helplinks":
				n := len(s.HelpLinks)
				if n == 0 {
					return
				}
				s.HelpLinkIdx = (s.HelpLinkIdx + delta + n) % n
			}
		} else {
			s.MoveLinksSection(delta)
		}
	case "keybindings":
		if s.KeybindTableFocused {
			n := len(s.FilteredKeybindings())
			if n == 0 {
				return
			}
			s.KeybindIdx = (s.KeybindIdx + delta + n) % n
		} else {
			s.MoveKeybindFilter(delta)
		}
	}
}

func (s *State) ActiveOmarchySection() OmarchySection {
	secs := OmarchySections()
	if s.OmarchySidebarIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.OmarchySidebarIdx]
}

func (s *State) SelectedWebLink() (links.WebLink, bool) {
	if len(s.WebLinks) == 0 {
		return links.WebLink{}, false
	}
	if s.WebLinkIdx >= len(s.WebLinks) {
		s.WebLinkIdx = 0
	}
	return s.WebLinks[s.WebLinkIdx], true
}

func (s *State) SelectedTUILink() (links.TUILink, bool) {
	if len(s.TUILinks) == 0 {
		return links.TUILink{}, false
	}
	if s.TUILinkIdx >= len(s.TUILinks) {
		s.TUILinkIdx = 0
	}
	return s.TUILinks[s.TUILinkIdx], true
}

func (s *State) SetKeybindings(snap keybind.Snapshot) {
	s.Keybindings = snap
	s.KeybindingsLoaded = true
	filtered := s.FilteredKeybindings()
	if s.KeybindIdx >= len(filtered) {
		s.KeybindIdx = 0
	}
}

func (s *State) FilteredKeybindings() []keybind.Binding {
	switch s.KeybindFilter {
	case 1:
		var out []keybind.Binding
		for _, b := range s.Keybindings.Bindings {
			if b.Source == keybind.SourceDefault {
				out = append(out, b)
			}
		}
		return out
	case 2:
		var out []keybind.Binding
		for _, b := range s.Keybindings.Bindings {
			if b.Source == keybind.SourceUser {
				out = append(out, b)
			}
		}
		return out
	default:
		return s.Keybindings.Bindings
	}
}

type KeybindSection struct {
	ID    string
	Icon  string
	Label string
}

func KeybindSections() []KeybindSection {
	return []KeybindSection{
		{"all", "󰌌", "All"},
		{"default", "󰒓", "Default"},
		{"user", "󰀉", "User"},
	}
}

func (s *State) CycleKeybindFilter() {
	s.KeybindFilter = (s.KeybindFilter + 1) % 3
	s.KeybindIdx = 0
}

func (s *State) MoveKeybindFilter(delta int) {
	n := len(KeybindSections())
	s.KeybindFilter = (s.KeybindFilter + delta + n) % n
	s.KeybindIdx = 0
}

func (s *State) SelectedKeybinding() (keybind.Binding, bool) {
	filtered := s.FilteredKeybindings()
	if len(filtered) == 0 {
		return keybind.Binding{}, false
	}
	if s.KeybindIdx >= len(filtered) {
		s.KeybindIdx = 0
	}
	return filtered[s.KeybindIdx], true
}
