package bluetooth

import "testing"

func TestLookupUUIDName(t *testing.T) {
	tests := []struct {
		uuid string
		want string
	}{
		{"0000110b-0000-1000-8000-00805f9b34fb", "Audio Sink (A2DP)"},
		{"0000110B-0000-1000-8000-00805F9B34FB", "Audio Sink (A2DP)"}, // case insensitive
		{"00001124-0000-1000-8000-00805f9b34fb", "Human Interface Device"},
		{"0000180f-0000-1000-8000-00805f9b34fb", "Battery Service"},
		{"deadbeef-dead-beef-dead-beefdeadbeef", ""},                    // unknown
		{"", ""},
	}
	for _, tt := range tests {
		if got := LookupUUIDName(tt.uuid); got != tt.want {
			t.Errorf("LookupUUIDName(%q) = %q, want %q", tt.uuid, got, tt.want)
		}
	}
}

func TestMajorClassFromClass(t *testing.T) {
	tests := []struct {
		class uint32
		want  string
	}{
		{0, ""},
		{0x240404, "Audio / Video"}, // typical headset
		{0x100, "Computer"},         // major class 1
		{0x200, "Phone"},            // major class 2
		{0x500, "Peripheral"},       // major class 5
		{0x700, "Wearable"},         // major class 7
		{0x1f00, "Uncategorized"},
		{0x0a00, "Unknown"}, // undefined major class
	}
	for _, tt := range tests {
		if got := MajorClassFromClass(tt.class); got != tt.want {
			t.Errorf("MajorClassFromClass(0x%x) = %q, want %q", tt.class, got, tt.want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		device Device
		want   string
	}{
		{Device{Alias: "My Headset", Name: "WH-1000XM5", Address: "AA:BB"}, "My Headset"},
		{Device{Name: "WH-1000XM5", Address: "AA:BB"}, "WH-1000XM5"},
		{Device{Address: "AA:BB"}, "AA:BB"},
		{Device{}, ""},
	}
	for _, tt := range tests {
		if got := tt.device.DisplayName(); got != tt.want {
			t.Errorf("DisplayName() = %q, want %q", got, tt.want)
		}
	}
}
