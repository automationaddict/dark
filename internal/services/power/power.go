package power

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/godbus/dbus/v5"
)

type Snapshot struct {
	Batteries    []Battery    `json:"batteries"`
	ACAdapters   []ACAdapter  `json:"ac_adapters"`
	Profile      string       `json:"profile"`
	Profiles     []string     `json:"profiles"`
	CPUs         []CPU        `json:"cpus"`
	Governor     string       `json:"governor"`
	Governors    []string     `json:"governors"`
	EPP          string       `json:"epp"`
	EPPs         []string     `json:"epps"`
	PState       string       `json:"pstate"`
	Thermals     []Thermal    `json:"thermals"`
	Fans         []Fan        `json:"fans"`
	GPU          GPUInfo      `json:"gpu"`
	Peripherals  []Peripheral `json:"peripherals"`
	SleepStates  []string       `json:"sleep_states"`
	MemSleep     string         `json:"mem_sleep"`
	Idle         IdleConfig     `json:"idle"`
	Buttons      SystemButtons  `json:"buttons"`
}

type IdleConfig struct {
	Running          bool `json:"running"`
	ScreensaverSec   int  `json:"screensaver_sec"`
	LockSec          int  `json:"lock_sec"`
	DPMSOffSec       int  `json:"dpms_off_sec"`
	KbdBacklightSec  int  `json:"kbd_backlight_sec"`
}

type SystemButtons struct {
	PowerKeyAction string `json:"power_key_action"`
	LidSwitch      string `json:"lid_switch"`
	LidSwitchPower string `json:"lid_switch_power"`
	LidSwitchDocked string `json:"lid_switch_docked"`
	ShowBatteryPct bool   `json:"show_battery_pct"`
}

type Battery struct {
	Name             string  `json:"name"`
	Status           string  `json:"status"`
	Capacity         int     `json:"capacity"`
	EnergyNow        float64 `json:"energy_now"`
	EnergyFull       float64 `json:"energy_full"`
	EnergyFullDesign float64 `json:"energy_full_design"`
	Voltage          float64 `json:"voltage"`
	PowerNow         float64 `json:"power_now"`
	Technology       string  `json:"technology"`
	CycleCount       int     `json:"cycle_count"`
}

func (b Battery) Health() int {
	if b.EnergyFullDesign <= 0 {
		return 100
	}
	h := int(b.EnergyFull / b.EnergyFullDesign * 100)
	if h > 100 {
		h = 100
	}
	return h
}

type ACAdapter struct {
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

type CPU struct {
	ID      int `json:"id"`
	CurFreq int `json:"cur_freq"` // kHz
	MinFreq int `json:"min_freq"`
	MaxFreq int `json:"max_freq"`
}

type Thermal struct {
	Label string  `json:"label"`
	Temp  float64 `json:"temp"` // celsius
}

type Fan struct {
	Label string `json:"label"`
	RPM   int    `json:"rpm"`
}

type GPUInfo struct {
	Temp     float64 `json:"temp"`
	PowerW   float64 `json:"power_w"`
	ClockMHz int     `json:"clock_mhz"`
}

type Peripheral struct {
	Name    string `json:"name"`
	Model   string `json:"model"`
	Charge  int    `json:"charge"`
	Status  string `json:"status"`
}

func ReadSnapshot() Snapshot {
	var s Snapshot
	s.Batteries = readBatteries()
	s.ACAdapters = readACAdapters()
	s.Profile, s.Profiles = readPowerProfiles()
	s.CPUs, s.Governor, s.Governors, s.EPP, s.EPPs, s.PState = readCPU()
	s.Thermals = readThermals()
	s.Fans = readFans()
	s.GPU = readGPU()
	s.Peripherals = readPeripherals()
	s.SleepStates, s.MemSleep = readSleep()
	s.Idle = readIdleConfig()
	s.Buttons = readSystemButtons()
	return s
}

// --- Power Profiles via D-Bus (net.hadess.PowerProfiles) ---

func SetProfile(profile string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	obj := conn.Object("net.hadess.PowerProfiles", "/net/hadess/PowerProfiles")
	return obj.SetProperty("net.hadess.PowerProfiles.ActiveProfile", dbus.MakeVariant(profile))
}

func readPowerProfiles() (string, []string) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return "", nil
	}
	obj := conn.Object("net.hadess.PowerProfiles", "/net/hadess/PowerProfiles")

	activeV, err := obj.GetProperty("net.hadess.PowerProfiles.ActiveProfile")
	if err != nil {
		return "", nil
	}
	active, _ := activeV.Value().(string)

	profilesV, err := obj.GetProperty("net.hadess.PowerProfiles.Profiles")
	if err != nil {
		return active, nil
	}

	var profiles []string
	if arr, ok := profilesV.Value().([]map[string]dbus.Variant); ok {
		for _, m := range arr {
			if p, ok := m["Profile"]; ok {
				if name, ok := p.Value().(string); ok {
					profiles = append(profiles, name)
				}
			}
		}
	}

	return active, profiles
}

