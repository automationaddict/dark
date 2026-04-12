package bluetooth

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	bluezAgentPath       = dbus.ObjectPath("/dark/bluetooth/agent")
	bluezAgentIface      = "org.bluez.Agent1"
	bluezAgentManager    = "org.bluez.AgentManager1"
	bluezAgentMgrPath    = dbus.ObjectPath("/org/bluez")
	bluezAgentCapability = "KeyboardDisplay"
)

// StartAgent exports our Agent on the system bus, registers it with
// BlueZ's AgentManager, and asks BlueZ to use it as the default agent
// for the session. After this call BlueZ delegates pairing prompts to
// us for any device we (or anything else) try to pair.
func (b *bluezBackend) StartAgent() error {
	if b.conn == nil {
		return fmt.Errorf("bluez: no D-Bus connection")
	}
	b.agent = newAgent(b)
	if err := b.conn.Export(b.agent, bluezAgentPath, bluezAgentIface); err != nil {
		b.agent = nil
		return fmt.Errorf("export agent: %w", err)
	}
	mgr := b.conn.Object(bluezBusName, bluezAgentMgrPath)
	if err := mgr.Call(bluezAgentManager+".RegisterAgent", 0, bluezAgentPath, bluezAgentCapability).Err; err != nil {
		_ = b.conn.Export(nil, bluezAgentPath, bluezAgentIface)
		b.agent = nil
		return fmt.Errorf("register agent: %w", err)
	}
	// Requesting default agent is optional but ensures BlueZ routes
	// prompts to us even when another agent (like bluetoothctl) is
	// also registered. Failure is non-fatal.
	_ = mgr.Call(bluezAgentManager+".RequestDefaultAgent", 0, bluezAgentPath).Err
	b.agentActive = true
	return nil
}

// StopAgent unregisters the agent with BlueZ and removes the D-Bus
// export. Errors from BlueZ are swallowed so shutdown doesn't stall.
func (b *bluezBackend) StopAgent() error {
	if !b.agentActive || b.conn == nil {
		return nil
	}
	mgr := b.conn.Object(bluezBusName, bluezAgentMgrPath)
	_ = mgr.Call(bluezAgentManager+".UnregisterAgent", 0, bluezAgentPath).Err
	_ = b.conn.Export(nil, bluezAgentPath, bluezAgentIface)
	b.agent = nil
	b.agentActive = false
	return nil
}
