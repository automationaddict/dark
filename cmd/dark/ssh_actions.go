package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	sshsvc "github.com/automationaddict/dark/internal/services/ssh"
	"github.com/automationaddict/dark/internal/tui"
)

// newSSHActions wires the TUI's SSH surface to NATS commands. Every
// command follows the {snapshot, error} reply shape wired in
// cmd/darkd/ssh.go. Requests that mutate state reply with the full
// refreshed snapshot so the client's cache is replaced in one step.
func newSSHActions(nc *nats.Conn) tui.SSHActions {
	return tui.SSHActions{
		GenerateKey: func(opts core.SSHGenerateKeyOptions) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(opts)
				return sshRequest(nc, bus.SubjectSSHGenerateKeyCmd, payload, "generate_key")
			}
		},
		DeleteKey: func(path string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"path": path})
				return sshRequest(nc, bus.SubjectSSHDeleteKeyCmd, payload, "delete_key")
			}
		},
		ChangePassphrase: func(path, oldPass, newPass string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{
					"path":            path,
					"old_passphrase":  oldPass,
					"new_passphrase":  newPass,
				})
				return sshRequest(nc, bus.SubjectSSHChangePassphraseCmd, payload, "change_passphrase")
			}
		},
		AgentStart: func() tea.Cmd {
			return func() tea.Msg {
				return sshRequest(nc, bus.SubjectSSHAgentStartCmd, nil, "agent_start")
			}
		},
		AgentStop: func() tea.Cmd {
			return func() tea.Msg {
				return sshRequest(nc, bus.SubjectSSHAgentStopCmd, nil, "agent_stop")
			}
		},
		AgentAdd: func(path, passphrase string, lifetimeSeconds int) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]interface{}{
					"path":             path,
					"passphrase":       passphrase,
					"lifetime_seconds": lifetimeSeconds,
				})
				return sshRequest(nc, bus.SubjectSSHAgentAddCmd, payload, "agent_add")
			}
		},
		AgentRemove: func(fingerprint string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"fingerprint": fingerprint})
				return sshRequest(nc, bus.SubjectSSHAgentRemoveCmd, payload, "agent_remove")
			}
		},
		AgentRemoveAll: func() tea.Cmd {
			return func() tea.Msg {
				return sshRequest(nc, bus.SubjectSSHAgentRemoveAllCmd, nil, "agent_remove_all")
			}
		},
		SaveHost: func(entry core.SSHHostEntry) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(coreHostToService(entry))
				return sshRequest(nc, bus.SubjectSSHSaveHostCmd, payload, "save_host")
			}
		},
		DeleteHost: func(pattern string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"pattern": pattern})
				return sshRequest(nc, bus.SubjectSSHDeleteHostCmd, payload, "delete_host")
			}
		},
		RemoveKnownHost: func(hostname string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"hostname": hostname})
				return sshRequest(nc, bus.SubjectSSHRemoveKnownHostCmd, payload, "remove_known_host")
			}
		},
		ScanHost: func(hostname string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"hostname": hostname})
				return sshRequest(nc, bus.SubjectSSHScanHostCmd, payload, "scan_host")
			}
		},
		AddAuthorizedKey: func(line string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"line": line})
				return sshRequest(nc, bus.SubjectSSHAddAuthorizedKeyCmd, payload, "add_authorized_key")
			}
		},
		RemoveAuthorizedKey: func(fingerprint string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"fingerprint": fingerprint})
				return sshRequest(nc, bus.SubjectSSHRemoveAuthorizedKeyCmd, payload, "remove_authorized_key")
			}
		},
		SaveServerConfig: func(edit core.SSHServerConfigEdit) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(edit)
				return sshRequest(nc, bus.SubjectSSHSaveServerConfigCmd, payload, "save_server_config")
			}
		},
		RestoreBackup: func(target string) tea.Cmd {
			return func() tea.Msg {
				payload, _ := json.Marshal(map[string]string{"target": target})
				return sshRequest(nc, bus.SubjectSSHRestoreBackupCmd, payload, "restore_backup")
			}
		},
	}
}

