package core

import "github.com/johnnelson/dark/internal/services/appearance"

func (s *State) SetAppearance(snap appearance.Snapshot) {
	s.Appearance = snap
	s.AppearanceLoaded = true
}
