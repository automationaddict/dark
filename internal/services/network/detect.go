package network

import "github.com/godbus/dbus/v5"

// pickBackend inspects which well-known D-Bus names are owned on the
// system bus and returns the best-matching Backend. Order matters:
// NetworkManager wins over systemd-networkd when both are present
// because NM is the more capable manager and almost always the user's
// intentional choice when it's running.
//
// Mirrors internal/services/wifi/detect.go in shape; if both pieces
// drift over time we should consider extracting a shared D-Bus name
// owner-set helper into a small package.
func pickBackend(conn *dbus.Conn) Backend {
	var names []string
	if err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return newNoopBackend(conn)
	}
	owned := make(map[string]bool, len(names))
	for _, n := range names {
		owned[n] = true
	}

	switch {
	case owned[nmBusName]:
		return newNetworkManagerBackend(conn)
	case owned[networkdBusName]:
		return newSystemdNetworkdBackend(conn)
	default:
		return newNoopBackend(conn)
	}
}
