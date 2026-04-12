package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	networkdBusName     = "org.freedesktop.network1"
	networkdManagerPath = dbus.ObjectPath("/org/freedesktop/network1")
	networkdMgrIface    = "org.freedesktop.network1.Manager"
	networkdLinkIface   = "org.freedesktop.network1.Link"
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

// ConfigureIPv4 writes a `.network` file describing the requested
// IPv4 configuration into /etc/systemd/network/, then asks networkd
// to reload its configuration set and reconfigure the interface so
// the new file takes effect.
//
// The file path is fixed at /etc/systemd/network/50-dark-<iface>.network
// so dark always knows where its own files live and can replace them
// in place. The "50-" weight puts dark's files after distro defaults
// (typically 10-, 20-) and before user overrides (90-, 99-) — the
// reasonable middle ground for a settings panel.
//
// Each call writes a comment header marking the file as managed by
// dark so a curious user editing files by hand sees what's going on.
//
// The actual write is delegated to the privileged helper via pkexec,
// which surfaces the standard polkit dialog. Reload + Reconfigure
// happen via the polkit-protected D-Bus methods on networkd that we
// already use for the bare Reconfigure verb.
func (b *systemdNetworkdBackend) ConfigureIPv4(ifaceName string, cfg IPv4Config) error {
	if b.conn == nil {
		return fmt.Errorf("systemd-networkd: no D-Bus connection")
	}
	if ifaceName == "" {
		return fmt.Errorf("systemd-networkd: missing interface name")
	}
	content, err := buildNetworkdFileContent(ifaceName, cfg)
	if err != nil {
		return err
	}
	path := darkNetworkFilePath(ifaceName)
	// The helper layer already returns user-friendly errors (e.g.
	// "authentication cancelled"). Wrapping with the file path here
	// would just add noise the user doesn't need to see when they
	// dismiss the polkit dialog.
	if err := writeNetworkdFile(path, []byte(content)); err != nil {
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

// buildNetworkdFileContent renders an IPv4Config into a systemd-networkd
// `.network` file body. The format is documented in `man 5 systemd.network`;
// we only emit the keys we explicitly support.
//
// Sections emitted:
//   - [Match] always — single Name= line
//   - [Link] only when MTU is non-zero
//   - [Network] always — DHCP=yes for dhcp mode, or Address/Gateway/DNS/
//     Domains lines for static mode
func buildNetworkdFileContent(iface string, cfg IPv4Config) (string, error) {
	v4Mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if v4Mode == "" {
		v4Mode = "dhcp"
	}
	if v4Mode != "dhcp" && v4Mode != "static" {
		return "", fmt.Errorf("systemd-networkd: mode must be dhcp or static, got %q", cfg.Mode)
	}
	v6Mode := strings.ToLower(strings.TrimSpace(cfg.IPv6Mode))
	switch v6Mode {
	case "", "dhcp", "static", "ra":
	default:
		return "", fmt.Errorf("systemd-networkd: ipv6 mode must be dhcp, static, ra, or empty, got %q", cfg.IPv6Mode)
	}
	if cfg.MTU < 0 {
		return "", fmt.Errorf("systemd-networkd: mtu cannot be negative")
	}

	var b strings.Builder
	fmt.Fprintln(&b, "# Managed by dark — do not edit by hand.")
	fmt.Fprintln(&b, "# Replace via the Network section or delete to revert.")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "[Match]")
	fmt.Fprintf(&b, "Name=%s\n\n", iface)

	if cfg.MTU > 0 {
		fmt.Fprintln(&b, "[Link]")
		fmt.Fprintf(&b, "MTUBytes=%d\n\n", cfg.MTU)
	}

	fmt.Fprintln(&b, "[Network]")

	// IPv4 family
	switch v4Mode {
	case "dhcp":
		// DHCP=ipv4 vs DHCP=yes: if v6 is also set to dhcp we'd
		// want DHCP=yes. Otherwise DHCP=ipv4 lets v6 follow its own
		// (possibly RA-driven) path independently.
		if v6Mode == "dhcp" {
			fmt.Fprintln(&b, "DHCP=yes")
		} else {
			fmt.Fprintln(&b, "DHCP=ipv4")
		}
	case "static":
		if cfg.Address == "" {
			return "", fmt.Errorf("systemd-networkd: static IPv4 mode requires an address")
		}
		fmt.Fprintf(&b, "Address=%s\n", cfg.Address)
		if cfg.Gateway != "" {
			fmt.Fprintf(&b, "Gateway=%s\n", cfg.Gateway)
		}
	}

	// IPv6 family
	switch v6Mode {
	case "dhcp":
		if v4Mode != "dhcp" {
			// v4 was static, v6 wants DHCP — write DHCP=ipv6.
			fmt.Fprintln(&b, "DHCP=ipv6")
		}
		// If both are dhcp we already wrote DHCP=yes above.
	case "static":
		if cfg.IPv6Address == "" {
			return "", fmt.Errorf("systemd-networkd: static IPv6 mode requires an address")
		}
		fmt.Fprintf(&b, "Address=%s\n", cfg.IPv6Address)
		if cfg.IPv6Gateway != "" {
			fmt.Fprintf(&b, "Gateway=%s\n", cfg.IPv6Gateway)
		}
		fmt.Fprintln(&b, "IPv6AcceptRA=no")
	case "ra":
		fmt.Fprintln(&b, "IPv6AcceptRA=yes")
	}

	// Shared DNS / search
	for _, dns := range cfg.DNS {
		dns = strings.TrimSpace(dns)
		if dns != "" {
			fmt.Fprintf(&b, "DNS=%s\n", dns)
		}
	}
	for _, search := range cfg.Search {
		search = strings.TrimSpace(search)
		if search != "" {
			fmt.Fprintf(&b, "Domains=%s\n", search)
		}
	}

	for _, r := range cfg.Routes {
		dest := strings.TrimSpace(r.Destination)
		if dest == "" {
			continue
		}
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "[Route]")
		fmt.Fprintf(&b, "Destination=%s\n", dest)
		if gw := strings.TrimSpace(r.Gateway); gw != "" {
			fmt.Fprintf(&b, "Gateway=%s\n", gw)
		}
		if r.Metric > 0 {
			fmt.Fprintf(&b, "Metric=%d\n", r.Metric)
		}
	}

	return b.String(), nil
}

