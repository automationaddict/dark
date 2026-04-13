package privacy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Snapshot struct {
	// Screen lock & idle.
	ScreensaverTimeout int  `json:"screensaver_timeout"` // seconds
	LockTimeout        int  `json:"lock_timeout"`
	ScreenOffTimeout   int  `json:"screen_off_timeout"`
	LockOnSleep        bool `json:"lock_on_sleep"`

	// DNS privacy.
	DNSServer    string `json:"dns_server"`
	DNSOverTLS   string `json:"dns_over_tls"`   // "yes", "no", "opportunistic"
	DNSSEC       string `json:"dnssec"`          // "yes", "no", "allow-downgrade"
	FallbackDNS  string `json:"fallback_dns"`
	DNSProtocols string `json:"dns_protocols"`

	// Firewall.
	FirewallInstalled bool     `json:"firewall_installed"`
	FirewallActive    bool     `json:"firewall_active"`
	FirewallRules     []string `json:"firewall_rules,omitempty"`

	// SSH server.
	SSHInstalled bool `json:"ssh_installed"`
	SSHActive    bool `json:"ssh_active"`
	SSHEnabled   bool `json:"ssh_enabled"`

	// Recent files.
	RecentFileCount int `json:"recent_file_count"`

	// Hyprlock.
	FingerprintEnabled bool `json:"fingerprint_enabled"`

	// Location services (geoclue).
	LocationInstalled bool `json:"location_installed"`
	LocationActive    bool `json:"location_active"`

	// WiFi MAC randomization (iwd).
	MACRandomization string `json:"mac_randomization"` // "disabled", "once", "network"

	// File indexer (localsearch/tracker).
	IndexerInstalled bool `json:"indexer_installed"`
	IndexerActive    bool `json:"indexer_active"`

	// Core dumps.
	CoredumpStorage string `json:"coredump_storage"` // "external", "journal", "none"
	JournalSize     string `json:"journal_size"`
}

func ReadSnapshot() Snapshot {
	s := Snapshot{}
	readHypridle(&s)
	readDNS(&s)
	readFirewall(&s)
	readSSH(&s)
	s.RecentFileCount = countRecentFiles()
	readHyprlock(&s)
	readLocation(&s)
	readMACRandomization(&s)
	readIndexer(&s)
	readCoredump(&s)
	return s
}

// --- Hypridle (screen lock & idle) ---

var timeoutRe = regexp.MustCompile(`timeout\s*=\s*(\d+)`)

func readHypridle(s *Snapshot) {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "hypr", "hypridle.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(data)

	// Parse general section for lock_on_sleep.
	s.LockOnSleep = strings.Contains(content, "before_sleep_cmd")

	// Parse listener blocks — identify by their on-timeout action.
	blocks := splitListenerBlocks(content)
	for _, block := range blocks {
		timeout := extractTimeout(block)
		if timeout == 0 {
			continue
		}
		lower := strings.ToLower(block)
		switch {
		case strings.Contains(lower, "screensaver"):
			s.ScreensaverTimeout = timeout
		case strings.Contains(lower, "lock-session") || strings.Contains(lower, "lock_session"):
			s.LockTimeout = timeout
		case strings.Contains(lower, "dpms off"):
			s.ScreenOffTimeout = timeout
		}
	}
}

func splitListenerBlocks(content string) []string {
	var blocks []string
	lines := strings.Split(content, "\n")
	inListener := false
	depth := 0
	var current strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "listener {" || strings.HasPrefix(trimmed, "listener {") {
			inListener = true
			depth = 1
			current.Reset()
			current.WriteString(line + "\n")
			continue
		}
		if inListener {
			current.WriteString(line + "\n")
			if strings.Contains(trimmed, "{") {
				depth++
			}
			if strings.Contains(trimmed, "}") {
				depth--
				if depth <= 0 {
					blocks = append(blocks, current.String())
					inListener = false
				}
			}
		}
	}
	return blocks
}

