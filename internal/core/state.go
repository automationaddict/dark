package core

import (
	"github.com/johnnelson/dark/internal/help"
	"github.com/johnnelson/dark/internal/services/audio"
	"github.com/johnnelson/dark/internal/services/bluetooth"
	"github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/services/sysinfo"
	"github.com/johnnelson/dark/internal/services/wifi"
)

const (
	SidebarItemWidthMin     = 14
	SidebarItemWidthMax     = 40
	SidebarItemWidthDefault = 22

	HelpWidthMin     = 28
	HelpWidthMax     = 80
	HelpWidthDefault = 60
)

// WifiFocus identifies which sub-table owns j/k and the action keys
// while the Wi-Fi section has content focus. Tab cycles between all
// three: Adapters → Networks → Known Networks.
type WifiFocus string

const (
	WifiFocusAdapters WifiFocus = "adapters"
	WifiFocusNetworks WifiFocus = "networks"
	WifiFocusKnown    WifiFocus = "known"
)

type State struct {
	ActiveTab        TabID
	SettingsFocus    int
	SidebarItemWidth int
	Rebuilding       bool
	BuildError       string
	RestartRequested bool
	BusConnected     bool
	SysInfo          sysinfo.SystemInfo
	SysInfoLoaded    bool
	Wifi                wifi.Snapshot
	WifiLoaded          bool
	WifiSelected        int
	WifiNetworkSelected int
	WifiKnownSelected   int
	WifiFocus           WifiFocus
	WifiDetailsOpen     bool
	WifiScanning        bool
	WifiScanError       string
	WifiBusy            bool
	WifiActionError     string
	RSSIHistory         map[string][]int16

	Bluetooth               bluetooth.Snapshot
	BluetoothLoaded         bool
	BluetoothSelected       int
	BluetoothDevSelected    int
	BluetoothFocus          BluetoothFocus
	BluetoothDetailsOpen    bool
	BluetoothDeviceInfoOpen bool
	BluetoothBusy           bool
	BluetoothActionError    string
	BluetoothScanFilter     bluetooth.DiscoveryFilter

	Network              network.Snapshot
	NetworkLoaded        bool
	NetworkSelected      int
	NetworkRoutesOpen    bool
	NetworkRouteSelected int
	NetworkBusy          bool
	NetworkActionError   string

	Audio                 audio.Snapshot
	AudioLoaded           bool
	AudioLevels           audio.Levels
	AudioFocus            AudioFocus
	AudioSinkIdx          int
	AudioSourceIdx        int
	AudioPlayAppIdx       int
	AudioRecordAppIdx     int
	AudioDeviceInfoOpen   bool
	AudioBusy             bool
	AudioActionError      string

	ContentFocused bool

	// SkipAutoExpand suppresses the single-adapter auto-drill on the
	// first snapshot of a section. Set by the restart path so ctrl+r
	// returns the user to the sidebar instead of dropping them back
	// into a drill-in they already navigated out of.
	SkipAutoExpand bool

	HelpOpen        bool
	HelpWidth       int
	HelpDoc         *help.Document
	HelpScroll      int
	HelpSearchMode  bool
	HelpSearchQuery string
	HelpMatches     []int
	HelpMatchIdx    int

	binPath  string
	sections []SettingsSection
}

// RSSIHistoryLen is the maximum number of samples kept per adapter.
// One sample is appended on every wifi snapshot tick from darkd
// (currently 30s), so a full buffer covers roughly 10 minutes of
// recent signal strength.
const RSSIHistoryLen = 20

func NewState(start TabID, binPath string) *State {
	return &State{
		ActiveTab:        start,
		SidebarItemWidth: SidebarItemWidthDefault,
		HelpWidth:        HelpWidthDefault,
		BusConnected:     true, // dark exits early if the initial connect fails
		RSSIHistory:      map[string][]int16{},
		binPath:          binPath,
		sections:         SettingsSections(),
	}
}

// SetBusConnected updates the daemon connection indicator. Called from the
// bus subscriber goroutine via tea.Program.Send.
func (s *State) SetBusConnected(ok bool) {
	s.BusConnected = ok
}

// SetSysInfo replaces the cached system snapshot with one received from the
// daemon. The TUI no longer gathers locally — darkd is the source of truth.
func (s *State) SetSysInfo(info sysinfo.SystemInfo) {
	s.SysInfo = info
	s.SysInfoLoaded = true
}

