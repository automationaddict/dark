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

// Snapshot is the network domain payload published on the bus.
type Snapshot struct {
	Hostname   string      `json:"hostname,omitempty"`
	Interfaces []Interface `json:"interfaces"`
	Routes     []Route     `json:"routes,omitempty"`
	DNS        DNS         `json:"dns,omitempty"`
}

// Service holds long-lived state across snapshot calls — currently
// just the previous traffic counter readings used to compute live
// per-interface bandwidth rates.
type Service struct {
	rateMu   sync.Mutex
	ratePrev map[string]rateSample
}

type rateSample struct {
	rxBytes uint64
	txBytes uint64
	at      time.Time
}

// NewService constructs a Service. Currently never errors — every
// dependency is a kernel filesystem path that's always available.
func NewService() (*Service, error) {
	return &Service{ratePrev: map[string]rateSample{}}, nil
}

// Close releases any resources the service holds. No-op today.
func (s *Service) Close() {}

// Snapshot builds a fresh network state snapshot.
func (s *Service) Snapshot() Snapshot {
	snap := Snapshot{
		Hostname:   readHostname(),
		Interfaces: scanInterfaces(),
		Routes:     readRoutes(),
		DNS:        readDNS(),
	}
	now := time.Now()
	for i := range snap.Interfaces {
		s.updateRateSample(&snap.Interfaces[i], now)
	}
	return snap
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
