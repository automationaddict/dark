package power

import "github.com/godbus/dbus/v5"

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
