package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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
