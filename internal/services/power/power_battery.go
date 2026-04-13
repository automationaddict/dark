package power

import (
	"os"
	"path/filepath"
	"strings"
)

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
