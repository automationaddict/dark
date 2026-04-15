package tui

import "github.com/automationaddict/dark/internal/core"

// EventsActions exposes a publish hook the Model uses to fire
// client-side UI events (tab switches, section changes, lifecycle
// transitions) out onto the bus so darkd can dispatch them into
// the Lua engine. Keeps the TUI package free of any nats import —
// cmd/dark owns the nats.Conn and passes a closure in.
//
// Publish is fire-and-forget; payload may be nil. Event names
// should follow the on_<snake_case> convention the rest of the
// scripting surface uses so user scripts can register with
// `dark.on("on_f1", ...)` consistently.
type EventsActions struct {
	Publish func(event string, payload map[string]interface{})
}

// publishEvent is a nil-safe shim so callers can fire events
// without checking whether the actions struct was wired in tests
// or other constructors that skip the events surface.
func (m *Model) publishEvent(event string, payload map[string]interface{}) {
	if m.events.Publish == nil {
		return
	}
	m.events.Publish(event, payload)
}

// switchTab is the single chokepoint for function-key tab changes.
// It flips the active tab, fires the tab-specific event, and fires a
// generic on_tab_change with the destination identifier so scripts
// that don't care which tab want a single hook.
func (m *Model) switchTab(id core.TabID, event string) {
	m.state.SelectTab(id)
	m.publishEvent(event, nil)
	m.publishEvent("on_tab_change", map[string]interface{}{"tab": event})
}
