package core

import (
	"testing"

	"github.com/automationaddict/dark/internal/services/audio"
	"github.com/automationaddict/dark/internal/services/bluetooth"
	"github.com/automationaddict/dark/internal/services/network"
	"github.com/automationaddict/dark/internal/services/wifi"
)

// --- Wi-Fi state ---

func TestMoveWifiSelection_Wraps(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetWifi(wifi.Snapshot{Adapters: []wifi.Adapter{{Name: "a"}, {Name: "b"}, {Name: "c"}}})

	s.MoveWifiSelection(1)
	if s.WifiSelected != 1 {
		t.Errorf("after +1: got %d, want 1", s.WifiSelected)
	}
	s.MoveWifiSelection(1)
	s.MoveWifiSelection(1)
	if s.WifiSelected != 0 {
		t.Errorf("after wrapping: got %d, want 0", s.WifiSelected)
	}
	s.MoveWifiSelection(-1)
	if s.WifiSelected != 2 {
		t.Errorf("after -1 from 0: got %d, want 2", s.WifiSelected)
	}
}

func TestMoveWifiSelection_EmptyList(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.MoveWifiSelection(1) // should not panic
	if s.WifiSelected != 0 {
		t.Errorf("got %d, want 0", s.WifiSelected)
	}
}

func TestSetWifi_ClampsIndices(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.WifiSelected = 5
	s.WifiNetworkSelected = 10
	s.WifiKnownSelected = 10
	s.SetWifi(wifi.Snapshot{
		Adapters:      []wifi.Adapter{{Name: "a", Networks: []wifi.Network{{SSID: "x"}}}},
		KnownNetworks: []wifi.KnownNetwork{{SSID: "y"}},
	})
	if s.WifiSelected != 0 {
		t.Errorf("WifiSelected not clamped: %d", s.WifiSelected)
	}
	if s.WifiNetworkSelected != 0 {
		t.Errorf("WifiNetworkSelected not clamped: %d", s.WifiNetworkSelected)
	}
	if s.WifiKnownSelected != 0 {
		t.Errorf("WifiKnownSelected not clamped: %d", s.WifiKnownSelected)
	}
}

func TestCycleWifiFocus(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.WifiDetailsOpen = true
	s.WifiFocus = WifiFocusAdapters

	s.CycleWifiFocus()
	if s.WifiFocus != WifiFocusNetworks {
		t.Errorf("after first cycle: %s", s.WifiFocus)
	}
	s.CycleWifiFocus()
	if s.WifiFocus != WifiFocusKnown {
		t.Errorf("after second cycle: %s", s.WifiFocus)
	}
	s.CycleWifiFocus()
	if s.WifiFocus != WifiFocusAdapters {
		t.Errorf("after third cycle: %s", s.WifiFocus)
	}
}

func TestSelectedAdapter_Empty(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	_, ok := s.SelectedAdapter()
	if ok {
		t.Error("expected false for empty list")
	}
}

func TestSelectedAdapter_OutOfBounds(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetWifi(wifi.Snapshot{Adapters: []wifi.Adapter{{Name: "a"}}})
	s.WifiSelected = 99
	a, ok := s.SelectedAdapter()
	if !ok || a.Name != "a" {
		t.Errorf("got %v, %v", a.Name, ok)
	}
	if s.WifiSelected != 0 {
		t.Errorf("not clamped: %d", s.WifiSelected)
	}
}

func TestOpenWifiDetails_ClampsSelection(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetWifi(wifi.Snapshot{Adapters: []wifi.Adapter{{Name: "a"}}})
	s.WifiSelected = 5 // out of bounds
	s.ContentFocused = true
	s.OpenWifiDetails()
	if s.WifiSelected != 0 {
		t.Errorf("WifiSelected not clamped: %d", s.WifiSelected)
	}
	if !s.WifiDetailsOpen {
		t.Error("details should be open")
	}
}