// --- CPU Governor / EPP via sysfs ---

// SetSystemButton updates a logind.conf handle key via the privileged
// helper. key is one of: HandlePowerKey, HandleLidSwitch,
// HandleLidSwitchExternalPower, HandleLidSwitchDocked.
func SetSystemButton(key, value string) error {
	return runHelper("logind-set", key, value)
}

func runHelper(args ...string) error {
	helper, err := helperPath()
	if err != nil {
		return err
	}
	full := append([]string{helper}, args...)
	cmd := exec.Command("pkexec", full...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}

func helperPath() (string, error) {
	if env := os.Getenv("DARK_HELPER"); env != "" {
		return env, nil
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "dark-helper")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	for _, p := range []string{"/usr/local/bin/dark-helper", "/usr/bin/dark-helper"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("dark-helper binary not found")
}

func SetGovernor(gov string) error {
	cpus, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/scaling_governor")
	for _, p := range cpus {
		os.WriteFile(p, []byte(gov), 0o644)
	}
	return nil
}

func SetEPP(epp string) error {
	cpus, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/energy_performance_preference")
	for _, p := range cpus {
		os.WriteFile(p, []byte(epp), 0o644)
	}
	return nil
}

// --- Batteries & AC via sysfs ---

func readBatteries() []Battery {
	entries, _ := os.ReadDir("/sys/class/power_supply")
	var bats []Battery
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "BAT") {
			continue
		}
		dir := filepath.Join("/sys/class/power_supply", e.Name())
		b := Battery{
			Name:             e.Name(),
			Status:           readSysStr(dir, "status"),
			Capacity:         readSysInt(dir, "capacity"),
			EnergyNow:        float64(readSysInt(dir, "energy_now")) / 1e6,
			EnergyFull:       float64(readSysInt(dir, "energy_full")) / 1e6,
			EnergyFullDesign: float64(readSysInt(dir, "energy_full_design")) / 1e6,
			Voltage:          float64(readSysInt(dir, "voltage_now")) / 1e6,
			PowerNow:         float64(readSysInt(dir, "power_now")) / 1e6,
			Technology:       readSysStr(dir, "technology"),
			CycleCount:       readSysInt(dir, "cycle_count"),
		}
		bats = append(bats, b)
	}
	return bats
}

func readACAdapters() []ACAdapter {
	entries, _ := os.ReadDir("/sys/class/power_supply")
	var acs []ACAdapter
	for _, e := range entries {
		dir := filepath.Join("/sys/class/power_supply", e.Name())
		typ := readSysStr(dir, "type")
		if typ != "Mains" {
			continue
		}
		acs = append(acs, ACAdapter{
			Name:   e.Name(),
			Online: readSysInt(dir, "online") == 1,
		})
	}
	return acs
}

// --- CPU info via sysfs ---

