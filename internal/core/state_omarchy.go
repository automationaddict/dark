package core

import (
	"github.com/johnnelson/dark/internal/services/keybind"
	"github.com/johnnelson/dark/internal/services/tuilink"
	"github.com/johnnelson/dark/internal/services/weblink"
)

type OmarchySection struct {
	ID    string
	Icon  string
	Label string
}

func OmarchySections() []OmarchySection {
	return []OmarchySection{
		{"weblinks", "󰖟", "Web Links"},
		{"tuilinks", "󰆍", "TUI Links"},
		{"keybindings", "󰌌", "Keybindings"},
	}
}

func (s *State) SetWebLinks(apps []weblink.WebApp) {
	s.WebLinks = apps
	s.WebLinksLoaded = true
	if s.WebLinkIdx >= len(apps) {
		s.WebLinkIdx = 0
	}
}

func (s *State) SetTUILinks(apps []tuilink.TUIApp) {
	s.TUILinks = apps
	s.TUILinksLoaded = true
	if s.TUILinkIdx >= len(apps) {
		s.TUILinkIdx = 0
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

func (s *State) SelectedWebLink() (weblink.WebApp, bool) {
	if len(s.WebLinks) == 0 {
		return weblink.WebApp{}, false
	}
	if s.WebLinkIdx >= len(s.WebLinks) {
		s.WebLinkIdx = 0
	}
	return s.WebLinks[s.WebLinkIdx], true
}

func (s *State) SelectedTUILink() (tuilink.TUIApp, bool) {
	if len(s.TUILinks) == 0 {
		return tuilink.TUIApp{}, false
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
