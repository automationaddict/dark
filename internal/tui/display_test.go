package tui

import (
	"testing"

	"github.com/johnnelson/dark/internal/services/display"
)

func TestParseMode(t *testing.T) {
	tests := []struct {
		input   string
		w, h    int
		rate    float64
		wantErr bool
	}{
		{"1920x1080@60.00Hz", 1920, 1080, 60.0, false},
		{"3840x2160@143.98Hz", 3840, 2160, 143.98, false},
		{"2560x1440@120Hz", 2560, 1440, 120.0, false},
		{"1920x1080@60.00", 1920, 1080, 60.0, false},
		{"bad", 0, 0, 0, true},
		{"1920x1080", 0, 0, 0, true},
		{"x1080@60Hz", 0, 0, 0, true},
		{"1920x@60Hz", 0, 0, 0, true},
		{"1920x1080@badHz", 0, 0, 0, true},
		{"", 0, 0, 0, true},
	}
	for _, tt := range tests {
		w, h, rate, err := parseMode(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseMode(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseMode(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if w != tt.w || h != tt.h || rate != tt.rate {
			t.Errorf("parseMode(%q) = (%d, %d, %.2f), want (%d, %d, %.2f)",
				tt.input, w, h, rate, tt.w, tt.h, tt.rate)
		}
	}
}

func TestSnapPosition(t *testing.T) {
	monitors := []display.Monitor{
		{Name: "DP-1", X: 0, Y: 0, Width: 1920, Height: 1080},
		{Name: "DP-2", X: 1920, Y: 0, Width: 2560, Height: 1440},
	}

	t.Run("snap to adjacent edge when close", func(t *testing.T) {
		mons := []display.Monitor{
			{Name: "A", X: 0, Y: 0, Width: 1920, Height: 1080},
			{Name: "B", X: 1950, Y: 0, Width: 2560, Height: 1440},
		}
		x, _ := snapPosition(mons, 1, -1, 0)
		if x != 1920 {
			t.Errorf("expected snap to 1920, got %d", x)
		}
	})

	t.Run("no movement on zero delta", func(t *testing.T) {
		x, y := snapPosition(monitors, 0, 0, 0)
		if x != 0 || y != 0 {
			t.Errorf("expected (0,0), got (%d,%d)", x, y)
		}
	})

	t.Run("nudge down", func(t *testing.T) {
		x, y := snapPosition(monitors, 0, 0, 1)
		if x != 0 {
			t.Errorf("x = %d, want 0", x)
		}
		if y <= 0 {
			t.Error("expected y > 0 when nudging down")
		}
	})

	t.Run("horizontal alignment snap when moving vertically", func(t *testing.T) {
		mons := []display.Monitor{
			{Name: "A", X: 0, Y: 0, Width: 1920, Height: 1080},
			{Name: "B", X: 10, Y: 1080, Width: 2560, Height: 1440},
		}
		x, _ := snapPosition(mons, 1, 0, -1)
		if x != 0 {
			t.Errorf("expected horizontal snap to x=0, got %d", x)
		}
	})
}