// SetWifi replaces the cached wifi snapshot with one received from darkd.
// Selection indices are clamped to the new list sizes so a Forget or a
// plugged-out adapter doesn't leave an out-of-bounds cursor. Also
// appends the current RSSI to each adapter's rolling history so the
// Details view can render a signal sparkline.
func (s *State) SetWifi(snap wifi.Snapshot) {
	firstLoad := !s.WifiLoaded
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

	// On the very first wifi snapshot, if the user is already on the Wi-Fi
	// section and there's exactly one powered adapter, drill in
	// automatically so they don't have to press Enter twice. Later
	// snapshots don't retrigger, and an off radio never auto-drills.
	if firstLoad && !s.SkipAutoExpand && s.ActiveTab == TabSettings && s.ActiveSection().ID == "wifi" &&
		len(snap.Adapters) == 1 && snap.Adapters[0].Powered {
		s.autoExpandSingleAdapter()
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

// autoExpandSingleAdapter is the "there's only one possible choice, just
// open it" shortcut. Mirrors OpenWifiDetails + FocusContent but skips
// their user-interaction gates.
func (s *State) autoExpandSingleAdapter() {
	s.ContentFocused = true
	s.WifiDetailsOpen = true
	s.WifiFocus = WifiFocusNetworks
	s.WifiNetworkSelected = 0
	s.WifiKnownSelected = 0
	adapter := s.Wifi.Adapters[0]
	for i, n := range adapter.Networks {
		if n.Connected {
			s.WifiNetworkSelected = i
			break
		}
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

// FocusContent enters the content pane for the current section if that
// section exposes selectable content. No-op otherwise.
func (s *State) FocusContent() {
	if s.ActiveTab != TabSettings {
		return
	}
	switch s.ActiveSection().ID {
	case "wifi":
		if len(s.Wifi.Adapters) > 0 {
			s.ContentFocused = true
			s.WifiDetailsOpen = true
			if s.WifiFocus == "" {
				s.WifiFocus = WifiFocusNetworks
			}
			s.WifiNetworkSelected = 0
			s.WifiKnownSelected = 0
			if adapter := s.Wifi.Adapters[s.WifiSelected]; len(adapter.Networks) > 0 {
				for i, n := range adapter.Networks {
					if n.Connected {
						s.WifiNetworkSelected = i
						break
					}
				}
			}
		}
	case "bluetooth":
		if len(s.Bluetooth.Adapters) > 0 {
			s.ContentFocused = true
			s.BluetoothDetailsOpen = true
			if s.BluetoothFocus == "" {
				s.BluetoothFocus = BluetoothFocusDevices
			}
			s.BluetoothDevSelected = 0
			if adapter := s.Bluetooth.Adapters[s.BluetoothSelected]; len(adapter.Devices) > 0 {
				for i, d := range adapter.Devices {
					if d.Connected {
						s.BluetoothDevSelected = i
						break
					}
				}
			}
		}
	case "sound":
		if len(s.Audio.Sinks) > 0 || len(s.Audio.Sources) > 0 {
			s.ContentFocused = true
			if s.AudioFocus == "" {
				s.AudioFocus = AudioFocusSinks
			}
		}
	case "network":
		if len(s.Network.Interfaces) > 0 {
			s.ContentFocused = true
		}
	}
}

// FocusSidebar returns key routing to the sidebar.
func (s *State) FocusSidebar() {
	s.ContentFocused = false
	s.WifiDetailsOpen = false
	s.BluetoothDetailsOpen = false
	s.BluetoothDeviceInfoOpen = false
	s.AudioDeviceInfoOpen = false
	s.NetworkRoutesOpen = false
}

// OpenWifiDetails drills into the currently highlighted adapter and shows
// the details panel. The network selection defaults to the currently
// connected network if there is one, otherwise the first in the list.
func (s *State) OpenWifiDetails() {
	if !s.ContentFocused || s.ActiveSection().ID != "wifi" || len(s.Wifi.Adapters) == 0 {
		return
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

// CycleWifiFocus cycles through Adapters → Networks → Known Networks.
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

// HelpKey returns the context key for the currently visible view.
// The help package looks this up in its embedded content directory.
func (s *State) HelpKey() string {
	if s.ActiveTab == TabSettings {
		return s.ActiveSection().ID
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

func (s *State) ResizeSidebar(delta int) {
	w := s.SidebarItemWidth + delta
	if w < SidebarItemWidthMin {
		w = SidebarItemWidthMin
	}
	if w > SidebarItemWidthMax {
		w = SidebarItemWidthMax
	}
	s.SidebarItemWidth = w
}

func (s *State) Sections() []SettingsSection { return s.sections }

func (s *State) ActiveSection() SettingsSection {
	return s.sections[s.SettingsFocus]
}

func (s *State) SelectTab(id TabID) { s.ActiveTab = id }

func (s *State) MoveSettingsFocus(delta int) {
	n := len(s.sections)
	if n == 0 {
		return
	}
	s.SettingsFocus = (s.SettingsFocus + delta + n) % n
}
