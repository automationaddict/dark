package core

import "github.com/automationaddict/dark/internal/services/wifi"

// SetWifi replaces the cached wifi snapshot with one received from darkd.
// Selection indices are clamped to the new list sizes so a Forget or a
// plugged-out adapter doesn't leave an out-of-bounds cursor. Also
// appends the current RSSI to each adapter's rolling history so the
// Details view can render a signal sparkline.
func (s *State) SetWifi(snap wifi.Snapshot) {
	s.Wifi = snap
	s.WifiLoaded = true
	s.appendRSSIHistory(snap)
	if s.WifiSelected >= len(snap.Adapters) {
		s.WifiSelected = 0
	}
	if s.WifiKnownSelected >= len(snap.KnownNetworks) {
		s.WifiKnownSelected = 0
	}
	if len(snap.Adapters) > 0 {
		if s.WifiNetworkSelected >= len(snap.Adapters[s.WifiSelected].Networks) {
			s.WifiNetworkSelected = 0
		}
	}
}

// appendRSSIHistory pushes the current RSSI for each adapter onto its
// rolling buffer. Disconnected adapters (RSSI = 0) are skipped so the
// buffer doesn't get "drawn down" by transient disconnects.
func (s *State) appendRSSIHistory(snap wifi.Snapshot) {
	if s.RSSIHistory == nil {
		s.RSSIHistory = map[string][]int16{}
	}
	for _, a := range snap.Adapters {
		if a.RSSI == 0 {
			continue
		}
		hist := s.RSSIHistory[a.Name]
		hist = append(hist, a.RSSI)
		if len(hist) > RSSIHistoryLen {
			hist = hist[len(hist)-RSSIHistoryLen:]
		}
		s.RSSIHistory[a.Name] = hist
	}
}

// MoveWifiSelection advances the selected adapter row, wrapping at the ends.
func (s *State) MoveWifiSelection(delta int) {
	n := len(s.Wifi.Adapters)
	if n == 0 {
		return
	}
	s.WifiSelected = (s.WifiSelected + delta + n) % n
}

// SelectedAdapter returns the currently highlighted adapter. The bool is
// false when the wifi list is empty.
func (s *State) SelectedAdapter() (wifi.Adapter, bool) {
	if len(s.Wifi.Adapters) == 0 {
		return wifi.Adapter{}, false
	}
	if s.WifiSelected >= len(s.Wifi.Adapters) {
		s.WifiSelected = 0
	}
	return s.Wifi.Adapters[s.WifiSelected], true
}

// OpenWifiDetails drills into the currently highlighted adapter and shows
// the details panel. The network selection defaults to the currently
// connected network if there is one, otherwise the first in the list.
func (s *State) OpenWifiDetails() {
	if !s.ContentFocused || s.ActiveSection().ID != "wifi" || len(s.Wifi.Adapters) == 0 {
		return
	}
	if s.WifiSelected >= len(s.Wifi.Adapters) {
		s.WifiSelected = 0
	}
	s.WifiDetailsOpen = true
	s.WifiFocus = WifiFocusNetworks
	s.WifiNetworkSelected = 0
	s.WifiKnownSelected = 0
	adapter := s.Wifi.Adapters[s.WifiSelected]
	for i, n := range adapter.Networks {
		if n.Connected {
			s.WifiNetworkSelected = i
			break
		}
	}
}

// CycleWifiFocus cycles through Adapters -> Networks -> Known Networks.
// Scroll reset is delegated to the TUI render pass, which knows the
// actual measured line offsets for each sub-table.
func (s *State) CycleWifiFocus() {
	if !s.WifiDetailsOpen {
		return
	}
	switch s.WifiFocus {
	case WifiFocusAdapters:
		s.WifiFocus = WifiFocusNetworks
	case WifiFocusNetworks:
		s.WifiFocus = WifiFocusKnown
	default:
		s.WifiFocus = WifiFocusAdapters
	}
	s.ContentScroll = 0
}

// MoveWifiNetworkSelection walks the network row highlight up or down
// within the selected adapter's scan list. No-op when there are no
// networks to move between.
func (s *State) MoveWifiNetworkSelection(delta int) {
	adapter, ok := s.SelectedAdapter()
	if !ok {
		return
	}
	n := len(adapter.Networks)
	if n == 0 {
		return
	}
	s.WifiNetworkSelected = (s.WifiNetworkSelected + delta + n) % n
}

// SelectedNetwork returns the currently highlighted network on the
// selected adapter. Returns false when the current adapter has no
// networks cached.
func (s *State) SelectedNetwork() (wifi.Network, bool) {
	adapter, ok := s.SelectedAdapter()
	if !ok || len(adapter.Networks) == 0 {
		return wifi.Network{}, false
	}
	if s.WifiNetworkSelected >= len(adapter.Networks) {
		s.WifiNetworkSelected = 0
	}
	return adapter.Networks[s.WifiNetworkSelected], true
}

// MoveWifiKnownSelection moves the highlight within the Known Networks
// list. No-op when the list is empty.
func (s *State) MoveWifiKnownSelection(delta int) {
	n := len(s.Wifi.KnownNetworks)
	if n == 0 {
		return
	}
	s.WifiKnownSelected = (s.WifiKnownSelected + delta + n) % n
}

// SelectedKnownNetwork returns the highlighted saved profile.
func (s *State) SelectedKnownNetwork() (wifi.KnownNetwork, bool) {
	n := len(s.Wifi.KnownNetworks)
	if n == 0 {
		return wifi.KnownNetwork{}, false
	}
	if s.WifiKnownSelected >= n {
		s.WifiKnownSelected = 0
	}
	return s.Wifi.KnownNetworks[s.WifiKnownSelected], true
}

// CloseWifiDetails hides the details panel but keeps content focus so the
// user can keep navigating adapters.
func (s *State) CloseWifiDetails() {
	s.WifiDetailsOpen = false
}
