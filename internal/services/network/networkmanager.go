package network

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	nmBusName    = "org.freedesktop.NetworkManager"
	nmObjectPath = dbus.ObjectPath("/org/freedesktop/NetworkManager")
	nmIface      = "org.freedesktop.NetworkManager"
	nmDeviceIf   = "org.freedesktop.NetworkManager.Device"
)

// networkManagerBackend talks to NetworkManager over the system D-Bus
// at org.freedesktop.NetworkManager. NM's model is connection-based:
// devices have an active connection profile, and the canonical "kick
// this thing" verb is Device.Reapply, which re-applies the current
// profile without tearing it down.
type networkManagerBackend struct {
	conn *dbus.Conn
}

func newNetworkManagerBackend(conn *dbus.Conn) *networkManagerBackend {
	return &networkManagerBackend{conn: conn}
}

func (b *networkManagerBackend) Name() string { return BackendNetworkManager }

func (b *networkManagerBackend) Close() {
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
}

// Augment fills in per-interface NM state on the snapshot. For each
// kernel interface we look up the matching NM Device, then read its
// State (an enum we map to a friendly label), the active connection
// name (used as the "source" the way networkd uses the .network file
// path), the IPv4 config's nameservers and search domains, and the
// device's autoconnect flag.
//
// Devices NM doesn't manage (loopback, virtual interfaces it doesn't
// recognize) silently produce no Management info — same handling as
// the systemd-networkd backend's no-match case.
func (b *networkManagerBackend) Augment(snap *Snapshot) {
	if b.conn == nil || snap == nil {
		return
	}
	for i := range snap.Interfaces {
		mi, ok := b.deviceManagementInfo(snap.Interfaces[i].Name)
		if !ok {
			continue
		}
		snap.Interfaces[i].Management = mi
	}
}

// deviceManagementInfo populates a ManagementInfo from NM's Device,
// IP4Config, and ActiveConnection objects for the named interface.
func (b *networkManagerBackend) deviceManagementInfo(ifaceName string) (*ManagementInfo, bool) {
	devicePath, err := b.findDevicePath(ifaceName)
	if err != nil || devicePath == "" {
		return nil, false
	}
	device := b.conn.Object(nmBusName, devicePath)

	mi := &ManagementInfo{BackendName: BackendNetworkManager}

	if v, err := device.GetProperty(nmDeviceIf + ".State"); err == nil {
		if state, ok := v.Value().(uint32); ok {
			mi.AdminState = nmDeviceStateString(state)
		}
	}
	if v, err := device.GetProperty(nmDeviceIf + ".Autoconnect"); err == nil {
		if auto, ok := v.Value().(bool); ok {
			mi.Required = &auto
		}
	}
	if v, err := device.GetProperty(nmDeviceIf + ".ActiveConnection"); err == nil {
		if path, ok := v.Value().(dbus.ObjectPath); ok && path != "/" && path != "" {
			mi.Source = b.activeConnectionLabel(path)
		}
	}
	if v, err := device.GetProperty(nmDeviceIf + ".Ip4Config"); err == nil {
		if path, ok := v.Value().(dbus.ObjectPath); ok && path != "/" && path != "" {
			servers, domains := b.ip4ConfigDNS(path)
			mi.DNS = servers
			mi.Domains = domains
		}
	}
	return mi, true
}

// activeConnectionLabel reads the Id property of the active
// connection object to get the user-facing connection name.
func (b *networkManagerBackend) activeConnectionLabel(path dbus.ObjectPath) string {
	const ifc = "org.freedesktop.NetworkManager.Connection.Active"
	obj := b.conn.Object(nmBusName, path)
	if v, err := obj.GetProperty(ifc + ".Id"); err == nil {
		if s, ok := v.Value().(string); ok {
			return s
		}
	}
	return ""
}

