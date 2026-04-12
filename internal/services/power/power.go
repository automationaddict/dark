package power

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	SleepStates  []string     `json:"sleep_states"`
	MemSleep     string       `json:"mem_sleep"`
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
	return s
}

func SetProfile(profile string) error {
	return exec.Command("powerprofilesctl", "set", profile).Run()
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

func readPowerProfiles() (string, []string) {
	out, err := exec.Command("powerprofilesctl", "list").Output()
	if err != nil {
		return "", nil
	}
	var profiles []string
	var active string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, ":") && !strings.Contains(line, "=") {
			name := strings.TrimSuffix(line, ":")
			isActive := strings.HasPrefix(name, "*")
			name = strings.TrimPrefix(name, "* ")
			name = strings.TrimSpace(name)
			profiles = append(profiles, name)
			if isActive {
				active = name
			}
		}
	}
	if active == "" {
		out2, err := exec.Command("powerprofilesctl", "get").Output()
		if err == nil {
			active = strings.TrimSpace(string(out2))
		}
	}
	return active, profiles
}

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

func readThermals() []Thermal {
	out, err := exec.Command("sensors").Output()
	if err != nil {
		return readThermalsFromSysfs()
	}
	var thermals []Thermal
	var currentChip string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.Contains(line, ":") {
			currentChip = line
			continue
		}
		if !strings.Contains(line, "°C") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		label := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		if idx := strings.Index(valStr, "°C"); idx > 0 {
			valStr = strings.TrimSpace(valStr[:idx])
			valStr = strings.TrimPrefix(valStr, "+")
			temp, err := strconv.ParseFloat(valStr, 64)
			if err == nil {
				name := label
				if currentChip != "" {
					name = currentChip + " · " + label
				}
				thermals = append(thermals, Thermal{Label: name, Temp: temp})
			}
		}
	}
	return thermals
}

func readThermalsFromSysfs() []Thermal {
	zones, _ := filepath.Glob("/sys/class/thermal/thermal_zone*")
	var thermals []Thermal
	for _, z := range zones {
		typ := readSysStr(z, "type")
		tempRaw := readSysInt(z, "temp")
		if tempRaw == 0 {
			continue
		}
		thermals = append(thermals, Thermal{
			Label: typ,
			Temp:  float64(tempRaw) / 1000.0,
		})
	}
	return thermals
}

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

func readPeripherals() []Peripheral {
	out, err := exec.Command("upower", "-e").Output()
	if err != nil {
		return nil
	}
	var periph []Peripheral
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "battery_BAT") ||
			strings.Contains(line, "line_power") ||
			strings.Contains(line, "DisplayDevice") {
			continue
		}
		info, err := exec.Command("upower", "-i", line).Output()
		if err != nil {
			continue
		}
		p := parseUPowerDevice(string(info))
		if p.Model != "" || p.Name != "" {
			if p.Name == "" {
				p.Name = filepath.Base(line)
			}
			periph = append(periph, p)
		}
	}
	return periph
}

func parseUPowerDevice(info string) Peripheral {
	var p Peripheral
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "model":
			p.Model = val
		case "native-path":
			if p.Name == "" {
				p.Name = val
			}
		case "percentage":
			pct, _ := strconv.Atoi(strings.TrimSuffix(val, "%"))
			p.Charge = pct
		case "state":
			p.Status = val
		}
	}
	return p
}

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