// sshRequest handles the common round-trip: dispatch the command,
// parse the {snapshot, error} reply, and return a SSHActionResultMsg
// the TUI can feed straight into state. Errors are wrapped with the
// action label so the user can tell which operation failed.
func sshRequest(nc *nats.Conn, subject string, payload []byte, action string) tui.SSHActionResultMsg {
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.SSHActionResultMsg{Action: action, Err: err.Error()}
	}
	var resp struct {
		Snapshot sshsvc.Snapshot `json:"snapshot"`
		Error    string          `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.SSHActionResultMsg{Action: action, Err: err.Error()}
	}
	return tui.SSHActionResultMsg{
		Action:   action,
		Snapshot: serviceSnapshotToCore(resp.Snapshot),
		Err:      resp.Error,
	}
}

// coreHostToService converts the core.SSHHostEntry mirror type back
// into the service-package type so JSON marshaling matches what the
// daemon expects.
func coreHostToService(e core.SSHHostEntry) sshsvc.HostEntry {
	return sshsvc.HostEntry{
		Pattern:               e.Pattern,
		HostName:              e.HostName,
		User:                  e.User,
		Port:                  e.Port,
		IdentityFile:          e.IdentityFile,
		ForwardAgent:          e.ForwardAgent,
		ProxyJump:             e.ProxyJump,
		StrictHostKeyChecking: e.StrictHostKeyChecking,
		Extras:                e.Extras,
	}
}

// serviceSnapshotToCore mirrors a service-package Snapshot into the
// core mirror type. The TUI never imports internal/services/ssh,
// matching the pattern used by scripting + every F1 service.
func serviceSnapshotToCore(s sshsvc.Snapshot) core.SSHSnapshot {
	out := core.SSHSnapshot{
		InstalledOK: s.InstalledOK,
		Backend:     s.Backend,
		LastError:   s.LastError,
		Agent: core.SSHAgentStatus{
			Running:           s.Agent.Running,
			SystemdManaged:    s.Agent.SystemdManaged,
			SystemdUnitExists: s.Agent.SystemdUnitExists,
			Forwarded:         s.Agent.Forwarded,
			SocketPath:        s.Agent.SocketPath,
			Pid:               s.Agent.Pid,
		},
		ClientConfig: core.SSHClientConfig{
			Path: s.ClientConfig.Path,
		},
		ServerConfig: core.SSHServerConfig{
			Path:                   s.ServerConfig.Path,
			Readable:               s.ServerConfig.Readable,
			Port:                   s.ServerConfig.Port,
			PermitRootLogin:        s.ServerConfig.PermitRootLogin,
			PasswordAuthentication: s.ServerConfig.PasswordAuthentication,
			PubkeyAuthentication:   s.ServerConfig.PubkeyAuthentication,
			X11Forwarding:          s.ServerConfig.X11Forwarding,
			AllowUsers:             s.ServerConfig.AllowUsers,
			AllowGroups:            s.ServerConfig.AllowGroups,
			ParseError:             s.ServerConfig.ParseError,
		},
	}
	for _, c := range s.Certificates {
		out.Certificates = append(out.Certificates, core.SSHCertificate{
			CertPath:       c.CertPath,
			Type:           c.Type,
			KeyID:          c.KeyID,
			Serial:         c.Serial,
			ValidAfter:     c.ValidAfter,
			ValidBefore:    c.ValidBefore,
			Principals:     c.Principals,
			CAFingerprint:  c.CAFingerprint,
			KeyFingerprint: c.KeyFingerprint,
			Expired:        c.Expired,
		})
	}
	for _, k := range s.Keys {
		out.Keys = append(out.Keys, core.SSHKey{
			Path:          k.Path,
			PublicPath:    k.PublicPath,
			Type:          k.Type,
			Bits:          k.Bits,
			Comment:       k.Comment,
			Fingerprint:   k.Fingerprint,
			PublicKey:     k.PublicKey,
			HasPassphrase: k.HasPassphrase,
			InAgent:       k.InAgent,
			ModTime:       k.ModTime,
		})
	}
	for _, lk := range s.Agent.LoadedKeys {
		out.Agent.LoadedKeys = append(out.Agent.LoadedKeys, core.SSHLoadedKey{
			Fingerprint: lk.Fingerprint,
			Comment:     lk.Comment,
			Type:        lk.Type,
		})
	}
	for _, h := range s.ClientConfig.Hosts {
		out.ClientConfig.Hosts = append(out.ClientConfig.Hosts, core.SSHHostEntry{
			Pattern:               h.Pattern,
			HostName:              h.HostName,
			User:                  h.User,
			Port:                  h.Port,
			IdentityFile:          h.IdentityFile,
			ForwardAgent:          h.ForwardAgent,
			ProxyJump:             h.ProxyJump,
			StrictHostKeyChecking: h.StrictHostKeyChecking,
			Extras:                h.Extras,
		})
	}
	for _, kh := range s.KnownHosts {
		out.KnownHosts = append(out.KnownHosts, core.SSHKnownHost{
			Hostname:    kh.Hostname,
			KeyType:     kh.KeyType,
			Fingerprint: kh.Fingerprint,
			Comment:     kh.Comment,
		})
	}
	for _, ak := range s.AuthorizedKeys {
		out.AuthorizedKeys = append(out.AuthorizedKeys, core.SSHAuthorizedKey{
			Options:     ak.Options,
			KeyType:     ak.KeyType,
			Fingerprint: ak.Fingerprint,
			Comment:     ak.Comment,
		})
	}
	return out
}

// sshInitialFetch fires the snapshot request at startup so F4 has
// data ready when the user first switches to the tab. Called from
// main.go alongside every other F1 service initial fetch.
func sshInitialFetch(nc *nats.Conn) (core.SSHSnapshot, bool) {
	reply, err := nc.Request(bus.SubjectSSHSnapshotCmd, nil, core.TimeoutFast)
	if err != nil {
		return core.SSHSnapshot{}, false
	}
	var snap sshsvc.Snapshot
	if err := json.Unmarshal(reply.Data, &snap); err != nil {
		return core.SSHSnapshot{}, false
	}
	return serviceSnapshotToCore(snap), true
}
