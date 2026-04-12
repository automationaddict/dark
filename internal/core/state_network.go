package core

import "github.com/johnnelson/dark/internal/services/network"

// SetNetwork replaces the cached network snapshot with one received
// from darkd. Selection index is clamped so an interface that hot-
// unplugs doesn't leave an out-of-bounds cursor.
func (s *State) SetNetwork(snap network.Snapshot) {
	s.Network = snap
	s.NetworkLoaded = true
	if s.NetworkSelected >= len(snap.Interfaces) {
		s.NetworkSelected = 0
	}
}

// MoveNetworkSelection walks the interface row highlight, wrapping
// at the ends.
func (s *State) MoveNetworkSelection(delta int) {
	n := len(s.Network.Interfaces)
	if n == 0 {
		return
	}
	s.NetworkSelected = (s.NetworkSelected + delta + n) % n
}

// SelectedNetworkInterface returns the currently highlighted interface
// or false when the list is empty.
func (s *State) SelectedNetworkInterface() (network.Interface, bool) {
	if len(s.Network.Interfaces) == 0 {
		return network.Interface{}, false
	}
	if s.NetworkSelected >= len(s.Network.Interfaces) {
		s.NetworkSelected = 0
	}
	return s.Network.Interfaces[s.NetworkSelected], true
}
