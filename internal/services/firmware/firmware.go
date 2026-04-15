// Package firmware queries and applies firmware updates via the fwupd
// D-Bus API (org.freedesktop.fwupd). When fwupd is not installed or
// the daemon is not running, the service degrades gracefully — all
// methods return empty results with no error.
package firmware

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	busName   = "org.freedesktop.fwupd"
	objPath   = "/"
	iface     = "org.freedesktop.fwupd"
	ifaceProps = "org.freedesktop.DBus.Properties"
)

// fwupd device flag bits. Full list lives in fwupd's libfwupd/fwupd-enums.h.
const (
	flagUpdatable       uint64 = 1 << 1 // FWUPD_DEVICE_FLAG_UPDATABLE
	flagUpdatableHidden uint64 = 1 << 3 // FWUPD_DEVICE_FLAG_UPDATABLE_HIDDEN
)

// Device represents a firmware-capable hardware component.
type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Vendor   string `json:"vendor"`
	Version  string `json:"version"`
	Plugin   string `json:"plugin"`
	Summary  string `json:"summary"`
	Updatable bool  `json:"updatable"`
}

// Release represents an available firmware version for a device.
type Release struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	URI         string `json:"uri"`
	Size        uint64 `json:"size"`
	Vendor      string `json:"vendor"`
}

// Snapshot is the wire snapshot of firmware state.
type Snapshot struct {
	Available bool     `json:"available"` // fwupd is installed and responding
	Devices   []Device `json:"devices"`
	Updates   int      `json:"updates"` // count of devices with pending updates
}

// Service wraps the fwupd D-Bus connection.
type Service struct {
	conn *dbus.Conn
}

// NewService connects to the system bus. Returns a service even if
// fwupd is not running — methods will return empty results.
func NewService() (*Service, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	return &Service{conn: conn}, nil
}

func (s *Service) Close() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}

func (s *Service) obj() dbus.BusObject {
	return s.conn.Object(busName, dbus.ObjectPath(objPath))
}

// Available checks whether the fwupd daemon is reachable.
func (s *Service) Available() bool {
	if s == nil || s.conn == nil {
		return false
	}
	var version dbus.Variant
	err := s.obj().Call(ifaceProps+".Get", 0, iface, "DaemonVersion").Store(&version)
	return err == nil
}

// GetDevices returns all firmware-capable devices.
func (s *Service) GetDevices() ([]Device, error) {
	if s == nil || s.conn == nil {
		return nil, nil
	}
	var raw []map[string]dbus.Variant
	err := s.obj().Call(iface+".GetDevices", 0).Store(&raw)
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetDevices: %w", err)
	}
	var devices []Device
	for _, d := range raw {
		dev := Device{
			ID:      varStr(d, "DeviceId"),
			Name:    varStr(d, "Name"),
			Vendor:  varStr(d, "Vendor"),
			Version: varStr(d, "Version"),
			Plugin:  varStr(d, "Plugin"),
			Summary: varStr(d, "Summary"),
		}
		flags := varUint64(d, "Flags")
		// fwupd splits "this device can be updated" across two flag
		// bits: UpdatableHidden is used for devices whose update path
		// is gated (e.g. requires AC power or a specific unlock). We
		// treat both as updatable so the UI can still surface them.
		dev.Updatable = flags&flagUpdatable != 0 || flags&flagUpdatableHidden != 0
		devices = append(devices, dev)
	}
	return devices, nil
}

// GetUpgrades returns available firmware updates for a device.
func (s *Service) GetUpgrades(deviceID string) ([]Release, error) {
	if s == nil || s.conn == nil {
		return nil, nil
	}
	var raw []map[string]dbus.Variant
	err := s.obj().Call(iface+".GetUpgrades", 0, deviceID).Store(&raw)
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetUpgrades(%s): %w", deviceID, err)
	}
	var releases []Release
	for _, r := range raw {
		releases = append(releases, Release{
			Version:     varStr(r, "Version"),
			Description: varStr(r, "Description"),
			URI:         varStr(r, "Uri"),
			Size:        varUint64(r, "Size"),
			Vendor:      varStr(r, "Vendor"),
		})
	}
	return releases, nil
}

// Snapshot gathers the current firmware state.
func (s *Service) Snapshot() Snapshot {
	if !s.Available() {
		return Snapshot{}
	}
	snap := Snapshot{Available: true}
	devices, err := s.GetDevices()
	if err != nil {
		return snap
	}
	for _, d := range devices {
		if !d.Updatable {
			continue
		}
		snap.Devices = append(snap.Devices, d)
		upgrades, _ := s.GetUpgrades(d.ID)
		if len(upgrades) > 0 {
			snap.Updates++
		}
	}
	return snap
}

func varStr(m map[string]dbus.Variant, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.Value().(string)
	if !ok {
		return ""
	}
	return s
}

func varUint64(m map[string]dbus.Variant, key string) uint64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.Value().(type) {
	case uint64:
		return val
	case uint32:
		return uint64(val)
	case int64:
		return uint64(val)
	case int32:
		return uint64(val)
	}
	return 0
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NothingToDo") ||
		strings.Contains(msg, "NotFound") ||
		strings.Contains(msg, "No devices")
}
