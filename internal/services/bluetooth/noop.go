package bluetooth

import "github.com/godbus/dbus/v5"

// noopBackend stands in when no bluetooth manager is detected on the
// bus. Snapshot returns an empty Adapters list and every action returns
// ErrBackendUnsupported so the TUI can show a clear inline error.
type noopBackend struct {
	conn *dbus.Conn
	name string
}

func newNoopBackend(conn *dbus.Conn) *noopBackend {
	return &noopBackend{conn: conn, name: BackendNone}
}

func (n *noopBackend) Name() string        { return n.name }
func (n *noopBackend) Snapshot() Snapshot  { return Snapshot{Backend: n.name} }
func (n *noopBackend) StartAgent() error   { return nil }
func (n *noopBackend) StopAgent() error    { return nil }
func (n *noopBackend) Close() {
	if n.conn != nil {
		_ = n.conn.Close()
		n.conn = nil
	}
}

func (n *noopBackend) SetPowered(string, bool) error                      { return ErrBackendUnsupported }
func (n *noopBackend) SetDiscoverable(string, bool) error                 { return ErrBackendUnsupported }
func (n *noopBackend) SetDiscoverableTimeout(string, uint32) error        { return ErrBackendUnsupported }
func (n *noopBackend) SetPairable(string, bool) error                     { return ErrBackendUnsupported }
func (n *noopBackend) SetAlias(string, string) error                      { return ErrBackendUnsupported }
func (n *noopBackend) SetDiscoveryFilter(string, DiscoveryFilter) error   { return ErrBackendUnsupported }
func (n *noopBackend) StartDiscovery(string) error                        { return ErrBackendUnsupported }
func (n *noopBackend) StopDiscovery(string) error           { return ErrBackendUnsupported }
func (n *noopBackend) Connect(string) error                 { return ErrBackendUnsupported }
func (n *noopBackend) Disconnect(string) error              { return ErrBackendUnsupported }
func (n *noopBackend) Pair(string, string) error            { return ErrBackendUnsupported }
func (n *noopBackend) CancelPairing(string) error           { return ErrBackendUnsupported }
func (n *noopBackend) Remove(string, string) error          { return ErrBackendUnsupported }
func (n *noopBackend) SetTrusted(string, bool) error        { return ErrBackendUnsupported }
func (n *noopBackend) SetBlocked(string, bool) error        { return ErrBackendUnsupported }
