package input

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// --- Device enumeration from /proc/bus/input/devices ---

type rawDevice struct {
	name    string
	phys    string
	uniq    string
	bus     string
	vendor  string
	product string
	ev      string
	key     string
	rel     string
	abs     string
	led     string
	handler string
}

func parseInputDevices() []Device {
	f, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return nil
	}
	defer f.Close()

	var devices []Device
	var cur rawDevice
	scanner := bufio.NewScanner(f)

	flush := func() {
		if cur.name == "" {
			return
		}
		event := ""
		for _, h := range strings.Fields(cur.handler) {
			if strings.HasPrefix(h, "event") {
				event = h
				break
			}
		}

		inhibited := false
		if event != "" {
			inh := readSysStr(filepath.Join("/sys/class/input", event, "device"), "inhibited")
			inhibited = inh == "1"
		}

		devices = append(devices, Device{
			Name:      cur.name,
			Event:     event,
			Bus:       busTypeName(cur.bus),
			VendorID:  cur.vendor,
			ProductID: cur.product,
			Phys:      cur.phys,
			Uniq:      cur.uniq,
			Inhibited: inhibited,
		})
		cur = rawDevice{}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if len(line) < 3 {
			continue
		}
		prefix := line[:3]
		rest := line[3:]
		switch prefix {
		case "N: ":
			cur.name = trimQuoted(strings.TrimPrefix(rest, "Name="))
		case "P: ":
			cur.phys = strings.TrimPrefix(rest, "Phys=")
		case "U: ":
			cur.uniq = strings.TrimPrefix(rest, "Uniq=")
		case "I: ":
			for _, field := range strings.Fields(rest) {
				kv := strings.SplitN(field, "=", 2)
				if len(kv) != 2 {
					continue
				}
				switch kv[0] {
				case "Bus":
					cur.bus = kv[1]
				case "Vendor":
					cur.vendor = kv[1]
				case "Product":
					cur.product = kv[1]
				}
			}
		case "H: ":
			cur.handler = strings.TrimPrefix(rest, "Handlers=")
		case "B: ":
			kv := strings.SplitN(rest, "=", 2)
			if len(kv) == 2 {
				switch kv[0] {
				case "EV":
					cur.ev = kv[1]
				case "KEY":
					cur.key = kv[1]
				case "REL":
					cur.rel = kv[1]
				case "ABS":
					cur.abs = kv[1]
				case "LED":
					cur.led = kv[1]
				}
			}
		}
	}
	flush()
	return devices
}

func classifyDevice(d Device) string {
	sysDir := filepath.Join("/sys/class/input", d.Event, "device", "capabilities")
	ev := readSysStr(sysDir, "ev")
	key := readSysStr(sysDir, "key")
	rel := readSysStr(sysDir, "rel")
	abs := readSysStr(sysDir, "abs")

	if d.Event == "" {
		return ""
	}

	evBits := parseHexBits(ev)
	hasBitEV := func(bit int) bool { return evBits&(1<<bit) != 0 }

	hasKey := hasBitEV(1)
	hasRel := hasBitEV(2)
	hasAbs := hasBitEV(3)

	nameLower := strings.ToLower(d.Name)

	if strings.Contains(nameLower, "touchpad") || strings.Contains(nameLower, "trackpad") {
		return "touchpad"
	}
	if hasAbs && hasKey && abs != "" && abs != "0" {
		if strings.Contains(nameLower, "touch") || strings.Contains(nameLower, "pad") {
			return "touchpad"
		}
	}

	if strings.Contains(nameLower, "mouse") || strings.Contains(nameLower, "trackball") {
		return "mouse"
	}
	if hasRel && rel != "" && rel != "0" && !hasAbs {
		if key != "" && key != "0" {
			return "mouse"
		}
	}

	if strings.Contains(nameLower, "keyboard") || strings.Contains(nameLower, "kbd") {
		return "keyboard"
	}
	if hasKey && !hasRel && !hasAbs {
		keyBits := parseHexBits(key)
		if keyBits > 0xffff {
			return "keyboard"
		}
	}

	if strings.Contains(nameLower, "button") ||
		strings.Contains(nameLower, "lid") ||
		strings.Contains(nameLower, "video") ||
		strings.Contains(nameLower, "speaker") ||
		strings.Contains(nameLower, "headphone") {
		return "other"
	}

	if hasKey || hasRel || hasAbs {
		return "other"
	}
	return ""
}

func hasCapability(d Device, cap string) bool {
	if d.Event == "" {
		return false
	}
	v := readSysStr(filepath.Join("/sys/class/input", d.Event, "device", "capabilities"), cap)
	return v != "" && v != "0"
}

// --- LED reading from /sys/class/leds ---

func readLEDs() []LED {
	entries, _ := os.ReadDir("/sys/class/leds")
	var leds []LED
	for _, e := range entries {
		name := e.Name()
		if !strings.Contains(name, "capslock") &&
			!strings.Contains(name, "numlock") &&
			!strings.Contains(name, "scrolllock") &&
			!strings.Contains(name, "kbd_backlight") {
			continue
		}
		dir := filepath.Join("/sys/class/leds", name)
		leds = append(leds, LED{
			Name:          name,
			Brightness:    readSysInt(dir, "brightness"),
			MaxBrightness: readSysInt(dir, "max_brightness"),
		})
	}
	return leds
}

// --- Helpers ---

func trimQuoted(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func busTypeName(hex string) string {
	switch hex {
	case "0003":
		return "USB"
	case "0005":
		return "Bluetooth"
	case "0011":
		return "ISA"
	case "0018":
		return "I2C"
	case "0019":
		return "ACPI"
	case "001D":
		return "Virtual"
	default:
		return hex
	}
}

func parseHexBits(s string) uint64 {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0
	}
	v, _ := strconv.ParseUint(parts[len(parts)-1], 16, 64)
	return v
}
