package bluetooth

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	bluezBusName   = "org.bluez"
	bluezRootPath  = "/org/bluez"
	bluezIfAdapter = "org.bluez.Adapter1"
	bluezIfDevice  = "org.bluez.Device1"
	bluezIfBattery = "org.bluez.Battery1"
	bluezIfProps   = "org.freedesktop.DBus.Properties"
)

// bluezBackend is the Backend implementation for BlueZ. It owns its own
// D-Bus connection and, when agent registration is active, its own
// Agent object exported on that connection.
type bluezBackend struct {
	conn        *dbus.Conn
	agent       *Agent
	agentActive bool
}

func newBluezBackend(conn *dbus.Conn) *bluezBackend {
	return &bluezBackend{conn: conn}
}

func (b *bluezBackend) Name() string { return BackendBlueZ }

func (b *bluezBackend) Close() {
	if b.agentActive {
		_ = b.StopAgent()
	}
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
}

// managedObjects is the shape of org.freedesktop.DBus.ObjectManager
// .GetManagedObjects: path → interface → properties.
type managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

func (b *bluezBackend) listManagedObjects() (managedObjects, error) {
	root := b.conn.Object(bluezBusName, "/")
	var result managedObjects
	err := root.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&result)
	return result, err
}

// Snapshot walks BlueZ's managed-object tree, collects Adapter1 objects
// as adapters, and groups Device1 objects under their parent adapter.
func (b *bluezBackend) Snapshot() Snapshot {
	if b.conn == nil {
		return Snapshot{Backend: BackendBlueZ}
	}
	objects, err := b.listManagedObjects()
	if err != nil {
		return Snapshot{Backend: BackendBlueZ}
	}

	adapters := map[dbus.ObjectPath]*Adapter{}
	var order []dbus.ObjectPath

	for p, ifaces := range objects {
		props, ok := ifaces[bluezIfAdapter]
		if !ok {
			continue
		}
		a := &Adapter{
			Path:    string(p),
			Name:    path.Base(string(p)),
			Backend: BackendBlueZ,
		}
		stringOpt(props, "Alias", &a.Alias)
		stringOpt(props, "Address", &a.Address)
		boolOpt(props, "Powered", &a.Powered)
		boolOpt(props, "Discoverable", &a.Discoverable)
		uint32Opt(props, "DiscoverableTimeout", &a.DiscoverableTimeout)
		boolOpt(props, "Pairable", &a.Pairable)
		uint32Opt(props, "PairableTimeout", &a.PairableTimeout)
		boolOpt(props, "Discovering", &a.Discovering)
		adapters[p] = a
		order = append(order, p)
	}

	for p, ifaces := range objects {
		props, ok := ifaces[bluezIfDevice]
		if !ok {
			continue
		}
		var adapterPath dbus.ObjectPath
		if v, ok := props["Adapter"].Value().(dbus.ObjectPath); ok {
			adapterPath = v
		}
		parent, ok := adapters[adapterPath]
		if !ok {
			continue
		}
		d := Device{Path: string(p), Battery: -1}
		stringOpt(props, "Address", &d.Address)
		stringOpt(props, "AddressType", &d.AddressType)
		stringOpt(props, "Name", &d.Name)
		stringOpt(props, "Alias", &d.Alias)
		stringOpt(props, "Icon", &d.Icon)
		stringOpt(props, "Modalias", &d.Modalias)
		uint32Opt(props, "Class", &d.Class)
		uint16Opt(props, "Appearance", &d.Appearance)
		boolOpt(props, "Paired", &d.Paired)
		boolOpt(props, "Bonded", &d.Bonded)
		boolOpt(props, "Trusted", &d.Trusted)
		boolOpt(props, "Blocked", &d.Blocked)
		boolOpt(props, "Connected", &d.Connected)
		boolOpt(props, "LegacyPairing", &d.LegacyPairing)
		boolOpt(props, "ServicesResolved", &d.ServicesResolved)
		int16Opt(props, "RSSI", &d.RSSI)
		int16Opt(props, "TxPower", &d.TxPower)
		if v, ok := props["UUIDs"].Value().([]string); ok {
			d.UUIDs = append([]string(nil), v...)
		}

		// Battery1 is a separate interface on the same path when the
		// device exposes a battery level. Percentage is a byte (0–100).
		if batProps, ok := ifaces[bluezIfBattery]; ok {
			if v, ok := batProps["Percentage"].Value().(byte); ok {
				d.Battery = int8(v)
			}
		}

		parent.Devices = append(parent.Devices, d)
	}

	// Order adapters by path for stable rendering.
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	out := make([]Adapter, 0, len(order))
	for _, p := range order {
		a := adapters[p]
		sortDevices(a.Devices)
		out = append(out, *a)
	}
	return Snapshot{Backend: BackendBlueZ, Adapters: out}
}