// ip4ConfigDNS reads the Nameservers (as a list of uint32 IPv4
// addresses encoded little-endian) and Domains from an IP4Config
// object. NM legacy IPv4 addresses are stored as packed uint32 in
// network byte order; we decode them back to dotted-quad strings.
func (b *networkManagerBackend) ip4ConfigDNS(path dbus.ObjectPath) ([]string, []string) {
	const ifc = "org.freedesktop.NetworkManager.IP4Config"
	obj := b.conn.Object(nmBusName, path)

	var servers []string
	if v, err := obj.GetProperty(ifc + ".Nameservers"); err == nil {
		if raw, ok := v.Value().([]uint32); ok {
			for _, n := range raw {
				servers = append(servers, nmUint32ToIPv4(n))
			}
		}
	}
	var domains []string
	if v, err := obj.GetProperty(ifc + ".Domains"); err == nil {
		if list, ok := v.Value().([]string); ok {
			domains = list
		}
	}
	return servers, domains
}

// nmUint32ToIPv4 decodes NM's packed-uint32 IPv4 representation to a
// dotted-quad string. NM stores addresses in network byte order so
// the lowest byte is the first octet of the address.
func nmUint32ToIPv4(n uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(n>>0), byte(n>>8), byte(n>>16), byte(n>>24))
}

// nmDeviceStateString maps NM's NM_DEVICE_STATE enum to a friendly
// label. Source: src/libnm-core-public/nm-dbus-interface.h.
func nmDeviceStateString(state uint32) string {
	switch state {
	case 0:
		return "unknown"
	case 10:
		return "unmanaged"
	case 20:
		return "unavailable"
	case 30:
		return "disconnected"
	case 40:
		return "preparing"
	case 50:
		return "configuring"
	case 60:
		return "need-auth"
	case 70:
		return "ip-config"
	case 80:
		return "ip-check"
	case 90:
		return "secondaries"
	case 100:
		return "activated"
	case 110:
		return "deactivating"
	case 120:
		return "failed"
	default:
		return ""
	}
}

// ConfigureIPv4 is not implemented for the NetworkManager backend
// yet. NM connection profile editing is involved enough (read existing
// settings, deep-merge the ipv4 sub-dictionary, write back via
// Connection.Settings.Update, reactivate via ActivateConnection) that
// it deserves its own focused pass once we have a working systemd-
// networkd path to compare against. The architecture is in place;
// only this method body is a stub.
func (b *networkManagerBackend) ConfigureIPv4(string, IPv4Config) error {
	return fmt.Errorf("NetworkManager IPv4 editing is not yet implemented (planned for a future wave)")
}

// ResetInterface is not implemented for the NetworkManager backend
// yet. The natural NM equivalent would be deleting the dark-created
// connection profile (if any) and reactivating the previous one,
// which depends on tracking which profile dark added — also
// deferred until the NM editing wave.
func (b *networkManagerBackend) ResetInterface(string) error {
	return fmt.Errorf("NetworkManager interface reset is not yet implemented (planned for a future wave)")
}

// Reconfigure looks up the NM Device object for the named interface
// and calls Device.Reapply with version_id=0 (use the connection's
// current settings) and flags=0 (default behavior). This re-applies
// the current connection profile to the device, which is the closest
// NM equivalent to systemd-networkd's ReconfigureLink.
//
// Polkit-protected on the NM side via the
// org.freedesktop.NetworkManager.network-control action, which is
// granted to active user sessions by default.
func (b *networkManagerBackend) Reconfigure(ifaceName string) error {
	if b.conn == nil {
		return fmt.Errorf("NetworkManager: no D-Bus connection")
	}
	devicePath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	obj := b.conn.Object(nmBusName, devicePath)
	// Reapply(connection a{sa{sv}}, version_id t, flags u). Passing
	// an empty connection dict tells NM to reuse what's currently
	// applied, which is the "refresh" semantic we want.
	emptyConn := map[string]map[string]dbus.Variant{}
	call := obj.Call(nmDeviceIf+".Reapply", 0, emptyConn, uint64(0), uint32(0))
	if call.Err != nil {
		return fmt.Errorf("NetworkManager reapply: %w", call.Err)
	}
	return nil
}

// findDevicePath asks NM's manager object for the device path that
// owns the named kernel interface.
func (b *networkManagerBackend) findDevicePath(ifaceName string) (dbus.ObjectPath, error) {
	mgr := b.conn.Object(nmBusName, nmObjectPath)
	var path dbus.ObjectPath
	if err := mgr.Call(nmIface+".GetDeviceByIpIface", 0, ifaceName).Store(&path); err != nil {
		return "", fmt.Errorf("NetworkManager: device lookup for %q: %w", ifaceName, err)
	}
	return path, nil
}
