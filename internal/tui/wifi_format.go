package tui

import (
	"fmt"
	"strings"
	"time"
)

// --- formatters ---

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func onOff(v bool) string {
	if v {
		return "On"
	}
	return "Off"
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func titleCase(s string) string {
	if s == "" || s == "—" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func formatFreq(mhz uint32) string {
	if mhz == 0 {
		return "—"
	}
	return fmt.Sprintf("%.2f GHz", float64(mhz)/1000.0)
}

func formatChannel(ch uint16) string {
	if ch == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", ch)
}

func formatRSSI(current, average int16) string {
	if current == 0 {
		return "—"
	}
	if average != 0 && average != current {
		return fmt.Sprintf("%d dBm  (avg %d)", current, average)
	}
	return fmt.Sprintf("%d dBm", current)
}

// formatSignal combines the RSSI text with a rolling sparkline of
// recent samples. With fewer than two samples the sparkline is
// omitted — a single bar doesn't tell the user anything useful.
func formatSignal(current, average int16, history []int16) string {
	text := formatRSSI(current, average)
	if len(history) < 2 {
		return text
	}
	return text + "  " + rssiSparkline(history)
}

// rssiSparkline maps a history of RSSI values (in dBm, always negative)
// to a Unicode bar-graph. The scale is pinned to -30 (excellent) and
// -90 (unusable) so changes across sessions are visually comparable
// rather than normalized to whatever's currently in the buffer.
func rssiSparkline(history []int16) string {
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	const best = -30
	const worst = -90
	const span = best - worst // 60

	var b strings.Builder
	for _, v := range history {
		if v == 0 {
			b.WriteRune(' ')
			continue
		}
		n := int(v)
		if n > best {
			n = best
		}
		if n < worst {
			n = worst
		}
		idx := (n - worst) * (len(bars) - 1) / span
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		b.WriteRune(bars[idx])
	}
	return b.String()
}

func formatLink(rxMode string, rxKbps uint32, txMode string, txKbps uint32) string {
	if rxKbps == 0 && txKbps == 0 {
		return "—"
	}
	mode := rxMode
	if mode == "" {
		mode = txMode
	}
	return fmt.Sprintf("%s  (TX %s · RX %s)", mode, formatBitrate(txKbps), formatBitrate(rxKbps))
}

// formatTraffic renders cumulative RX/TX byte totals as a short string.
func formatTraffic(rxBytes, txBytes uint64) string {
	if rxBytes == 0 && txBytes == 0 {
		return "—"
	}
	return fmt.Sprintf("%s ↓  %s ↑", formatBytes(rxBytes), formatBytes(txBytes))
}

// formatRate renders RX/TX rates in human-friendly units.
func formatRate(rxBps, txBps uint64) string {
	if rxBps == 0 && txBps == 0 {
		return "—"
	}
	return fmt.Sprintf("%s ↓  %s ↑", formatBitsPerSec(rxBps), formatBitsPerSec(txBps))
}

// formatBytes renders a byte count as KiB/MiB/GiB/TiB.
func formatBytes(b uint64) string {
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

// formatBitsPerSec renders a byte-per-second rate as bits per second in
// kb/s or Mb/s. Network rates are conventionally reported in bits.
func formatBitsPerSec(bytesPerSec uint64) string {
	bitsPerSec := bytesPerSec * 8
	switch {
	case bitsPerSec == 0:
		return "0 bps"
	case bitsPerSec < 1_000:
		return fmt.Sprintf("%d bps", bitsPerSec)
	case bitsPerSec < 1_000_000:
		return fmt.Sprintf("%.1f kbps", float64(bitsPerSec)/1_000)
	case bitsPerSec < 1_000_000_000:
		return fmt.Sprintf("%.1f Mbps", float64(bitsPerSec)/1_000_000)
	default:
		return fmt.Sprintf("%.2f Gbps", float64(bitsPerSec)/1_000_000_000)
	}
}

func formatBitrate(kbps uint32) string {
	if kbps == 0 {
		return "—"
	}
	if kbps >= 10000 {
		return fmt.Sprintf("%.1f Mbps", float64(kbps)/1000.0)
	}
	return fmt.Sprintf("%d kbps", kbps)
}

func formatDuration(secs uint32) string {
	if secs == 0 {
		return "—"
	}
	d := int(secs)
	h := d / 3600
	m := (d % 3600) / 60
	s := d % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// formatAge renders an RFC3339 timestamp as a short "time ago" string.
// iwd reports LastConnectedTime as RFC3339 UTC. Unset / unparseable
// values return a dash.
func formatAge(rfc3339 string) string {
	if rfc3339 == "" {
		return "—"
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	d := time.Since(t)
	switch {
	case d < 0:
		return "just now"
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// formatNetSecurity maps iwd's network type to a display label.
func formatNetSecurity(t string) string {
	switch t {
	case "open":
		return "Open"
	case "psk":
		return "WPA2/3"
	case "8021x":
		return "Enterprise"
	case "wep":
		return "WEP"
	case "":
		return "—"
	default:
		return t
	}
}

// signalBars returns a Unicode bar-graph glyph approximating signal
// strength from an RSSI value in dBm.
func signalBars(dbm int) string {
	switch {
	case dbm == 0:
		return "—"
	case dbm >= -50:
		return "▁▂▃▄▅▆▇█"
	case dbm >= -60:
		return "▁▂▃▄▅▆▇"
	case dbm >= -67:
		return "▁▂▃▄▅▆"
	case dbm >= -70:
		return "▁▂▃▄▅"
	case dbm >= -75:
		return "▁▂▃▄"
	case dbm >= -80:
		return "▁▂▃"
	case dbm >= -85:
		return "▁▂"
	default:
		return "▁"
	}
}

// rowMarker builds a 3-cell prefix: selection marker · status glyph ·
// trailing space. The status glyph is the Nerd Font Wi-Fi icon when the
// row represents the currently associated network, matching the icon
// used for the Wi-Fi section in the sidebar.
func rowMarker(selected, connected bool) string {
	sel := " "
	if selected {
		sel = tableSelectionMarker.Render("▸")
	}
	status := " "
	if connected {
		status = tableSelectionMarker.Render("󰖩")
	}
	return sel + status + " "
}
