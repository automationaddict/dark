// Package sysinfo gathers and exposes a snapshot of the host system. It runs
// inside darkd; the TUI receives serialized snapshots over the bus.
package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SystemInfo is the wire snapshot of the host. JSON tags are kept short
// because this rides every periodic publish on the bus.
type SystemInfo struct {
	Hostname    string        `json:"hostname"`
	OSPretty    string        `json:"os"`
	Kernel      string        `json:"kernel"`
	Arch        string        `json:"arch"`
	Uptime      time.Duration `json:"uptime_ns"`
	CPUModel    string        `json:"cpu_model"`
	CPUCores    int           `json:"cpu_cores"`
	MemTotal    uint64        `json:"mem_total"`
	MemUsed     uint64        `json:"mem_used"`
	SwapTotal   uint64        `json:"swap_total"`
	SwapUsed    uint64        `json:"swap_used"`
	User        string        `json:"user"`
	Shell       string        `json:"shell"`
	Terminal    string        `json:"terminal"`
	Desktop     string        `json:"desktop"`
	GoVersion      string        `json:"go_version"`
	DarkVersion    string        `json:"dark_version"`
	BinaryPath     string        `json:"binary_path"`
	BinaryMTime    time.Time     `json:"binary_mtime"`
	OmarchyVersion string        `json:"omarchy_version"`
	OmarchyBranch  string        `json:"omarchy_branch"`
	OmarchyChannel string        `json:"omarchy_channel"`
	OmarchyTheme   string        `json:"omarchy_theme"`
	OmarchyAuthor  string        `json:"omarchy_author"`
	PackageCount   int           `json:"package_count"`
	Compositor     string        `json:"compositor"`
	Font           string        `json:"font"`
	InstallAge     time.Duration `json:"install_age_ns"`
	LastUpdate     time.Time     `json:"last_update"`
}

// Gather reads the current system snapshot from the host. binPath is the
// path of the running daemon binary so we can stamp it on the snapshot;
// callers can pass "" if they don't care.
func Gather(binPath string) SystemInfo {
	info := SystemInfo{
		GoVersion:  runtime.Version(),
		BinaryPath: binPath,
	}

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}
	info.OSPretty = readOSPretty()

	var uts syscall.Utsname
	if err := syscall.Uname(&uts); err == nil {
		info.Kernel = utsString(uts.Release[:])
		info.Arch = utsString(uts.Machine[:])
	}

	info.Uptime = readUptime()
	info.CPUModel, info.CPUCores = readCPUInfo()
	info.MemTotal, info.MemUsed, info.SwapTotal, info.SwapUsed = readMemInfo()

	info.User = firstNonEmpty(os.Getenv("USER"), os.Getenv("LOGNAME"))
	info.Shell = baseName(os.Getenv("SHELL"))
	info.Terminal = firstNonEmpty(os.Getenv("TERM_PROGRAM"), os.Getenv("TERM"))
	info.Desktop = firstNonEmpty(
		os.Getenv("XDG_CURRENT_DESKTOP"),
		os.Getenv("XDG_SESSION_DESKTOP"),
		os.Getenv("DESKTOP_SESSION"),
	)
	if info.Desktop == "" && os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		info.Desktop = "Hyprland"
	}

	if fi, err := os.Stat(binPath); err == nil {
		info.BinaryMTime = fi.ModTime()
	}

	info.DarkVersion = DarkVersion
	info.OmarchyVersion, info.OmarchyTheme, info.OmarchyAuthor = readOmarchyInfo()
	info.OmarchyBranch = readOmarchyBranch()
	info.OmarchyChannel = readOmarchyChannel()
	info.PackageCount = readPackageCount()
	info.Compositor = readCompositor()
	info.Font = readFont()
	info.InstallAge = readInstallAge()
	info.LastUpdate = readLastUpdate()

	return info
}

// DarkVersion is the current dark release. Set at build time via
// -ldflags or kept at the dev default. The release workflow injects
// a clean semver like "0.0.1"; local go build/install keeps the
// "-dev" suffix so IsDevBuild can tell them apart.
var DarkVersion = "0.1.0-dev"

// IsDevBuild reports whether the binary was built from a working
// tree rather than a tagged release. True when DarkVersion carries
// a "-dev" suffix — the release workflow strips that when it
// injects the tag value via ldflags.
func IsDevBuild() bool {
	return strings.Contains(DarkVersion, "-dev")
}

