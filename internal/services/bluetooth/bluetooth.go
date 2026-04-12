// Package bluetooth talks to the host Bluetooth stack (currently BlueZ)
// and publishes a snapshot of adapters + their known/discovered devices.
// All stack-specific logic lives behind the Backend interface so a new
// implementation can slot in without touching the TUI or the daemon.
package bluetooth

import (
	"github.com/godbus/dbus/v5"
)

// Adapter is one controller (typically hci0) plus the devices BlueZ has
// cataloged under it, whether currently visible or previously paired.
type Adapter struct {
	Path                string `json:"path"`
	Name                string `json:"name"`
	Alias               string `json:"alias,omitempty"`
	Address             string `json:"address,omitempty"`
	Powered             bool   `json:"powered"`
	Discoverable        bool   `json:"discoverable"`
	DiscoverableTimeout uint32 `json:"discoverable_timeout,omitempty"`
	Pairable            bool   `json:"pairable"`
	PairableTimeout     uint32 `json:"pairable_timeout,omitempty"`
	Discovering         bool   `json:"discovering"`
	Backend             string `json:"backend"`

	Devices []Device `json:"devices,omitempty"`
}

// Device is a single Device1 object from BlueZ: anything the adapter
// currently knows about. Paired == true means there's a bond on disk.
type Device struct {
	Path             string   `json:"path"`
	Address          string   `json:"address,omitempty"`
	AddressType      string   `json:"address_type,omitempty"`
	Name             string   `json:"name,omitempty"`
	Alias            string   `json:"alias,omitempty"`
	Icon             string   `json:"icon,omitempty"`
	Class            uint32   `json:"class,omitempty"`
	Appearance       uint16   `json:"appearance,omitempty"`
	Modalias         string   `json:"modalias,omitempty"`
	Paired           bool     `json:"paired"`
	Bonded           bool     `json:"bonded"`
	Trusted          bool     `json:"trusted"`
	Blocked          bool     `json:"blocked"`
	Connected        bool     `json:"connected"`
	LegacyPairing    bool     `json:"legacy_pairing,omitempty"`
	ServicesResolved bool     `json:"services_resolved,omitempty"`
	RSSI             int16    `json:"rssi,omitempty"`
	TxPower          int16    `json:"tx_power,omitempty"`
	Battery          int8     `json:"battery,omitempty"` // -1 when unknown
	UUIDs            []string `json:"uuids,omitempty"`
}

// Snapshot is the bluetooth domain payload published on the bus.
type Snapshot struct {
	Backend  string    `json:"backend"`
	Adapters []Adapter `json:"adapters"`
}

// DiscoveryFilter is the subset of BlueZ's SetDiscoveryFilter parameters
// dark exposes to the user. An empty DiscoveryFilter (zero value) clears
// all filters via SetDiscoveryFilter({}).
type DiscoveryFilter struct {
	// Transport is "auto", "bredr", or "le". Empty means "auto".
	Transport string `json:"transport,omitempty"`
	// RSSI floor in dBm. Negative values filter out weaker signals.
	// Zero disables the RSSI filter.
	RSSI int16 `json:"rssi,omitempty"`
	// Pattern is a name prefix/substring. Empty disables the filter.
	Pattern string `json:"pattern,omitempty"`
}

// IsEmpty reports whether the filter carries any non-default fields.
// An empty filter maps to SetDiscoveryFilter({}) which clears all
// filters on the adapter.
func (f DiscoveryFilter) IsEmpty() bool {
	return f.Transport == "" && f.RSSI == 0 && f.Pattern == ""
}

// Backend identifiers.
const (
	BackendNone  = "none"
	BackendBlueZ = "bluez"
)

// Service owns the chosen Backend and is the single entry point the
// daemon uses to read or mutate bluetooth state.
type Service struct {
	backend Backend
}

// NewService opens the system bus, detects which bluetooth manager is
// running, and returns a Service wired to the right backend.
func NewService() (*Service, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	return &Service{backend: pickBackend(conn)}, nil
}

func (s *Service) Close() {
	if s.backend != nil {
		s.backend.Close()
		s.backend = nil
	}
}

func (s *Service) StartAgent() error {
	if s.backend == nil {
		return nil
	}
	return s.backend.StartAgent()
}

func (s *Service) StopAgent() error {
	if s.backend == nil {
		return nil
	}
	return s.backend.StopAgent()
}

func (s *Service) Snapshot() Snapshot {
	if s.backend == nil {
		return Snapshot{Backend: BackendNone}
	}
	return s.backend.Snapshot()
}

// Detect is a one-shot convenience for callers that don't manage Service
// lifetime. Used by the daemon snapshot reply when the long-lived
// Service couldn't be built.
func Detect() Snapshot {
	svc, err := NewService()
	if err != nil {
		return Snapshot{Backend: BackendNone}
	}
	defer svc.Close()
	return svc.Snapshot()
}

// --- action delegation ---

func (s *Service) SetPowered(adapter string, powered bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetPowered(adapter, powered)
}

func (s *Service) SetDiscoverable(adapter string, discoverable bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetDiscoverable(adapter, discoverable)
}

func (s *Service) SetPairable(adapter string, pairable bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetPairable(adapter, pairable)
}

func (s *Service) SetDiscoverableTimeout(adapter string, seconds uint32) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetDiscoverableTimeout(adapter, seconds)
}

func (s *Service) SetDiscoveryFilter(adapter string, filter DiscoveryFilter) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetDiscoveryFilter(adapter, filter)
}

func (s *Service) SetAlias(adapter, alias string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetAlias(adapter, alias)
}

func (s *Service) StartDiscovery(adapter string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.StartDiscovery(adapter)
}

func (s *Service) StopDiscovery(adapter string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.StopDiscovery(adapter)
}

func (s *Service) Connect(device string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Connect(device)
}

func (s *Service) Disconnect(device string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Disconnect(device)
}

func (s *Service) Pair(device, pin string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Pair(device, pin)
}

func (s *Service) CancelPairing(device string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.CancelPairing(device)
}

func (s *Service) Remove(adapter, device string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Remove(adapter, device)
}

func (s *Service) SetTrusted(device string, trusted bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetTrusted(device, trusted)
}

func (s *Service) SetBlocked(device string, blocked bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetBlocked(device, blocked)
}