func extractTimeout(block string) int {
	m := timeoutRe.FindStringSubmatch(block)
	if len(m) < 2 {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

func SetIdleTimeout(field string, seconds int) error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "hypr", "hypridle.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	var marker string
	switch field {
	case "screensaver":
		marker = "screensaver"
	case "lock":
		marker = "lock-session"
	case "screen_off":
		marker = "dpms off"
	default:
		return fmt.Errorf("unknown field %q", field)
	}

	// Find the listener block containing the marker and replace its timeout.
	lines := strings.Split(content, "\n")
	inTarget := false
	replaced := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "listener {" || strings.HasPrefix(trimmed, "listener {") {
			// Look ahead to see if this block contains our marker.
			inTarget = blockContains(lines[i:], marker)
		}
		if inTarget && !replaced && timeoutRe.MatchString(trimmed) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			comment := ""
			if idx := strings.Index(line, "#"); idx >= 0 {
				comment = " " + strings.TrimSpace(line[idx:])
			}
			lines[i] = fmt.Sprintf("%stimeout = %d%s", indent, seconds, comment)
			replaced = true
		}
		if inTarget && strings.Contains(trimmed, "}") {
			inTarget = false
		}
	}

	if !replaced {
		return fmt.Errorf("could not find %s listener block", field)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func blockContains(lines []string, marker string) bool {
	depth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "{") {
			depth++
		}
		if strings.Contains(strings.ToLower(trimmed), marker) {
			return true
		}
		if strings.Contains(trimmed, "}") {
			depth--
			if depth <= 0 {
				return false
			}
		}
	}
	return false
}

// --- DNS Privacy ---

func readDNS(s *Snapshot) {
	// Read live status from resolvectl.
	out, err := exec.Command("resolvectl", "status", "--no-pager").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Current DNS Server:") {
				s.DNSServer = strings.TrimSpace(strings.TrimPrefix(line, "Current DNS Server:"))
			}
			if strings.HasPrefix(line, "Fallback DNS Servers:") {
				s.FallbackDNS = strings.TrimSpace(strings.TrimPrefix(line, "Fallback DNS Servers:"))
			}
			if strings.HasPrefix(line, "Protocols:") {
				s.DNSProtocols = strings.TrimSpace(strings.TrimPrefix(line, "Protocols:"))
			}
		}
	}

	// Read config for editable values.
	cfg := readResolvedConf()
	if v, ok := cfg["DNSOverTLS"]; ok {
		s.DNSOverTLS = v
	} else {
		s.DNSOverTLS = "no"
	}
	if v, ok := cfg["DNSSEC"]; ok {
		s.DNSSEC = v
	} else {
		s.DNSSEC = "no"
	}
}

func readResolvedConf() map[string]string {
	m := make(map[string]string)
	f, err := os.Open("/etc/systemd/resolved.conf")
	if err != nil {
		return m
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m
}

// --- Firewall ---

func readFirewall(s *Snapshot) {
	_, err := exec.LookPath("ufw")
	s.FirewallInstalled = err == nil
	if !s.FirewallInstalled {
		return
	}

	// Try reading status (needs root, may fail).
	out, err := exec.Command("ufw", "status").Output()
	if err != nil {
		return
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Status:") {
			s.FirewallActive = strings.Contains(line, "active") && !strings.Contains(line, "inactive")
		}
	}
	if s.FirewallActive && len(lines) > 4 {
		for _, line := range lines[4:] {
			line = strings.TrimSpace(line)
			if line != "" {
				s.FirewallRules = append(s.FirewallRules, line)
			}
		}
	}
}

// --- SSH Server ---

func readSSH(s *Snapshot) {
	_, err := os.Stat("/etc/ssh/sshd_config")
	s.SSHInstalled = err == nil
	if !s.SSHInstalled {
		return
	}

	if out, err := exec.Command("systemctl", "is-active", "sshd").Output(); err == nil {
		s.SSHActive = strings.TrimSpace(string(out)) == "active"
	}
	if out, err := exec.Command("systemctl", "is-enabled", "sshd").Output(); err == nil {
		s.SSHEnabled = strings.TrimSpace(string(out)) == "enabled"
	}
}