func readCPU() ([]CPU, string, []string, string, []string, string) {
	cpuDirs, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	var cpus []CPU
	var governor string
	var governors []string
	var epp string
	var epps []string

	for _, d := range cpuDirs {
		name := filepath.Base(d)
		id, err := strconv.Atoi(strings.TrimPrefix(name, "cpu"))
		if err != nil {
			continue
		}
		freq := filepath.Join(d, "cpufreq")
		if _, err := os.Stat(freq); err != nil {
			continue
		}
		cpus = append(cpus, CPU{
			ID:      id,
			CurFreq: readSysInt(freq, "scaling_cur_freq"),
			MinFreq: readSysInt(freq, "scaling_min_freq"),
			MaxFreq: readSysInt(freq, "scaling_max_freq"),
		})
		if governor == "" {
			governor = readSysStr(freq, "scaling_governor")
		}
		if len(governors) == 0 {
			g := readSysStr(freq, "scaling_available_governors")
			if g != "" {
				governors = strings.Fields(g)
			}
		}
		if epp == "" {
			epp = readSysStr(freq, "energy_performance_preference")
		}
		if len(epps) == 0 {
			e := readSysStr(freq, "energy_performance_available_preferences")
			if e != "" {
				epps = strings.Fields(e)
			}
		}
	}

	pstate := ""
	if s := readSysStr("/sys/devices/system/cpu/amd_pstate", "status"); s != "" {
		pstate = "amd_pstate (" + s + ")"
	} else if _, err := os.Stat("/sys/devices/system/cpu/intel_pstate"); err == nil {
		pstate = "intel_pstate"
	}

	return cpus, governor, governors, epp, epps, pstate
}

// --- Thermals via /sys/class/hwmon ---

func readThermals() []Thermal {
	hwmons, _ := filepath.Glob("/sys/class/hwmon/hwmon*")
	var thermals []Thermal
	for _, dir := range hwmons {
		chipName := readSysStr(dir, "name")
		for i := 1; ; i++ {
			tempFile := fmt.Sprintf("temp%d_input", i)
			path := filepath.Join(dir, tempFile)
			if _, err := os.Stat(path); err != nil {
				break
			}
			raw := readFileInt(path)
			if raw == 0 {
				continue
			}
			labelFile := fmt.Sprintf("temp%d_label", i)
			label := readSysStr(dir, labelFile)
			if label == "" {
				label = fmt.Sprintf("temp%d", i)
			}
			name := label
			if chipName != "" {
				name = chipName + " · " + label
			}
			thermals = append(thermals, Thermal{
				Label: name,
				Temp:  float64(raw) / 1000.0,
			})
		}
	}
	return thermals
}

// --- Fans via /sys/class/hwmon ---

func readFans() []Fan {
	inputs, _ := filepath.Glob("/sys/class/hwmon/hwmon*/fan*_input")
	var fans []Fan
	for _, p := range inputs {
		dir := filepath.Dir(p)
		base := filepath.Base(p)
		num := strings.TrimSuffix(strings.TrimPrefix(base, "fan"), "_input")
		labelFile := fmt.Sprintf("fan%s_label", num)
		label := readSysStr(dir, labelFile)
		if label == "" {
			hwmonName := readSysStr(dir, "name")
			label = hwmonName + " fan" + num
		}
		rpm := readFileInt(p)
		fans = append(fans, Fan{Label: label, RPM: rpm})
	}
	return fans
}

// --- GPU via /sys/class/hwmon (amdgpu) ---

func readGPU() GPUInfo {
	var gpu GPUInfo
	hwmons, _ := filepath.Glob("/sys/class/hwmon/hwmon*/name")
	for _, n := range hwmons {
		name, _ := os.ReadFile(n)
		if strings.TrimSpace(string(name)) == "amdgpu" {
			dir := filepath.Dir(n)
			tempRaw := readFileInt(filepath.Join(dir, "temp1_input"))
			gpu.Temp = float64(tempRaw) / 1000.0
			powerRaw := readFileInt(filepath.Join(dir, "power1_average"))
			gpu.PowerW = float64(powerRaw) / 1e6
			freqRaw := readFileInt(filepath.Join(dir, "freq1_input"))
			gpu.ClockMHz = freqRaw / 1_000_000
			break
		}
	}
	return gpu
}

