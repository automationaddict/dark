package core

import "github.com/johnnelson/dark/internal/services/display"

type DisplayFocus string

const (
	DisplayFocusMonitors DisplayFocus = "monitors"
	DisplayFocusLayout   DisplayFocus = "layout"
)

// SetDisplay replaces the cached display snapshot. Selection indices
// are clamped to the new list sizes.
func (s *State) SetDisplay(snap display.Snapshot) {
	s.Display = snap
	s.DisplayLoaded = true
	if s.DisplayFocus == "" {
		s.DisplayFocus = DisplayFocusMonitors
	}
	if s.DisplayMonitorIdx >= len(snap.Monitors) {
		s.DisplayMonitorIdx = 0
	}
}

// MoveDisplaySelection walks the selected monitor row, wrapping at ends.
func (s *State) MoveDisplaySelection(delta int) {
	n := len(s.Display.Monitors)
	if n == 0 {
		return
	}
	s.DisplayMonitorIdx = (s.DisplayMonitorIdx + delta + n) % n
}

// SelectedMonitor returns the highlighted monitor.
func (s *State) SelectedMonitor() (display.Monitor, bool) {
	if len(s.Display.Monitors) == 0 {
		return display.Monitor{}, false
	}
	if s.DisplayMonitorIdx >= len(s.Display.Monitors) {
		s.DisplayMonitorIdx = 0
	}
	return s.Display.Monitors[s.DisplayMonitorIdx], true
}

// OpenDisplayInfo drills into the info panel for the selected monitor.
func (s *State) OpenDisplayInfo() {
	if !s.ContentFocused || s.ActiveSection().ID != "display" {
		return
	}
	if _, ok := s.SelectedMonitor(); !ok {
		return
	}
	s.DisplayInfoOpen = true
}

// CloseDisplayInfo backs out of the monitor info panel.
func (s *State) CloseDisplayInfo() {
	s.DisplayInfoOpen = false
}

// OpenDisplayLayout enters the visual arrangement mode.
func (s *State) OpenDisplayLayout() {
	if !s.ContentFocused || s.ActiveSection().ID != "display" {
		return
	}
	if len(s.Display.Monitors) < 1 {
		return
	}
	s.DisplayLayoutOpen = true
	s.DisplayFocus = DisplayFocusLayout
}

// CloseDisplayLayout exits the visual arrangement mode.
func (s *State) CloseDisplayLayout() {
	s.DisplayLayoutOpen = false
	s.DisplayFocus = DisplayFocusMonitors
}
