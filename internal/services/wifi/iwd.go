package wifi

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	iwdScanPollInterval = 300 * time.Millisecond // poll scanning state after triggering a scan
	iwdAPTransitionWait = 400 * time.Millisecond // wait for iwd to switch mode to AP
)

const (
	iwdBusName  = "net.connman.iwd"
	iwdIfDevice = "net.connman.iwd.Device"
	iwdIfStatn  = "net.connman.iwd.Station"
	iwdIfDiag   = "net.connman.iwd.StationDiagnostic"
	iwdIfNet    = "net.connman.iwd.Network"
	iwdIfAdapt  = "net.connman.iwd.Adapter"
	iwdIfKnown  = "net.connman.iwd.KnownNetwork"
	iwdIfAP     = "net.connman.iwd.AccessPoint"
)

// iwdBackend is the Backend implementation for Intel's iwd daemon. It
// owns its own D-Bus connection and, when agents are registered, its
// own Agent object exported on that connection. done is closed by
// Close so in-flight polling loops (notably TriggerScan) can bail out
// promptly on daemon shutdown instead of sleeping through the rest of
// their timeout window.
type iwdBackend struct {
	conn        *dbus.Conn
	agent       *Agent
	agentActive bool
	done        chan struct{}
}

func newIwdBackend(conn *dbus.Conn) *iwdBackend {
	return &iwdBackend{conn: conn, done: make(chan struct{})}
}

// Name implements Backend.
func (b *iwdBackend) Name() string { return BackendIWD }

// Close releases the D-Bus connection and signals in-flight polling
// loops to exit. StopAgent is called implicitly if an agent was
// registered.
func (b *iwdBackend) Close() {
	if b.done != nil {
		select {
		case <-b.done: // already closed
		default:
			close(b.done)
		}
	}
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

func (b *iwdBackend) listManagedObjects() (managedObjects, error) {
	root := b.conn.Object(iwdBusName, "/")
	var result managedObjects
	err := root.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&result)
	return result, err
}

// Augment walks iwd's managed-object tree and fills in Device, Station,
// Adapter, and StationDiagnostic state on each Adapter passed in.
func (b *iwdBackend) Augment(adapters []Adapter) {
	if b.conn == nil || len(adapters) == 0 {
		return
	}
	objects, err := b.listManagedObjects()
	if err != nil {
		return
	}

	deviceByName := map[string]dbus.ObjectPath{}
	deviceProps := map[dbus.ObjectPath]map[string]dbus.Variant{}
	stationProps := map[dbus.ObjectPath]map[string]dbus.Variant{}
	adapterProps := map[dbus.ObjectPath]map[string]dbus.Variant{}
	apProps := map[dbus.ObjectPath]map[string]dbus.Variant{}

	for path, ifaces := range objects {
		if dev, ok := ifaces[iwdIfDevice]; ok {
			deviceProps[path] = dev
			if name, ok := dev["Name"].Value().(string); ok {
				deviceByName[name] = path
			}
		}
		if st, ok := ifaces[iwdIfStatn]; ok {
			stationProps[path] = st
		}
		if ad, ok := ifaces[iwdIfAdapt]; ok {
			adapterProps[path] = ad
		}
		if ap, ok := ifaces[iwdIfAP]; ok {
			apProps[path] = ap
		}
	}

	for i := range adapters {
		devPath, ok := deviceByName[adapters[i].Name]
		if !ok {
			continue
		}
		b.fillFromDevice(&adapters[i], devPath, deviceProps, adapterProps, stationProps, apProps, objects)
	}
}