// --- Peripherals via D-Bus (org.freedesktop.UPower) ---

func readPeripherals() []Peripheral {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil
	}

	obj := conn.Object("org.freedesktop.UPower", "/org/freedesktop/UPower")
	var paths []dbus.ObjectPath
	if err := obj.Call("org.freedesktop.UPower.EnumerateDevices", 0).Store(&paths); err != nil {
		return nil
	}

	var periph []Peripheral
	for _, path := range paths {
		sp := string(path)
		if strings.Contains(sp, "battery_BAT") ||
			strings.Contains(sp, "line_power") ||
			strings.Contains(sp, "DisplayDevice") {
			continue
		}

		dev := conn.Object("org.freedesktop.UPower", path)
		p := Peripheral{}

		if v, err := dev.GetProperty("org.freedesktop.UPower.Device.Model"); err == nil {
			p.Model, _ = v.Value().(string)
		}
		if v, err := dev.GetProperty("org.freedesktop.UPower.Device.NativePath"); err == nil {
			p.Name, _ = v.Value().(string)
		}
		if v, err := dev.GetProperty("org.freedesktop.UPower.Device.Percentage"); err == nil {
			if pct, ok := v.Value().(float64); ok {
				p.Charge = int(pct)
			}
		}
		if v, err := dev.GetProperty("org.freedesktop.UPower.Device.State"); err == nil {
			if state, ok := v.Value().(uint32); ok {
				switch state {
				case 1:
					p.Status = "charging"
				case 2:
					p.Status = "discharging"
				case 4:
					p.Status = "fully-charged"
				default:
					p.Status = "unknown"
				}
			}
		}

		if p.Model != "" || p.Name != "" {
			if p.Name == "" {
				p.Name = filepath.Base(sp)
			}
			periph = append(periph, p)
		}
	}
	return periph
}

// --- Sleep states via sysfs ---

func readSleep() ([]string, string) {
	states := readSysStr("/sys/power", "state")
	memSleep := readSysStr("/sys/power", "mem_sleep")
	var stateList []string
	if states != "" {
		stateList = strings.Fields(states)
	}
	active := ""
	if idx := strings.Index(memSleep, "["); idx >= 0 {
		end := strings.Index(memSleep, "]")
		if end > idx {
			active = memSleep[idx+1 : end]
		}
	}
	return stateList, active
}

// --- Idle config from hypridle.conf ---

func readIdleConfig() IdleConfig {
	cfg := IdleConfig{}

	// Check if hypridle is running
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) == "hypridle" {
			cfg.Running = true
			break
		}
	}

	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "hypr", "hypridle.conf")
	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inListener := false
	var currentTimeout int
	var currentOnTimeout string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if strings.HasPrefix(line, "listener") && strings.Contains(line, "{") {
			inListener = true
			currentTimeout = 0
			currentOnTimeout = ""
			continue
		}

		if line == "}" && inListener {
			inListener = false
			if currentTimeout > 0 {
				lower := strings.ToLower(currentOnTimeout)
				switch {
				case strings.Contains(lower, "screensaver"):
					cfg.ScreensaverSec = currentTimeout
				case strings.Contains(lower, "lock"):
					cfg.LockSec = currentTimeout
				case strings.Contains(lower, "dpms off"):
					cfg.DPMSOffSec = currentTimeout
				case strings.Contains(lower, "kbd_backlight") || strings.Contains(lower, "brightnessctl"):
					cfg.KbdBacklightSec = currentTimeout
				}
			}
			continue
		}

		if !inListener {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.SplitN(val, "#", 2)[0]
		val = strings.TrimSpace(val)

		switch key {
		case "timeout":
			v, _ := strconv.Atoi(val)
			currentTimeout = v
		case "on-timeout":
			currentOnTimeout = val
		}
	}

	return cfg
}

// --- System buttons from logind.conf + waybar ---

