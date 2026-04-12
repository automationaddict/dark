package core

import (
	"testing"

	"github.com/johnnelson/dark/internal/services/audio"
	"github.com/johnnelson/dark/internal/services/bluetooth"
	"github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/services/wifi"
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

func TestSelectedBluetoothDevice_Empty(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetBluetooth(bluetooth.Snapshot{Adapters: []bluetooth.Adapter{{Name: "hci0"}}})
	_, ok := s.SelectedBluetoothDevice()
	if ok {
		t.Error("expected false for empty device list")
	}
}

// --- Audio state ---

func TestCycleAudioFocus_SkipsEmpty(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetAudio(audio.Snapshot{
		Sinks:   []audio.Device{{Name: "speakers"}},
		Sources: []audio.Device{}, // empty — should be skipped
	})
	s.AudioFocus = AudioFocusSinks

	s.CycleAudioFocus()
	// Should skip sources (empty) and play/record apps (empty)
	// and wrap back to sinks.
	if s.AudioFocus != AudioFocusSinks {
		t.Errorf("expected sinks (only non-empty), got %s", s.AudioFocus)
	}
}

func TestCycleAudioFocus_AllPresent(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.SetAudio(audio.Snapshot{
		Sinks:         []audio.Device{{Name: "out"}},
		Sources:       []audio.Device{{Name: "in"}},
		SinkInputs:    []audio.Stream{{Index: 1}},
		SourceOutputs: []audio.Stream{{Index: 2}},
	})
	s.AudioFocus = AudioFocusSinks

	expected := []AudioFocus{AudioFocusSources, AudioFocusPlayApps, AudioFocusRecordApps, AudioFocusSinks}
	for _, want := range expected {
		s.CycleAudioFocus()
		if s.AudioFocus != want {
			t.Errorf("expected %s, got %s", want, s.AudioFocus)
		}
	}
}

func TestSetAudio_FallsBackFromEmptyApps(t *testing.T) {
	s := NewState(TabSettings, "/bin/dark")
	s.AudioFocus = AudioFocusPlayApps
	s.SetAudio(audio.Snapshot{
		Sinks:      []audio.Device{{Name: "out"}},
		SinkInputs: []audio.Stream{}, // was focused, now empty
	})
	if s.AudioFocus != AudioFocusSinks {
		t.Errorf("should fall back to sinks, got %s", s.AudioFocus)
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
