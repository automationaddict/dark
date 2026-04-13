package display

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type hyprlandBackend struct {
	backlightDev string
	eventCh      chan struct{}
	closeCh      chan struct{}
}

func newHyprlandBackend() (*hyprlandBackend, error) {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") == "" {
		return nil, fmt.Errorf("not running under Hyprland")
	}
	if _, err := exec.LookPath("hyprctl"); err != nil {
		return nil, fmt.Errorf("hyprctl not found: %w", err)
	}
	b := &hyprlandBackend{
		backlightDev: detectBacklightDevice(),
		eventCh:      make(chan struct{}, 1),
		closeCh:      make(chan struct{}),
	}
	go b.watchEventSocket()
	return b, nil
}

func (b *hyprlandBackend) Name() string { return "hyprland" }

func (b *hyprlandBackend) Snapshot() Snapshot {
	out, err := exec.Command("hyprctl", "monitors", "-j").Output()
	if err != nil {
		return Snapshot{}
	}
	var monitors []Monitor
	if err := json.Unmarshal(out, &monitors); err != nil {
		return Snapshot{}
	}
	snap := Snapshot{Monitors: monitors, NightLightGamma: 100}
	b.readBrightness(&snap)
	b.readNightLight(&snap)
	if profiles, err := ListProfiles(); err == nil {
		snap.Profiles = profiles
	}
	return snap
}

func (b *hyprlandBackend) Close() {
	close(b.closeCh)
}

func (b *hyprlandBackend) Events() <-chan struct{} { return b.eventCh }

func (b *hyprlandBackend) SetResolution(name string, width, height int, refreshRate float64) error {
	mode := fmt.Sprintf("%dx%d@%.2f", width, height, refreshRate)
	return hyprctl("keyword", "monitor", name+","+mode+",auto,auto")
}

func (b *hyprlandBackend) SetScale(name string, scale float64) error {
	snap := b.Snapshot()
	for _, m := range snap.Monitors {
		if m.Name == name {
			mode := fmt.Sprintf("%dx%d@%.2f", m.Width, m.Height, m.RefreshRate)
			pos := fmt.Sprintf("%dx%d", m.X, m.Y)
			scaleStr := fmt.Sprintf("%.2f", scale)
			return hyprctl("keyword", "monitor", name+","+mode+","+pos+","+scaleStr)
		}
	}
	return fmt.Errorf("monitor %s not found", name)
}

func (b *hyprlandBackend) SetTransform(name string, transform int) error {
	return hyprctl("keyword", "monitor", name+",transform,"+fmt.Sprint(transform))
}

func (b *hyprlandBackend) SetPosition(name string, x, y int) error {
	snap := b.Snapshot()
	for _, m := range snap.Monitors {
		if m.Name == name {
			mode := fmt.Sprintf("%dx%d@%.2f", m.Width, m.Height, m.RefreshRate)
			pos := fmt.Sprintf("%dx%d", x, y)
			scaleStr := fmt.Sprintf("%.2f", m.Scale)
			return hyprctl("keyword", "monitor", name+","+mode+","+pos+","+scaleStr)
		}
	}
	return fmt.Errorf("monitor %s not found", name)
}

func (b *hyprlandBackend) SetDpms(name string, on bool) error {
	flag := "off"
	if on {
		flag = "on"
	}
	return hyprctl("dispatch", "dpms", flag, name)
}

func (b *hyprlandBackend) SetVrr(name string, mode int) error {
	return hyprctl("keyword", "monitor", name+",vrr,"+fmt.Sprint(mode))
}

func (b *hyprlandBackend) SetMirror(name, mirrorOf string) error {
	return hyprctl("keyword", "monitor", name+",preferred,auto,1,mirror,"+mirrorOf)
}

func (b *hyprlandBackend) ToggleEnabled(name string) error {
	snap := b.Snapshot()
	for _, m := range snap.Monitors {
		if m.Name == name {
			if m.Disabled {
				return hyprctl("keyword", "monitor", name+",preferred,auto,auto")
			}
			return hyprctl("keyword", "monitor", name+",disabled")
		}
	}
	return fmt.Errorf("monitor %s not found", name)
}

func (b *hyprlandBackend) Identify() error {
	snap := b.Snapshot()
	if len(snap.Monitors) == 0 {
		return fmt.Errorf("no monitors to identify")
	}

	// hyprctl notify can't target a specific monitor. Instead, use
	// `hyprctl dispatch exec` with inline window rules to spawn a
	// small floating terminal on each monitor showing its label.
	// The inline rules [float;monitor <name>;center] pin each
	// window to the correct output.
	for i, m := range snap.Monitors {
		label := fmt.Sprintf("%d: %s (%dx%d)", i+1, m.Name, m.Width, m.Height)
		script := fmt.Sprintf(`echo; echo "    %s"; sleep 3`, label)
		rules := fmt.Sprintf("[float;monitor %s;size 400 80;center;noanim]", m.Name)
		_ = hyprctl("dispatch", "exec", rules+" alacritty -T dark-identify -e sh -c '"+script+"'")
	}
	return nil
}

