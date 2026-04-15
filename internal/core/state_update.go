package core

import (
	"github.com/automationaddict/dark/internal/services/darkupdate"
	"github.com/automationaddict/dark/internal/services/firmware"
	"github.com/automationaddict/dark/internal/services/update"
)

// SetDarkUpdate replaces the cached dark self-update snapshot
// and clears the busy flags. Called from the TUI message
// handlers on reply receipt and from the periodic publish path.
func (s *State) SetDarkUpdate(snap darkupdate.Snapshot) {
	s.DarkUpdate = snap
	s.DarkUpdateLoaded = true
	s.DarkUpdateChecking = false
	s.DarkUpdateApplying = false
}

type UpdateSection struct {
	ID    string
	Icon  string
	Label string
}

func UpdateSections() []UpdateSection {
	return []UpdateSection{
		{"omarchy", "󰣇", "Omarchy"},
		{"firmware", "󰍛", "Firmware"},
		{"dark", "󰓎", "Dark"},
	}
}

func (s *State) ActiveUpdateSection() UpdateSection {
	secs := UpdateSections()
	if s.UpdateSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.UpdateSectionIdx]
}

func (s *State) MoveUpdateSection(delta int) {
	n := len(UpdateSections())
	if n == 0 {
		return
	}
	s.UpdateSectionIdx = (s.UpdateSectionIdx + delta + n) % n
}

func (s *State) SetUpdate(snap update.Snapshot) {
	s.Update = snap
	s.UpdateLoaded = true
}

func (s *State) MarkUpdateBusy() {
	s.UpdateBusy = true
	s.UpdateStatusMsg = ""
	s.UpdateResult = nil
}

func (s *State) SetUpdateResult(r update.RunResult) {
	s.UpdateBusy = false
	s.UpdateResult = &r
	if r.Error != "" {
		s.UpdateStatusMsg = r.Error
	} else if r.RebootNeeded {
		s.UpdateStatusMsg = "Update complete — reboot recommended"
	} else {
		s.UpdateStatusMsg = "Update complete"
	}
}

func (s *State) SetFirmware(snap firmware.Snapshot) {
	s.Firmware = snap
	s.FirmwareLoaded = true
	if s.FirmwareDeviceIdx >= len(snap.Devices) {
		s.FirmwareDeviceIdx = 0
	}
}

func (s *State) MoveFirmwareDevice(delta int) {
	n := len(s.Firmware.Devices)
	if n == 0 {
		return
	}
	s.FirmwareDeviceIdx = (s.FirmwareDeviceIdx + delta + n) % n
}

func (s *State) SelectedFirmwareDevice() (firmware.Device, bool) {
	if len(s.Firmware.Devices) == 0 {
		return firmware.Device{}, false
	}
	if s.FirmwareDeviceIdx >= len(s.Firmware.Devices) {
		s.FirmwareDeviceIdx = 0
	}
	return s.Firmware.Devices[s.FirmwareDeviceIdx], true
}
