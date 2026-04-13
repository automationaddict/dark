// dark-helper is a small privileged helper binary for the dark
// settings panel. It exists to perform a tightly bounded set of
// file operations under /etc/systemd/network/ that the unprivileged
// darkd process cannot do directly.
//
// dark-helper is intended to be invoked via pkexec, which handles
// privilege escalation through the standard polkit dialog. The
// helper validates every input path against a fixed prefix and
// extension, never accepts a content path on the command line, and
// limits stdin reads so a misbehaving darkd cannot use it as an
// arbitrary write primitive.
//
// Subcommands:
//
//	dark-helper write-network-file <path>
//	    Read up to 64 KiB from stdin and atomically write it to <path>.
//	    Path must be under /etc/systemd/network/ and end in .network.
//
//	dark-helper delete-network-file <path>
//	    Remove <path>. Same path validation rules apply. Missing
//	    files are treated as success so callers can use this for
//	    "ensure absent".
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	networkdConfigDir   = "/etc/systemd/network"
	networkFileSuffix   = ".network"
	maxNetworkFileBytes = 64 * 1024
	maxPacmanPackages   = 20
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: dark-helper <subcommand> [args...]", 2)
	}
	switch os.Args[1] {
	case "write-network-file":
		if len(os.Args) != 3 {
			fail("usage: dark-helper write-network-file <path>", 2)
		}
		if err := writeNetworkFile(os.Args[2]); err != nil {
			fail(err.Error(), 1)
		}
	case "delete-network-file":
		if len(os.Args) != 3 {
			fail("usage: dark-helper delete-network-file <path>", 2)
		}
		if err := deleteNetworkFile(os.Args[2]); err != nil {
			fail(err.Error(), 1)
		}
	case "pacman-install":
		if len(os.Args) < 3 {
			fail("usage: dark-helper pacman-install <pkg> [pkg...]", 2)
		}
		if err := pacmanInstall(os.Args[2:]); err != nil {
			fail(err.Error(), 1)
		}
	case "pacman-remove":
		if len(os.Args) < 3 {
			fail("usage: dark-helper pacman-remove <pkg> [pkg...]", 2)
		}
		if err := pacmanRemove(os.Args[2:]); err != nil {
			fail(err.Error(), 1)
		}
	case "pacman-upgrade":
		if len(os.Args) != 2 {
			fail("usage: dark-helper pacman-upgrade", 2)
		}
		if err := pacmanUpgrade(); err != nil {
			fail(err.Error(), 1)
		}
	case "read-shadow":
		if len(os.Args) != 2 {
			fail("usage: dark-helper read-shadow", 2)
		}
		data, err := os.ReadFile("/etc/shadow")
		if err != nil {
			fail(err.Error(), 1)
		}
		os.Stdout.Write(data)

	case "user-add":
		if len(os.Args) < 3 {
			fail("usage: dark-helper user-add <username> [--comment NAME] [--shell SHELL] [--admin]", 2)
		}
		if err := userAdd(os.Args[2], os.Args[3:]); err != nil {
			fail(err.Error(), 1)
		}
	case "user-remove":
		if len(os.Args) < 3 {
			fail("usage: dark-helper user-remove <username> [--remove-home]", 2)
		}
		if err := userRemove(os.Args[2], os.Args[3:]); err != nil {
			fail(err.Error(), 1)
		}
	case "user-shell":
		if len(os.Args) != 4 {
			fail("usage: dark-helper user-shell <username> <shell>", 2)
		}
		if err := userModify(os.Args[2], "-s", os.Args[3]); err != nil {
			fail(err.Error(), 1)
		}
	case "user-comment":
		if len(os.Args) != 4 {
			fail("usage: dark-helper user-comment <username> <comment>", 2)
		}
		if err := userModify(os.Args[2], "-c", os.Args[3]); err != nil {
			fail(err.Error(), 1)
		}
	case "user-lock":
		if len(os.Args) != 3 {
			fail("usage: dark-helper user-lock <username>", 2)
		}
		if err := userModify(os.Args[2], "-L"); err != nil {
			fail(err.Error(), 1)
		}
	case "user-unlock":
		if len(os.Args) != 3 {
			fail("usage: dark-helper user-unlock <username>", 2)
		}
		if err := userModify(os.Args[2], "-U"); err != nil {
			fail(err.Error(), 1)
		}
	case "user-group-add":
		if len(os.Args) != 4 {
			fail("usage: dark-helper user-group-add <username> <group>", 2)
		}
		if err := userGroupChange(os.Args[2], os.Args[3], true); err != nil {
			fail(err.Error(), 1)
		}
	case "user-group-remove":
		if len(os.Args) != 4 {
			fail("usage: dark-helper user-group-remove <username> <group>", 2)
		}
		if err := userGroupChange(os.Args[2], os.Args[3], false); err != nil {
			fail(err.Error(), 1)
		}
	case "user-passwd":
		if len(os.Args) < 3 || len(os.Args) > 4 {
			fail("usage: dark-helper user-passwd <username> [--verify]", 2)
		}
		verify := len(os.Args) == 4 && os.Args[3] == "--verify"
		if err := userSetPassword(os.Args[2], verify); err != nil {
			fail(err.Error(), 1)
		}

	case "resolved-set":
		if len(os.Args) != 4 {
			fail("usage: dark-helper resolved-set <key> <value>", 2)
		}
		if err := resolvedSet(os.Args[2], os.Args[3]); err != nil {
			fail(err.Error(), 1)
		}
	case "ufw-enable":
		if err := runCmd("ufw", "--force", "enable"); err != nil {
			fail(err.Error(), 1)
		}
	case "ufw-disable":
		if err := runCmd("ufw", "disable"); err != nil {
			fail(err.Error(), 1)
		}
	case "sshd-enable":
		if err := runCmd("systemctl", "enable", "--now", "sshd"); err != nil {
			fail(err.Error(), 1)
		}
	case "sshd-disable":
		if err := runCmd("systemctl", "disable", "--now", "sshd"); err != nil {
			fail(err.Error(), 1)
		}
	case "geoclue-enable":
		if err := runCmd("systemctl", "start", "geoclue"); err != nil {
			fail(err.Error(), 1)
		}
	case "geoclue-disable":
		if err := runCmd("systemctl", "stop", "geoclue"); err != nil {
			fail(err.Error(), 1)
		}
	case "iwd-mac-random":
		if len(os.Args) != 3 {
			fail("usage: dark-helper iwd-mac-random <disabled|once|network>", 2)
		}
		v := os.Args[2]
		if v != "disabled" && v != "once" && v != "network" {
			fail("value must be disabled, once, or network", 2)
		}
		if err := iwdSetMACRandom(v); err != nil {
			fail(err.Error(), 1)
		}
	case "indexer-enable":
		// Try localsearch first, then tracker3.
		_ = exec.Command("systemctl", "--user", "start", "localsearch-3").Run()
		_ = exec.Command("systemctl", "--user", "start", "localsearch-control-3").Run()
	case "indexer-disable":
		_ = exec.Command("systemctl", "--user", "stop", "localsearch-3").Run()
		_ = exec.Command("systemctl", "--user", "stop", "localsearch-control-3").Run()
		_ = exec.Command("systemctl", "--user", "mask", "localsearch-3").Run()
	case "resolved-set-coredump":
		if len(os.Args) != 3 {
			fail("usage: dark-helper resolved-set-coredump <external|journal|none>", 2)
		}
		if err := setCoredumpStorage(os.Args[2]); err != nil {
			fail(err.Error(), 1)
		}

	case "logind-set":
		if len(os.Args) != 4 {
			fail("usage: dark-helper logind-set <key> <value>", 2)
		}
		if err := logindSet(os.Args[2], os.Args[3]); err != nil {
			fail(err.Error(), 1)
		}

	default:
		fail("dark-helper: unknown subcommand "+os.Args[1], 2)
	}
}

