package core

import "github.com/automationaddict/dark/internal/services/screensaver"

// SetScreensaver replaces the cached screensaver snapshot and clears
// the busy flag. Called from the TUI's AppstoreActionResultMsg-style
// message handlers after a set-enabled / set-content / preview reply
// arrives, and from the periodic snapshot publish path.
func (s *State) SetScreensaver(snap screensaver.Snapshot) {
	s.Screensaver = snap
	s.ScreensaverLoaded = true
	s.ScreensaverBusy = false
}
