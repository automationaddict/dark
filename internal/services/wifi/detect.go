package wifi

import (
	"github.com/godbus/dbus/v5"
)

// pickBackend inspects which well-known D-Bus names are owned on the
// system bus and returns the best-matching Backend. iwd takes
// precedence if both iwd and NetworkManager are running. If nothing
// is running, a noop backend is returned so the caller can still
// produce a (mostly empty) Snapshot from sysfs.
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
	case owned["net.connman.iwd"]:
		return newIwdBackend(conn)
	case owned["org.freedesktop.NetworkManager"]:
		// NetworkManager support is planned but not yet implemented.
		// Fall through to the noop backend which at least reports the
		// backend name so the TUI can display "NetworkManager" in the
		// Adapters table even while actions return unsupported.
		return newNoopBackend(conn).withName(BackendNetworkManager)
	case owned["fi.w1.wpa_supplicant1"]:
		return newNoopBackend(conn).withName(BackendWpaSupplicant)
	default:
		return newNoopBackend(conn)
	}
}
