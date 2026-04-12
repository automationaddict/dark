// Package network builds a read-only snapshot of the host's network
// state — interfaces, addresses, routes, DNS — by reading the kernel
// directly via sysfs, /proc, /etc/resolv.conf, and the Go net package.
//
// No daemon dependency: dark talks to the same kernel objects whether
// you're running systemd-networkd, NetworkManager, dhcpcd, iwd's
// built-in DHCP, or nothing at all. This is the Tier 1 surface; Tier 3
// will add a pluggable Backend interface for *writing* configuration
// (the daemons disagree about that part), but the readout side has
// only one source of truth.
package network

import (
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// Interface is everything we observe about one network device.
type Interface struct {
	Name      string `json:"name"`
	Type      string `json:"type"`            // ethernet, wireless, loopback, bridge, bond, virtual, unknown
	Driver    string `json:"driver,omitempty"` // kernel module name from sysfs
	MAC       string `json:"mac,omitempty"`
	MTU       int    `json:"mtu,omitempty"`
	SpeedMbps int    `json:"speed_mbps,omitempty"` // -1 when not advertised; many wireless and virtual ifaces report this
	Duplex    string `json:"duplex,omitempty"`     // "full" / "half" / "" when unknown
	State     string `json:"state,omitempty"`      // sysfs operstate: up, down, unknown, dormant, lowerlayerdown
	Carrier   bool   `json:"carrier"`              // sysfs carrier — true means a link is detected
	IPv4      []Address `json:"ipv4,omitempty"`
	IPv6      []Address `json:"ipv6,omitempty"`
	RxBytes   uint64 `json:"rx_bytes,omitempty"`
	TxBytes   uint64 `json:"tx_bytes,omitempty"`
	RxPackets uint64 `json:"rx_packets,omitempty"`
	TxPackets uint64 `json:"tx_packets,omitempty"`
	RxRateBps uint64 `json:"rx_rate_bps,omitempty"` // computed delta against previous snapshot
	TxRateBps uint64 `json:"tx_rate_bps,omitempty"`

	// Management is the per-interface state the active backend
	// (systemd-networkd, NetworkManager) reports about how this
	// device is being managed. Nil when no backend is detected, or
	// when the backend doesn't recognize this interface (loopback
	// for example, on networkd, has no managed config).
	Management *ManagementInfo `json:"management,omitempty"`

	// Managed is dark's own view of the configuration it has written
	// for this interface — parsed back out of our `.network` file by
	// the systemd-networkd backend. Nil when there is no dark-managed
	// config for the interface yet. Used by the editor dialogs to
	// prefill from what we wrote (which is the source of truth for
	// dark's intent) rather than from kernel state (which mixes our
	// config with everything else).
	Managed *IPv4Config `json:"managed,omitempty"`
}

// ManagementInfo is the backend's view of how an interface is being
// configured. Two managers report mostly-overlapping information; we
// normalize the field names so the TUI doesn't care which is active.
type ManagementInfo struct {
	BackendName string `json:"backend"`               // "systemd-networkd" / "NetworkManager"
	AdminState  string `json:"admin_state,omitempty"` // configured / configuring / failed / unmanaged / activated / disconnected / ...
	OnlineState string `json:"online_state,omitempty"` // online / offline / partial
	Source      string `json:"source,omitempty"`       // .network file path or NM connection name
	DHCPv4      string `json:"dhcpv4,omitempty"`       // DHCP client state when one is active
	DHCPv6      string `json:"dhcpv6,omitempty"`
	DNS         []string `json:"dns,omitempty"`         // per-interface DNS configured by the manager
	Domains     []string `json:"domains,omitempty"`     // per-interface search/route domains
	Required    *bool  `json:"required,omitempty"`     // RequiredForOnline / autoconnect
}

// Address is one IP address bound to an interface, formatted with its
// CIDR prefix length so the rendering layer doesn't have to remember
// the netmask.
type Address struct {
	Address string `json:"address"`        // e.g. "192.168.1.42"
	CIDR    string `json:"cidr"`           // e.g. "192.168.1.42/24"
	Scope   string `json:"scope,omitempty"` // global, link, host
}

// Route is one entry from the kernel's routing table.
type Route struct {
	Family      string `json:"family"`            // ipv4 or ipv6
	Destination string `json:"destination"`       // "default" or a CIDR like "192.168.1.0/24"
	Gateway     string `json:"gateway,omitempty"` // empty for on-link routes
	Interface   string `json:"interface"`
	Metric      int    `json:"metric,omitempty"`
	Source      string `json:"source,omitempty"` // preferred source address when set
}

// DNS is the resolver configuration the host is using right now.
type DNS struct {
	Servers []string `json:"servers,omitempty"`
	Search  []string `json:"search,omitempty"`
	Source  string   `json:"source,omitempty"` // resolv.conf path that fed this — typically /etc/resolv.conf or a symlink target
}

// IPv4Config is a request to set the layer-3 configuration of an
// interface. Originally IPv4-only (hence the name we kept for type-
// stability), it now also carries IPv6 fields and link-level MTU.
// The naming is preserved across the codebase to avoid touching every
// call site every time we extend it; mentally read it as "interface
// L3 config".
//
// Mode is "dhcp" or "static" and applies to the IPv4 family. For
// static mode, Address/Gateway/DNS/Search apply.
//
// IPv6Mode is the parallel for IPv6: "dhcp" (DHCPv6), "static",
// "ra" (router advertisements only — the typical SLAAC case), or
// empty meaning "leave IPv6 unspecified, fall through to systemd-
// networkd defaults". When set to "static", IPv6Address and
// IPv6Gateway apply.
//
// DNS and Search are shared across families — a v4 and a v6
// nameserver can both appear in the same DNS list.
//
// MTU is a layer-2 setting bundled here because the .network files
// we write already cover the whole link config and a single editor
// dialog is friendlier than two. Zero means "don't write an MTU line
// at all".
//
// Routes is the list of static routes dark should add for the
// interface. Routes can target either family — the destination CIDR
// determines which.
type IPv4Config struct {
	Mode        string        `json:"mode"`
	Address     string        `json:"address,omitempty"` // IPv4 CIDR like 192.168.1.10/24
	Gateway     string        `json:"gateway,omitempty"`
	DNS         []string      `json:"dns,omitempty"`
	Search      []string      `json:"search,omitempty"`
	MTU         int           `json:"mtu,omitempty"` // 0 = leave unset
	Routes      []RouteConfig `json:"routes,omitempty"`
	IPv6Mode    string        `json:"ipv6_mode,omitempty"`    // dhcp / static / ra / "" (unset)
	IPv6Address string        `json:"ipv6_address,omitempty"` // IPv6 CIDR like 2001:db8::1/64
	IPv6Gateway string        `json:"ipv6_gateway,omitempty"`
}

// RouteConfig is one static route to add to an interface. Destination
// is required (a CIDR like "10.0.0.0/8" or "0.0.0.0/0" for a default
// route). Gateway is optional — leaving it empty produces an on-link
// route. Metric is optional and zero means "let the kernel pick".
type RouteConfig struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway,omitempty"`
	Metric      int    `json:"metric,omitempty"`
}