func readOmarchyInfo() (version, theme, author string) {
	home, _ := os.UserHomeDir()
	if home == "" {
		return
	}
	base := home + "/.local/share/omarchy"

	if b, err := os.ReadFile(base + "/version"); err == nil {
		version = strings.TrimSpace(string(b))
	}

	themePath := home + "/.config/omarchy/current/theme.name"
	if b, err := os.ReadFile(themePath); err == nil {
		theme = strings.TrimSpace(string(b))
	}

	author = "DHH"
	return
}

func readOSPretty() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(v, `"`)
		}
	}
	return "Linux"
}

func readUptime() time.Duration {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}

func readCPUInfo() (string, int) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", 0
	}
	defer f.Close()

	model := ""
	cores := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "processor") {
			cores++
			continue
		}
		if model == "" && strings.HasPrefix(line, "model name") {
			if i := strings.IndexByte(line, ':'); i >= 0 {
				model = strings.TrimSpace(line[i+1:])
			}
		}
	}
	return model, cores
}

func readMemInfo() (memTotal, memUsed, swapTotal, swapUsed uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()

	vals := map[string]uint64{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := line[:colon]
		fields := strings.Fields(line[colon+1:])
		if len(fields) == 0 {
			continue
		}
		kb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		vals[key] = kb * 1024
	}

	memTotal = vals["MemTotal"]
	memAvail := vals["MemAvailable"]
	if memAvail > memTotal {
		memAvail = memTotal
	}
	memUsed = memTotal - memAvail

	swapTotal = vals["SwapTotal"]
	swapFree := vals["SwapFree"]
	if swapFree > swapTotal {
		swapFree = swapTotal
	}
	swapUsed = swapTotal - swapFree
	return
}

func readOmarchyBranch() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	b, err := os.ReadFile(home + "/.local/share/omarchy/.git/HEAD")
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(b))
	if strings.HasPrefix(ref, "ref: refs/heads/") {
		return strings.TrimPrefix(ref, "ref: refs/heads/")
	}
	return ref
}

func readOmarchyChannel() string {
	data, err := os.ReadFile("/etc/pacman.d/mirrorlist")
	if err != nil {
		return "unknown"
	}
	content := string(data)
	switch {
	case strings.Contains(content, "stable-mirror.omarchy.org"):
		return "stable"
	case strings.Contains(content, "rc-mirror.omarchy.org"):
		return "rc"
	case strings.Contains(content, "mirror.omarchy.org"):
		return "edge"
	}
	return "unknown"
}

func readPackageCount() int {
	entries, err := os.ReadDir("/var/lib/pacman/local")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() && e.Name() != "." && e.Name() != "ALPM_DB_VERSION" {
			count++
		}
	}
	return count
}

func readCompositor() string {
	sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if sig == "" {
		return ""
	}
	// Read version from hyprctl socket
	b, err := os.ReadFile("/tmp/hypr/" + sig + "/.version")
	if err == nil {
		return "Hyprland " + strings.TrimSpace(string(b))
	}
	// Fallback: parse from hyprctl
	outBytes, err := exec.Command("hyprctl", "version").Output()
	out := string(outBytes)
	if err != nil {
		return "Hyprland"
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Hyprland") || strings.Contains(line, "Tag:") {
			return strings.TrimSpace(line)
		}
	}
	return "Hyprland"
}

func readFont() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	// Try ghostty first, then alacritty, then kitty
	for _, path := range []string{
		home + "/.config/ghostty/config",
		home + "/.config/alacritty/alacritty.toml",
		home + "/.config/kitty/kitty.conf",
	} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "font-family") || strings.Contains(line, "font_family") {
				f.Close()
				// Extract value after = or space
				if i := strings.IndexByte(line, '='); i >= 0 {
					return strings.Trim(strings.TrimSpace(line[i+1:]), `"'`)
				}
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					return strings.Join(fields[1:], " ")
				}
			}
		}
		f.Close()
	}
	return ""
}

func readInstallAge() time.Duration {
	var st syscall.Stat_t
	if err := syscall.Stat("/", &st); err != nil {
		return 0
	}
	birth := time.Unix(st.Ctim.Sec, st.Ctim.Nsec)
	if birth.IsZero() || birth.Year() < 2000 {
		return 0
	}
	return time.Since(birth)
}

func readLastUpdate() time.Time {
	fi, err := os.Stat("/var/lib/pacman/sync/core.db")
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

func utsString(b []int8) string {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c == 0 {
			break
		}
		out = append(out, byte(c))
	}
	return string(out)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func baseName(p string) string {
	if p == "" {
		return ""
	}
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "unknown"
	}
	days := int(d / (24 * time.Hour))
	d -= time.Duration(days) * 24 * time.Hour
	hours := int(d / time.Hour)
	d -= time.Duration(hours) * time.Hour
	mins := int(d / time.Minute)

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}
