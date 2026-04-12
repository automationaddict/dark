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
	s.NightLightActive = snap.NightLightActive
	if snap.NightLightTemp > 0 {
		s.NightLightTemp = snap.NightLightTemp
	} else if s.NightLightTemp == 0 {
		s.NightLightTemp = 4500
	}
	if snap.NightLightGamma > 0 {
		s.NightLightGamma = snap.NightLightGamma
	} else if s.NightLightGamma == 0 {
		s.NightLightGamma = 100
	}
	if s.DisplayFocus == "" {
		s.DisplayFocus = DisplayFocusMonitors
	}
	if s.DisplayMonitorIdx >= len(snap.Monitors) {
		s.DisplayMonitorIdx = 0
	}
}

func (s *State) SetNightLight(active bool, temp int) {
	s.NightLightActive = active
	s.NightLightTemp = temp
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
	if len(s.Display.Monitors) < 2 {
		s.DisplayActionError = "layout view requires multiple monitors"
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
