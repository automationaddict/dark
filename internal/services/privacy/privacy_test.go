package privacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleHypridle = `general {
    lock_cmd = pidof hyprlock || hyprlock
    before_sleep_cmd = loginctl lock-session
    after_sleep_cmd = hyprctl dispatch dpms on
}

listener {
    timeout = 300
    on-timeout = notify-send "screensaver"
    on-resume = notify-send "welcome back"
}

listener {
    timeout = 600
    on-timeout = loginctl lock-session
}

listener {
    timeout = 900
    on-timeout = hyprctl dispatch dpms off
    on-resume = hyprctl dispatch dpms on
}
`

func TestSplitListenerBlocks(t *testing.T) {
	blocks := splitListenerBlocks(sampleHypridle)
	if len(blocks) != 3 {
		t.Fatalf("got %d blocks, want 3", len(blocks))
	}
	if !strings.Contains(blocks[0], "screensaver") {
		t.Errorf("block 0 missing screensaver marker: %q", blocks[0])
	}
	if !strings.Contains(blocks[1], "lock-session") {
		t.Errorf("block 1 missing lock-session marker: %q", blocks[1])
	}
	if !strings.Contains(blocks[2], "dpms off") {
		t.Errorf("block 2 missing dpms off marker: %q", blocks[2])
	}
}

func TestExtractTimeout(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"timeout = 300", 300},
		{"  timeout=600", 600},
		{"timeout   =   120\n", 120},
		{"no timeout here", 0},
		{"", 0},
	}
	for _, tt := range tests {
		if got := extractTimeout(tt.in); got != tt.want {
			t.Errorf("extractTimeout(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestReadHypridleParsesTimeouts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".config", "hypr")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "hypridle.conf")
	if err := os.WriteFile(path, []byte(sampleHypridle), 0o644); err != nil {
		t.Fatal(err)
	}

	var s Snapshot
	readHypridle(&s)

	if s.ScreensaverTimeout != 300 {
		t.Errorf("ScreensaverTimeout = %d, want 300", s.ScreensaverTimeout)
	}
	if s.LockTimeout != 600 {
		t.Errorf("LockTimeout = %d, want 600", s.LockTimeout)
	}
	if s.ScreenOffTimeout != 900 {
		t.Errorf("ScreenOffTimeout = %d, want 900", s.ScreenOffTimeout)
	}
	if !s.LockOnSleep {
		t.Error("LockOnSleep = false, want true (before_sleep_cmd present)")
	}
}

func TestReadHypridleMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var s Snapshot
	readHypridle(&s) // must not panic, must leave zero values
	if s.ScreensaverTimeout != 0 || s.LockTimeout != 0 || s.ScreenOffTimeout != 0 {
		t.Errorf("missing file should leave zero timeouts, got %+v", s)
	}
}

func TestSetIdleTimeoutReplacesCorrectBlock(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".config", "hypr")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "hypridle.conf")
	if err := os.WriteFile(path, []byte(sampleHypridle), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetIdleTimeout("lock", 1800); err != nil {
		t.Fatalf("SetIdleTimeout: %v", err)
	}

	// Re-read and verify only LockTimeout changed.
	var s Snapshot
	readHypridle(&s)
	if s.LockTimeout != 1800 {
		t.Errorf("LockTimeout = %d, want 1800", s.LockTimeout)
	}
	if s.ScreensaverTimeout != 300 {
		t.Errorf("ScreensaverTimeout = %d, want 300 (unchanged)", s.ScreensaverTimeout)
	}
	if s.ScreenOffTimeout != 900 {
		t.Errorf("ScreenOffTimeout = %d, want 900 (unchanged)", s.ScreenOffTimeout)
	}
}

func TestSetIdleTimeoutUnknownField(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".config", "hypr")
	_ = os.MkdirAll(cfgDir, 0o755)
	_ = os.WriteFile(filepath.Join(cfgDir, "hypridle.conf"), []byte(sampleHypridle), 0o644)

	if err := SetIdleTimeout("bogus", 100); err == nil {
		t.Error("expected error for unknown field, got nil")
	}
}

func TestBlockContains(t *testing.T) {
	lines := strings.Split(`listener {
    timeout = 300
    on-timeout = loginctl lock-session
}
listener {
    timeout = 600
    on-timeout = hyprctl dispatch dpms off
}`, "\n")
	if !blockContains(lines, "lock-session") {
		t.Error("expected block to contain lock-session")
	}
	if blockContains(lines, "dpms off") {
		t.Error("first block should not match dpms off from second block")
	}
}

func TestReadResolvedConfMissingFile(t *testing.T) {
	// Function hardcodes /etc/systemd/resolved.conf; just verify it
	// handles the not-installed case gracefully.
	m := readResolvedConf()
	if m == nil {
		t.Error("readResolvedConf should return non-nil map even on read failure")
	}
}