// Snapshot is the network domain payload published on the bus.
type Snapshot struct {
	Backend    string      `json:"backend"`
	Hostname   string      `json:"hostname,omitempty"`
	Interfaces []Interface `json:"interfaces"`
	Routes     []Route     `json:"routes,omitempty"`
	DNS        DNS         `json:"dns,omitempty"`
}

// Service holds long-lived state across snapshot calls: the previous
// traffic counter readings for live bandwidth rate computation, plus
// the active Backend chosen at construction time. The kernel-scrape
// path that powers Snapshot is backend-agnostic; the backend only
// kicks in for mutating operations and for optional state augmentation.
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

// NewService opens a system bus connection, detects which network
// manager (if any) is running, and returns a Service wired to the
// right backend. The backend selection is recoverable — even when no
// manager is detected, the Service still works for read-only kernel
// scrapes.
func NewService() (*Service, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		// No D-Bus connection means no backend, but reads still work.
		return &Service{backend: newNoopBackend(nil), ratePrev: map[string]rateSample{}}, nil
	}
	return &Service{
		backend:  pickBackend(conn),
		ratePrev: map[string]rateSample{},
	}, nil
}

// Close releases the backend (which closes its D-Bus connection).
func (s *Service) Close() {
	if s.backend != nil {
		s.backend.Close()
		s.backend = nil
	}
}

// Snapshot builds a fresh network state snapshot. The kernel scrape
// always runs first; the backend then gets a chance to augment.
func (s *Service) Snapshot() Snapshot {
	snap := Snapshot{
		Hostname:   readHostname(),
		Interfaces: scanInterfaces(),
		Routes:     readRoutes(),
		DNS:        readDNS(),
	}
	if s.backend != nil {
		snap.Backend = s.backend.Name()
		s.backend.Augment(&snap)
	} else {
		snap.Backend = BackendNone
	}
	now := time.Now()
	for i := range snap.Interfaces {
		s.updateRateSample(&snap.Interfaces[i], now)
	}
	return snap
}

// Reconfigure delegates to the active backend. Returns
// ErrBackendUnsupported when no manager is detected — the TUI shows
// that error inline so the user knows the section is read-only on
// this system.
func (s *Service) Reconfigure(iface string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Reconfigure(iface)
}

// ConfigureIPv4 commits a new IPv4 layer-3 configuration to an
// interface via the active backend.
func (s *Service) ConfigureIPv4(iface string, cfg IPv4Config) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.ConfigureIPv4(iface, cfg)
}

// ResetInterface removes any dark-managed configuration for an
// interface and re-applies the system defaults via the active backend.
func (s *Service) ResetInterface(iface string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.ResetInterface(iface)
}

// updateRateSample mirrors the wifi service's rate computation: store
// the previous (rx, tx, time) per interface, compute bytes-per-second
// against the new reading.
func (s *Service) updateRateSample(iface *Interface, now time.Time) {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	if s.ratePrev == nil {
		s.ratePrev = map[string]rateSample{}
	}
	prev, ok := s.ratePrev[iface.Name]
	s.ratePrev[iface.Name] = rateSample{rxBytes: iface.RxBytes, txBytes: iface.TxBytes, at: now}
	if !ok {
		return
	}
	elapsed := now.Sub(prev.at).Seconds()
	if elapsed <= 0 {
		return
	}
	if iface.RxBytes >= prev.rxBytes {
		iface.RxRateBps = uint64(float64(iface.RxBytes-prev.rxBytes) / elapsed)
	}
	if iface.TxBytes >= prev.txBytes {
		iface.TxRateBps = uint64(float64(iface.TxBytes-prev.txBytes) / elapsed)
	}
}

// Detect is a one-shot convenience for callers that don't manage
// Service lifetime — used by the daemon snapshot reply when the
// long-lived Service couldn't be built.
func Detect() Snapshot {
	svc, err := NewService()
	if err != nil {
		return Snapshot{}
	}
	defer svc.Close()
	return svc.Snapshot()
}
