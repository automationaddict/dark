package core

import "github.com/johnnelson/dark/internal/services/bluetooth"

// SetBluetooth replaces the cached bluetooth snapshot with one received
// from darkd. Selection indices are clamped to the new list sizes so an
// unpair or a removed device doesn't leave an out-of-bounds cursor.
// Mirrors SetWifi, including the "single adapter + on Bluetooth section"
// auto-expand on first load.
func (s *State) SetBluetooth(snap bluetooth.Snapshot) {
	firstLoad := !s.BluetoothLoaded
	s.Bluetooth = snap
	s.BluetoothLoaded = true

	if s.BluetoothSelected >= len(snap.Adapters) {
		s.BluetoothSelected = 0
	}
	if len(snap.Adapters) > 0 {
		if s.BluetoothDevSelected >= len(snap.Adapters[s.BluetoothSelected].Devices) {
			s.BluetoothDevSelected = 0
		}
	}

	if firstLoad && !s.SkipAutoExpand && s.ActiveTab == TabSettings && s.ActiveSection().ID == "bluetooth" &&
		len(snap.Adapters) == 1 && snap.Adapters[0].Powered {
		s.autoExpandSingleBluetoothAdapter()
	}
}

func (s *State) autoExpandSingleBluetoothAdapter() {
	s.ContentFocused = true
	s.BluetoothDetailsOpen = true
	s.BluetoothDevSelected = 0
}

// MoveBluetoothSelection walks the adapter row highlight.
func (s *State) MoveBluetoothSelection(delta int) {
	n := len(s.Bluetooth.Adapters)
	if n == 0 {
		return
	}
	s.BluetoothSelected = (s.BluetoothSelected + delta + n) % n
	s.BluetoothDevSelected = 0
}

// SelectedBluetoothAdapter returns the currently highlighted adapter.
func (s *State) SelectedBluetoothAdapter() (bluetooth.Adapter, bool) {
	if len(s.Bluetooth.Adapters) == 0 {
		return bluetooth.Adapter{}, false
	}
	if s.BluetoothSelected >= len(s.Bluetooth.Adapters) {
		s.BluetoothSelected = 0
	}
	return s.Bluetooth.Adapters[s.BluetoothSelected], true
}

// MoveBluetoothDeviceSelection walks the device row highlight within
// the currently selected adapter's device list.
func (s *State) MoveBluetoothDeviceSelection(delta int) {
	adapter, ok := s.SelectedBluetoothAdapter()
	if !ok {
		return
	}
	n := len(adapter.Devices)
	if n == 0 {
		return
	}
	s.BluetoothDevSelected = (s.BluetoothDevSelected + delta + n) % n
}

// SelectedBluetoothDevice returns the currently highlighted device on
// the selected adapter.
func (s *State) SelectedBluetoothDevice() (bluetooth.Device, bool) {
	adapter, ok := s.SelectedBluetoothAdapter()
	if !ok || len(adapter.Devices) == 0 {
		return bluetooth.Device{}, false
	}
	if s.BluetoothDevSelected >= len(adapter.Devices) {
		s.BluetoothDevSelected = 0
	}
	return adapter.Devices[s.BluetoothDevSelected], true
}

// OpenBluetoothDetails drills into the highlighted adapter and shows
// its device list. Default device selection lands on the first
// connected device if there is one.
func (s *State) OpenBluetoothDetails() {
	if !s.ContentFocused || s.ActiveSection().ID != "bluetooth" || len(s.Bluetooth.Adapters) == 0 {
		return
	}
	s.BluetoothDetailsOpen = true
	s.BluetoothDevSelected = 0
	adapter := s.Bluetooth.Adapters[s.BluetoothSelected]
	for i, d := range adapter.Devices {
		if d.Connected {
			s.BluetoothDevSelected = i
			break
		}
	}
}

// CloseBluetoothDetails hides the device list but keeps content focus.
func (s *State) CloseBluetoothDetails() {
	s.BluetoothDetailsOpen = false
	s.BluetoothDeviceInfoOpen = false
}

// OpenBluetoothDeviceInfo drills a second level into the currently
// highlighted device, expanding the full property readout. Only valid
// when the Devices list is already visible.
func (s *State) OpenBluetoothDeviceInfo() {
	if !s.BluetoothDetailsOpen {
		return
	}
	if _, ok := s.SelectedBluetoothDevice(); !ok {
		return
	}
	s.BluetoothDeviceInfoOpen = true
}

// CloseBluetoothDeviceInfo backs out of the info panel to the Devices list.
func (s *State) CloseBluetoothDeviceInfo() {
	s.BluetoothDeviceInfoOpen = false
}
