package main

import "testing"

func TestValidateNetworkdPath(t *testing.T) {
	valid := []string{
		"/etc/systemd/network/50-dark-eth0.network",
		"/etc/systemd/network/99-custom.network",
	}
	for _, p := range valid {
		if err := validateNetworkdPath(p); err != nil {
			t.Errorf("expected valid: %q → %v", p, err)
		}
	}

	invalid := []struct {
		path   string
		reason string
	}{
		{"", "empty"},
		{"relative.network", "relative path"},
		{"/etc/dark/foo.network", "wrong directory"},
		{"/etc/systemd/network/../passwd", "parent traversal"},
		{"/etc/systemd/network/foo", "no .network extension"},
		{"/etc/systemd/network/sub/foo.network", "subdirectory"},
		{"/etc/systemd/network/.network", "dot-only name"},
	}
	for _, tt := range invalid {
		if err := validateNetworkdPath(tt.path); err == nil {
			t.Errorf("expected rejection for %s: %q", tt.reason, tt.path)
		}
	}
}
