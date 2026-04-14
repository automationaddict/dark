package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolvedSet updates a key in /etc/systemd/resolved.conf and restarts
// systemd-resolved. Only a small set of keys is allowed.
func resolvedSet(key, value string) error {
	allowed := map[string]bool{"DNSOverTLS": true, "DNSSEC": true, "DNS": true, "FallbackDNS": true}
	if !allowed[key] {
		return fmt.Errorf("key %q not allowed", key)
	}

	data, err := os.ReadFile("/etc/systemd/resolved.conf")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match both active and commented-out lines.
		bare := strings.TrimPrefix(trimmed, "#")
		k, _, ok := strings.Cut(bare, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == key {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}
	if !found {
		// Insert after [Resolve] header.
		for i, line := range lines {
			if strings.TrimSpace(line) == "[Resolve]" {
				insert := key + "=" + value
				lines = append(lines[:i+1], append([]string{insert}, lines[i+1:]...)...)
				break
			}
		}
	}

	if err := os.WriteFile("/etc/systemd/resolved.conf", []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	return exec.Command("systemctl", "restart", "systemd-resolved").Run()
}

func iwdSetMACRandom(value string) error {
	path := "/etc/iwd/main.conf"
	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "[General]") {
		content = "[General]\nAddressRandomization=" + value + "\n" + content
	} else {
		lines := strings.Split(content, "\n")
		found := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			bare := strings.TrimPrefix(trimmed, "#")
			k, _, ok := strings.Cut(bare, "=")
			if ok && strings.TrimSpace(k) == "AddressRandomization" {
				lines[i] = "AddressRandomization=" + value
				found = true
				break
			}
		}
		if !found {
			for i, line := range lines {
				if strings.TrimSpace(line) == "[General]" {
					lines = append(lines[:i+1], append([]string{"AddressRandomization=" + value}, lines[i+1:]...)...)
					break
				}
			}
		}
		content = strings.Join(lines, "\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return exec.Command("systemctl", "restart", "iwd").Run()
}

func setCoredumpStorage(value string) error {
	allowed := map[string]bool{"external": true, "journal": true, "none": true}
	if !allowed[value] {
		return fmt.Errorf("value must be external, journal, or none")
	}

	path := "/etc/systemd/coredump.conf"
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		bare := strings.TrimPrefix(trimmed, "#")
		k, _, ok := strings.Cut(bare, "=")
		if ok && strings.TrimSpace(k) == "Storage" {
			lines[i] = "Storage=" + value
			found = true
			break
		}
	}
	if !found {
		for i, line := range lines {
			if strings.TrimSpace(line) == "[Coredump]" {
				lines = append(lines[:i+1], append([]string{"Storage=" + value}, lines[i+1:]...)...)
				break
			}
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func logindSet(key, value string) error {
	allowed := map[string]bool{
		"HandlePowerKey":               true,
		"HandleLidSwitch":              true,
		"HandleLidSwitchExternalPower": true,
		"HandleLidSwitchDocked":        true,
	}
	if !allowed[key] {
		return fmt.Errorf("key %q not allowed for logind", key)
	}
	validValues := map[string]bool{
		"ignore": true, "poweroff": true, "reboot": true,
		"halt": true, "suspend": true, "hibernate": true,
		"hybrid-sleep": true, "suspend-then-hibernate": true,
		"lock": true,
	}
	if !validValues[value] {
		return fmt.Errorf("value %q not allowed for logind", value)
	}

	const confPath = "/etc/systemd/logind.conf"
	data, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		bare := strings.TrimPrefix(trimmed, "#")
		k, _, ok := strings.Cut(bare, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == key {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}
	if !found {
		for i, line := range lines {
			if strings.TrimSpace(line) == "[Login]" {
				insert := key + "=" + value
				lines = append(lines[:i+1], append([]string{insert}, lines[i+1:]...)...)
				break
			}
		}
	}

	if err := os.WriteFile(confPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	// Reload logind so changes take effect immediately.
	_ = exec.Command("systemctl", "kill", "-s", "HUP", "systemd-logind").Run()
	return nil
}