// sortDevices puts connected devices first, then paired, then by
// strongest RSSI, then by display name for stability.
func sortDevices(ds []Device) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.Connected != b.Connected {
			return a.Connected
		}
		if a.Paired != b.Paired {
			return a.Paired
		}
		if a.RSSI != b.RSSI {
			// Larger (less negative) RSSI is stronger.
			return a.RSSI > b.RSSI
		}
		return strings.ToLower(a.DisplayName()) < strings.ToLower(b.DisplayName())
	})
}

// DisplayName returns Alias if set, falling back to Name, then Address.
// BlueZ defaults Alias to the device's advertised Name, but the user
// can override it via the bluetoothctl `set-alias` command, so Alias
// is the correct user-facing label.
func (d Device) DisplayName() string {
	if d.Alias != "" {
		return d.Alias
	}
	if d.Name != "" {
		return d.Name
	}
	return d.Address
}

// --- actions ---

func (b *bluezBackend) SetPowered(adapterPath string, powered bool) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if err := obj.SetProperty(bluezIfAdapter+".Powered", dbus.MakeVariant(powered)); err != nil {
		return fmt.Errorf("set Adapter.Powered: %w", err)
	}
	return nil
}

func (b *bluezBackend) SetDiscoverable(adapterPath string, discoverable bool) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if err := obj.SetProperty(bluezIfAdapter+".Discoverable", dbus.MakeVariant(discoverable)); err != nil {
		return fmt.Errorf("set Adapter.Discoverable: %w", err)
	}
	return nil
}

func (b *bluezBackend) SetPairable(adapterPath string, pairable bool) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if err := obj.SetProperty(bluezIfAdapter+".Pairable", dbus.MakeVariant(pairable)); err != nil {
		return fmt.Errorf("set Adapter.Pairable: %w", err)
	}
	return nil
}

// SetDiscoverableTimeout writes Adapter1.DiscoverableTimeout in seconds.
// Zero means "no timeout" (stays discoverable until explicitly toggled
// off). BlueZ defaults this to 180 seconds.
func (b *bluezBackend) SetDiscoverableTimeout(adapterPath string, seconds uint32) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if err := obj.SetProperty(bluezIfAdapter+".DiscoverableTimeout", dbus.MakeVariant(seconds)); err != nil {
		return fmt.Errorf("set Adapter.DiscoverableTimeout: %w", err)
	}
	return nil
}

// SetDiscoveryFilter calls Adapter1.SetDiscoveryFilter with a dict
// built from the non-zero fields of DiscoveryFilter. An empty filter
// clears all filters on the adapter.
//
// Per BlueZ docs, the filter only affects subsequent StartDiscovery
// calls on the same D-Bus connection. Since the daemon holds a single
// long-lived connection and we set the filter before the user triggers
// the next scan, that maps cleanly to dark's flow.
func (b *bluezBackend) SetDiscoveryFilter(adapterPath string, filter DiscoveryFilter) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	args := map[string]dbus.Variant{}
	if filter.Transport != "" {
		args["Transport"] = dbus.MakeVariant(filter.Transport)
	}
	if filter.RSSI != 0 {
		args["RSSI"] = dbus.MakeVariant(filter.RSSI)
	}
	if filter.Pattern != "" {
		args["Pattern"] = dbus.MakeVariant(filter.Pattern)
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if call := obj.Call(bluezIfAdapter+".SetDiscoveryFilter", 0, args); call.Err != nil {
		return fmt.Errorf("bluez set discovery filter: %w", call.Err)
	}
	return nil
}

func (b *bluezBackend) SetAlias(adapterPath, alias string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if err := obj.SetProperty(bluezIfAdapter+".Alias", dbus.MakeVariant(alias)); err != nil {
		return fmt.Errorf("set Adapter.Alias: %w", err)
	}
	return nil
}

