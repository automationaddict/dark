package wifi

import (
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
		_ = b.conn.Export(nil, iwdAgentPath, "net.connman.iwd.Agent")
		b.agent = nil
		return fmt.Errorf("register agent: %w", err)
	}
	b.agentActive = true
	return nil
}

// StopAgent unregisters the agent with iwd and removes the D-Bus
// export. Errors from iwd are swallowed so shutdown doesn't stall on
// a broken bus.
func (b *iwdBackend) StopAgent() error {
	if !b.agentActive || b.conn == nil {
		return nil
	}
	manager := b.conn.Object(iwdBusName, "/net/connman/iwd")
	_ = manager.Call("net.connman.iwd.AgentManager.UnregisterAgent", 0, iwdAgentPath).Err
	_ = b.conn.Export(nil, iwdAgentPath, "net.connman.iwd.Agent")
	b.agent = nil
	b.agentActive = false
	return nil
}
