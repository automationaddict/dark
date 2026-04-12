package network

import "github.com/godbus/dbus/v5"

// noopBackend is the placeholder used when no manager daemon is
// detected on the system bus. The kernel scrape still works on these
// systems for *reading* — Snapshot/Routes/DNS all come from /sys and
// /proc and don't need a backend at all. noop only kicks in when the
// user tries to *do* something the kernel scrape can't do alone.
type noopBackend struct {
	conn *dbus.Conn
}

func newNoopBackend(conn *dbus.Conn) *noopBackend {
	return &noopBackend{conn: conn}
}

func (n *noopBackend) Name() string                           { return BackendNone }
func (n *noopBackend) Augment(*Snapshot)                      {}
func (n *noopBackend) Reconfigure(string) error               { return ErrBackendUnsupported }
func (n *noopBackend) ConfigureIPv4(string, IPv4Config) error { return ErrBackendUnsupported }
func (n *noopBackend) ResetInterface(string) error            { return ErrBackendUnsupported }
func (n *noopBackend) Close() {
	if n.conn != nil {
		_ = n.conn.Close()
		n.conn = nil
	}
}
