package tui

import (
	"fmt"

	"github.com/johnnelson/dark/internal/services/network"
)

func formatInterfaceState(iface network.Interface) string {
	if iface.State == "" {
		return "—"
	}
	if iface.State == "up" && !iface.Carrier {
		return "up · no link"
	}
	return iface.State
}

// primaryAddress returns the first global-scope address for the
// interface, falling back to whatever's available, or "—" when none.
func primaryAddress(addrs []network.Address) string {
	for _, a := range addrs {
		if a.Scope == "global" {
			return a.Address
		}
	}
	if len(addrs) > 0 {
		return addrs[0].Address
	}
	return "—"
}

// addressList returns the formatted CIDR strings for an address slice.
func addressList(addrs []network.Address) []string {
	if len(addrs) == 0 {
		return []string{"—"}
	}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a.Scope != "" && a.Scope != "global" {
			out = append(out, a.CIDR+" ("+a.Scope+")")
		} else {
			out = append(out, a.CIDR)
		}
	}
	return out
}

func formatLinkSpeed(mbps int) string {
	if mbps <= 0 {
		return "—"
	}
	if mbps >= 1000 {
		return fmt.Sprintf("%.1f Gbps", float64(mbps)/1000.0)
	}
	return fmt.Sprintf("%d Mbps", mbps)
}

func formatCarrier(carrier bool, mbps int, duplex string) string {
	if !carrier {
		return placeholderStyle.Render("no link")
	}
	speed := formatLinkSpeed(mbps)
	if duplex != "" && duplex != "unknown" {
		return fmt.Sprintf("%s  %s-duplex", speed, duplex)
	}
	return speed
}

func formatMTU(mtu int) string {
	if mtu <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d", mtu)
}

func formatPackets(n uint64) string {
	if n == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}

// formatNetworkBytes mirrors the wifi formatBytes function.
func formatNetworkBytes(b uint64) string {
	if b == 0 {
		return "—"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatNetworkRate(rx, tx uint64) string {
	if rx == 0 && tx == 0 {
		return "—"
	}
	return fmt.Sprintf("%s ↓  %s ↑", formatBitsPerSec(rx), formatBitsPerSec(tx))
}

