package input

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Snapshot struct {
	Keyboards []Keyboard `json:"keyboards"`
	Touchpads []Touchpad `json:"touchpads"`
	Mice      []Mouse    `json:"mice"`
	Others    []Device   `json:"others"`
	LEDs      []LED      `json:"leds"`
	Config    InputConfig `json:"config"`
}

type Device struct {
	Name      string `json:"name"`
	Event     string `json:"event"`
	Bus       string `json:"bus"`
	VendorID  string `json:"vendor_id"`
	ProductID string `json:"product_id"`
	Phys      string `json:"phys"`
	Uniq      string `json:"uniq"`
	Inhibited bool   `json:"inhibited"`
}

type Keyboard struct {
	Device
	HasLEDs bool `json:"has_leds"`
}

type Touchpad struct {
	Device
}

type Mouse struct {
	Device
}

type LED struct {
	Name          string `json:"name"`
	Brightness    int    `json:"brightness"`
	MaxBrightness int    `json:"max_brightness"`
}

type InputConfig struct {
	// Keyboard
	KBLayout       string `json:"kb_layout"`
	KBVariant      string `json:"kb_variant"`
	KBModel        string `json:"kb_model"`
	KBOptions      string `json:"kb_options"`
	RepeatRate     int    `json:"repeat_rate"`
	RepeatDelay    int    `json:"repeat_delay"`
	NumlockDefault bool   `json:"numlock_default"`

	// Mouse
	Sensitivity  float64 `json:"sensitivity"`
	AccelProfile string  `json:"accel_profile"`
	ForceNoAccel bool    `json:"force_no_accel"`
	LeftHanded   bool    `json:"left_handed"`
	ScrollMethod string  `json:"scroll_method"`
	FollowMouse  int     `json:"follow_mouse"`

	// Touchpad
	NaturalScroll       bool    `json:"natural_scroll"`
	ScrollFactor        float64 `json:"scroll_factor"`
	DisableWhileTyping  bool    `json:"disable_while_typing"`
	TapToClick          bool    `json:"tap_to_click"`
	TapAndDrag          bool    `json:"tap_and_drag"`
	DragLock            bool    `json:"drag_lock"`
	MiddleButtonEmu     bool    `json:"middle_button_emulation"`
	ClickfingerBehavior bool    `json:"clickfinger_behavior"`
}

func ReadSnapshot() Snapshot {
	devices := parseInputDevices()
	var s Snapshot

	for _, d := range devices {
		switch classifyDevice(d) {
		case "keyboard":
			s.Keyboards = append(s.Keyboards, Keyboard{
				Device:  d,
				HasLEDs: hasCapability(d, "led"),
			})
		case "touchpad":
			s.Touchpads = append(s.Touchpads, Touchpad{Device: d})
		case "mouse":
			s.Mice = append(s.Mice, Mouse{Device: d})
		case "other":
			s.Others = append(s.Others, d)
		}
	}

	s.LEDs = readLEDs()
	s.Config = readHyprlandConfig()
	return s
}

// --- Hyprland keyword mutations ---

func SetRepeatRate(rate int) error {
	return hyprctl("keyword", "input:repeat_rate", strconv.Itoa(rate))
}

func SetRepeatDelay(delay int) error {
	return hyprctl("keyword", "input:repeat_delay", strconv.Itoa(delay))
}

func SetSensitivity(sens float64) error {
	return hyprctl("keyword", "input:sensitivity", fmt.Sprintf("%.2f", sens))
}

func SetNaturalScroll(enabled bool) error {
	v := "false"
	if enabled {
		v = "true"
	}
	return hyprctl("keyword", "input:touchpad:natural_scroll", v)
}

func SetScrollFactor(factor float64) error {
	return hyprctl("keyword", "input:touchpad:scroll_factor", fmt.Sprintf("%.2f", factor))
}

func SetKBLayout(layout string) error {
	return hyprctl("keyword", "input:kb_layout", layout)
}

func SetAccelProfile(profile string) error {
	return hyprctl("keyword", "input:accel_profile", profile)
}

func SetForceNoAccel(enabled bool) error {
	return hyprctl("keyword", "input:force_no_accel", boolStr(enabled))
}

func SetLeftHanded(enabled bool) error {
	return hyprctl("keyword", "input:left_handed", boolStr(enabled))
}

func SetFollowMouse(mode int) error {
	return hyprctl("keyword", "input:follow_mouse", strconv.Itoa(mode))
}

