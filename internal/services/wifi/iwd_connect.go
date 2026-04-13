package wifi

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

// Connect implements Backend. When passphrase is non-empty the agent
// stashes it before calling Network.Connect; iwd invokes the agent
// during the connect to retrieve it.
func (b *iwdBackend) Connect(ifaceName, ssid, passphrase string, timeout time.Duration) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	if passphrase != "" && b.agent != nil {
		b.agent.SetPending(ssid, passphrase)
		defer b.agent.ClearPending(ssid)
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	netPath, err := b.findNetworkPath(devPath, ssid)
	if err != nil {
		return err
	}
	call := b.conn.Object(iwdBusName, netPath).Call(iwdIfNet+".Connect", 0)
	if call.Err != nil {
		return fmt.Errorf("iwd connect: %w", call.Err)
	}
	return nil
}

// ConnectHidden implements Backend.
func (b *iwdBackend) ConnectHidden(ifaceName, ssid, passphrase string) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	if ssid == "" {
		return fmt.Errorf("iwd: missing ssid for hidden connect")
	}
	if passphrase != "" && b.agent != nil {
		b.agent.SetPending(ssid, passphrase)
		defer b.agent.ClearPending(ssid)
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	call := b.conn.Object(iwdBusName, devPath).Call(iwdIfStatn+".ConnectHiddenNetwork", 0, ssid)
	if call.Err != nil {
		return fmt.Errorf("iwd connect hidden: %w", call.Err)
	}
	return nil
}

// Disconnect implements Backend.
func (b *iwdBackend) Disconnect(ifaceName string) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	call := b.conn.Object(iwdBusName, devPath).Call(iwdIfStatn+".Disconnect", 0)
	if call.Err != nil {
		return fmt.Errorf("iwd disconnect: %w", call.Err)
	}
	return nil
}

// Forget implements Backend.
func (b *iwdBackend) Forget(ifaceName, ssid string) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	objects, err := b.listManagedObjects()
	if err != nil {
		return err
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	netPath, err := b.findNetworkPath(devPath, ssid)
	if err != nil {
		return err
	}
	netIfaces, ok := objects[netPath]
	if !ok {
		return fmt.Errorf("iwd: network %q not in managed objects", ssid)
	}
	netProps, ok := netIfaces[iwdIfNet]
	if !ok {
		return fmt.Errorf("iwd: network %q has no Network interface", ssid)
	}
	known, ok := netProps["KnownNetwork"].Value().(dbus.ObjectPath)
	if !ok || known == "" || known == "/" {
		return fmt.Errorf("iwd: %q is not a saved network", ssid)
	}
	call := b.conn.Object(iwdBusName, known).Call("net.connman.iwd.KnownNetwork.Forget", 0)
	if call.Err != nil {
		return fmt.Errorf("iwd forget: %w", call.Err)
	}
	return nil
}

// SetAutoConnect implements Backend by writing KnownNetwork.AutoConnect.
func (b *iwdBackend) SetAutoConnect(ssid string, autoConnect bool) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	objects, err := b.listManagedObjects()
	if err != nil {
		return err
	}
	for path, ifaces := range objects {
		props, ok := ifaces[iwdIfKnown]
		if !ok {
			continue
		}
		name, ok := props["Name"].Value().(string)
		if !ok || name != ssid {
			continue
		}
		known := b.conn.Object(iwdBusName, path)
		if err := known.SetProperty(iwdIfKnown+".AutoConnect", dbus.MakeVariant(autoConnect)); err != nil {
			return fmt.Errorf("set KnownNetwork.AutoConnect: %w", err)
		}
		return nil
	}
	return fmt.Errorf("iwd: %q is not a saved network", ssid)
}
