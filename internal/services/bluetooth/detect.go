package bluetooth

import "github.com/godbus/dbus/v5"

// pickBackend inspects the well-known D-Bus names on the system bus and
// returns the best-matching Backend. BlueZ is the only bluetooth stack
// we implement; any other case gets the noop backend.
func pickBackend(conn *dbus.Conn) Backend {
	var names []string
	if err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return newNoopBackend(conn)
	}
	for _, n := range names {
		if n == bluezBusName {
			return newBluezBackend(conn)
		}
	}
	return newNoopBackend(conn)
}
