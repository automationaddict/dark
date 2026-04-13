package power

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
