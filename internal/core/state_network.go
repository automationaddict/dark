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

// OpenNetworkRoutes drills into the routes-management view for the
// highlighted interface. Only valid when the interface has a dark-
// managed config (otherwise there's nothing to manage routes against
// — the user needs to set up basic IPv4 first via h or e).
func (s *State) OpenNetworkRoutes() bool {
	iface, ok := s.SelectedNetworkInterface()
	if !ok || iface.Managed == nil {
		return false
	}
	s.NetworkRoutesOpen = true
	s.NetworkRouteSelected = 0
	return true
}

// CloseNetworkRoutes backs out of the routes drill-in.
func (s *State) CloseNetworkRoutes() {
	s.NetworkRoutesOpen = false
}

// MoveNetworkRouteSelection walks the route highlight up or down
// within the highlighted interface's dark-managed route list.
func (s *State) MoveNetworkRouteSelection(delta int) {
	iface, ok := s.SelectedNetworkInterface()
	if !ok || iface.Managed == nil {
		return
	}
	n := len(iface.Managed.Routes)
	if n == 0 {
		return
	}
	s.NetworkRouteSelected = (s.NetworkRouteSelected + delta + n) % n
}

// SelectedNetworkRoute returns the currently highlighted route on the
// selected interface's dark-managed route list, plus the route's
// index within that list. Returns false when there's no current
// selection (no routes, no managed config, or no selected interface).
func (s *State) SelectedNetworkRoute() (network.RouteConfig, int, bool) {
	iface, ok := s.SelectedNetworkInterface()
	if !ok || iface.Managed == nil || len(iface.Managed.Routes) == 0 {
		return network.RouteConfig{}, 0, false
	}
	if s.NetworkRouteSelected >= len(iface.Managed.Routes) {
		s.NetworkRouteSelected = 0
	}
	return iface.Managed.Routes[s.NetworkRouteSelected], s.NetworkRouteSelected, true
}
