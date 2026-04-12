package network

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestBuildAndParse_StaticIPv4Only(t *testing.T) {
	cfg := IPv4Config{
		Mode:    "static",
		Address: "192.168.1.10/24",
		Gateway: "192.168.1.1",
		DNS:     []string{"1.1.1.1", "8.8.8.8"},
		Search:  []string{"lan", "corp"},
		MTU:     1500,
	}
	content, err := buildNetworkdFileContent("eth0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	parsed := mustParseTemp(t, content)
	assertEqual(t, "Mode", parsed.Mode, "static")
	assertEqual(t, "Address", parsed.Address, "192.168.1.10/24")
	assertEqual(t, "Gateway", parsed.Gateway, "192.168.1.1")
	assertEqual(t, "MTU", itoa(parsed.MTU), "1500")
	assertSlice(t, "DNS", parsed.DNS, []string{"1.1.1.1", "8.8.8.8"})
	assertSlice(t, "Search", parsed.Search, []string{"lan", "corp"})
}

func TestBuildAndParse_DHCPOnly(t *testing.T) {
	cfg := IPv4Config{Mode: "dhcp"}
	content, err := buildNetworkdFileContent("wlan0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	parsed := mustParseTemp(t, content)
	assertEqual(t, "Mode", parsed.Mode, "dhcp")
}

func TestBuildAndParse_DualStackStatic(t *testing.T) {
	cfg := IPv4Config{
		Mode:        "static",
		Address:     "10.0.0.5/24",
		Gateway:     "10.0.0.1",
		IPv6Mode:    "static",
		IPv6Address: "2001:db8::5/64",
		IPv6Gateway: "fe80::1",
		DNS:         []string{"1.1.1.1", "2606:4700:4700::1111"},
	}
	content, err := buildNetworkdFileContent("eth0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "IPv6AcceptRA=no") {
		t.Error("static IPv6 should disable RA")
	}
	parsed := mustParseTemp(t, content)
	assertEqual(t, "IPv6Mode", parsed.IPv6Mode, "static")
	assertEqual(t, "IPv6Address", parsed.IPv6Address, "2001:db8::5/64")
	assertEqual(t, "IPv6Gateway", parsed.IPv6Gateway, "fe80::1")
}

func TestBuildAndParse_BothDHCP(t *testing.T) {
	cfg := IPv4Config{Mode: "dhcp", IPv6Mode: "dhcp"}
	content, err := buildNetworkdFileContent("eth0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "DHCP=yes") {
		t.Error("both DHCP should produce DHCP=yes")
	}
	parsed := mustParseTemp(t, content)
	assertEqual(t, "Mode", parsed.Mode, "dhcp")
	assertEqual(t, "IPv6Mode", parsed.IPv6Mode, "dhcp")
}

func TestBuildAndParse_IPv6RA(t *testing.T) {
	cfg := IPv4Config{Mode: "static", Address: "10.0.0.5/24", IPv6Mode: "ra"}
	content, err := buildNetworkdFileContent("eth0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "IPv6AcceptRA=yes") {
		t.Error("RA mode should produce IPv6AcceptRA=yes")
	}
	parsed := mustParseTemp(t, content)
	assertEqual(t, "IPv6Mode", parsed.IPv6Mode, "ra")
}

func TestBuildAndParse_Routes(t *testing.T) {
	cfg := IPv4Config{
		Mode:    "static",
		Address: "10.0.0.5/24",
		Routes: []RouteConfig{
			{Destination: "10.0.0.0/8", Gateway: "10.0.0.1", Metric: 100},
			{Destination: "192.168.5.0/24", Gateway: "10.0.0.254"},
			{Destination: "0.0.0.0/0"},
		},
	}
	content, err := buildNetworkdFileContent("eth0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	parsed := mustParseTemp(t, content)
	if len(parsed.Routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(parsed.Routes))
	}
	assertEqual(t, "Route[0].Dest", parsed.Routes[0].Destination, "10.0.0.0/8")
	assertEqual(t, "Route[0].GW", parsed.Routes[0].Gateway, "10.0.0.1")
	if parsed.Routes[0].Metric != 100 {
		t.Errorf("Route[0].Metric = %d, want 100", parsed.Routes[0].Metric)
	}
	assertEqual(t, "Route[2].Dest", parsed.Routes[2].Destination, "0.0.0.0/0")
}

func TestBuild_InvalidMode(t *testing.T) {
	_, err := buildNetworkdFileContent("eth0", IPv4Config{Mode: "bogus"})
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestBuild_StaticRequiresAddress(t *testing.T) {
	_, err := buildNetworkdFileContent("eth0", IPv4Config{Mode: "static"})
	if err == nil {
		t.Error("expected error for static without address")
	}
}

func TestParseMissingFile(t *testing.T) {
	cfg, err := parseDarkNetworkFile("/nonexistent/file.network")
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("expected nil for missing file")
	}
}

func TestIsIPv6Address(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"192.168.1.1/24", false},
		{"10.0.0.0/8", false},
		{"2001:db8::1/64", true},
		{"fe80::1", true},
		{"::1/128", true},
	}
	for _, tt := range tests {
		if got := isIPv6Address(tt.input); got != tt.want {
			t.Errorf("isIPv6Address(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- helpers ---

func mustParseTemp(t *testing.T, content string) *IPv4Config {
	t.Helper()
	f, err := os.CreateTemp("", "dark-test-*.network")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	cfg, err := parseDarkNetworkFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("parsed config is nil")
	}
	return cfg
}

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertSlice(t *testing.T, field string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s len = %d, want %d: %v", field, len(got), len(want), got)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", field, i, got[i], want[i])
		}
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
