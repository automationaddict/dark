package core

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
		}
	case "bluetooth":
		if len(s.Bluetooth.Adapters) > 0 {
			s.ContentFocused = true
		}
	case "display":
		if len(s.Display.Monitors) > 0 {
			s.ContentFocused = true
			if s.DisplayFocus == "" {
				s.DisplayFocus = DisplayFocusMonitors
			}
		}
	case "sound":
		if len(s.Audio.Sinks) > 0 || len(s.Audio.Sources) > 0 {
			s.ContentFocused = true
			if s.AudioFocus == "" {
				s.AudioFocus = AudioFocusSinks
			}
		}
	case "power":
		if s.PowerLoaded {
			s.ContentFocused = true
		}
	case "input":
		if s.InputDevicesLoaded {
			s.ContentFocused = true
		}
	case "notifications":
		if s.NotifyLoaded {
			s.ContentFocused = true
		}
	case "datetime":
		if s.DateTimeLoaded {
			s.ContentFocused = true
		}
	case "network":
		if len(s.Network.Interfaces) > 0 {
			s.ContentFocused = true
		}
	case "privacy":
		if s.PrivacyLoaded {
			s.ContentFocused = true
		}
	case "users":
		if s.UsersLoaded {
			s.ContentFocused = true
		}
	case "appearance":
		if s.AppearanceLoaded {
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
	s.DisplayInfoOpen = false
	s.DisplayLayoutOpen = false
	s.AudioDeviceInfoOpen = false
	s.NetworkRoutesOpen = false
	s.AppstoreDetailOpen = false
	s.AppstoreFocus = AppstoreFocusSidebar
	s.KeybindTableFocused = false
	s.WifiContentFocused = false
	s.BluetoothContentFocused = false
}

func (s *State) MoveSettingsFocus(delta int) {
	n := len(s.sections)
	if n == 0 {
		return
	}
	s.SettingsFocus = (s.SettingsFocus + delta + n) % n
	s.ContentScroll = 0
}

func (s *State) ScrollContent(delta int) {
	s.ContentScroll += delta
	if s.ContentScroll < 0 {
		s.ContentScroll = 0
	}
	if s.ContentScroll > s.ContentScrollMax {
		s.ContentScroll = s.ContentScrollMax
	}
}

func (s *State) Sections() []SettingsSection { return s.sections }

func (s *State) ActiveSection() SettingsSection {
	return s.sections[s.SettingsFocus]
}

func (s *State) SelectTab(id TabID) { s.ActiveTab = id }

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
