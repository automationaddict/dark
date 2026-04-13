package network

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"

	"github.com/godbus/dbus/v5"
)

const (
	networkdBusName     = "org.freedesktop.network1"
	networkdManagerPath = dbus.ObjectPath("/org/freedesktop/network1")
	networkdMgrIface    = "org.freedesktop.network1.Manager"
	networkdLinkIface   = "org.freedesktop.network1.Link"

	networkdConfigDir  = "/etc/systemd/network"
	networkdFilePrefix = "50-dark-"
	networkdFileSuffix = ".network"
)

// systemdNetworkdBackend talks to systemd-networkd over the system
// D-Bus at org.freedesktop.network1. systemd-networkd is declarative —
// configuration lives in .network files under /etc/systemd/network/ —
// so the D-Bus surface is mostly about state queries and operational
// commands like "reconfigure this link" or "renew its DHCP lease".
type systemdNetworkdBackend struct {
	conn *dbus.Conn
}

func newSystemdNetworkdBackend(conn *dbus.Conn) *systemdNetworkdBackend {
	return &systemdNetworkdBackend{conn: conn}
}

func (b *systemdNetworkdBackend) Name() string { return BackendSystemdNetworkd }

func (b *systemdNetworkdBackend) Close() {
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
}

// Augment walks the snapshot's interfaces and asks systemd-networkd
// for the rich per-link state via Link.Describe (which returns a JSON
// blob with everything networkd knows about the link). We populate
// Management with the fields the TUI displays today: admin state,
// online state, the .network file managing the link, per-link DNS
// and search domains, DHCP client state, and required-for-online.
//
// We also parse our own dark-managed `.network` file if it exists and
// attach it as Interface.Managed so the editor dialogs can prefill
// from dark's actual written state instead of inferring from kernel
// state (which mixes our config with everything else).
func (b *systemdNetworkdBackend) Augment(snap *Snapshot) {
	if b.conn == nil || snap == nil {
		return
	}
	for i := range snap.Interfaces {
		name := snap.Interfaces[i].Name
		if desc, ok := b.describeLink(name); ok {
			snap.Interfaces[i].Management = describeToManagementInfo(desc)
		}
		// Parse failures are silently dropped — the file is dark-
		// generated, so a parse error means either someone hand-
		// edited it into something we don't understand or there's
		// a bug in our writer. Either way, leaving Managed nil is
		// safer than surfacing a partial parse.
		if managed, _ := parseDarkNetworkFile(darkNetworkFilePath(name)); managed != nil {
			snap.Interfaces[i].Managed = managed
		}
	}
}

// describeLink calls Link.Describe and parses its JSON return. We
// only decode the fields we actually consume — networkd's full
// Describe payload is much larger and changes between versions.
func (b *systemdNetworkdBackend) describeLink(ifaceName string) (linkDescribe, bool) {
	var out linkDescribe
	idx, err := ifindex(ifaceName)
	if err != nil {
		return out, false
	}
	mgr := b.conn.Object(networkdBusName, networkdManagerPath)
	var linkPath dbus.ObjectPath
	var ifindexReturned int32
	if err := mgr.Call(networkdMgrIface+".GetLinkByIndex", 0, int32(idx)).Store(&ifindexReturned, &linkPath); err != nil {
		return out, false
	}
	link := b.conn.Object(networkdBusName, linkPath)
	var raw string
	if err := link.Call(networkdLinkIface+".Describe", 0).Store(&raw); err != nil {
		return out, false
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, false
	}
	return out, true
}

// linkDescribe is the subset of the JSON Link.Describe returns that
// dark consumes. Anonymous structs handle the nested DNS / DHCP
// objects so we don't pollute the package with single-use types.
type linkDescribe struct {
	Name              string `json:"Name"`
	NetworkFile       string `json:"NetworkFile"`
	OperationalState  string `json:"OperationalState"`
	AdministrativeState string `json:"AdministrativeState"`
	OnlineState       string `json:"OnlineState"`
	RequiredForOnline bool   `json:"RequiredForOnline"`
	ActivationPolicy  string `json:"ActivationPolicy"`
	DHCPv4Client      struct {
		State string `json:"State"`
	} `json:"DHCPv4Client"`
	DHCPv6Client struct {
		State string `json:"State"`
	} `json:"DHCPv6Client"`
	DNS []struct {
		AddressString string `json:"AddressString"`
	} `json:"DNS"`
	SearchDomains []struct {
		Domain string `json:"Domain"`
	} `json:"SearchDomains"`
}

