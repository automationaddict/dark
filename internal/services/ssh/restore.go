package ssh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// RestoreTarget identifies which backup dark is being asked to
// roll back. Only three files create .bak files today: the client
// config, authorized_keys, and the server config. Known_hosts is
// edited via `ssh-keygen -R` which doesn't leave a backup.
type RestoreTarget string

const (
	RestoreClientConfig   RestoreTarget = "client_config"
	RestoreAuthorizedKeys RestoreTarget = "authorized_keys"
	RestoreServerConfig   RestoreTarget = "server_config"
)

// RestoreBackup rolls the named file back to its `.bak` sibling.
// The user-scoped files (~/.ssh/config, ~/.ssh/authorized_keys) are
// restored in-process via a plain rename. The root-owned
// /etc/ssh/sshd_config goes through dark-helper's
// `sshd-config-restore` subcommand which runs `sshd -t` on the
// backup before installing it so a broken .bak doesn't brick ssh.
//
// An absent .bak returns a clear "no backup to restore" error so
// the TUI can surface something the user can act on.
func (b *OpenSSHBackend) RestoreBackup(ctx context.Context, target RestoreTarget) error {
	switch target {
	case RestoreClientConfig:
		return restoreUserFile(filepath.Join(b.sshDir, "config"))
	case RestoreAuthorizedKeys:
		return restoreUserFile(filepath.Join(b.sshDir, "authorized_keys"))
	case RestoreServerConfig:
		return runSSHDHelperRestore(ctx)
	}
	return fmt.Errorf("unknown restore target %q", target)
}

// restoreUserFile handles the user-owned backup rollback. Atomic
// via rename so a crash halfway through can't leave the file in an
// inconsistent state. The .bak is consumed (removed) on success so
// a second restore without a subsequent edit fails cleanly instead
// of restoring the same state twice.
func restoreUserFile(path string) error {
	bak := path + ".bak"
	if _, err := os.Stat(bak); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no backup at %s", bak)
		}
		return err
	}
	data, err := os.ReadFile(bak)
	if err != nil {
		return err
	}
	if err := atomicWrite(path, data, 0o600); err != nil {
		return err
	}
	_ = os.Remove(bak)
	return nil
}
