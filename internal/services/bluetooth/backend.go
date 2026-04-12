package bluetooth

import "fmt"

// Backend abstracts the bluetooth stack dark is talking to. BlueZ is
// the current implementation. Mirrors wifi.Backend's shape so the
// daemon/TUI patterns transfer 1:1.
type Backend interface {
	Name() string

	// Snapshot returns the full adapter + device tree. Unlike wifi,
	// bluetooth has no separate kernel enumeration step — all adapters
	// come from the backend directly.
	Snapshot() Snapshot

	StartAgent() error
	StopAgent() error

	SetPowered(adapter string, powered bool) error
	SetDiscoverable(adapter string, discoverable bool) error
	SetDiscoverableTimeout(adapter string, seconds uint32) error
	SetPairable(adapter string, pairable bool) error
	SetAlias(adapter, alias string) error
	SetDiscoveryFilter(adapter string, filter DiscoveryFilter) error
	StartDiscovery(adapter string) error
	StopDiscovery(adapter string) error
	Connect(device string) error
	Disconnect(device string) error
	Pair(device, pin string) error
	CancelPairing(device string) error
	Remove(adapter, device string) error
	SetTrusted(device string, trusted bool) error
	SetBlocked(device string, blocked bool) error

	Close()
}

// ErrBackendUnsupported is returned by backends that don't implement
// a particular operation.
var ErrBackendUnsupported = fmt.Errorf("operation not supported by this bluetooth backend")