func readSystemButtons() SystemButtons {
	sb := SystemButtons{
		PowerKeyAction:  "suspend",
		LidSwitch:       "suspend",
		LidSwitchPower:  "suspend",
		LidSwitchDocked: "ignore",
	}

	f, err := os.Open("/etc/systemd/logind.conf")
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			switch key {
			case "HandlePowerKey":
				sb.PowerKeyAction = val
			case "HandleLidSwitch":
				sb.LidSwitch = val
			case "HandleLidSwitchExternalPower":
				sb.LidSwitchPower = val
			case "HandleLidSwitchDocked":
				sb.LidSwitchDocked = val
			}
		}
	}

	sb.ShowBatteryPct = detectBatteryPctVisible()
	return sb
}

func detectBatteryPctVisible() bool {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "waybar", "config.jsonc")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	// The default format shows {capacity}% — if discharging format
	// omits it, the percentage is effectively hidden during use.
	// Check if the main format includes {capacity}.
	if strings.Contains(content, `"format-discharging"`) {
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, `"format-discharging"`) {
				return strings.Contains(line, "{capacity}")
			}
		}
	}
	return true
}

// SetIdleTimeout updates a specific listener's timeout in hypridle.conf.
// The kind parameter identifies the listener: "screensaver", "lock",
// "dpms", or "kbd". The value is in seconds. After writing, sends
// SIGUSR1 to hypridle to reload the config.
func SetIdleTimeout(kind string, seconds int) error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "hypr", "hypridle.conf")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read hypridle.conf: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	result := patchIdleTimeout(lines, kind, seconds)

	if err := os.WriteFile(path, []byte(strings.Join(result, "\n")), 0o644); err != nil {
		return fmt.Errorf("write hypridle.conf: %w", err)
	}

	// Signal hypridle to reload. It re-reads the config on SIGUSR1.
	reloadHypridle()
	return nil
}

// patchIdleTimeout scans hypridle.conf lines for the listener whose
// on-timeout matches kind, then updates its timeout value.
func patchIdleTimeout(lines []string, kind string, seconds int) []string {
	// Two-pass: first find which listener block matches, then patch it.
	type listenerBlock struct {
		startLine   int
		timeoutLine int
		onTimeout   string
	}

	var blocks []listenerBlock
	inListener := false
	var cur listenerBlock

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "listener") && strings.Contains(trimmed, "{") {
			inListener = true
			cur = listenerBlock{startLine: i, timeoutLine: -1}
			continue
		}
		if trimmed == "}" && inListener {
			inListener = false
			blocks = append(blocks, cur)
			continue
		}
		if !inListener {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.SplitN(val, "#", 2)[0]
		val = strings.TrimSpace(val)
		switch key {
		case "timeout":
			cur.timeoutLine = i
		case "on-timeout":
			cur.onTimeout = val
		}
	}

	// Match kind to the correct block.
	for _, b := range blocks {
		lower := strings.ToLower(b.onTimeout)
		matched := false
		switch kind {
		case "screensaver":
			matched = strings.Contains(lower, "screensaver")
		case "lock":
			matched = strings.Contains(lower, "lock")
		case "dpms":
			matched = strings.Contains(lower, "dpms off")
		case "kbd":
			matched = strings.Contains(lower, "kbd_backlight") || strings.Contains(lower, "brightnessctl")
		}
		if matched && b.timeoutLine >= 0 {
			// Preserve indent and any inline comment.
			orig := lines[b.timeoutLine]
			indent := orig[:len(orig)-len(strings.TrimLeft(orig, " \t"))]
			comment := ""
			if idx := strings.Index(orig, "#"); idx >= 0 {
				comment = " " + strings.TrimSpace(orig[idx:])
			}
			lines[b.timeoutLine] = fmt.Sprintf("%stimeout = %d%s", indent, seconds, comment)
			break
		}
	}

	return lines
}

func reloadHypridle() {
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) == "hypridle" {
			pid, err := strconv.Atoi(e.Name())
			if err != nil {
				continue
			}
			proc, err := os.FindProcess(pid)
			if err != nil {
				continue
			}
			_ = proc.Signal(syscall.SIGUSR1)
			return
		}
	}
}

// --- sysfs helpers ---

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

func readFileInt(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return v
}
