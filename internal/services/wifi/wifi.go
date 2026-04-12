// Package wifi enumerates wireless adapters on the host and joins their
// kernel-level state (sysfs, netlink) with their user-space backend
// state (currently iwd; NetworkManager planned). All backend-specific
// logic lives behind the Backend interface so adding a new manager is
// additive rather than structural.
package wifi

import (
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// Adapter is the full wire snapshot of a single wireless interface.
type Adapter struct {
	// Hardware, from sysfs
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	MAC     string `json:"mac"`
	Phy     string `json:"phy,omitempty"`
	Model   string `json:"model,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
	Backend string `json:"backend"`

	// Backend-populated device state
	Mode           string   `json:"mode,omitempty"`
	Powered        bool     `json:"powered"`
	SupportedModes []string `json:"supported_modes,omitempty"`

	// Access Point state. Populated only when Mode == "ap" and iwd has
	// an AccessPoint interface exposed on the device.
	APActive       bool   `json:"ap_active,omitempty"`
	APSSID         string `json:"ap_ssid,omitempty"`
	APFrequencyMHz uint32 `json:"ap_freq_mhz,omitempty"`

	// Backend-populated station state
	State    string `json:"state,omitempty"`
	Scanning bool   `json:"scanning"`
	SSID     string `json:"ssid,omitempty"`

	// Backend-populated connection diagnostics
	BSSID         string `json:"bssid,omitempty"`
	FrequencyMHz  uint32 `json:"frequency_mhz,omitempty"`
	Channel       uint16 `json:"channel,omitempty"`
	Security      string `json:"security,omitempty"`
	RSSI          int16  `json:"rssi,omitempty"`
	AverageRSSI   int16  `json:"avg_rssi,omitempty"`
	RxMode        string `json:"rx_mode,omitempty"`
	TxMode        string `json:"tx_mode,omitempty"`
	TxBitrateKbps uint32 `json:"tx_bitrate,omitempty"`
	RxBitrateKbps uint32 `json:"rx_bitrate,omitempty"`
	ConnectedSecs uint32 `json:"connected_secs,omitempty"`

	// Kernel / netlink
	IPv4    string   `json:"ipv4,omitempty"`
	IPv6    string   `json:"ipv6,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,omitempty"`

	// Traffic counters from sysfs statistics files.
	RxBytes   uint64 `json:"rx_bytes,omitempty"`
	TxBytes   uint64 `json:"tx_bytes,omitempty"`
	RxRateBps uint64 `json:"rx_rate_bps,omitempty"`
	TxRateBps uint64 `json:"tx_rate_bps,omitempty"`

	// Scan results cached by the backend.
	Networks []Network `json:"networks,omitempty"`
}

// Network is one entry from a Station's ordered network list.
type Network struct {
	SSID      string `json:"ssid"`
	Security  string `json:"security"`
	SignalDBm int    `json:"signal_dbm"`
	Known     bool   `json:"known"`
	Connected bool   `json:"connected"`
	BSSCount  int    `json:"bss_count"`
}

// KnownNetwork is one entry from the backend's saved-profile list.
type KnownNetwork struct {
	SSID              string `json:"ssid"`
	Security          string `json:"security"`
	AutoConnect       bool   `json:"autoconnect"`
	Hidden            bool   `json:"hidden,omitempty"`
	LastConnectedTime string `json:"last_connected,omitempty"`
}

// Snapshot is the wifi domain payload published over the bus.
type Snapshot struct {
	Backend       string         `json:"backend"`
	Adapters      []Adapter      `json:"adapters"`
	KnownNetworks []KnownNetwork `json:"known_networks,omitempty"`
}

// Backend identifiers.
const (
	BackendNone           = "none"
	BackendIWD            = "iwd"
	BackendNetworkManager = "networkmanager"
	BackendWpaSupplicant  = "wpa_supplicant"
	BackendUnknown        = "unknown"
)

// Service is the long-lived wifi service. It owns a Backend (chosen at
// construction by pickBackend), plus the backend-agnostic state
// tracking for traffic rate calculation.
type Service struct {
	backend Backend

	rateMu   sync.Mutex
	ratePrev map[string]rateSample
}

type rateSample struct {
	rxBytes uint64
	txBytes uint64
	at      time.Time
}

// NewService opens the system bus, detects which wifi manager is
// running, and returns a Service wired to the right backend. The
// caller should Close the Service on shutdown.
func NewService() (*Service, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	return &Service{
		backend:  pickBackend(conn),
		ratePrev: map[string]rateSample{},
	}, nil
}

// Close releases the underlying Backend and any resources it owns.
func (s *Service) Close() {
	if s.backend != nil {
		s.backend.Close()
		s.backend = nil
	}
}

// StartAgent asks the backend to register its credential agent. For
// backends without an agent concept this is a silent no-op.
func (s *Service) StartAgent() error {
	if s.backend == nil {
		return nil
	}
	return s.backend.StartAgent()
}

// StopAgent unregisters the credential agent.
func (s *Service) StopAgent() error {
	if s.backend == nil {
		return nil
	}
	return s.backend.StopAgent()
}

// Snapshot builds the current wifi payload: enumerate adapters from
// sysfs, let the backend augment them with its own state and fetch
// known networks, then fill in kernel-level network config.
func (s *Service) Snapshot() Snapshot {
	adapters := scanAdaptersFromSysfs()

	backendName := BackendNone
	var knownNets []KnownNetwork
	if s.backend != nil {
		s.backend.Augment(adapters)
		knownNets = s.backend.FetchKnownNetworks()
		backendName = s.backend.Name()
	}

	dns := readDNSServers()
	now := time.Now()
	for i := range adapters {
		adapters[i].Backend = backendName
		readKernelNet(&adapters[i])
		adapters[i].DNS = dns

		rx, tx := readTrafficCounters(adapters[i].Name)
		adapters[i].RxBytes = rx
		adapters[i].TxBytes = tx
		s.updateRateSample(&adapters[i], rx, tx, now)
	}

	return Snapshot{Backend: backendName, Adapters: adapters, KnownNetworks: knownNets}
}

// Detect is a one-shot convenience for callers that don't want to
// manage Service lifetime. Opens a bus connection, takes one snapshot,
// closes.
func Detect() Snapshot {
	svc, err := NewService()
	if err != nil {
		return Snapshot{
			Backend:  BackendUnknown,
			Adapters: scanAdaptersFromSysfs(),
		}
	}
	defer svc.Close()
	return svc.Snapshot()
}

// --- action delegation to backend ---

func (s *Service) TriggerScan(iface string, timeout time.Duration) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.TriggerScan(iface, timeout)
}

func (s *Service) Connect(iface, ssid string, timeout time.Duration) error {
	return s.ConnectWithPassphrase(iface, ssid, "", timeout)
}

func (s *Service) ConnectWithPassphrase(iface, ssid, passphrase string, timeout time.Duration) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Connect(iface, ssid, passphrase, timeout)
}

func (s *Service) ConnectHidden(iface, ssid, passphrase string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.ConnectHidden(iface, ssid, passphrase)
}

func (s *Service) Disconnect(iface string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Disconnect(iface)
}

func (s *Service) Forget(iface, ssid string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Forget(iface, ssid)
}

func (s *Service) SetRadioPowered(iface string, powered bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetRadioPowered(iface, powered)
}

func (s *Service) SetAutoConnect(ssid string, enabled bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetAutoConnect(ssid, enabled)
}

func (s *Service) SetMode(iface, mode string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetMode(iface, mode)
}

func (s *Service) StartAP(iface, ssid, passphrase string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.StartAP(iface, ssid, passphrase)
}

func (s *Service) StopAP(iface string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.StopAP(iface)
}

// updateRateSample converts cumulative counter differences between the
// previous sample and the current reading into average byte-per-second
// rates for the interface.
func (s *Service) updateRateSample(a *Adapter, rx, tx uint64, now time.Time) {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	if s.ratePrev == nil {
		s.ratePrev = map[string]rateSample{}
	}
	prev, ok := s.ratePrev[a.Name]
	s.ratePrev[a.Name] = rateSample{rxBytes: rx, txBytes: tx, at: now}
	if !ok {
		return
	}
	elapsed := now.Sub(prev.at).Seconds()
	if elapsed <= 0 {
		return
	}
	if rx >= prev.rxBytes {
		a.RxRateBps = uint64(float64(rx-prev.rxBytes) / elapsed)
	}
	if tx >= prev.txBytes {
		a.TxRateBps = uint64(float64(tx-prev.txBytes) / elapsed)
	}
}
