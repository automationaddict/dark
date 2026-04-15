package tui

import (
	"fmt"

	"github.com/automationaddict/dark/internal/services/bluetooth"
)

// formatBluetoothIcon maps BlueZ's icon hint strings to a short display
// label. BlueZ uses freedesktop icon names like "audio-headset".
func formatBluetoothIcon(icon string) string {
	switch icon {
	case "":
		return "—"
	case "audio-headset", "audio-headphones":
		return "headset"
	case "audio-card":
		return "audio"
	case "input-keyboard":
		return "keyboard"
	case "input-mouse":
		return "mouse"
	case "input-gaming":
		return "gamepad"
	case "input-tablet":
		return "tablet"
	case "phone":
		return "phone"
	case "computer":
		return "computer"
	case "camera-video", "camera-photo":
		return "camera"
	case "printer":
		return "printer"
	case "network-wireless":
		return "network"
	default:
		return icon
	}
}

func formatBluetoothRSSI(rssi int16) string {
	if rssi == 0 {
		return "—"
	}
	return fmt.Sprintf("%d dBm", rssi)
}

// formatBluetoothTimeout renders a BlueZ timeout value in seconds as a
// human-friendly string. Zero is "never" — BlueZ treats it as "no
// timeout, stay in this state until explicitly toggled".
func formatBluetoothTimeout(seconds uint32) string {
	if seconds == 0 {
		return "never"
	}
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	default:
		h := seconds / 3600
		m := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", h, m)
	}
}

func formatBluetoothTxPower(tx int16) string {
	if tx == 0 {
		return "—"
	}
	return fmt.Sprintf("%d dBm", tx)
}

// formatBluetoothAddress joins a MAC with its address type (public or
// random) when BlueZ reports one. LE devices commonly expose a random
// resolvable address so the type is useful debugging context.
func formatBluetoothAddress(addr, addrType string) string {
	if addr == "" {
		return "—"
	}
	if addrType == "" {
		return addr
	}
	return addr + " (" + addrType + ")"
}

// formatBluetoothClass renders the Class of Device as hex plus the
// decoded major class name.
func formatBluetoothClass(class uint32) string {
	if class == 0 {
		return "—"
	}
	major := bluetooth.MajorClassFromClass(class)
	if major == "" {
		return fmt.Sprintf("0x%06x", class)
	}
	return fmt.Sprintf("0x%06x  (%s)", class, major)
}

func formatBluetoothBattery(pct int8) string {
	if pct < 0 {
		return "—"
	}
	return fmt.Sprintf("%d%%", pct)
}