func TestOpenWifiDetails_EmptyAdapters(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.ContentFocused = true
	s.OpenWifiDetails() // should not panic
	if s.WifiDetailsOpen {
		t.Error("details should not open with empty adapters")
	}
}

// --- Bluetooth state ---

func TestSetBluetooth_ClampsIndices(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.BluetoothSelected = 5
	s.BluetoothDevSelected = 5
	s.SetBluetooth(bluetooth.Snapshot{
		Adapters: []bluetooth.Adapter{{Name: "hci0", Devices: []bluetooth.Device{{Address: "aa"}}}},
	})
	if s.BluetoothSelected != 0 {
		t.Errorf("BluetoothSelected not clamped: %d", s.BluetoothSelected)
	}
	if s.BluetoothDevSelected != 0 {
		t.Errorf("BluetoothDevSelected not clamped: %d", s.BluetoothDevSelected)
	}
}

func TestCycleBluetoothFocus(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.BluetoothFocus = BluetoothFocusAdapters
	s.CycleBluetoothFocus()
	if s.BluetoothFocus != BluetoothFocusDevices {
		t.Errorf("expected devices, got %s", s.BluetoothFocus)
	}
	s.CycleBluetoothFocus()
	if s.BluetoothFocus != BluetoothFocusAdapters {
		t.Errorf("expected adapters, got %s", s.BluetoothFocus)
	}
}

func TestOpenBluetoothDetails_ClampsSelection(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetBluetooth(bluetooth.Snapshot{
		Adapters: []bluetooth.Adapter{{Name: "hci0", Devices: []bluetooth.Device{{Address: "aa"}}}},
	})
	s.BluetoothSelected = 5 // out of bounds
	s.ContentFocused = true
	s.SettingsFocus = 1 // bluetooth section
	s.OpenBluetoothDetails()
	if s.BluetoothSelected != 0 {
		t.Errorf("BluetoothSelected not clamped: %d", s.BluetoothSelected)
	}
	if !s.BluetoothDetailsOpen {
		t.Error("details should be open")
	}
}

func TestOpenBluetoothDetails_EmptyAdapters(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.ContentFocused = true
	s.SettingsFocus = 1
	s.OpenBluetoothDetails() // should not panic
	if s.BluetoothDetailsOpen {
		t.Error("details should not open with empty adapters")
	}
}

func TestSelectedBluetoothDevice_Empty(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetBluetooth(bluetooth.Snapshot{Adapters: []bluetooth.Adapter{{Name: "hci0"}}})
	_, ok := s.SelectedBluetoothDevice()
	if ok {
		t.Error("expected false for empty device list")
	}
}

// --- Audio state ---

func TestSyncAudioFocus_FromSection(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetAudio(audio.Snapshot{
		Sinks:   []audio.Device{{Name: "speakers"}},
		Sources: []audio.Device{{Name: "mic"}},
	})

	s.AudioSectionIdx = 0
	s.SyncAudioFocus()
	if s.AudioFocus != AudioFocusSinks {
		t.Errorf("section 0 should sync to sinks, got %s", s.AudioFocus)
	}

	s.AudioSectionIdx = 1
	s.SyncAudioFocus()
	if s.AudioFocus != AudioFocusSources {
		t.Errorf("section 1 should sync to sources, got %s", s.AudioFocus)
	}
}

func TestMoveAudioSection_Wraps(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.AudioSectionIdx = 3

	s.MoveAudioSection(1)
	if s.AudioSectionIdx != 0 {
		t.Errorf("expected wrap to 0, got %d", s.AudioSectionIdx)
	}

	s.MoveAudioSection(-1)
	if s.AudioSectionIdx != 3 {
		t.Errorf("expected wrap to 3, got %d", s.AudioSectionIdx)
	}
}

