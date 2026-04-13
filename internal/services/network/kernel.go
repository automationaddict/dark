package network

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// scanInterfaces walks /sys/class/net and builds an Interface entry
// for every device the kernel has registered. The Go net package
// gives us interfaces and addresses; sysfs gives us everything else
// (driver, MTU, MAC, state, carrier, traffic).
func scanInterfaces() []Interface {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil
	}

	// Pre-fetch addresses from the Go net package once and bucket
	// them by interface name. Calling InterfaceByName per device works
	// but does the same lookup repeatedly.
	addrsByIface := map[string][]net.Addr{}
	if ifaces, err := net.Interfaces(); err == nil {
		for _, ni := range ifaces {
			if a, err := ni.Addrs(); err == nil {
				addrsByIface[ni.Name] = a
			}
		}
	}

	out := make([]Interface, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		iface := Interface{Name: name}
		fillInterfaceFromSysfs(&iface)
		fillInterfaceAddresses(&iface, addrsByIface[name])
		out = append(out, iface)
	}

	sort.Slice(out, func(i, j int) bool {
		return interfaceSortKey(out[i]) < interfaceSortKey(out[j])
	})
	return out
}

// interfaceSortKey orders interfaces in the same way `ip addr` tends
// to: real hardware first, then virtual, then loopback at the end.
func interfaceSortKey(iface Interface) string {
	rank := "5"
	switch iface.Type {
	case "ethernet":
		rank = "1"
	case "wireless":
		rank = "2"
	case "bridge", "bond":
		rank = "3"
	case "virtual":
		rank = "4"
	case "loopback":
		rank = "9"
	}
	return rank + iface.Name
}

// fillInterfaceFromSysfs populates everything we read out of
// /sys/class/net/<name>/. Errors on individual files are silently
// swallowed because some devices (loopback, virtual) legitimately
// don't expose every property.
func fillInterfaceFromSysfs(iface *Interface) {
	base := "/sys/class/net/" + iface.Name

	if v := readSysfsString(base + "/address"); v != "" {
		iface.MAC = v
	}
	if v := readSysfsInt(base + "/mtu"); v >= 0 {
		iface.MTU = v
	}
	if v := readSysfsInt(base + "/speed"); v > 0 {
		iface.SpeedMbps = v
	}
	if v := readSysfsString(base + "/duplex"); v != "" {
		iface.Duplex = v
	}
	if v := readSysfsString(base + "/operstate"); v != "" {
		iface.State = v
	}
	if v := readSysfsString(base + "/carrier"); v == "1" {
		iface.Carrier = true
	}

	iface.Type = detectInterfaceType(iface.Name, base)
	iface.Driver = readDriver(base)

	// Traffic counters live under statistics/.
	iface.RxBytes = readSysfsUint(base + "/statistics/rx_bytes")
	iface.TxBytes = readSysfsUint(base + "/statistics/tx_bytes")
	iface.RxPackets = readSysfsUint(base + "/statistics/rx_packets")
	iface.TxPackets = readSysfsUint(base + "/statistics/tx_packets")
}

// detectInterfaceType resolves the kind of interface using a small
// priority chain. Wireless and bridge are detected by the existence
// of a known sysfs subdirectory; loopback by name; "virtual" is the
// fallback for anything that doesn't have a backing hardware device.
func detectInterfaceType(name, base string) string {
	if name == "lo" {
		return "loopback"
	}
	if dirExists(base + "/wireless") {
		return "wireless"
	}
	if dirExists(base + "/bridge") {
		return "bridge"
	}
	if dirExists(base + "/bonding") {
		return "bond"
	}
	if !dirExists(base + "/device") {
		return "virtual"
	}
	return "ethernet"
}

// readDriver follows the device/driver symlink and returns its
// basename, which is the kernel module name (e.g. "iwlwifi", "e1000e",
// "r8169"). Empty for virtual interfaces with no backing device.
func readDriver(base string) string {
	link, err := os.Readlink(base + "/device/driver/module")
	if err == nil {
		return filepath.Base(link)
	}
	link, err = os.Readlink(base + "/device/driver")
	if err == nil {
		return filepath.Base(link)
	}
	return ""
}

// fillInterfaceAddresses parses the Go net package address list into
// our typed Address slices. The net.Addr stringification already
// includes the CIDR — we split it into address-only and CIDR forms
// for the renderer.
func fillInterfaceAddresses(iface *Interface, addrs []net.Addr) {
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP
		entry := Address{
			Address: ip.String(),
			CIDR:    ipnet.String(),
			Scope:   addressScope(ip),
		}
		if ip.To4() != nil {
			iface.IPv4 = append(iface.IPv4, entry)
		} else {
			iface.IPv6 = append(iface.IPv6, entry)
		}
	}
}

// addressScope returns the human-readable scope label for an IP. We
// distinguish global / link / host because users care about it for
// debugging, especially with IPv6 link-local addresses.
func addressScope(ip net.IP) string {
	switch {
	case ip.IsLoopback():
		return "host"
	case ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast():
		return "link"
	case ip.IsPrivate(), ip.IsGlobalUnicast():
		return "global"
	default:
		return ""
	}
}

// readDNS parses /etc/resolv.conf for nameserver and search lines.
// On systemd-resolved systems /etc/resolv.conf is typically a symlink
// to /run/systemd/resolve/stub-resolv.conf which lists 127.0.0.53;
// the upstream servers are visible to systemd-resolved itself, not in
// the file. That stub address is the truth from the perspective of
// every application on the system, so showing it is correct.
func readDNS() DNS {
	const path = "/etc/resolv.conf"
	f, err := os.Open(path)
	if err != nil {
		return DNS{Source: path}
	}
	defer f.Close()

	out := DNS{Source: path}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "nameserver":
			if len(fields) >= 2 {
				out.Servers = append(out.Servers, fields[1])
			}
		case "search", "domain":
			out.Search = append(out.Search, fields[1:]...)
		}
	}
	if target, err := os.Readlink(path); err == nil {
		out.Source = target
	}
	return out
}

// readHostname returns the kernel-reported hostname (sethostname).
// Equivalent to /proc/sys/kernel/hostname.
func readHostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return ""
}
