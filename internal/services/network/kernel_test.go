package network

import (
	"net"
	"testing"
)

func TestParseHexLE32(t *testing.T) {
	tests := []struct {
		input string
		want  uint32
	}{
		{"00000000", 0},
		{"0100A8C0", 0xC0A80001}, // 192.168.0.1 in LE hex
		{"FFFFFFFF", 0xFFFFFFFF},
	}
	for _, tt := range tests {
		if got := parseHexLE32(tt.input); got != tt.want {
			t.Errorf("parseHexLE32(%q) = 0x%x, want 0x%x", tt.input, got, tt.want)
		}
	}
}

func TestUint32ToIP(t *testing.T) {
	ip := uint32ToIP(0xC0A80001) // 192.168.0.1 in network order
	if ip.String() != "192.168.0.1" {
		t.Errorf("uint32ToIP(0xC0A80001) = %s, want 192.168.0.1", ip)
	}
}

func TestBitsInMask(t *testing.T) {
	tests := []struct {
		mask uint32
		want int
	}{
		{0, 0},
		{0xFFFFFFFF, 32},
		{0xFFFFFF00, 24},
		{0xFFFF0000, 16},
		{0xFF000000, 8},
	}
	for _, tt := range tests {
		if got := bitsInMask(tt.mask); got != tt.want {
			t.Errorf("bitsInMask(0x%x) = %d, want %d", tt.mask, got, tt.want)
		}
	}
}

func TestParseHexIPv6(t *testing.T) {
	// fe80::1 = fe800000 00000000 00000000 00000001
	input := "fe800000000000000000000000000001"
	ip, ok := parseHexIPv6(input)
	if !ok {
		t.Fatal("parseHexIPv6 failed")
	}
	if ip.String() != "fe80::1" {
		t.Errorf("got %s, want fe80::1", ip)
	}

	// invalid length
	_, ok = parseHexIPv6("short")
	if ok {
		t.Error("expected failure for short input")
	}
}

func TestAddressScope(t *testing.T) {
	tests := []struct {
		ip   string
		want string
	}{
		{"127.0.0.1", "host"},
		{"192.168.1.1", "global"},
		{"10.0.0.1", "global"},
		{"fe80::1", "link"},
		{"2001:db8::1", "global"},
	}
	for _, tt := range tests {
		if got := addressScope(net.ParseIP(tt.ip)); got != tt.want {
			t.Errorf("addressScope(%s) = %q, want %q", tt.ip, got, tt.want)
		}
	}
}

func TestDetectInterfaceType(t *testing.T) {
	if got := detectInterfaceType("lo", "/sys/class/net/lo"); got != "loopback" {
		t.Errorf("lo: got %q, want loopback", got)
	}
}
