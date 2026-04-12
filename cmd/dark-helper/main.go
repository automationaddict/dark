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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	networkdConfigDir   = "/etc/systemd/network"
	networkFileSuffix   = ".network"
	maxNetworkFileBytes = 64 * 1024
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

func fail(msg string, code int) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}