// describeToManagementInfo converts the parsed Describe blob into the
// flat ManagementInfo shape the TUI consumes. RequiredForOnline is
// passed through as a *bool so the renderer can distinguish "false"
// from "not reported".
func describeToManagementInfo(d linkDescribe) *ManagementInfo {
	mi := &ManagementInfo{
		BackendName: BackendSystemdNetworkd,
		AdminState:  d.AdministrativeState,
		OnlineState: d.OnlineState,
		Source:      d.NetworkFile,
		DHCPv4:      d.DHCPv4Client.State,
		DHCPv6:      d.DHCPv6Client.State,
	}
	for _, e := range d.DNS {
		if e.AddressString != "" {
			mi.DNS = append(mi.DNS, e.AddressString)
		}
	}
	for _, e := range d.SearchDomains {
		if e.Domain != "" {
			mi.Domains = append(mi.Domains, e.Domain)
		}
	}
	required := d.RequiredForOnline
	mi.Required = &required
	return mi
}

// darkNetworkFilePath returns the canonical path of dark's managed
// `.network` file for the named interface. Centralized so the writer
// (ConfigureIPv4), the deleter (ResetInterface), and the parser
// (Augment) all agree on where to look.
func darkNetworkFilePath(iface string) string {
	return filepath.Join(networkdConfigDir, fmt.Sprintf("%s%s%s", networkdFilePrefix, iface, networkdFileSuffix))
}

// ResetInterface deletes the dark-managed `.network` file for the
// interface (if any) and triggers a reload + reconfigure so the
// system falls back to whatever else matches that link — typically
// a distro default `.network` file or systemd-networkd's implicit
// fallback behavior.
//
// The delete is delegated to the privileged helper via pkexec, same
// path as ConfigureIPv4 — the user gets one polkit dialog, the file
// disappears as root, and we then call the polkit-protected D-Bus
// methods on networkd to apply the change.
func (b *systemdNetworkdBackend) ResetInterface(ifaceName string) error {
	if b.conn == nil {
		return fmt.Errorf("systemd-networkd: no D-Bus connection")
	}
	if ifaceName == "" {
		return fmt.Errorf("systemd-networkd: missing interface name")
	}
	path := darkNetworkFilePath(ifaceName)
	if err := deleteNetworkdFile(path); err != nil {
		return err
	}
	mgr := b.conn.Object(networkdBusName, networkdManagerPath)
	if call := mgr.Call(networkdMgrIface+".Reload", 0); call.Err != nil {
		return fmt.Errorf("networkd reload: %w", call.Err)
	}
	if err := b.Reconfigure(ifaceName); err != nil {
		return fmt.Errorf("networkd reconfigure: %w", err)
	}
	return nil
}

// Reconfigure calls Manager.ReconfigureLink, which makes systemd-networkd
// re-read the .network files for that link and re-apply them. This
// triggers a fresh DHCP request when DHCP is enabled — exactly the
// "kick this thing and make it try again" semantics we want.
//
// Polkit-protected on the daemon side: the user's session has the
// `org.freedesktop.network1.reconfigure-link` action by default in
// most distros, so godbus's call goes through without dark needing to
// know anything about polkit. If polkit denies it we get a clean
// AccessDenied error to surface in the TUI.
func (b *systemdNetworkdBackend) Reconfigure(ifaceName string) error {
	if b.conn == nil {
		return fmt.Errorf("systemd-networkd: no D-Bus connection")
	}
	idx, err := ifindex(ifaceName)
	if err != nil {
		return err
	}
	obj := b.conn.Object(networkdBusName, networkdManagerPath)
	call := obj.Call(networkdMgrIface+".ReconfigureLink", 0, int32(idx))
	if call.Err != nil {
		return fmt.Errorf("systemd-networkd reconfigure: %w", call.Err)
	}
	return nil
}

// ifindex resolves a kernel interface name to its integer index,
// which is the form networkd's D-Bus methods take.
func ifindex(name string) (int, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return 0, fmt.Errorf("lookup %q: %w", name, err)
	}
	return iface.Index, nil
}
