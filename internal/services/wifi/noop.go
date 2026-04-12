package wifi

import (
	"time"

	"github.com/godbus/dbus/v5"
)

// noopBackend is the "we don't have a wifi manager we can talk to"
// placeholder. Augment and FetchKnownNetworks return nothing, and
// every action returns ErrBackendUnsupported so the TUI shows a clear
// inline error instead of silently failing.
//
// It's also the stand-in for backends we've detected but not yet
// implemented (currently: NetworkManager, wpa_supplicant). The name
// field lets detect.go report the real daemon in Snapshot.Backend so
// the Adapters table shows the truth even while actions are stubbed.
type noopBackend struct {
	conn *dbus.Conn
	name string
}

func newNoopBackend(conn *dbus.Conn) *noopBackend {
	return &noopBackend{conn: conn, name: BackendNone}
}

func (n *noopBackend) withName(name string) *noopBackend {
	n.name = name
	return n
}

func (n *noopBackend) Name() string                      { return n.name }
func (n *noopBackend) Augment(adapters []Adapter)        {}
func (n *noopBackend) FetchKnownNetworks() []KnownNetwork { return nil }
func (n *noopBackend) StartAgent() error                 { return nil }
func (n *noopBackend) StopAgent() error                  { return nil }
func (n *noopBackend) Close() {
	if n.conn != nil {
		_ = n.conn.Close()
		n.conn = nil
	}
}

func (n *noopBackend) TriggerScan(iface string, timeout time.Duration) error {
	return ErrBackendUnsupported
}
func (n *noopBackend) Connect(iface, ssid, passphrase string, timeout time.Duration) error {
	return ErrBackendUnsupported
}
func (n *noopBackend) ConnectHidden(iface, ssid, passphrase string) error {
	return ErrBackendUnsupported
}
func (n *noopBackend) Disconnect(iface string) error            { return ErrBackendUnsupported }
func (n *noopBackend) Forget(iface, ssid string) error          { return ErrBackendUnsupported }
func (n *noopBackend) SetRadioPowered(iface string, on bool) error { return ErrBackendUnsupported }
func (n *noopBackend) SetAutoConnect(ssid string, on bool) error { return ErrBackendUnsupported }
func (n *noopBackend) SetMode(iface, mode string) error         { return ErrBackendUnsupported }
func (n *noopBackend) StartAP(iface, ssid, passphrase string) error {
	return ErrBackendUnsupported
}
func (n *noopBackend) StopAP(iface string) error { return ErrBackendUnsupported }
