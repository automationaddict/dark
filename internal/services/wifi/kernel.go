package wifi

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// readTrafficCounters reads the cumulative rx/tx byte counters for the
// interface from /sys/class/net/<iface>/statistics/. Zero values on read
// failure so the TUI just shows "—" instead of an error.
func readTrafficCounters(iface string) (rx, tx uint64) {
	rx = readUint64(filepath.Join("/sys/class/net", iface, "statistics", "rx_bytes"))
	tx = readUint64(filepath.Join("/sys/class/net", iface, "statistics", "tx_bytes"))
	return
}

func readUint64(path string) uint64 {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// readKernelNet fills in IP addresses and the default gateway for the
// interface. DNS servers are system-wide, read separately.
func readKernelNet(a *Adapter) {
	iface, err := net.InterfaceByName(a.Name)
	if err != nil {
		return
	}
	addrs, err := iface.Addrs()
	if err == nil {
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if v4 := ip.To4(); v4 != nil {
				if a.IPv4 == "" {
					a.IPv4 = ipnet.String()
				}
				continue
			}
			if ip.IsGlobalUnicast() && a.IPv6 == "" {
				a.IPv6 = ipnet.String()
			}
		}
	}
	a.Gateway = readDefaultGateway(a.Name)
}

// readDNSServers parses /etc/resolv.conf for nameserver entries. systemd-
// resolved users will typically see 127.0.0.53 here which is still correct;
// we can expand this to follow the resolved D-Bus path if a user asks.
func readDNSServers() []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close()
	var servers []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "nameserver ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			servers = append(servers, parts[1])
		}
	}
	return servers
}

// readDefaultGateway walks /proc/net/route looking for the default route
// (destination 0.0.0.0) scoped to the named interface. Gateway is stored
// in the kernel's native byte order, which on x86/ARM is little-endian.
func readDefaultGateway(ifname string) string {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan() // header
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		iface, destHex, gwHex := fields[0], fields[1], fields[2]
		if iface != ifname || destHex != "00000000" {
			continue
		}
		if ip, ok := parseRouteHexIP(gwHex); ok {
			return ip
		}
	}
	return ""
}

func parseRouteHexIP(hex string) (string, bool) {
	if len(hex) != 8 {
		return "", false
	}
	b := [4]uint64{}
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return "", false
		}
		b[i] = v
	}
	// /proc/net/route stores the gateway in little-endian order, so the
	// first byte in the hex string is the lowest-order octet.
	return fmt.Sprintf("%d.%d.%d.%d", b[3], b[2], b[1], b[0]), true
}