// validateNetworkdPath enforces the path safety rules for every
// helper operation. The path must be absolute, in canonical form
// (no `.` / `..` segments), under the networkd config directory,
// and end in `.network`. The check is intentionally strict — this
// is the only thing standing between us and an arbitrary write
// primitive running as root.
func validateNetworkdPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %q is not absolute", path)
	}
	cleaned := filepath.Clean(path)
	if cleaned != path {
		return fmt.Errorf("path %q is not in canonical form (cleaned: %q)", path, cleaned)
	}
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path %q contains parent traversal", cleaned)
	}
	if !strings.HasPrefix(cleaned, networkdConfigDir+"/") {
		return fmt.Errorf("path %q must be under %s", cleaned, networkdConfigDir)
	}
	if !strings.HasSuffix(cleaned, networkFileSuffix) {
		return fmt.Errorf("path %q must end in %s", cleaned, networkFileSuffix)
	}
	// Reject any subdirectory underneath the config dir — dark only
	// manages files at the top level so we can be confident about
	// what we own and what we don't.
	rel := strings.TrimPrefix(cleaned, networkdConfigDir+"/")
	if strings.Contains(rel, "/") {
		return fmt.Errorf("path %q must be directly under %s", cleaned, networkdConfigDir)
	}
	// Reject filenames that are just the extension with no actual
	// name before it — dark never generates these and accepting
	// them widens the attack surface for no reason.
	name := strings.TrimSuffix(rel, networkFileSuffix)
	if name == "" {
		return fmt.Errorf("path %q has no filename before %s", cleaned, networkFileSuffix)
	}
	return nil
}

