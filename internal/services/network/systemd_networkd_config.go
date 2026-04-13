package network

import (
	"fmt"
	"strings"
)

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
