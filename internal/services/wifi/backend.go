package wifi

import (
	"fmt"
	"time"
)

// Backend abstracts the user-space wifi manager dark is talking to. The
// current implementation is iwd, but the interface is shaped so that
// NetworkManager, wpa_supplicant, or a future backend can slot in
// without touching the TUI or the daemon's command handlers.
//
// The Service holds a single Backend and delegates every operation to
// it. Sysfs enumeration and kernel-level reads (IP addresses, traffic
// counters, DNS) happen in the Service itself and are backend-agnostic.
type Backend interface {
	// Name returns the short identifier advertised in Snapshot.Backend.
	// Must match the BackendIWD / BackendNetworkManager / … constants.
	Name() string

	// Augment fills in backend-specific fields on each Adapter the
	// Service has already populated from sysfs. This is where iwd's
	// Device/Station/Diagnostics properties, or NetworkManager's
	// Device/ActiveConnection properties, land on our wire type.
	Augment(adapters []Adapter)

	// FetchKnownNetworks returns the list of saved profiles known to
	// the backend. Independent of whether those networks are currently
	// in range.
	FetchKnownNetworks() []KnownNetwork

	// StartAgent registers any credential agent the backend needs so
	// unknown-network connects can supply a passphrase. No-op for
	// backends that don't use an agent pattern.
	StartAgent() error

	// StopAgent unregisters whatever StartAgent set up.
	StopAgent() error

	// Action methods. Implementations should block until the backend
	// reports success or failure and return a typed error the TUI can
	// surface inline.
	TriggerScan(iface string, timeout time.Duration) error
	Connect(iface, ssid, passphrase string, timeout time.Duration) error
	ConnectHidden(iface, ssid, passphrase string) error
	Disconnect(iface string) error
	Forget(iface, ssid string) error
	SetRadioPowered(iface string, powered bool) error
	SetAutoConnect(ssid string, enabled bool) error

	// Access Point operations. SetMode switches the device between
	// station/ap/ad-hoc. StartAP and StopAP require the device to be
	// in AP mode; implementations typically call SetMode("ap") as part
	// of StartAP so the caller doesn't have to sequence it manually.
	SetMode(iface, mode string) error
	StartAP(iface, ssid, passphrase string) error
	StopAP(iface string) error

	// Close releases any long-lived resources (D-Bus connection,
	// exported agent objects, pending-request state).
	Close()
}

// ErrBackendUnsupported is returned by backends that don't implement a
// particular operation. The TUI shows this inline as an action error.
var ErrBackendUnsupported = fmt.Errorf("operation not supported by this wifi backend")
