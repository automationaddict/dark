package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// sshdConfigPath is the canonical location dark writes to. Hardcoded
// (rather than accepted as an argument) so an unprivileged caller
// can't redirect the write to a different root-owned file.
const sshdConfigPath = "/etc/ssh/sshd_config"

// maxSSHDConfigBytes caps the stdin read so a misbehaving darkd can't
// use the helper as an arbitrary write primitive. Real sshd_config
// files are a few kilobytes; 128 KiB gives generous headroom.
const maxSSHDConfigBytes = 128 * 1024

// writeSSHDConfig reads a proposed sshd_config from stdin, validates
// it with `sshd -t -f <tmp>`, and atomically installs it over
// /etc/ssh/sshd_config. The previous contents are saved to a .bak
// alongside the target so a caller can recover by hand if the new
// config proves to be broken in a way `sshd -t` didn't catch (e.g. a
// directive that's valid syntactically but locks the user out).
//
// Validation happens BEFORE the rename, never after. A failed
// validation leaves the on-disk file untouched and returns the
// sshd -t stderr verbatim so the user can see which line broke.
func writeSSHDConfig() error {
	data, err := io.ReadAll(io.LimitReader(os.Stdin, maxSSHDConfigBytes+1))
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if len(data) > maxSSHDConfigBytes {
		return fmt.Errorf("input too large (max %d bytes)", maxSSHDConfigBytes)
	}
	if len(data) == 0 {
		return fmt.Errorf("refusing to write empty sshd_config")
	}

	// Stage the proposed content in a tmp file next to the target
	// so `sshd -t -f` reads the real file (same inode, same fs).
	// Using /tmp would mean sshd reads from a different mount,
	// which is fine but less hygienic.
	tmp := sshdConfigPath + ".dark-tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	defer os.Remove(tmp)

	if stderr, err := runSSHDTest(tmp); err != nil {
		return fmt.Errorf("sshd -t rejected the proposed config:\n%s", stderr)
	}

	// Save a backup of the existing file before overwriting. Best
	// effort — an absent original (fresh install?) isn't an error.
	if orig, err := os.ReadFile(sshdConfigPath); err == nil {
		_ = os.WriteFile(sshdConfigPath+".bak", orig, 0o600)
	}

	if err := os.Rename(tmp, sshdConfigPath); err != nil {
		return fmt.Errorf("rename into place: %w", err)
	}
	// Enforce the invariant even if the caller shipped a wider mode.
	_ = os.Chmod(sshdConfigPath, 0o600)
	return nil
}

// restoreSSHDConfig rolls /etc/ssh/sshd_config back to its .bak
// sibling. Validation runs against the backup before the rename
// so a .bak that's been rendered obsolete by a kernel update (new
// directive names, dropped ones) doesn't brick ssh. The original
// file is preserved at .bak.prev so two consecutive restores can
// still recover the state just before the last restore.
func restoreSSHDConfig() error {
	bak := sshdConfigPath + ".bak"
	data, err := os.ReadFile(bak)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no backup at %s", bak)
		}
		return fmt.Errorf("read backup: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("backup at %s is empty", bak)
	}

	tmp := sshdConfigPath + ".dark-restore"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("stage restore: %w", err)
	}
	defer os.Remove(tmp)

	if stderr, err := runSSHDTest(tmp); err != nil {
		return fmt.Errorf("sshd -t rejected the backup:\n%s", stderr)
	}

	// Keep a .prev of the current file before overwriting, so the
	// user has one more level of rollback available.
	if cur, err := os.ReadFile(sshdConfigPath); err == nil {
		_ = os.WriteFile(sshdConfigPath+".bak.prev", cur, 0o600)
	}
	if err := os.Rename(tmp, sshdConfigPath); err != nil {
		return fmt.Errorf("rename into place: %w", err)
	}
	_ = os.Chmod(sshdConfigPath, 0o600)
	return nil
}

// runSSHDTest invokes `sshd -t -f <path>` and returns any stderr
// output on failure. `sshd` itself exits 0 on a clean parse, non-zero
// on any error, and writes its diagnostic to stderr. The full stderr
// surface bubbles up to the user so they see the line number and
// directive name that broke.
func runSSHDTest(path string) (string, error) {
	cmd := exec.Command("sshd", "-t", "-f", path)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return errBuf.String(), err
}