func (b *bluezBackend) StartDiscovery(adapterPath string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if call := obj.Call(bluezIfAdapter+".StartDiscovery", 0); call.Err != nil {
		if !isAlreadyInProgress(call.Err) {
			return fmt.Errorf("bluez start discovery: %w", call.Err)
		}
	}
	return nil
}

func (b *bluezBackend) StopDiscovery(adapterPath string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if call := obj.Call(bluezIfAdapter+".StopDiscovery", 0); call.Err != nil {
		return fmt.Errorf("bluez stop discovery: %w", call.Err)
	}
	return nil
}

func (b *bluezBackend) Connect(devicePath string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(devicePath))
	if call := obj.Call(bluezIfDevice+".Connect", 0); call.Err != nil {
		return fmt.Errorf("bluez connect: %w", call.Err)
	}
	return nil
}

func (b *bluezBackend) Disconnect(devicePath string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(devicePath))
	if call := obj.Call(bluezIfDevice+".Disconnect", 0); call.Err != nil {
		return fmt.Errorf("bluez disconnect: %w", call.Err)
	}
	return nil
}

// Pair calls Device1.Pair. When pin is non-empty, the agent is
// pre-seeded so RequestPinCode can return it without round-tripping to
// the TUI. The seed is cleared after the call completes (success or
// failure) so it can't leak into a subsequent unrelated pair.
func (b *bluezBackend) Pair(devicePath, pin string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	if pin != "" && b.agent != nil {
		b.agent.SetPendingPIN(devicePath, pin)
		defer b.agent.ClearPendingPIN(devicePath)
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(devicePath))
	if call := obj.Call(bluezIfDevice+".Pair", 0); call.Err != nil {
		return fmt.Errorf("bluez pair: %w", call.Err)
	}
	return nil
}

// CancelPairing aborts an in-flight Device1.Pair.
func (b *bluezBackend) CancelPairing(devicePath string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(devicePath))
	if call := obj.Call(bluezIfDevice+".CancelPairing", 0); call.Err != nil {
		return fmt.Errorf("bluez cancel pair: %w", call.Err)
	}
	return nil
}

// Remove is "unpair" from the user's perspective: it deletes the bond
// and removes the Device1 object from the adapter.
func (b *bluezBackend) Remove(adapterPath, devicePath string) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(adapterPath))
	if call := obj.Call(bluezIfAdapter+".RemoveDevice", 0, dbus.ObjectPath(devicePath)); call.Err != nil {
		return fmt.Errorf("bluez remove device: %w", call.Err)
	}
	return nil
}

func (b *bluezBackend) SetTrusted(devicePath string, trusted bool) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(devicePath))
	if err := obj.SetProperty(bluezIfDevice+".Trusted", dbus.MakeVariant(trusted)); err != nil {
		return fmt.Errorf("set Device.Trusted: %w", err)
	}
	return nil
}

func (b *bluezBackend) SetBlocked(devicePath string, blocked bool) error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	obj := b.conn.Object(bluezBusName, dbus.ObjectPath(devicePath))
	if err := obj.SetProperty(bluezIfDevice+".Blocked", dbus.MakeVariant(blocked)); err != nil {
		return fmt.Errorf("set Device.Blocked: %w", err)
	}
	return nil
}

func isAlreadyInProgress(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "InProgress") || strings.Contains(msg, "AlreadyExists")
}

// --- variant extractors (local copies to keep the package self-contained) ---

func stringOpt(m map[string]dbus.Variant, key string, dst *string) {
	if v, ok := m[key]; ok {
		if s, ok := v.Value().(string); ok {
			*dst = s
		}
	}
}

func boolOpt(m map[string]dbus.Variant, key string, dst *bool) {
	if v, ok := m[key]; ok {
		if b, ok := v.Value().(bool); ok {
			*dst = b
		}
	}
}

func int16Opt(m map[string]dbus.Variant, key string, dst *int16) {
	if v, ok := m[key]; ok {
		if n, ok := v.Value().(int16); ok {
			*dst = n
		}
	}
}

func uint16Opt(m map[string]dbus.Variant, key string, dst *uint16) {
	if v, ok := m[key]; ok {
		if n, ok := v.Value().(uint16); ok {
			*dst = n
		}
	}
}

func uint32Opt(m map[string]dbus.Variant, key string, dst *uint32) {
	if v, ok := m[key]; ok {
		if n, ok := v.Value().(uint32); ok {
			*dst = n
		}
	}
}
