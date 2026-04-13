package network

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

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