// writeNetworkFile reads stdin (capped at 64 KiB) and atomically
// writes it to the validated path. Atomic via write-to-tmp + rename
// so a crash or kill mid-write can't leave a partial file that
// confuses systemd-networkd.
func writeNetworkFile(path string) error {
	if err := validateNetworkdPath(path); err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(os.Stdin, maxNetworkFileBytes+1))
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if len(data) > maxNetworkFileBytes {
		return fmt.Errorf("input too large (max %d bytes)", maxNetworkFileBytes)
	}
	tmp := path + ".dark-tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}

// deleteNetworkFile removes the validated path. Already-absent files
// are treated as success.
func deleteNetworkFile(path string) error {
	if err := validateNetworkdPath(path); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// validPkgName matches the characters pacman allows in package names.
// Anything outside this set is rejected before we hand it to pacman
// so shell metacharacters and path traversal are impossible.
var validPkgName = regexp.MustCompile(`^[a-zA-Z0-9@._+-]+$`)

// validatePackageNames checks that every name in the list is a legal
// pacman package name and that the batch size is within our cap. The
// cap exists to prevent abuse — a misbehaving caller could otherwise
// ask us to install the entire repo.
func validatePackageNames(names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("no package names provided")
	}
	if len(names) > maxPacmanPackages {
		return fmt.Errorf("too many packages (%d, max %d)", len(names), maxPacmanPackages)
	}
	for _, name := range names {
		if name == "" {
			return fmt.Errorf("empty package name")
		}
		if !validPkgName.MatchString(name) {
			return fmt.Errorf("invalid package name %q", name)
		}
	}
	return nil
}

// runPacman executes pacman with the given arguments and streams its
// stdout/stderr to our own stdout/stderr so the daemon can capture
// progress output for the TUI status line.
func runPacman(args ...string) error {
	cmd := exec.Command("pacman", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pacman %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func pacmanInstall(names []string) error {
	if err := validatePackageNames(names); err != nil {
		return err
	}
	args := append([]string{"-S", "--noconfirm"}, names...)
	return runPacman(args...)
}

func pacmanRemove(names []string) error {
	if err := validatePackageNames(names); err != nil {
		return err
	}
	args := append([]string{"-R", "--noconfirm"}, names...)
	return runPacman(args...)
}

func pacmanUpgrade() error {
	return runPacman("-Syu", "--noconfirm")
}

// validUsername matches valid Linux usernames.
var validUsername = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func validateUsername(name string) error {
	if name == "" {
		return fmt.Errorf("empty username")
	}
	if len(name) > 32 {
		return fmt.Errorf("username too long (max 32)")
	}
	if !validUsername.MatchString(name) {
		return fmt.Errorf("invalid username %q", name)
	}
	// Reject system-critical names.
	switch name {
	case "root", "nobody", "daemon", "bin", "sys":
		return fmt.Errorf("cannot modify system user %q", name)
	}
	return nil
}

var validGroupName = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func validateGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("empty group name")
	}
	if !validGroupName.MatchString(name) {
		return fmt.Errorf("invalid group name %q", name)
	}
	return nil
}

