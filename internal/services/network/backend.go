package network

import "fmt"

// Backend abstracts the network manager dark is talking to. The
// kernel scrape (interfaces, addresses, routes, DNS) works without
// any backend at all — Backend is for *mutations* and for adding
// manager-specific augmentation on top of the kernel snapshot.
//
// Implementations: systemd-networkd, NetworkManager, noop. Detected
// at startup by which D-Bus name is owned. Mirrors the wifi and
// bluetooth Backend pattern exactly.
type Backend interface {
	// Name is the short identifier reported in Snapshot.Backend so
	// the TUI can show "systemd-networkd" or "NetworkManager" in
	// status text.
	Name() string

	// Augment fills in any manager-specific fields on top of the
	// kernel-scrape snapshot. Optional — backends with nothing extra
	// to add leave it empty.
	Augment(snap *Snapshot)

	// Reconfigure tells the backend to re-apply current configuration
	// to an interface. Includes a DHCP refresh on backends where that
	// happens automatically as part of reapplying. The verb the user
	// reaches for when network is acting up.
	Reconfigure(iface string) error

	// ConfigureIPv4 commits a new IPv4 layer-3 configuration to an
	// interface. For systemd-networkd this writes a `.network` file
	// via the privileged helper and triggers a reload + reconfigure.
	// For NetworkManager this updates the device's connection profile
	// over D-Bus and reactivates it. Backends that don't support
	// configuration writes return ErrBackendUnsupported.
	ConfigureIPv4(iface string, cfg IPv4Config) error

	// ResetInterface removes any dark-managed configuration for an
	// interface and re-applies whatever the system would have done
	// without dark in the picture. The "I changed my mind, put it
	// back the way it was" verb. For systemd-networkd this deletes
	// the dark-managed `.network` file and triggers reload +
	// reconfigure so the interface falls back to whatever else
	// matches it (a distro default `.network` file, the implicit
	// fallback, etc.).
	ResetInterface(iface string) error

	Close()
}

// ErrBackendUnsupported is returned when a backend doesn't implement
// an operation. The TUI surfaces this inline as an action error.
var ErrBackendUnsupported = fmt.Errorf("operation not supported by this network backend")

// Backend identifiers.
const (
	BackendNone            = "none"
	BackendSystemdNetworkd = "systemd-networkd"
	BackendNetworkManager  = "NetworkManager"
)