func TestSetAudio_SyncsFocusFromSection(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.AudioSectionIdx = 2 // play_apps
	s.SetAudio(audio.Snapshot{
		Sinks:      []audio.Device{{Name: "out"}},
		SinkInputs: []audio.Stream{}, // empty — section stays, focus syncs
	})
	// AudioFocus should match the section, not fall back.
	if s.AudioFocus != AudioFocusPlayApps {
		t.Errorf("focus should sync to play_apps from section, got %s", s.AudioFocus)
	}
}

func TestSelectedAudioDevice_SinkFocus(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetAudio(audio.Snapshot{
		Sinks: []audio.Device{{Name: "out", Index: 42}},
	})
	s.AudioFocus = AudioFocusSinks
	dev, isSink, ok := s.SelectedAudioDevice()
	if !ok || !isSink || dev.Index != 42 {
		t.Errorf("got %v, %v, %v", dev.Index, isSink, ok)
	}
}

func TestSelectedAudioDevice_SourceFocus(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetAudio(audio.Snapshot{
		Sources: []audio.Device{{Name: "mic", Index: 7}},
	})
	s.AudioFocus = AudioFocusSources
	dev, isSink, ok := s.SelectedAudioDevice()
	if !ok || isSink || dev.Index != 7 {
		t.Errorf("got %v, %v, %v", dev.Index, isSink, ok)
	}
}

// --- Network state ---

func TestSetNetwork_ClampsIndex(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.NetworkSelected = 10
	s.SetNetwork(network.Snapshot{
		Interfaces: []network.Interface{{Name: "eth0"}},
	})
	if s.NetworkSelected != 0 {
		t.Errorf("not clamped: %d", s.NetworkSelected)
	}
}

func TestMoveNetworkSelection_Wraps(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetNetwork(network.Snapshot{
		Interfaces: []network.Interface{{Name: "lo"}, {Name: "eth0"}},
	})
	s.MoveNetworkSelection(1)
	if s.NetworkSelected != 1 {
		t.Errorf("got %d, want 1", s.NetworkSelected)
	}
	s.MoveNetworkSelection(1)
	if s.NetworkSelected != 0 {
		t.Errorf("got %d, want 0 (wrap)", s.NetworkSelected)
	}
}

func TestOpenNetworkRoutes_RequiresManaged(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.ContentFocused = true
	s.SetNetwork(network.Snapshot{
		Interfaces: []network.Interface{{Name: "eth0"}}, // no Managed
	})
	s.SettingsFocus = 2 // network section index
	if s.OpenNetworkRoutes() {
		t.Error("should fail without Managed config")
	}
}

// --- FocusSidebar resets ---

func TestFocusSidebar_ResetsAll(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.ContentFocused = true
	s.WifiDetailsOpen = true
	s.BluetoothDetailsOpen = true
	s.BluetoothDeviceInfoOpen = true
	s.AudioDeviceInfoOpen = true
	s.NetworkRoutesOpen = true
	s.AppstoreDetailOpen = true
	s.AppstoreFocus = AppstoreFocusResults

	s.FocusSidebar()

	if s.ContentFocused {
		t.Error("ContentFocused should be false")
	}
	if s.WifiDetailsOpen {
		t.Error("WifiDetailsOpen should be false")
	}
	if s.BluetoothDetailsOpen {
		t.Error("BluetoothDetailsOpen should be false")
	}
	if s.BluetoothDeviceInfoOpen {
		t.Error("BluetoothDeviceInfoOpen should be false")
	}
	if s.AudioDeviceInfoOpen {
		t.Error("AudioDeviceInfoOpen should be false")
	}
	if s.NetworkRoutesOpen {
		t.Error("NetworkRoutesOpen should be false")
	}
	if s.AppstoreDetailOpen {
		t.Error("AppstoreDetailOpen should be false")
	}
	if s.AppstoreFocus != AppstoreFocusSidebar {
		t.Errorf("AppstoreFocus should be sidebar, got %s", s.AppstoreFocus)
	}
}