func (b *iwdBackend) fillFromDevice(
	a *Adapter,
	devPath dbus.ObjectPath,
	deviceProps, adapterProps, stationProps, apProps map[dbus.ObjectPath]map[string]dbus.Variant,
	objects managedObjects,
) {
	if dev, ok := deviceProps[devPath]; ok {
		stringOpt(dev, "Mode", &a.Mode)
		if a.MAC == "" {
			stringOpt(dev, "Address", &a.MAC)
		}
		// Follow Device.Adapter to the phy-level iwd.Adapter for
		// vendor/model, supported modes list, AND the real radio
		// powered state.
		if v, ok := dev["Adapter"].Value().(dbus.ObjectPath); ok {
			if props, ok := adapterProps[v]; ok {
				stringOpt(props, "Model", &a.Model)
				stringOpt(props, "Vendor", &a.Vendor)
				boolOpt(props, "Powered", &a.Powered)
				if modes, ok := props["SupportedModes"].Value().([]string); ok {
					a.SupportedModes = append([]string(nil), modes...)
				}
			}
		}
	}

	// AP state: only when the device currently has Mode == "ap" and the
	// AccessPoint interface is exported.
	if ap, ok := apProps[devPath]; ok {
		boolOpt(ap, "Started", &a.APActive)
		stringOpt(ap, "Name", &a.APSSID)
		uint32Opt(ap, "Frequency", &a.APFrequencyMHz)
	}

	st, hasStation := stationProps[devPath]
	if !hasStation {
		return
	}
	stringOpt(st, "State", &a.State)
	boolOpt(st, "Scanning", &a.Scanning)

	if v, ok := st["ConnectedNetwork"].Value().(dbus.ObjectPath); ok && v != "/" && v != "" {
		if netIfaces, ok := objects[v]; ok {
			if netProps, ok := netIfaces[iwdIfNet]; ok {
				stringOpt(netProps, "Name", &a.SSID)
			}
		}
	}

	if a.State == "connected" {
		b.fillDiagnostics(devPath, a)
	}
	a.Networks = b.fetchNetworks(devPath, objects)
}

func (b *iwdBackend) fillDiagnostics(devPath dbus.ObjectPath, a *Adapter) {
	obj := b.conn.Object(iwdBusName, devPath)
	var diag map[string]dbus.Variant
	err := obj.Call(iwdIfDiag+".GetDiagnostics", 0).Store(&diag)
	if err != nil {
		return
	}
	stringOpt(diag, "ConnectedBss", &a.BSSID)
	uint32Opt(diag, "Frequency", &a.FrequencyMHz)
	uint16Opt(diag, "Channel", &a.Channel)
	stringOpt(diag, "Security", &a.Security)
	int16Opt(diag, "RSSI", &a.RSSI)
	int16Opt(diag, "AverageRSSI", &a.AverageRSSI)
	stringOpt(diag, "RxMode", &a.RxMode)
	stringOpt(diag, "TxMode", &a.TxMode)
	uint32Opt(diag, "TxBitrate", &a.TxBitrateKbps)
	uint32Opt(diag, "RxBitrate", &a.RxBitrateKbps)
	uint32Opt(diag, "ConnectedTime", &a.ConnectedSecs)
}

// FetchKnownNetworks implements Backend by walking the managed-object
// tree for KnownNetwork interfaces.
func (b *iwdBackend) FetchKnownNetworks() []KnownNetwork {
	if b.conn == nil {
		return nil
	}
	objects, err := b.listManagedObjects()
	if err != nil {
		return nil
	}
	var out []KnownNetwork
	for _, ifaces := range objects {
		props, ok := ifaces[iwdIfKnown]
		if !ok {
			continue
		}
		var k KnownNetwork
		stringOpt(props, "Name", &k.SSID)
		stringOpt(props, "Type", &k.Security)
		boolOpt(props, "AutoConnect", &k.AutoConnect)
		boolOpt(props, "Hidden", &k.Hidden)
		stringOpt(props, "LastConnectedTime", &k.LastConnectedTime)
		if k.SSID != "" {
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastConnectedTime > out[j].LastConnectedTime
	})
	return out
}

func (b *iwdBackend) fetchNetworks(devPath dbus.ObjectPath, objects managedObjects) []Network {
	obj := b.conn.Object(iwdBusName, devPath)
	var entries []struct {
		Path   dbus.ObjectPath
		Signal int16
	}
	if err := obj.Call(iwdIfStatn+".GetOrderedNetworks", 0).Store(&entries); err != nil {
		return nil
	}
	out := make([]Network, 0, len(entries))
	for _, e := range entries {
		netIfaces, ok := objects[e.Path]
		if !ok {
			continue
		}
		props, ok := netIfaces[iwdIfNet]
		if !ok {
			continue
		}
		var n Network
		stringOpt(props, "Name", &n.SSID)
		stringOpt(props, "Type", &n.Security)
		boolOpt(props, "Connected", &n.Connected)
		if v, ok := props["KnownNetwork"].Value().(dbus.ObjectPath); ok && v != "" && v != "/" {
			n.Known = true
		}
		if v, ok := props["ExtendedServiceSet"].Value().([]dbus.ObjectPath); ok {
			n.BSSCount = len(v)
		}
		n.SignalDBm = int(e.Signal) / 100
		out = append(out, n)
	}
	return out
}

