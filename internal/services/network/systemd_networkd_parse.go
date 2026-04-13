package network

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

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