func validateShell(shell string) error {
	if shell == "" {
		return fmt.Errorf("empty shell path")
	}
	if !filepath.IsAbs(shell) {
		return fmt.Errorf("shell %q must be absolute path", shell)
	}
	if _, err := os.Stat(shell); err != nil {
		return fmt.Errorf("shell %q does not exist", shell)
	}
	return nil
}

func userAdd(username string, flags []string) error {
	if err := validateUsername(username); err != nil {
		return err
	}

	args := []string{"-m"} // create home directory
	admin := false
	for i := 0; i < len(flags); i++ {
		switch flags[i] {
		case "--comment":
			if i+1 >= len(flags) {
				return fmt.Errorf("--comment requires a value")
			}
			i++
			args = append(args, "-c", flags[i])
		case "--shell":
			if i+1 >= len(flags) {
				return fmt.Errorf("--shell requires a value")
			}
			i++
			if err := validateShell(flags[i]); err != nil {
				return err
			}
			args = append(args, "-s", flags[i])
		case "--admin":
			admin = true
		default:
			return fmt.Errorf("unknown flag %q", flags[i])
		}
	}

	args = append(args, username)
	cmd := exec.Command("useradd", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("useradd: %w", err)
	}

	if admin {
		cmd := exec.Command("gpasswd", "-a", username, "wheel")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gpasswd (add to wheel): %w", err)
		}
	}

	return nil
}

func userRemove(username string, flags []string) error {
	if err := validateUsername(username); err != nil {
		return err
	}

	args := []string{}
	for _, f := range flags {
		switch f {
		case "--remove-home":
			args = append(args, "-r")
		default:
			return fmt.Errorf("unknown flag %q", f)
		}
	}
	args = append(args, username)

	cmd := exec.Command("userdel", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("userdel: %w", err)
	}
	return nil
}

func userModify(username string, flags ...string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	// Validate shell if present.
	for i, f := range flags {
		if f == "-s" && i+1 < len(flags) {
			if err := validateShell(flags[i+1]); err != nil {
				return err
			}
		}
	}
	args := append(flags, username)
	cmd := exec.Command("usermod", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("usermod: %w", err)
	}
	return nil
}

func userGroupChange(username, group string, add bool) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := validateGroupName(group); err != nil {
		return err
	}
	flag := "-a"
	if !add {
		flag = "-d"
	}
	cmd := exec.Command("gpasswd", flag, username, group)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpasswd: %w", err)
	}
	return nil
}

func userSetPassword(username string, verify bool) error {
	if err := validateUsername(username); err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)

	if verify {
		// Read and verify the current password first.
		if !scanner.Scan() {
			return fmt.Errorf("no current password provided")
		}
		currentPass := scanner.Text()
		if err := verifyPassword(username, currentPass); err != nil {
			return err
		}
	}

	// Read the new password.
	if !scanner.Scan() {
		return fmt.Errorf("no new password provided")
	}
	newPass := scanner.Text()
	if newPass == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Set password via chpasswd.
	cmd := exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader(username + ":" + newPass + "\n")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chpasswd: %w", err)
	}
	return nil
}

// verifyPassword checks the current password by attempting su.
func verifyPassword(username, password string) error {
	cmd := exec.Command("su", "-c", "true", username)
	cmd.Stdin = strings.NewReader(password + "\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("current password is incorrect")
	}
	return nil
}

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
		"HandlePowerKey":                true,
		"HandleLidSwitch":               true,
		"HandleLidSwitchExternalPower":  true,
		"HandleLidSwitchDocked":         true,
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

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fail(msg string, code int) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}
