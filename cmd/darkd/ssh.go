package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	sshsvc "github.com/automationaddict/dark/internal/services/ssh"
)

// sshResponse is the shared reply shape for every ssh.* command so
// callers see a consistent {snapshot, error} envelope regardless of
// which operation they triggered.
type sshResponse struct {
	Snapshot sshsvc.Snapshot `json:"snapshot"`
	Error    string          `json:"error,omitempty"`
}

// wireSSH subscribes the full SSH command surface. The snapshot
// publisher returned by this function is fired once at daemon
// startup and again on SIGHUP (see main.go's SIGHUP block) so the
// TUI sees current state on cold start without waiting for a
// periodic tick.
func wireSSH(nc *nats.Conn) func() {
	svc := sshsvc.NewService()

	// Snapshot — bare shape, matches the client's startup request
	// decoder which reads directly into sshsvc.Snapshot.
	if _, err := nc.Subscribe(bus.SubjectSSHSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(svc.Snapshot())
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectSSHSnapshotCmd, "error", err)
		os.Exit(1)
	}

	// sub wraps a command handler in the common flow: run it,
	// collect any error, build a fresh {snapshot, error} reply,
	// respond on the request path, and optionally broadcast the new
	// snapshot for subscribers. Keeps every handler body to just
	// the operation-specific plumbing.
	sub := func(subject string, timeout time.Duration, broadcast bool, op func(ctx context.Context, m *nats.Msg) error) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			resp := sshResponse{}
			if err := op(ctx, m); err != nil {
				resp.Error = err.Error()
			}
			resp.Snapshot = svc.Snapshot()
			data, _ := json.Marshal(resp)
			respond(m, data)
			if broadcast {
				snapData, _ := json.Marshal(resp.Snapshot)
				publish(nc, bus.SubjectSSHSnapshot, snapData)
			}
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	sub(bus.SubjectSSHGenerateKeyCmd, 30*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req sshsvc.GenerateKeyOptions
		_ = json.Unmarshal(m.Data, &req)
		_, err := svc.Backend().GenerateKey(ctx, req)
		return err
	})

	sub(bus.SubjectSSHDeleteKeyCmd, 10*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().DeleteKey(ctx, req.Path)
	})

	sub(bus.SubjectSSHChangePassphraseCmd, 10*time.Second, false, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Path          string `json:"path"`
			OldPassphrase string `json:"old_passphrase"`
			NewPassphrase string `json:"new_passphrase"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().ChangePassphrase(ctx, req.Path, req.OldPassphrase, req.NewPassphrase)
	})

	sub(bus.SubjectSSHAgentStartCmd, 10*time.Second, true, func(ctx context.Context, _ *nats.Msg) error {
		return svc.Backend().AgentStart(ctx)
	})

	sub(bus.SubjectSSHAgentStopCmd, 10*time.Second, true, func(ctx context.Context, _ *nats.Msg) error {
		return svc.Backend().AgentStop(ctx)
	})

	sub(bus.SubjectSSHAgentAddCmd, 15*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Path            string `json:"path"`
			Passphrase      string `json:"passphrase"`
			LifetimeSeconds int    `json:"lifetime_seconds"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().AgentAdd(ctx, req.Path, req.Passphrase, req.LifetimeSeconds)
	})

	sub(bus.SubjectSSHAgentRemoveCmd, 10*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Fingerprint string `json:"fingerprint"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().AgentRemove(ctx, req.Fingerprint)
	})

	sub(bus.SubjectSSHAgentRemoveAllCmd, 10*time.Second, true, func(ctx context.Context, _ *nats.Msg) error {
		return svc.Backend().AgentRemoveAll(ctx)
	})

	sub(bus.SubjectSSHSaveHostCmd, 15*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var entry sshsvc.HostEntry
		_ = json.Unmarshal(m.Data, &entry)
		if err := svc.Backend().SaveHostEntry(ctx, entry); err != nil {
			return err
		}
		// Auto-scan after save. Best effort — a host that isn't
		// reachable shouldn't fail the whole save.
		if entry.HostName != "" {
			_, _ = svc.Backend().ScanHost(ctx, entry.HostName)
		}
		return nil
	})

	sub(bus.SubjectSSHDeleteHostCmd, 10*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Pattern string `json:"pattern"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().DeleteHostEntry(ctx, req.Pattern)
	})

	sub(bus.SubjectSSHScanHostCmd, 15*time.Second, false, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Hostname string `json:"hostname"`
		}
		_ = json.Unmarshal(m.Data, &req)
		_, err := svc.Backend().ScanHost(ctx, req.Hostname)
		return err
	})

	sub(bus.SubjectSSHRemoveKnownHostCmd, 10*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Hostname string `json:"hostname"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().RemoveKnownHost(ctx, req.Hostname)
	})

	sub(bus.SubjectSSHAddAuthorizedKeyCmd, 10*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Line string `json:"line"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().AddAuthorizedKey(ctx, req.Line)
	})

	sub(bus.SubjectSSHRemoveAuthorizedKeyCmd, 10*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Fingerprint string `json:"fingerprint"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().RemoveAuthorizedKey(ctx, req.Fingerprint)
	})

	// Server config edits run sshd -t against the staged content
	// before the atomic rename. Timeout is generous because pkexec
	// may wait on the user to authenticate via the polkit dialog.
	sub(bus.SubjectSSHSaveServerConfigCmd, 60*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var edit sshsvc.ServerConfigEdit
		if err := json.Unmarshal(m.Data, &edit); err != nil {
			return err
		}
		return svc.Backend().SaveServerConfig(ctx, edit)
	})

	sub(bus.SubjectSSHRestoreBackupCmd, 60*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var req struct {
			Target string `json:"target"`
		}
		_ = json.Unmarshal(m.Data, &req)
		return svc.Backend().RestoreBackup(ctx, sshsvc.RestoreTarget(req.Target))
	})

	sub(bus.SubjectSSHSignKeyCmd, 30*time.Second, true, func(ctx context.Context, m *nats.Msg) error {
		var opts sshsvc.SignKeyOptions
		_ = json.Unmarshal(m.Data, &opts)
		_, err := svc.Backend().SignKey(ctx, opts)
		return err
	})

	publishSnapshot := func() {
		data, err := json.Marshal(svc.Snapshot())
		if err != nil {
			slog.Warn("ssh: marshal snapshot", "err", err)
			return
		}
		if err := nc.Publish(bus.SubjectSSHSnapshot, data); err != nil {
			slog.Warn("ssh: publish snapshot", "err", err)
		}
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			publishSnapshot()
		}
	}()

	return publishSnapshot
}
