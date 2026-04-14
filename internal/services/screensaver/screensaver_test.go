package screensaver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupHome points HOME at a temp directory and returns that path.
// Every test isolates its filesystem mutations under its own HOME so
// nothing touches the real user config.
func setupHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestReadSnapshotMissingFiles(t *testing.T) {
	setupHome(t)
	s := ReadSnapshot()
	if !s.Enabled {
		t.Error("expected Enabled=true when flag file is absent")
	}
	if s.Content != "" {
		t.Errorf("expected empty Content, got %q", s.Content)
	}
	if !strings.HasSuffix(s.ContentPath, contentRelPath) {
		t.Errorf("ContentPath = %q, want suffix %q", s.ContentPath, contentRelPath)
	}
	// TTEInstalled / TerminalName depend on the host environment, so
	// we don't assert on them here — only the file-backed bits.
}

func TestSetEnabledTogglesFlagFile(t *testing.T) {
	home := setupHome(t)
	flagPath := filepath.Join(home, stateFlagPath)

	// Start enabled (no flag file).
	if _, err := os.Stat(flagPath); !os.IsNotExist(err) {
		t.Fatalf("flag file should not exist at start, got err=%v", err)
	}

	// Disable → flag file should exist.
	if err := SetEnabled(false); err != nil {
		t.Fatalf("SetEnabled(false): %v", err)
	}
	if _, err := os.Stat(flagPath); err != nil {
		t.Fatalf("flag file should exist after SetEnabled(false): %v", err)
	}
	if s := ReadSnapshot(); s.Enabled {
		t.Error("ReadSnapshot should show Enabled=false after disable")
	}

	// Re-enable → flag file should be gone.
	if err := SetEnabled(true); err != nil {
		t.Fatalf("SetEnabled(true): %v", err)
	}
	if _, err := os.Stat(flagPath); !os.IsNotExist(err) {
		t.Fatalf("flag file should not exist after SetEnabled(true), got err=%v", err)
	}
	if s := ReadSnapshot(); !s.Enabled {
		t.Error("ReadSnapshot should show Enabled=true after enable")
	}
}

func TestSetEnabledIsIdempotent(t *testing.T) {
	setupHome(t)
	// Enable twice — the second call must not error even though the
	// flag file doesn't exist to remove.
	if err := SetEnabled(true); err != nil {
		t.Fatalf("first SetEnabled(true): %v", err)
	}
	if err := SetEnabled(true); err != nil {
		t.Fatalf("second SetEnabled(true): %v", err)
	}
	// Disable twice — the second call must overwrite the existing
	// flag file without erroring.
	if err := SetEnabled(false); err != nil {
		t.Fatalf("first SetEnabled(false): %v", err)
	}
	if err := SetEnabled(false); err != nil {
		t.Fatalf("second SetEnabled(false): %v", err)
	}
}

func TestWriteContentReadBack(t *testing.T) {
	home := setupHome(t)
	want := "  ┌──────────┐\n  │  HELLO   │\n  └──────────┘\n"
	if err := WriteContent(want); err != nil {
		t.Fatalf("WriteContent: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(home, contentRelPath))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Errorf("content mismatch:\nwant: %q\ngot:  %q", want, string(got))
	}

	// ReadSnapshot should return the same content.
	if snap := ReadSnapshot(); snap.Content != want {
		t.Errorf("snapshot content mismatch:\nwant: %q\ngot:  %q", want, snap.Content)
	}
}

func TestWriteContentRejectsOversized(t *testing.T) {
	setupHome(t)
	huge := strings.Repeat("x", maxContentBytes+1)
	err := WriteContent(huge)
	if err == nil {
		t.Fatal("expected error for oversized content, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' in error, got %v", err)
	}
}

func TestWriteContentIsAtomic(t *testing.T) {
	home := setupHome(t)
	// Write once, overwrite, ensure the previous temp file is gone.
	if err := WriteContent("first"); err != nil {
		t.Fatal(err)
	}
	if err := WriteContent("second"); err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(home, contentRelPath+".dark-tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("temp file should not persist after successful write, got err=%v", err)
	}
}

func TestIsSupportedTerminal(t *testing.T) {
	cases := map[string]bool{
		"alacritty": true,
		"ghostty":   true,
		"kitty":     true,
		"foot":      false,
		"wezterm":   false,
		"":          false,
		"xterm":     false,
	}
	for in, want := range cases {
		if got := isSupportedTerminal(in); got != want {
			t.Errorf("isSupportedTerminal(%q) = %v, want %v", in, got, want)
		}
	}
}
