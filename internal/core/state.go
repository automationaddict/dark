package core

import (
	"github.com/johnnelson/dark/internal/help"
	"github.com/johnnelson/dark/internal/services/appearance"
	"github.com/johnnelson/dark/internal/services/appstore"
	"github.com/johnnelson/dark/internal/services/firmware"
	"github.com/johnnelson/dark/internal/services/keybind"
	"github.com/johnnelson/dark/internal/services/limine"
	"github.com/johnnelson/dark/internal/services/links"
	"github.com/johnnelson/dark/internal/services/update"
	"github.com/johnnelson/dark/internal/services/audio"
	"github.com/johnnelson/dark/internal/services/bluetooth"
	"github.com/johnnelson/dark/internal/services/display"
	"github.com/johnnelson/dark/internal/services/datetime"
	inputsvc "github.com/johnnelson/dark/internal/services/input"
	"github.com/johnnelson/dark/internal/services/notifycfg"
	"github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/services/power"
	"github.com/johnnelson/dark/internal/services/privacy"
	"github.com/johnnelson/dark/internal/services/screensaver"
	"github.com/johnnelson/dark/internal/services/sysinfo"
	"github.com/johnnelson/dark/internal/services/topbar"
	"github.com/johnnelson/dark/internal/services/users"
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
	ActiveTab          TabID
	SettingsFocus      int
	SidebarItemWidth   int
	ContentScroll      int
	ContentScrollMax   int
	Rebuilding       bool
	BuildError       string
	RestartRequested bool
	BusConnected     bool
	SysInfo          sysinfo.SystemInfo
	SysInfoLoaded    bool
	AboutSectionIdx  int
	Wifi                wifi.Snapshot
	WifiLoaded          bool
	WifiSelected        int
	WifiNetworkSelected int
	WifiKnownSelected   int
	WifiFocus           WifiFocus
	WifiSectionIdx      int
	WifiContentFocused  bool
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
	BluetoothFocus            BluetoothFocus
	BluetoothSectionIdx       int
	BluetoothContentFocused   bool
	BluetoothDetailsOpen    bool
	BluetoothDeviceInfoOpen bool
	BluetoothBusy           bool
	BluetoothActionError    string
	BluetoothScanFilter     bluetooth.DiscoveryFilter

	Network              network.Snapshot
	NetworkLoaded        bool
	NetworkSelected      int
	NetworkSectionIdx    int
	NetworkContentFocused bool
	NetworkRoutesOpen    bool
	NetworkRouteSelected int
	NetworkBusy          bool
	NetworkActionError   string

	Display               display.Snapshot
	DisplayLoaded         bool
	DisplayMonitorIdx     int
	DisplaySectionIdx     int
	DisplayContentFocused bool
	DisplayFocus          DisplayFocus
	DisplayInfoOpen       bool
	DisplayLayoutOpen     bool
	DisplayBusy           bool
	DisplayActionError    string
	NightLightActive      bool
	NightLightTemp        int
	NightLightGamma       int

	Audio                 audio.Snapshot
	AudioLoaded           bool
	AudioLevels           audio.Levels
	AudioFocus            AudioFocus
	AudioSectionIdx       int
	AudioContentFocused   bool
	AudioSinkIdx          int
	AudioSourceIdx        int
	AudioPlayAppIdx       int
	AudioRecordAppIdx     int
	AudioDeviceInfoOpen   bool
	AudioBusy             bool
	AudioActionError      string

	Power           power.Snapshot
	PowerLoaded     bool
	PowerFocus      PowerFocus
	PowerSectionIdx int

	InputDevices       inputsvc.Snapshot
	InputDevicesLoaded bool
	InputSectionIdx    int

	Notify           notifycfg.Snapshot
	NotifyLoaded     bool
	NotifySectionIdx int

	DateTime          datetime.Snapshot
	DateTimeLoaded    bool
	DateTimeSectionIdx int

	Privacy          privacy.Snapshot
	PrivacyLoaded    bool
	PrivacySectionIdx int

	Users              users.Snapshot
	UsersLoaded        bool
	UsersIdx           int
	UsersSectionIdx    int
	UsersContentFocused bool

	Appearance          appearance.Snapshot
	AppearanceLoaded    bool
	AppearanceSectionIdx int

	Screensaver         screensaver.Snapshot
	ScreensaverLoaded   bool
	ScreensaverBusy     bool
	ScreensaverPreviewing bool
	ScreensaverActionError string

	TopBar            topbar.Snapshot
	TopBarLoaded      bool
	TopBarBusy        bool
	TopBarActionError string

	F2SidebarIdx          int
	Appstore              appstore.Snapshot
	AppstoreLoaded        bool
	AppstoreCategoryIdx   int
	AppstoreResults       appstore.SearchResult
	AppstoreResultsLoaded bool
	AppstoreResultIdx     int
	AppstoreDetail        appstore.Detail
	AppstoreDetailLoaded  bool
	AppstoreDetailOpen    bool
	AppstoreDetailScroll  int
	AppstoreDetailLines   int
	AppstoreDetailViewH   int
	AppstoreSearchInput   string
	AppstoreSearchActive  bool
	AppstoreFocus         AppstoreFocus
	AppstoreStatusMsg     string
	AppstoreBusy          bool
	AppstoreIncludeAUR    bool

	Update              update.Snapshot
	UpdateLoaded        bool
	UpdateBusy          bool
	UpdateResult        *update.RunResult
	UpdateStatusMsg     string
	UpdateSectionIdx    int

	Firmware            firmware.Snapshot
	FirmwareLoaded      bool
	FirmwareDeviceIdx   int

	WebLinks          []links.WebLink
	TUILinks          []links.TUILink
	HelpLinks         []links.HelpLink
	LinksLoaded       bool
	WebLinkIdx        int
	TUILinkIdx        int
	HelpLinkIdx       int
	Keybindings       keybind.Snapshot
	KeybindingsLoaded bool
	KeybindIdx          int
	KeybindFilter       int // 0=All, 1=Default, 2=User
	KeybindTableFocused bool
	OmarchySidebarIdx    int
	OmarchyLinksIdx      int
	OmarchyLinksFocused  bool

	Limine               limine.Snapshot
	LimineLoaded         bool
	LimineSubIdx         int
	LimineContentFocused bool
	LimineSnapshotIdx    int
	LimineBootCfgIdx     int
	LimineSyncCfgIdx     int
	LimineOmarchyCfgIdx  int
	LimineBusy           bool
	LimineActionError    string

	ContentFocused bool

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

func (s *State) SetDateTime(snap datetime.Snapshot) {
	s.DateTime = snap
	s.DateTimeLoaded = true
}

func (s *State) SetPrivacy(snap privacy.Snapshot) {
	s.Privacy = snap
	s.PrivacyLoaded = true
}

func (s *State) SetUsers(snap users.Snapshot) {
	s.Users = snap
	s.UsersLoaded = true
	if s.UsersIdx >= len(snap.Users) {
		s.UsersIdx = 0
	}
}

func (s *State) MoveUsersIdx(delta int) {
	n := len(s.Users.Users)
	if n == 0 {
		return
	}
	s.UsersIdx = (s.UsersIdx + delta + n) % n
}

func (s *State) SelectedUser() (users.User, bool) {
	if len(s.Users.Users) == 0 {
		return users.User{}, false
	}
	if s.UsersIdx >= len(s.Users.Users) {
		s.UsersIdx = 0
	}
	return s.Users.Users[s.UsersIdx], true
}

func (s *State) SetNotify(snap notifycfg.Snapshot) {
	s.Notify = snap
	s.NotifyLoaded = true
}

func (s *State) SetInputDevices(snap inputsvc.Snapshot) {
	s.InputDevices = snap
	s.InputDevicesLoaded = true
}

func (s *State) SetPower(snap power.Snapshot) {
	s.Power = snap
	s.PowerLoaded = true
}

// SetSysInfo replaces the cached system snapshot with one received from the
// daemon. The TUI no longer gathers locally — darkd is the source of truth.
func (s *State) SetSysInfo(info sysinfo.SystemInfo) {
	s.SysInfo = info
	s.SysInfoLoaded = true
}
