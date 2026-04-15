package core

import "github.com/automationaddict/dark/internal/help"

// HelpKey returns the context key for the currently visible view.
// The help package looks this up in its embedded content directory.
func (s *State) HelpKey() string {
	switch s.ActiveTab {
	case TabSettings:
		return s.ActiveSection().ID
	case TabF2:
		if s.F2OnUpdates() {
			return "updates"
		}
		return "appstore"
	case TabF3:
		switch s.ActiveOmarchySection().ID {
		case "limine":
			return "limine"
		case "keybindings":
			return "keybindings"
		case "links":
			return "links"
		}
	}
	return "default"
}

func (s *State) OpenHelp() {
	doc, err := help.Load(s.HelpKey(), s.HelpWidth)
	if err != nil {
		return
	}
	s.HelpDoc = doc
	s.HelpOpen = true
	s.HelpScroll = 0
	s.HelpSearchMode = false
	s.HelpSearchQuery = ""
	s.HelpMatches = nil
	s.HelpMatchIdx = 0
}

func (s *State) CloseHelp() {
	s.HelpOpen = false
	s.HelpSearchMode = false
	s.HelpSearchQuery = ""
	s.HelpMatches = nil
}

func (s *State) ResizeHelp(delta int) {
	w := s.HelpWidth + delta
	if w < HelpWidthMin {
		w = HelpWidthMin
	}
	if w > HelpWidthMax {
		w = HelpWidthMax
	}
	if w == s.HelpWidth {
		return
	}
	s.HelpWidth = w
	if s.HelpOpen {
		if doc, err := help.Load(s.HelpKey(), s.HelpWidth); err == nil {
			s.HelpDoc = doc
			if s.HelpSearchQuery != "" {
				s.refreshSearchMatches()
			}
		}
	}
}

func (s *State) ScrollHelp(delta int) {
	if s.HelpDoc == nil {
		return
	}
	s.HelpScroll += delta
	s.clampScroll()
}

func (s *State) ScrollHelpTo(line int) {
	if s.HelpDoc == nil {
		return
	}
	s.HelpScroll = line
	s.clampScroll()
}

func (s *State) JumpHelpSection(delta int) {
	if s.HelpDoc == nil || len(s.HelpDoc.TOC) == 0 {
		return
	}
	current := -1
	for i, e := range s.HelpDoc.TOC {
		if e.Line <= s.HelpScroll {
			current = i
		} else {
			break
		}
	}
	next := current + delta
	if next < 0 {
		next = 0
	}
	if next >= len(s.HelpDoc.TOC) {
		next = len(s.HelpDoc.TOC) - 1
	}
	s.ScrollHelpTo(s.HelpDoc.TOC[next].Line)
}

func (s *State) clampScroll() {
	if s.HelpDoc == nil {
		s.HelpScroll = 0
		return
	}
	if s.HelpScroll < 0 {
		s.HelpScroll = 0
	}
	max := len(s.HelpDoc.Lines) - 1
	if s.HelpScroll > max {
		s.HelpScroll = max
	}
}

func (s *State) BeginHelpSearch() {
	if s.HelpDoc == nil {
		return
	}
	s.HelpSearchMode = true
	s.HelpSearchQuery = ""
}

func (s *State) AppendSearchRune(r rune) {
	if !s.HelpSearchMode {
		return
	}
	s.HelpSearchQuery += string(r)
}

func (s *State) BackspaceSearch() {
	if !s.HelpSearchMode || s.HelpSearchQuery == "" {
		return
	}
	q := []rune(s.HelpSearchQuery)
	s.HelpSearchQuery = string(q[:len(q)-1])
}

func (s *State) CommitHelpSearch() {
	s.HelpSearchMode = false
	s.refreshSearchMatches()
	if len(s.HelpMatches) > 0 {
		s.HelpMatchIdx = 0
		s.ScrollHelpTo(s.HelpMatches[0])
	}
}

func (s *State) CancelHelpSearch() {
	s.HelpSearchMode = false
	s.HelpSearchQuery = ""
	s.HelpMatches = nil
}

func (s *State) NextHelpMatch(delta int) {
	if len(s.HelpMatches) == 0 {
		return
	}
	s.HelpMatchIdx = (s.HelpMatchIdx + delta + len(s.HelpMatches)) % len(s.HelpMatches)
	s.ScrollHelpTo(s.HelpMatches[s.HelpMatchIdx])
}

func (s *State) refreshSearchMatches() {
	if s.HelpDoc == nil {
		s.HelpMatches = nil
		return
	}
	s.HelpMatches = s.HelpDoc.Search(s.HelpSearchQuery)
	s.HelpMatchIdx = 0
}
