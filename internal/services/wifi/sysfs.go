package wifi

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// scanAdaptersFromSysfs walks /sys/class/net for interfaces that expose a
// wireless/ subdirectory — the canonical kernel signal that an interface
// has IEEE 802.11 capabilities.
func scanAdaptersFromSysfs() []Adapter {
	const root = "/sys/class/net"
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var out []Adapter
	for _, e := range entries {
		name := e.Name()
		if name == "lo" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, name, "wireless")); err != nil {
			continue
		}
		out = append(out, readAdapterFromSysfs(name))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func readAdapterFromSysfs(name string) Adapter {
	a := Adapter{Name: name}

	if mac, err := os.ReadFile(filepath.Join("/sys/class/net", name, "address")); err == nil {
		a.MAC = strings.TrimSpace(string(mac))
	}

	if target, err := os.Readlink(filepath.Join("/sys/class/net", name, "device/driver")); err == nil {
		a.Driver = filepath.Base(target)
	}

	if target, err := os.Readlink(filepath.Join("/sys/class/net", name, "phy80211")); err == nil {
		a.Phy = filepath.Base(target)
	}

	return a
}
