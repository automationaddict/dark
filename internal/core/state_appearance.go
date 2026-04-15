package core

import "github.com/automationaddict/dark/internal/services/appearance"

func (s *State) SetAppearance(snap appearance.Snapshot) {
	s.Appearance = snap
	s.AppearanceLoaded = true
}