// TriggerScan implements Backend.
func (b *iwdBackend) TriggerScan(ifaceName string, timeout time.Duration) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	obj := b.conn.Object(iwdBusName, devPath)
	if call := obj.Call(iwdIfStatn+".Scan", 0); call.Err != nil {
		if !isAlreadyScanning(call.Err) {
			return fmt.Errorf("iwd scan: %w", call.Err)
		}
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		scanning, err := b.readScanning(devPath)
		if err != nil {
			return err
		}
		if !scanning {
			return nil
		}
		// Sleep responsively: wake on shutdown so daemon exit
		// doesn't have to drain the full timeout window.
		select {
		case <-b.done:
			return fmt.Errorf("iwd scan: cancelled")
		case <-time.After(iwdScanPollInterval):
		}
	}
	return fmt.Errorf("iwd scan: timeout after %s", timeout)
}

// SetRadioPowered implements Backend by writing iwd.Adapter.Powered.
func (b *iwdBackend) SetRadioPowered(ifaceName string, powered bool) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	dev := b.conn.Object(iwdBusName, devPath)
	adapterProp, err := dev.GetProperty(iwdIfDevice + ".Adapter")
	if err != nil {
		return fmt.Errorf("read Device.Adapter: %w", err)
	}
	adapterPath, ok := adapterProp.Value().(dbus.ObjectPath)
	if !ok {
		return fmt.Errorf("iwd: unexpected type for Device.Adapter")
	}
	adapter := b.conn.Object(iwdBusName, adapterPath)
	if err := adapter.SetProperty(iwdIfAdapt+".Powered", dbus.MakeVariant(powered)); err != nil {
		return fmt.Errorf("set Adapter.Powered: %w", err)
	}
	return nil
}

// SetMode writes Device.Mode on the named adapter. iwd tears down and
// rebuilds the interface when the mode changes — a station with an
// active connection will disconnect. Valid modes: "station", "ap",
// "ad-hoc".
func (b *iwdBackend) SetMode(ifaceName, mode string) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	dev := b.conn.Object(iwdBusName, devPath)
	if err := dev.SetProperty(iwdIfDevice+".Mode", dbus.MakeVariant(mode)); err != nil {
		return fmt.Errorf("set Device.Mode: %w", err)
	}
	return nil
}

// findDevicePath returns the iwd Device object path whose Name matches
// the kernel interface name.
func (b *iwdBackend) findDevicePath(ifaceName string) (dbus.ObjectPath, error) {
	objects, err := b.listManagedObjects()
	if err != nil {
		return "", err
	}
	for path, ifaces := range objects {
		dev, ok := ifaces[iwdIfDevice]
		if !ok {
			continue
		}
		if name, ok := dev["Name"].Value().(string); ok && name == ifaceName {
			return path, nil
		}
	}
	return "", fmt.Errorf("iwd: no device for interface %q", ifaceName)
}

// findNetworkPath returns the Network object path with the given SSID
// under the given device.
func (b *iwdBackend) findNetworkPath(devPath dbus.ObjectPath, ssid string) (dbus.ObjectPath, error) {
	objects, err := b.listManagedObjects()
	if err != nil {
		return "", err
	}
	prefix := string(devPath) + "/"
	for path, ifaces := range objects {
		if !strings.HasPrefix(string(path), prefix) {
			continue
		}
		netProps, ok := ifaces[iwdIfNet]
		if !ok {
			continue
		}
		if name, ok := netProps["Name"].Value().(string); ok && name == ssid {
			return path, nil
		}
	}
	return "", fmt.Errorf("iwd: network %q not found on %s", ssid, devPath)
}

// resolveNetworkSSID is used by the Agent to turn a network object path
// into its advertised SSID for pending-passphrase lookup.
func (b *iwdBackend) resolveNetworkSSID(path dbus.ObjectPath) (string, error) {
	objects, err := b.listManagedObjects()
	if err != nil {
		return "", err
	}
	ifaces, ok := objects[path]
	if !ok {
		return "", fmt.Errorf("iwd: object %s not found", path)
	}
	netProps, ok := ifaces[iwdIfNet]
	if !ok {
		return "", fmt.Errorf("iwd: object %s has no Network interface", path)
	}
	name, ok := netProps["Name"].Value().(string)
	if !ok {
		return "", fmt.Errorf("iwd: object %s has no Name", path)
	}
	return name, nil
}

func (b *iwdBackend) readScanning(devPath dbus.ObjectPath) (bool, error) {
	obj := b.conn.Object(iwdBusName, devPath)
	v, err := obj.GetProperty(iwdIfStatn + ".Scanning")
	if err != nil {
		return false, err
	}
	bv, ok := v.Value().(bool)
	if !ok {
		return false, fmt.Errorf("iwd: unexpected type for Scanning")
	}
	return bv, nil
}
