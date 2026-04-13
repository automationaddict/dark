package datetime

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

type Snapshot struct {
	LocalTime    string `json:"local_time"`
	UTCTime      string `json:"utc_time"`
	Timezone     string `json:"timezone"`
	TZAbbrev     string `json:"tz_abbrev"`
	UTCOffset    string `json:"utc_offset"`
	NTPEnabled   bool   `json:"ntp_enabled"`
	NTPSynced    bool   `json:"ntp_synced"`
	NTPServer    string `json:"ntp_server"`
	PollInterval string `json:"poll_interval"`
	Jitter       string `json:"jitter"`
	RTCTime      string `json:"rtc_time"`
	RTCDate      string `json:"rtc_date"`
	RTCInUTC     bool   `json:"rtc_in_utc"`
	CanNTP       bool   `json:"can_ntp"`
	Locale       string `json:"locale"`
	ClockFormat  string `json:"clock_format"` // "12h" or "24h"
	Uptime       string `json:"uptime"`
	Timezones    []string `json:"timezones,omitempty"`
}

func ReadSnapshot() Snapshot {
	s := Snapshot{}

	now := time.Now()
	s.LocalTime = now.Format("Mon 2006-01-02 15:04:05")
	s.UTCTime = now.UTC().Format("Mon 2006-01-02 15:04:05")

	zone, offset := now.Zone()
	s.TZAbbrev = zone
	hours := offset / 3600
	mins := (offset % 3600) / 60
	if mins < 0 {
		mins = -mins
	}
	s.UTCOffset = fmt.Sprintf("%+03d:%02d", hours, mins)

	readTimedateD(&s)
	readTimesyncD(&s)
	readRTC(&s)
	s.Locale = readLocale()
	s.ClockFormat = readClockFormat()
	s.Uptime = readUptime()

	if zones, err := ListTimezones(); err == nil {
		s.Timezones = zones
	}

	return s
}

func SetTimezone(tz string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	obj := conn.Object("org.freedesktop.timedate1", "/org/freedesktop/timedate1")
	return obj.Call("org.freedesktop.timedate1.SetTimezone", 0, tz, false).Err
}

func SetNTP(enabled bool) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	obj := conn.Object("org.freedesktop.timedate1", "/org/freedesktop/timedate1")
	return obj.Call("org.freedesktop.timedate1.SetNTP", 0, enabled, false).Err
}

// SetTime sets the system clock. timeStr must be in "2006-01-02 15:04:05" format.
// NTP must be disabled before calling this.
func SetTime(timeStr string) error {
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return fmt.Errorf("invalid time format (use YYYY-MM-DD HH:MM:SS): %w", err)
	}
	usec := t.UnixMicro()
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	obj := conn.Object("org.freedesktop.timedate1", "/org/freedesktop/timedate1")
	return obj.Call("org.freedesktop.timedate1.SetTime", 0, usec, false, false).Err
}

// SetLocalRTC switches the hardware clock between local time and UTC.
func SetLocalRTC(local bool) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	obj := conn.Object("org.freedesktop.timedate1", "/org/freedesktop/timedate1")
	return obj.Call("org.freedesktop.timedate1.SetLocalRTC", 0, local, true, false).Err
}

// ListTimezones returns all available timezones from systemd.
func ListTimezones() ([]string, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	obj := conn.Object("org.freedesktop.timedate1", "/org/freedesktop/timedate1")
	call := obj.Call("org.freedesktop.timedate1.ListTimezones", 0)
	if call.Err != nil {
		return nil, call.Err
	}
	var zones []string
	if err := call.Store(&zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func ToggleClockFormat() error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "waybar", "config.jsonc")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	if strings.Contains(content, "%I") {
		// Switch from 12h to 24h
		content = strings.ReplaceAll(content, "%I:%M %p", "%H:%M")
		content = strings.ReplaceAll(content, "%I", "%H")
		// Remove %p (AM/PM) if still present
		content = strings.ReplaceAll(content, " %p", "")
		content = strings.ReplaceAll(content, "%p", "")
	} else {
		// Switch from 24h to 12h
		content = strings.ReplaceAll(content, "%H:%M", "%I:%M %p")
		content = strings.ReplaceAll(content, "%H", "%I")
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

// --- D-Bus reads ---

func readTimedateD(s *Snapshot) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return
	}
	obj := conn.Object("org.freedesktop.timedate1", "/org/freedesktop/timedate1")

	if v, err := obj.GetProperty("org.freedesktop.timedate1.Timezone"); err == nil {
		s.Timezone, _ = v.Value().(string)
	}
	if v, err := obj.GetProperty("org.freedesktop.timedate1.NTP"); err == nil {
		s.NTPEnabled, _ = v.Value().(bool)
	}
	if v, err := obj.GetProperty("org.freedesktop.timedate1.NTPSynchronized"); err == nil {
		s.NTPSynced, _ = v.Value().(bool)
	}

	localRTC := false
	if v, err := obj.GetProperty("org.freedesktop.timedate1.LocalRTC"); err == nil {
		localRTC, _ = v.Value().(bool)
	}
	s.RTCInUTC = !localRTC

	if v, err := obj.GetProperty("org.freedesktop.timedate1.CanNTP"); err == nil {
		s.CanNTP, _ = v.Value().(bool)
	}
}

func readTimesyncD(s *Snapshot) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return
	}
	obj := conn.Object("org.freedesktop.timesync1", "/org/freedesktop/timesync1")

	if v, err := obj.GetProperty("org.freedesktop.timesync1.Manager.ServerName"); err == nil {
		s.NTPServer, _ = v.Value().(string)
	}
	if v, err := obj.GetProperty("org.freedesktop.timesync1.Manager.PollIntervalUSec"); err == nil {
		if usec, ok := v.Value().(uint64); ok {
			dur := time.Duration(usec) * time.Microsecond
			if dur >= time.Minute {
				s.PollInterval = fmt.Sprintf("%.0fmin", dur.Minutes())
			} else {
				s.PollInterval = fmt.Sprintf("%.0fs", dur.Seconds())
			}
		}
	}
	if v, err := obj.GetProperty("org.freedesktop.timesync1.Manager.Jitter"); err == nil {
		if usec, ok := v.Value().(uint64); ok {
			s.Jitter = fmt.Sprintf("%.1fms", float64(usec)/1000)
		}
	}
}

// --- Sysfs / config reads ---

func readRTC(s *Snapshot) {
	s.RTCTime = readSysStr("/sys/class/rtc/rtc0", "time")
	s.RTCDate = readSysStr("/sys/class/rtc/rtc0", "date")
}

func readLocale() string {
	f, err := os.Open("/etc/locale.conf")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "LANG=") {
			return strings.TrimPrefix(line, "LANG=")
		}
	}
	return ""
}

func readClockFormat() string {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "waybar", "config.jsonc")
	data, err := os.ReadFile(path)
	if err != nil {
		return "24h"
	}
	if strings.Contains(string(data), "%I") {
		return "12h"
	}
	return "24h"
}

func readUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return ""
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return ""
	}
	var secs float64
	fmt.Sscanf(parts[0], "%f", &secs)
	dur := time.Duration(secs * float64(time.Second))
	days := int(dur.Hours()) / 24
	hours := int(dur.Hours()) % 24
	mins := int(dur.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func readSysStr(dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
