// Package topbar wraps the Wayland top bar on Omarchy systems
// (waybar). The omarchy shell helpers (omarchy-toggle-waybar,
// omarchy-restart-waybar, omarchy-refresh-waybar) are deliberately
// not used — this package reimplements the same semantics in Go so
// dark doesn't depend on those wrappers being on PATH and can
// integrate with the rest of the service layer cleanly.
//
// Everything here is a user-file / user-process operation. No pkexec.
package topbar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Relative paths under $HOME. Absolute paths are resolved at call time
// so tests can swap HOME under t.TempDir.
const (
	configRel      = ".config/waybar/config.jsonc"
	styleRel       = ".config/waybar/style.css"
	defaultSubPath = ".local/share/omarchy/config/waybar"
)

// Snapshot is the state dark shows on the Appearance → Top Bar
// sub-section. It's a mix of process state (whether waybar is
// running), parsed scalar knobs (position, layer, height, spacing),
// and the module arrays the user has configured. The raw config
// file content is carried too so the full-screen editor can open
// it without a round trip.
type Snapshot struct {
	Running bool `json:"running"`

	// Scalar settings pulled from the top-level of config.jsonc.
	// Missing fields are left as zero values; the view shows "—".
	Position string `json:"position,omitempty"`
	Layer    string `json:"layer,omitempty"`
	Height   int    `json:"height,omitempty"`
	Spacing  int    `json:"spacing,omitempty"`

	// Module arrays in display order. These are always strings even
	// when the underlying config uses objects for custom modules —
	// waybar's modules-left/center/right arrays are documented as
	// string arrays, and all Omarchy config files follow that shape.
	ModulesLeft   []string `json:"modules_left,omitempty"`
	ModulesCenter []string `json:"modules_center,omitempty"`
	ModulesRight  []string `json:"modules_right,omitempty"`

	// Absolute paths so the UI can show them and callers can open
	// the files in the editor overlay without recomputing.
	ConfigPath string `json:"config_path"`
	StylePath  string `json:"style_path"`

	// Full file contents. The editor overlay reads from Content /
	// Style rather than re-reading disk so the edit-save cycle
	// round-trips cleanly through the snapshot.
	Config string `json:"config,omitempty"`
	Style  string `json:"style,omitempty"`

	// DefaultsAvailable is true when the Omarchy defaults directory
	// exists — it's the source for the Reset action. Without it,
	// Reset is a no-op and the UI greys the action out.
	DefaultsAvailable bool `json:"defaults_available"`
}

// Maximum bytes dark will load into the snapshot for either file.
// Both waybar config and style files are well under this; the cap
// keeps snapshot payloads bounded when a misconfigured file is huge.
const maxFileBytes = 256 * 1024

// ReadSnapshot walks the process table and reads the two config
// files, returning a best-effort view. Missing files produce zero
// values rather than errors.
func ReadSnapshot() Snapshot {
	home, _ := os.UserHomeDir()

	configPath := filepath.Join(home, configRel)
	stylePath := filepath.Join(home, styleRel)

	snap := Snapshot{
		Running:           waybarRunning(),
		ConfigPath:        configPath,
		StylePath:         stylePath,
		DefaultsAvailable: defaultsDir(home) != "",
	}

	if data, err := readFile(configPath); err == nil {
		snap.Config = data
		if parsed, ok := parseWaybarConfig(data); ok {
			snap.Position = parsed.Position
			snap.Layer = parsed.Layer
			snap.Height = parsed.Height
			snap.Spacing = parsed.Spacing
			snap.ModulesLeft = parsed.ModulesLeft
			snap.ModulesCenter = parsed.ModulesCenter
			snap.ModulesRight = parsed.ModulesRight
		}
	}

	if data, err := readFile(stylePath); err == nil {
		snap.Style = data
	}

	return snap
}

// SetRunning toggles the waybar daemon on or off. The omarchy
// wrapper shell scripts are deliberately not used — we do the same
// pkill/spawn dance in Go so the logic lives next to the rest of
// the backend and the code is directly testable.
func SetRunning(running bool) error {
	pid, already := waybarPID()
	if already == running {
		return nil
	}

	if !running {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("find waybar pid %d: %w", pid, err)
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("stop waybar: %w", err)
		}
		return nil
	}

	return spawnSessionDaemon("waybar")
}

// Restart stops waybar and starts it again. We stop, give the
// process a moment to exit, then start — rather than SIGUSR2 (live
// reload) — because restart is the right action when the user has
// changed something dark-side that requires a clean reparse of the
// full config.
func Restart() error {
	if _, running := waybarPID(); running {
		if err := SetRunning(false); err != nil {
			return err
		}
		// Give waybar a beat to unmap its surfaces and exit. 200ms
		// is well above the observed shutdown time on test hardware.
		time.Sleep(200 * time.Millisecond)
	}
	return SetRunning(true)
}

