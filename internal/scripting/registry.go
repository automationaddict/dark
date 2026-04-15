package scripting

import "sync"

// RegistryEntry describes one symbol the scripting layer exposes to
// Lua user scripts. Kind is either "function" (a host-provided Go
// callable injected as a Lua global) or "event" (a hook point that
// user scripts can register for). The UI enumerates these to build
// the F5 Scripting > Lua reference tab.
// RegistryField mirrors bus.CommandField on the registry side so
// the scripting package doesn't pull in the bus import. Fields
// describe a single payload parameter for action entries.
type RegistryField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Desc     string `json:"desc"`
}

type RegistryEntry struct {
	Kind    string          `json:"kind"`
	Name    string          `json:"name"`
	Args    string          `json:"args"`
	Summary string          `json:"summary"`
	Subject string          `json:"subject,omitempty"`
	Fields  []RegistryField `json:"fields,omitempty"`
}

// Registry is a small thread-safe catalog of exposed Lua symbols. The
// Engine owns one and populates it while registering host functions
// and seeded event hooks. The UI reads a snapshot via Entries().
type Registry struct {
	mu      sync.RWMutex
	entries []RegistryEntry
}

// RegisterFunction records a Go→Lua function binding.
func (r *Registry) RegisterFunction(name, args, summary string) {
	r.add(RegistryEntry{Kind: "function", Name: name, Args: args, Summary: summary})
}

// RegisterEvent records a hook point user scripts can register for.
func (r *Registry) RegisterEvent(name, args, summary string) {
	r.add(RegistryEntry{Kind: "event", Name: name, Args: args, Summary: summary})
}

// RegisterAction records a first-class Lua wrapper around a bus
// command subject. The Subject field keeps the underlying
// `dark.cmd.*` subject so the F5 reference doc can show both the
// Lua call form and the raw NATS round-trip. Fields describe the
// documented payload schema so docs can render real parameter
// tables instead of an empty example.
func (r *Registry) RegisterAction(name, subject, summary string, fields []RegistryField) {
	r.add(RegistryEntry{
		Kind:    "action",
		Name:    name,
		Subject: subject,
		Summary: summary,
		Fields:  fields,
	})
}

// Entries returns a copy of the current registry contents.
func (r *Registry) Entries() []RegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RegistryEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

func (r *Registry) add(e RegistryEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.entries {
		if existing.Kind == e.Kind && existing.Name == e.Name {
			return
		}
	}
	r.entries = append(r.entries, e)
}
