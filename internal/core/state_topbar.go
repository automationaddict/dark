package core

import "github.com/johnnelson/dark/internal/services/topbar"

// SetTopBar replaces the cached top bar snapshot. Called from the
// NATS subscriber when darkd publishes a new snapshot, and from the
// action-result handler after an edit command completes.
func (s *State) SetTopBar(snap topbar.Snapshot) {
	s.TopBar = snap
	s.TopBarLoaded = true
	s.TopBarBusy = false
}
