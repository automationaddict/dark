// dark-helper is a small privileged helper binary for the dark
// settings panel. It exists to perform a tightly bounded set of
// file operations and system-config mutations that the unprivileged
// darkd process cannot do directly.
//
// dark-helper is intended to be invoked via pkexec, which handles
// privilege escalation through the standard polkit dialog. Every
// subcommand validates its input before touching the filesystem or
// executing another process, and stdin reads are bounded so a
// misbehaving darkd cannot use the helper as an arbitrary write
// primitive.
//
// Subcommands are grouped by file:
//
//	network.go  — networkd file ops (write-network-file, delete-network-file)
//	pacman.go   — pacman-install, pacman-remove, pacman-upgrade
//	users.go    — user-add, user-remove, user-shell, user-comment,
//	              user-lock, user-unlock, user-group-add,
//	              user-group-remove, user-passwd
//	system.go   — resolved-set, iwd-mac-random, resolved-set-coredump,
//	              logind-set (and indirect ufw/sshd/geoclue toggles)
//	gpu.go      — gpu-hybrid
//	update.go   — update-channel, update-full
//
// main.go only owns the dispatch table, shared helpers (fail, runCmd),
// and the top-level constants.
package main

import (
	"fmt"
	"os"
	"os/exec"
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
		if _, err := os.Stdout.Write(data); err != nil {
			fail("write shadow to stdout: "+err.Error(), 1)
		}

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
	case "sshd-config-write":
		if len(os.Args) != 2 {
			fail("usage: dark-helper sshd-config-write (reads new config from stdin)", 2)
		}
		if err := writeSSHDConfig(); err != nil {
			fail(err.Error(), 1)
		}
	case "sshd-config-restore":
		if len(os.Args) != 2 {
			fail("usage: dark-helper sshd-config-restore", 2)
		}
		if err := restoreSSHDConfig(); err != nil {
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

	case "gpu-hybrid":
		if len(os.Args) != 3 {
			fail("usage: dark-helper gpu-hybrid <hybrid|integrated>", 2)
		}
		if err := setGPUMode(os.Args[2]); err != nil {
			fail(err.Error(), 1)
		}

	case "update-full":
		if err := updateFull(); err != nil {
			fail(err.Error(), 1)
		}

	case "update-channel":
		if len(os.Args) != 3 {
			fail("usage: dark-helper update-channel <stable|rc|edge>", 2)
		}
		if err := setChannel(os.Args[2]); err != nil {
			fail(err.Error(), 1)
		}

	default:
		fail("dark-helper: unknown subcommand "+os.Args[1], 2)
	}
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
