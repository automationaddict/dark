package tui

import "testing"

func TestOrDash(t *testing.T) {
	if got := orDash(""); got != "—" {
		t.Errorf("orDash empty = %q", got)
	}
	if got := orDash("  "); got != "—" {
		t.Errorf("orDash whitespace = %q", got)
	}
	if got := orDash("hello"); got != "hello" {
		t.Errorf("orDash value = %q", got)
	}
}

func TestOnOff(t *testing.T) {
	if got := onOff(true); got != "On" {
		t.Errorf("onOff(true) = %q", got)
	}
	if got := onOff(false); got != "Off" {
		t.Errorf("onOff(false) = %q", got)
	}
}

func TestYesNo(t *testing.T) {
	if got := yesNo(true); got != "Yes" {
		t.Errorf("yesNo(true) = %q", got)
	}
	if got := yesNo(false); got != "No" {
		t.Errorf("yesNo(false) = %q", got)
	}
}

func TestFormatFreq(t *testing.T) {
	tests := []struct {
		mhz  uint32
		want string
	}{
		{0, "—"},
		{2412, "2.41 GHz"},
		{5180, "5.18 GHz"},
	}
	for _, tt := range tests {
		if got := formatFreq(tt.mhz); got != tt.want {
			t.Errorf("formatFreq(%d) = %q, want %q", tt.mhz, got, tt.want)
		}
	}
}

func TestSignalBars(t *testing.T) {
	if got := signalBars(0); got != "—" {
		t.Errorf("signalBars(0) = %q, want —", got)
	}
	if got := signalBars(-40); len(got) == 0 {
		t.Error("signalBars(-40) should produce bars")
	}
	if got := signalBars(-90); len(got) == 0 {
		t.Error("signalBars(-90) should produce bars")
	}
}

func TestFormatNetworkBytes(t *testing.T) {
	tests := []struct {
		b    uint64
		want string
	}{
		{0, "—"},
		{500, "500 B"},
		{1024, "1.0 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}
	for _, tt := range tests {
		if got := formatNetworkBytes(tt.b); got != tt.want {
			t.Errorf("formatNetworkBytes(%d) = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"  ", 0},
		{"a, b, c", 3},
		{"one,,two", 2},
		{",,,", 0},
	}
	for _, tt := range tests {
		if got := splitCSV(tt.input); len(got) != tt.want {
			t.Errorf("splitCSV(%q) len = %d, want %d: %v", tt.input, len(got), tt.want, got)
		}
	}
}