func SetDisableWhileTyping(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:disable_while_typing", boolStr(enabled))
}

func SetTapToClick(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:tap-to-click", boolStr(enabled))
}

func SetTapAndDrag(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:tap-and-drag", boolStr(enabled))
}

func SetDragLock(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:drag_lock", boolStr(enabled))
}

func SetMiddleButtonEmu(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:middle_button_emulation", boolStr(enabled))
}

func SetClickfingerBehavior(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:clickfinger_behavior", boolStr(enabled))
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func hyprctl(args ...string) error {
	cmd := exec.Command("hyprctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hyprctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

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

// --- Hyprland input config parsing ---

func readHyprlandConfig() InputConfig {
	cfg := InputConfig{
		KBLayout:           "us",
		RepeatRate:         25,
		RepeatDelay:        600,
		FollowMouse:        1,
		DisableWhileTyping: true,
		TapToClick:         true,
		TapAndDrag:         true,
	}

	cfg.KBLayout = hyprOptionStr("input:kb_layout", cfg.KBLayout)
	cfg.KBVariant = hyprOptionStr("input:kb_variant", "")
	cfg.KBModel = hyprOptionStr("input:kb_model", "")
	cfg.KBOptions = hyprOptionStr("input:kb_options", "")
	cfg.RepeatRate = hyprOptionInt("input:repeat_rate", cfg.RepeatRate)
	cfg.RepeatDelay = hyprOptionInt("input:repeat_delay", cfg.RepeatDelay)
	cfg.NumlockDefault = hyprOptionBool("input:numlock_by_default", false)

	cfg.Sensitivity = hyprOptionFloat("input:sensitivity", 0)
	cfg.AccelProfile = hyprOptionStr("input:accel_profile", "")
	cfg.ForceNoAccel = hyprOptionBool("input:force_no_accel", false)
	cfg.LeftHanded = hyprOptionBool("input:left_handed", false)
	cfg.ScrollMethod = hyprOptionStr("input:scroll_method", "")
	cfg.FollowMouse = hyprOptionInt("input:follow_mouse", 1)

	cfg.NaturalScroll = hyprOptionBool("input:touchpad:natural_scroll", false)
	cfg.ScrollFactor = hyprOptionFloat("input:touchpad:scroll_factor", 1.0)
	cfg.DisableWhileTyping = hyprOptionBool("input:touchpad:disable_while_typing", true)
	cfg.TapToClick = hyprOptionBool("input:touchpad:tap-to-click", true)
	cfg.TapAndDrag = hyprOptionBool("input:touchpad:tap-and-drag", true)
	cfg.DragLock = hyprOptionBool("input:touchpad:drag_lock", false)
	cfg.MiddleButtonEmu = hyprOptionBool("input:touchpad:middle_button_emulation", false)
	cfg.ClickfingerBehavior = hyprOptionBool("input:touchpad:clickfinger_behavior", false)

	return cfg
}

func hyprOptionStr(key, fallback string) string {
	out, err := exec.Command("hyprctl", "getoption", key, "-j").Output()
	if err != nil {
		return fallback
	}
	// Parse JSON: {"option":"...","str":"value","int":0,"float":0.0,"set":true}
	var result struct {
		Str string `json:"str"`
	}
	if err := parseJSON(out, &result); err != nil {
		return fallback
	}
	if result.Str == "[EMPTY]" || result.Str == "" {
		return fallback
	}
	return result.Str
}

func hyprOptionInt(key string, fallback int) int {
	out, err := exec.Command("hyprctl", "getoption", key, "-j").Output()
	if err != nil {
		return fallback
	}
	var result struct {
		Int int `json:"int"`
	}
	if err := parseJSON(out, &result); err != nil {
		return fallback
	}
	return result.Int
}

func hyprOptionFloat(key string, fallback float64) float64 {
	out, err := exec.Command("hyprctl", "getoption", key, "-j").Output()
	if err != nil {
		return fallback
	}
	var result struct {
		Float float64 `json:"float"`
	}
	if err := parseJSON(out, &result); err != nil {
		return fallback
	}
	return result.Float
}

func hyprOptionBool(key string, fallback bool) bool {
	v := hyprOptionInt(key, -1)
	if v == -1 {
		return fallback
	}
	return v == 1
}

func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
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

func readSysStr(dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readSysInt(dir, name string) int {
	s := readSysStr(dir, name)
	v, _ := strconv.Atoi(s)
	return v
}