// Reset copies the Omarchy defaults over the user's config and
// style files, with a timestamped backup of each existing file so
// nothing is lost. After writing, it triggers a restart so the
// change takes effect immediately. Returns an error only if the
// defaults directory is missing or a filesystem op fails — the
// restart error is logged by the caller but not fatal.
func Reset() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	defaults := defaultsDir(home)
	if defaults == "" {
		return fmt.Errorf("omarchy defaults directory not found")
	}

	pairs := []struct {
		src  string
		dest string
	}{
		{filepath.Join(defaults, "config.jsonc"), filepath.Join(home, configRel)},
		{filepath.Join(defaults, "style.css"), filepath.Join(home, styleRel)},
	}

	for _, p := range pairs {
		if _, err := os.Stat(p.src); err != nil {
			return fmt.Errorf("default %s missing: %w", p.src, err)
		}
		if err := backupAndCopy(p.src, p.dest); err != nil {
			return err
		}
	}

	return Restart()
}

// SetConfig overwrites config.jsonc atomically and restarts waybar
// so the change takes effect. Called from the editor overlay's
// submit path.
func SetConfig(content string) error {
	if len(content) > maxFileBytes {
		return fmt.Errorf("config too large (%d bytes, max %d)", len(content), maxFileBytes)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	if err := writeAtomic(filepath.Join(home, configRel), []byte(content)); err != nil {
		return err
	}
	return Restart()
}

// SetStyle overwrites style.css atomically. Waybar watches the
// style file when reload_style_on_change is true, so we don't
// restart after this — the live reload picks up the change. If the
// user has disabled reload_style_on_change, they can hit `r` on
// the Top Bar sub-section to force a full restart.
func SetStyle(content string) error {
	if len(content) > maxFileBytes {
		return fmt.Errorf("style too large (%d bytes, max %d)", len(content), maxFileBytes)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	return writeAtomic(filepath.Join(home, styleRel), []byte(content))
}

// ─── process control helpers ───────────────────────────────────────

func waybarRunning() bool {
	_, ok := waybarPID()
	return ok
}

// waybarPID walks /proc/*/comm for a matching entry and returns the
// first PID found. The backend only needs to signal one process so
// we don't care about multiples — the omarchy convention is one
// waybar per session anyway.
func waybarPID() (int, bool) {
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) == "waybar" {
			pid := 0
			for _, ch := range e.Name() {
				if ch < '0' || ch > '9' {
					pid = 0
					break
				}
				pid = pid*10 + int(ch-'0')
			}
			if pid > 0 {
				return pid, true
			}
		}
	}
	return 0, false
}

// spawnSessionDaemon starts bin with the right Wayland session
// environment. Prefers uwsm-app (Omarchy's session-aware launcher)
// when available, falls back to the binary directly. Detaches via
// setsid and Releases the Process handle so the child survives a
// darkd restart — mirrors the helper in power_idle.go.
func spawnSessionDaemon(bin string) error {
	var argv []string
	if _, err := exec.LookPath("uwsm-app"); err == nil {
		argv = []string{"uwsm-app", "--", bin}
	} else {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s not installed", bin)
		}
		argv = []string{bin}
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", bin, err)
	}
	_ = cmd.Process.Release()
	return nil
}

// ─── file helpers ────────────────────────────────────────────────

func readFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, maxFileBytes+1)
	n, _ := f.Read(buf)
	if n > maxFileBytes {
		n = maxFileBytes
	}
	return string(buf[:n]), nil
}

// writeAtomic writes data to path via a tmp file + rename so a
// crash mid-write can't leave the config truncated and break waybar.
func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	tmp := path + ".dark-tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}

// backupAndCopy is the Reset primitive: if dest exists, copy it to
// dest.bak.<unix>, then overwrite dest with src's contents via an
// atomic write. Mirrors the semantics of omarchy-refresh-config but
// stays in Go so dark doesn't depend on that wrapper script.
func backupAndCopy(src, dest string) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if _, err := os.Stat(dest); err == nil {
		destData, readErr := os.ReadFile(dest)
		if readErr == nil {
			backup := fmt.Sprintf("%s.bak.%d", dest, time.Now().Unix())
			if err := os.WriteFile(backup, destData, 0o644); err != nil {
				return fmt.Errorf("backup %s: %w", dest, err)
			}
		}
	}
	return writeAtomic(dest, srcData)
}

// defaultsDir returns the Omarchy waybar defaults directory if it
// exists and is readable, or "" otherwise. Used by ReadSnapshot to
// populate Snapshot.DefaultsAvailable and by Reset as the source.
func defaultsDir(home string) string {
	dir := filepath.Join(home, defaultSubPath)
	if _, err := os.Stat(dir); err != nil {
		return ""
	}
	return dir
}
