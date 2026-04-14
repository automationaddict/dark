package wifi

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const iwdAgentPath = dbus.ObjectPath("/dark/agent")

// StartAgent exports our Agent on the system bus and registers it with
// iwd's AgentManager. After this call, iwd delegates credential prompts
// to us for any network we try to connect.
func (b *iwdBackend) StartAgent() error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	b.agent = newAgent(b)
	if err := b.conn.Export(b.agent, iwdAgentPath, "net.connman.iwd.Agent"); err != nil {
		b.agent = nil
		return fmt.Errorf("export agent: %w", err)
	}
	manager := b.conn.Object(iwdBusName, "/net/connman/iwd")
	if err := manager.Call("net.connman.iwd.AgentManager.RegisterAgent", 0, iwdAgentPath).Err; err != nil {
		if unexportErr := b.conn.Export(nil, iwdAgentPath, "net.connman.iwd.Agent"); unexportErr != nil {
			err = errors.Join(err, fmt.Errorf("unexport after failed register: %w", unexportErr))
		}
		b.agent = nil
		return fmt.Errorf("register agent: %w", err)
	}
	b.agentActive = true
	return nil
}

// StopAgent unregisters the agent with iwd and removes the D-Bus
// export. Both steps always run so a broken bus doesn't leave the
// export dangling; any errors are joined and returned for the caller
// to log without stalling shutdown.
func (b *iwdBackend) StopAgent() error {
	if !b.agentActive || b.conn == nil {
		return nil
	}
	var errs []error
	manager := b.conn.Object(iwdBusName, "/net/connman/iwd")
	if err := manager.Call("net.connman.iwd.AgentManager.UnregisterAgent", 0, iwdAgentPath).Err; err != nil {
		errs = append(errs, fmt.Errorf("unregister agent: %w", err))
	}
	if err := b.conn.Export(nil, iwdAgentPath, "net.connman.iwd.Agent"); err != nil {
		errs = append(errs, fmt.Errorf("unexport agent: %w", err))
	}
	b.agent = nil
	b.agentActive = false
	return errors.Join(errs...)
}
