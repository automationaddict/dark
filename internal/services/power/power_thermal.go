package power

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
)

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