// parseDarkNetworkFile reads one of dark's own `.network` files back
// into an IPv4Config. The parser is intentionally minimal — it only
// understands the keys we generate in buildNetworkdFileContent, and
// silently ignores anything else. The file is world-readable so this
// uses os.ReadFile directly without going through the privileged
// helper.
//
// Returns nil when the file doesn't exist (the "no dark config yet"
// case), and an error only on parse failure of an existing file.
func parseDarkNetworkFile(path string) (*IPv4Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cfg := &IPv4Config{Mode: "dhcp"}
	var section string
	var currentRoute *RouteConfig
	flushRoute := func() {
		if currentRoute != nil && currentRoute.Destination != "" {
			cfg.Routes = append(cfg.Routes, *currentRoute)
		}
		currentRoute = nil
	}

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flushRoute()
			section = line[1 : len(line)-1]
			if section == "Route" {
				currentRoute = &RouteConfig{}
			}
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])

		switch section {
		case "Link":
			if key == "MTUBytes" {
				if n, err := strconv.Atoi(val); err == nil && n > 0 {
					cfg.MTU = n
				}
			}
		case "Network":
			switch key {
			case "DHCP":
				switch strings.ToLower(val) {
				case "yes":
					cfg.Mode = "dhcp"
					cfg.IPv6Mode = "dhcp"
				case "ipv4":
					cfg.Mode = "dhcp"
				case "ipv6":
					cfg.IPv6Mode = "dhcp"
				}
			case "Address":
				if isIPv6Address(val) {
					cfg.IPv6Mode = "static"
					cfg.IPv6Address = val
				} else {
					cfg.Mode = "static"
					cfg.Address = val
				}
			case "Gateway":
				if isIPv6Address(val) {
					cfg.IPv6Gateway = val
				} else {
					cfg.Gateway = val
				}
			case "IPv6AcceptRA":
				if strings.EqualFold(val, "yes") && cfg.IPv6Mode == "" {
					cfg.IPv6Mode = "ra"
				}
			case "DNS":
				cfg.DNS = append(cfg.DNS, val)
			case "Domains":
				cfg.Search = append(cfg.Search, val)
			}
		case "Route":
			if currentRoute == nil {
				currentRoute = &RouteConfig{}
			}
			switch key {
			case "Destination":
				currentRoute.Destination = val
			case "Gateway":
				currentRoute.Gateway = val
			case "Metric":
				if n, err := strconv.Atoi(val); err == nil {
					currentRoute.Metric = n
				}
			}
		}
	}
	flushRoute()
	return cfg, nil
}

// isIPv6Address detects whether a CIDR or address string is IPv6.
// systemd.network uses Address= for both families and only the
// payload tells us which. Detection by colon presence is sufficient
// here because IPv4 addresses cannot contain colons.
func isIPv6Address(s string) bool {
	return strings.Contains(s, ":")
}

// darkNetworkFilePath returns the canonical path of dark's managed
// `.network` file for the named interface. Centralized so the writer
// (ConfigureIPv4), the deleter (ResetInterface), and the parser
// (Augment) all agree on where to look.
func darkNetworkFilePath(iface string) string {
	return filepath.Join("/etc/systemd/network", fmt.Sprintf("50-dark-%s.network", iface))
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
