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
)

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
	if pid, ok := hypridlePID(); ok {
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Signal(syscall.SIGUSR1)
		}
	}
}

// hypridlePID returns the PID of the running hypridle process and a
// bool indicating whether one was found. Walks /proc/*/comm so it
// doesn't need pgrep on the daemon's PATH.
func hypridlePID() (int, bool) {
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
			return pid, true
		}
	}
	return 0, false
}

// SetIdleRunning toggles the hypridle daemon on or off. When running
// is false and hypridle is running, it sends SIGTERM and lets the
// process exit cleanly. When running is true and hypridle is not
// running, it launches it via omarchy-toggle-idle (which handles the
// uwsm-app environment setup the Wayland session needs). No-op when
// the process is already in the desired state — callers get a clear
// error only when the flip actually failed.
func SetIdleRunning(running bool) error {
	pid, already := hypridlePID()
	if already == running {
		return nil
	}

	if !running {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("find hypridle pid %d: %w", pid, err)
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("stop hypridle: %w", err)
		}
		return nil
	}

	// Starting hypridle needs the Wayland session environment
	// (WAYLAND_DISPLAY, XDG_RUNTIME_DIR, DBUS_SESSION_BUS_ADDRESS,
	// etc.). The omarchy-toggle-idle script uses uwsm-app for that,
	// which is the correct path on a stock Omarchy install. Fall
	// back to a direct hypridle launch if omarchy-toggle-idle is
	// missing — that keeps dark working on systems that have
	// hypridle installed without the Omarchy wrapper scripts.
	if _, err := exec.LookPath("omarchy-toggle-idle"); err == nil {
		// omarchy-toggle-idle flips state, so calling it when
		// hypridle is off will turn it on. The above state check
		// guarantees we only reach here when that's what we want.
		cmd := exec.Command("omarchy-toggle-idle")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("omarchy-toggle-idle: %w", err)
		}
		return nil
	}
	if _, err := exec.LookPath("hypridle"); err != nil {
		return fmt.Errorf("hypridle not installed")
	}
	// Detached launch — we don't want darkd to be the parent because
	// the hypridle process should outlive any daemon restart.
	cmd := exec.Command("hypridle")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start hypridle: %w", err)
	}
	// Don't Wait — we want this child fully orphaned. Release the
	// Go-side handle so the runtime doesn't keep a zombie waiting.
	_ = cmd.Process.Release()
	return nil
}

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