// --- Recent files ---

func countRecentFiles() int {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "recently-used.xbel")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(data), "<bookmark ")
}

func ClearRecentFiles() error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "recently-used.xbel")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<xbel version="1.0"
      xmlns:bookmark="http://www.freedesktop.org/standards/desktop-bookmarks"
      xmlns:mime="http://www.freedesktop.org/standards/shared-mime-info">
</xbel>
`
	return os.WriteFile(path, []byte(content), 0o644)
}

// --- Hyprlock ---

func readHyprlock(s *Snapshot) {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "hypr", "hyprlock.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := strings.ToLower(string(data))
	// Check for fingerprint = true in the config.
	s.FingerprintEnabled = strings.Contains(content, "fingerprint") &&
		!strings.Contains(content, "fingerprint = false")
}

// --- Location services (geoclue) ---

func readLocation(s *Snapshot) {
	_, err := exec.LookPath("geoclue")
	if err != nil {
		// Also check for the demo agent binary.
		if _, err2 := os.Stat("/usr/lib/geoclue-2.0/demos/agent"); err2 != nil {
			// Check if the package is installed.
			if out, err3 := exec.Command("pacman", "-Q", "geoclue").Output(); err3 == nil && len(out) > 0 {
				s.LocationInstalled = true
			}
			return
		}
	}
	s.LocationInstalled = true

	if out, err := exec.Command("systemctl", "is-active", "geoclue").Output(); err == nil {
		s.LocationActive = strings.TrimSpace(string(out)) == "active"
	}
}

// --- WiFi MAC randomization (iwd) ---

func readMACRandomization(s *Snapshot) {
	s.MACRandomization = "disabled"
	data, err := os.ReadFile("/etc/iwd/main.conf")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		k, v, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == "AddressRandomization" {
			s.MACRandomization = strings.TrimSpace(v)
		}
	}
}

func SetMACRandomization(value string) error {
	path := "/etc/iwd/main.conf"
	data, _ := os.ReadFile(path)
	content := string(data)

	// Ensure [General] section exists and set AddressRandomization.
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
			// Insert after [General].
			for i, line := range lines {
				if strings.TrimSpace(line) == "[General]" {
					insert := "AddressRandomization=" + value
					lines = append(lines[:i+1], append([]string{insert}, lines[i+1:]...)...)
					break
				}
			}
		}
		content = strings.Join(lines, "\n")
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

// --- File indexer (localsearch/tracker) ---

func readIndexer(s *Snapshot) {
	// Check for localsearch (tracker3 successor) or tracker-miner-fs.
	for _, pkg := range []string{"localsearch", "tracker3-miners"} {
		if out, err := exec.Command("pacman", "-Q", pkg).Output(); err == nil && len(out) > 0 {
			s.IndexerInstalled = true
			break
		}
	}
	if !s.IndexerInstalled {
		return
	}

	for _, svc := range []string{"localsearch-3", "tracker-miner-fs-3"} {
		if out, err := exec.Command("systemctl", "--user", "is-active", svc).Output(); err == nil {
			if strings.TrimSpace(string(out)) == "active" {
				s.IndexerActive = true
				return
			}
		}
	}
}

// --- Core dumps ---

func readCoredump(s *Snapshot) {
	s.CoredumpStorage = "external" // systemd default
	data, err := os.ReadFile("/etc/systemd/coredump.conf")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		k, v, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == "Storage" {
			s.CoredumpStorage = strings.TrimSpace(v)
		}
	}

	if out, err := exec.Command("journalctl", "--disk-usage").Output(); err == nil {
		// Output: "Archived and active journals take up 112.0M in the file system."
		line := strings.TrimSpace(string(out))
		if idx := strings.Index(line, "take up "); idx >= 0 {
			rest := line[idx+8:]
			if end := strings.Index(rest, " in"); end >= 0 {
				s.JournalSize = rest[:end]
			}
		}
	}
}
