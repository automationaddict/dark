package network

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

// readRoutes parses both /proc/net/route (IPv4) and
// /proc/net/ipv6_route (IPv6) into a single sorted Route slice with
// IPv4 default routes first, since that's almost always what the user
// is checking when they open the page.
func readRoutes() []Route {
	var routes []Route
	routes = append(routes, readIPv4Routes()...)
	routes = append(routes, readIPv6Routes()...)
	sort.SliceStable(routes, func(i, j int) bool {
		// Default routes first, then by family, then destination.
		ai := routes[i].Destination == "default"
		aj := routes[j].Destination == "default"
		if ai != aj {
			return ai
		}
		if routes[i].Family != routes[j].Family {
			return routes[i].Family < routes[j].Family
		}
		return routes[i].Destination < routes[j].Destination
	})
	return routes
}

// readIPv4Routes parses /proc/net/route. The file is a fixed-width
// table where every field is little-endian hex — destination, gateway,
// flags, metric, mask. The first row is a header.
func readIPv4Routes() []Route {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return nil
	}
	defer f.Close()

	var routes []Route
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 11 {
			continue
		}
		dest := parseHexLE32(fields[1])
		gateway := parseHexLE32(fields[2])
		mask := parseHexLE32(fields[7])
		metric, _ := strconv.Atoi(fields[6])

		var destStr string
		if dest == 0 && mask == 0 {
			destStr = "default"
		} else {
			destStr = fmt.Sprintf("%s/%d", uint32ToIP(dest).String(), bitsInMask(mask))
		}
		gwStr := ""
		if gateway != 0 {
			gwStr = uint32ToIP(gateway).String()
		}
		routes = append(routes, Route{
			Family:      "ipv4",
			Destination: destStr,
			Gateway:     gwStr,
			Interface:   fields[0],
			Metric:      metric,
		})
	}
	return routes
}

// readIPv6Routes parses /proc/net/ipv6_route. The format is similar
// to ipv4 but the destination/gateway are 32-char hex strings (16
// bytes each) and there's a different field layout.
//
// Fields: dest_net dest_prefix src_net src_prefix next_hop metric refcnt use flags iface
func readIPv6Routes() []Route {
	f, err := os.Open("/proc/net/ipv6_route")
	if err != nil {
		return nil
	}
	defer f.Close()

	var routes []Route
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		dest, ok := parseHexIPv6(fields[0])
		if !ok {
			continue
		}
		prefix, _ := strconv.ParseUint(fields[1], 16, 32)
		nexthop, _ := parseHexIPv6(fields[4])
		metric, _ := strconv.ParseUint(fields[5], 16, 32)
		flagsRaw, _ := strconv.ParseUint(fields[8], 16, 64)
		const RTFCache = 0x01000000
		if flagsRaw&RTFCache != 0 {
			// Cache entries; not interesting for the user.
			continue
		}

		var destStr string
		if dest.IsUnspecified() && prefix == 0 {
			destStr = "default"
		} else {
			destStr = fmt.Sprintf("%s/%d", dest.String(), prefix)
		}
		gw := ""
		if !nexthop.IsUnspecified() {
			gw = nexthop.String()
		}
		routes = append(routes, Route{
			Family:      "ipv6",
			Destination: destStr,
			Gateway:     gw,
			Interface:   fields[9],
			Metric:      int(metric),
		})
	}
	return routes
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

// --- low level helpers ---

func readSysfsString(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readSysfsInt(path string) int {
	s := readSysfsString(path)
	if s == "" {
		return -1
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}

func readSysfsUint(path string) uint64 {
	s := readSysfsString(path)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// parseHexLE32 reads an 8-character little-endian hex string as a
// uint32. /proc/net/route stores IPv4 addresses this way.
func parseHexLE32(s string) uint32 {
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0
	}
	// Bytes are in network order in the integer, but the file writes
	// them in host order — flip them.
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return binary.LittleEndian.Uint32(b)
}

func uint32ToIP(n uint32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return net.IP(b)
}

func bitsInMask(mask uint32) int {
	count := 0
	for mask != 0 {
		count += int(mask & 1)
		mask >>= 1
	}
	return count
}

// parseHexIPv6 decodes a 32-character hex string into a net.IP.
// /proc/net/ipv6_route uses this format with no separators.
func parseHexIPv6(s string) (net.IP, bool) {
	if len(s) != 32 {
		return nil, false
	}
	out := make(net.IP, 16)
	for i := 0; i < 16; i++ {
		b, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, false
		}
		out[i] = byte(b)
	}
	return out, true
}