func (b *hyprlandBackend) SetBrightness(pct int) error {
	if b.backlightDev == "" {
		return fmt.Errorf("no backlight device found")
	}
	return exec.Command("brightnessctl", "set", fmt.Sprintf("%d%%", pct), "-d", b.backlightDev).Run()
}

func (b *hyprlandBackend) SetKbdBrightness(pct int) error {
	return exec.Command("brightnessctl", "set", fmt.Sprintf("%d%%", pct), "-d", "rgb:kbd_backlight").Run()
}

func (b *hyprlandBackend) SetNightLight(enable bool, tempK int, gamma int) error {
	_ = exec.Command("pkill", "-x", "hyprsunset").Run()
	if !enable {
		return nil
	}
	args := []string{"-t", strconv.Itoa(tempK)}
	if gamma > 0 && gamma != 100 {
		args = append(args, "-g", strconv.Itoa(gamma))
	}
	cmd := exec.Command("hyprsunset", args...)
	cmd.SysProcAttr = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func (b *hyprlandBackend) SetGamma(pct int) error {
	if pct < 0 {
		pct = 0
	}
	if pct > 200 {
		pct = 200
	}
	snap := b.Snapshot()
	temp := snap.NightLightTemp
	if temp == 0 {
		temp = 6500
	}
	_ = exec.Command("pkill", "-x", "hyprsunset").Run()
	args := []string{"-t", strconv.Itoa(temp)}
	if pct != 100 {
		args = append(args, "-g", strconv.Itoa(pct))
	}
	cmd := exec.Command("hyprsunset", args...)
	cmd.SysProcAttr = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func (b *hyprlandBackend) SaveProfile(name string) error {
	snap := b.Snapshot()
	return SaveProfile(name, snap)
}

func (b *hyprlandBackend) ApplyProfile(name string) error {
	profile, err := LoadProfile(name)
	if err != nil {
		return err
	}
	return ApplyProfile(profile)
}

func (b *hyprlandBackend) DeleteProfile(name string) error {
	return DeleteProfile(name)
}

func (b *hyprlandBackend) readBrightness(snap *Snapshot) {
	if b.backlightDev != "" {
		if cur, max, ok := parseBrightnessctl(b.backlightDev); ok {
			snap.HasBacklight = true
			snap.MaxBrightness = max
			if max > 0 {
				snap.Brightness = cur * 100 / max
			}
		}
	}
	if cur, max, ok := parseBrightnessctl("rgb:kbd_backlight"); ok {
		snap.HasKbdLight = true
		snap.KbdMaxBright = max
		if max > 0 {
			snap.KbdBrightness = cur * 100 / max
		}
	}
}

func (b *hyprlandBackend) readNightLight(snap *Snapshot) {
	out, err := exec.Command("pgrep", "-x", "hyprsunset").Output()
	if err != nil {
		return
	}
	snap.NightLightActive = true
	snap.NightLightTemp = 4500

	pid := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	cmdline, err := os.ReadFile("/proc/" + pid + "/cmdline")
	if err != nil {
		return
	}
	parts := bytes.Split(cmdline, []byte{0})
	for i, p := range parts {
		if string(p) == "-t" && i+1 < len(parts) {
			if t, err := strconv.Atoi(string(parts[i+1])); err == nil && t > 0 {
				snap.NightLightTemp = t
			}
		}
		if string(p) == "-g" && i+1 < len(parts) {
			if g, err := strconv.Atoi(string(parts[i+1])); err == nil && g > 0 {
				snap.NightLightGamma = g
			}
		}
	}
}

func (b *hyprlandBackend) watchEventSocket() {
	sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if sig == "" || runtime == "" {
		return
	}
	socketPath := runtime + "/hypr/" + sig + "/.socket2.sock"

	for {
		select {
		case <-b.closeCh:
			return
		default:
		}
		b.listenSocket(socketPath)
		select {
		case <-b.closeCh:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (b *hyprlandBackend) listenSocket(path string) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		select {
		case <-b.closeCh:
			return
		default:
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "monitoradded") ||
			strings.HasPrefix(line, "monitorremoved") ||
			strings.HasPrefix(line, "configreloaded") {
			select {
			case b.eventCh <- struct{}{}:
			default:
			}
		}
	}
}

func detectBacklightDevice() string {
	if out, err := exec.Command("brightnessctl", "-m", "-d", "amdgpu_bl1").Output(); err == nil {
		if len(out) > 0 {
			return "amdgpu_bl1"
		}
	}
	out, err := exec.Command("brightnessctl", "-m", "-c", "backlight").Output()
	if err != nil {
		return ""
	}
	line := strings.SplitN(string(out), "\n", 2)[0]
	fields := strings.Split(line, ",")
	if len(fields) > 0 && fields[0] != "" {
		return fields[0]
	}
	return ""
}

func parseBrightnessctl(device string) (current, max int, ok bool) {
	out, err := exec.Command("brightnessctl", "-m", "-d", device).Output()
	if err != nil {
		return 0, 0, false
	}
	fields := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(fields) < 5 {
		return 0, 0, false
	}
	cur, err1 := strconv.Atoi(fields[2])
	mx, err2 := strconv.Atoi(fields[4])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return cur, mx, true
}

func hyprctl(args ...string) error {
	cmd := exec.Command("hyprctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hyprctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
