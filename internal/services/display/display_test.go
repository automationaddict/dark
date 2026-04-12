package display

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMonitorResolution(t *testing.T) {
	tests := []struct {
		w, h int
		want string
	}{
		{1920, 1080, "1920x1080"},
		{3840, 2160, "3840x2160"},
		{0, 0, "0x0"},
	}
	for _, tt := range tests {
		m := Monitor{Width: tt.w, Height: tt.h}
		if got := m.Resolution(); got != tt.want {
			t.Errorf("Resolution(%d,%d) = %q, want %q", tt.w, tt.h, got, tt.want)
		}
	}
}

func TestMonitorRefreshRateHz(t *testing.T) {
	tests := []struct {
		rate float64
		want string
	}{
		{60.0, "60.00Hz"},
		{143.98, "143.98Hz"},
		{0, "0.00Hz"},
	}
	for _, tt := range tests {
		m := Monitor{RefreshRate: tt.rate}
		if got := m.RefreshRateHz(); got != tt.want {
			t.Errorf("RefreshRateHz(%v) = %q, want %q", tt.rate, got, tt.want)
		}
	}
}

func TestMonitorTransformLabel(t *testing.T) {
	tests := []struct {
		transform int
		want      string
	}{
		{0, "Normal"},
		{1, "90°"},
		{2, "180°"},
		{3, "270°"},
		{4, "Flipped"},
		{5, "Flipped 90°"},
		{6, "Flipped 180°"},
		{7, "Flipped 270°"},
		{99, "Unknown (99)"},
	}
	for _, tt := range tests {
		m := Monitor{Transform: tt.transform}
		if got := m.TransformLabel(); got != tt.want {
			t.Errorf("TransformLabel(%d) = %q, want %q", tt.transform, got, tt.want)
		}
	}
}

func TestMonitorDPI(t *testing.T) {
	tests := []struct {
		width, physW int
		want         int
	}{
		{1920, 530, 92},
		{3840, 600, 163},
		{0, 0, 0},
		{1920, 0, 0},
	}
	for _, tt := range tests {
		m := Monitor{Width: tt.width, PhysicalWidth: tt.physW}
		got := m.DPI()
		if got != tt.want {
			t.Errorf("DPI(w=%d, physW=%d) = %d, want %d", tt.width, tt.physW, got, tt.want)
		}
	}
}

func TestMonitorDiagonalInches(t *testing.T) {
	m := Monitor{PhysicalWidth: 530, PhysicalHeight: 300}
	diag := m.DiagonalInches()
	if diag < 23.9 || diag > 24.1 {
		t.Errorf("DiagonalInches() = %.2f, expected ~24.0", diag)
	}

	zero := Monitor{}
	if zero.DiagonalInches() != 0 {
		t.Error("DiagonalInches() should be 0 for zero physical dims")
	}
}

func TestFormatMonitorLine(t *testing.T) {
	mon := Monitor{
		Name: "DP-1", Width: 1920, Height: 1080, RefreshRate: 60.0,
		X: 0, Y: 0, Scale: 1.0,
	}
	got := formatMonitorLine(mon)
	want := "monitor = DP-1, 1920x1080@60.00, 0x0, 1.00"
	if got != want {
		t.Errorf("formatMonitorLine = %q, want %q", got, want)
	}

	mon.Transform = 1
	mon.Vrr = true
	got = formatMonitorLine(mon)
	want = "monitor = DP-1, 1920x1080@60.00, 0x0, 1.00, transform, 1, vrr, 1"
	if got != want {
		t.Errorf("formatMonitorLine (transform+vrr) = %q, want %q", got, want)
	}
}

func TestMatchesMonitorLine(t *testing.T) {
	tests := []struct {
		line   string
		prefix string
		want   bool
	}{
		{"monitor = DP-1, 1920x1080@60.00, 0x0, 1.00", "monitor=DP-1,", true},
		{"monitor=DP-1,preferred,auto,1", "monitor=DP-1,", true},
		{"monitor = HDMI-A-1, 3840x2160, 0x0, 2.00", "monitor=DP-1,", false},
		{"# monitor = DP-1, something", "monitor=DP-1,", false},
		{"", "monitor=DP-1,", false},
	}
	for _, tt := range tests {
		got := matchesMonitorLine(tt.line, tt.prefix)
		if got != tt.want {
			t.Errorf("matchesMonitorLine(%q, %q) = %v, want %v", tt.line, tt.prefix, got, tt.want)
		}
	}
}

func TestProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	_ = os.MkdirAll(filepath.Join(dir, ".config", "dark", "display-profiles"), 0755)

	snap := Snapshot{
		Monitors: []Monitor{
			{Name: "DP-1", Width: 1920, Height: 1080, RefreshRate: 60, Scale: 1.0, Vrr: true},
			{Name: "HDMI-A-1", Width: 3840, Height: 2160, RefreshRate: 60, Scale: 2.0, Disabled: true},
		},
	}

	if err := SaveProfile("test-profile", snap); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	profiles, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(profiles) != 1 || profiles[0] != "test-profile" {
		t.Errorf("ListProfiles = %v, want [test-profile]", profiles)
	}

	loaded, err := LoadProfile("test-profile")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if loaded.Name != "test-profile" {
		t.Errorf("loaded.Name = %q, want test-profile", loaded.Name)
	}
	if len(loaded.Monitors) != 2 {
		t.Fatalf("loaded.Monitors len = %d, want 2", len(loaded.Monitors))
	}
	if loaded.Monitors[0].Vrr != true {
		t.Error("expected Vrr=true for first monitor")
	}
	if loaded.Monitors[1].Enabled != false {
		t.Error("expected Enabled=false for disabled monitor")
	}

	if err := DeleteProfile("test-profile"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	profiles, _ = ListProfiles()
	if len(profiles) != 0 {
		t.Errorf("after delete, ListProfiles = %v, want empty", profiles)
	}
}

func TestClosestScaleIndex(t *testing.T) {
	opts := []string{"0.50", "0.75", "1.00", "1.25", "1.50", "1.75", "2.00"}
	tests := []struct {
		current float64
		want    int
	}{
		{1.0, 2},
		{1.1, 2},
		{1.3, 3},
		{0.5, 0},
		{2.0, 6},
		{1.62, 4},
	}
	for _, tt := range tests {
		got := ClosestScaleIndex(opts, tt.current)
		if got != tt.want {
			t.Errorf("ClosestScaleIndex(%.2f) = %d, want %d", tt.current, got, tt.want)
		}
	}
}
