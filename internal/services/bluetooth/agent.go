package bluetooth

import (
	"sync"

	"github.com/godbus/dbus/v5"
)

// Agent implements org.bluez.Agent1. Numeric-comparison pairings
// (modern phones, headphones, most keyboards) auto-confirm because the
// user already pressed 'p' to initiate. Legacy PIN pairings are handled
// by stashing a PIN ahead of the pair call — the TUI prompts when it
// sees LegacyPairing=true, the backend calls SetPendingPIN, and the
// agent hands the stashed value back to BlueZ when it asks.
//
// Passkey-entry SSP flows (where BlueZ asks us to display a number for
// the user to type on a keyboard) are not implemented yet; they return
// Rejected so the pair fails cleanly instead of hanging.
type Agent struct {
	backend *bluezBackend
	mu      sync.Mutex
	pending map[string]string // device path → PIN
}

func newAgent(backend *bluezBackend) *Agent {
	return &Agent{backend: backend, pending: map[string]string{}}
}

// SetPendingPIN stashes a PIN for a device about to be paired. The
// backend calls this before issuing Device1.Pair so the agent callback
// can return the value without round-tripping back to the TUI.
func (a *Agent) SetPendingPIN(device, pin string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pending[device] = pin
}

// ClearPendingPIN removes the stashed PIN after the pair attempt finishes.
func (a *Agent) ClearPendingPIN(device string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.pending, device)
}

func (a *Agent) lookupPIN(device dbus.ObjectPath) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	pin, ok := a.pending[string(device)]
	return pin, ok
}

const agentErrRejected = "org.bluez.Error.Rejected"

// Release is called by BlueZ when it unregisters us or shuts down.
func (a *Agent) Release() *dbus.Error { return nil }

// RequestPinCode is used by legacy devices. We return the PIN stashed
// via SetPendingPIN if the user supplied one; otherwise we refuse so
// the pair call fails quickly instead of hanging.
func (a *Agent) RequestPinCode(device dbus.ObjectPath) (string, *dbus.Error) {
	if pin, ok := a.lookupPIN(device); ok {
		return pin, nil
	}
	return "", dbus.NewError(agentErrRejected, nil)
}

// DisplayPinCode is informational — the remote device asked us to show
// a PIN to the user. We have no UI for this yet; accept silently.
func (a *Agent) DisplayPinCode(device dbus.ObjectPath, pincode string) *dbus.Error {
	return nil
}

// RequestPasskey is the numeric equivalent of RequestPinCode. Same
// policy: no dialog yet, so refuse.
func (a *Agent) RequestPasskey(device dbus.ObjectPath) (uint32, *dbus.Error) {
	return 0, dbus.NewError(agentErrRejected, nil)
}

// DisplayPasskey is informational: show this number so the user can
// compare it with the number on the remote device.
func (a *Agent) DisplayPasskey(device dbus.ObjectPath, passkey uint32, entered uint16) *dbus.Error {
	return nil
}

// RequestConfirmation is the numeric-comparison flow: "is the number
// 123456 shown on both devices the same?". Since the user explicitly
// pressed 'p' to initiate the pair, we treat that as consent and
// auto-confirm.
func (a *Agent) RequestConfirmation(device dbus.ObjectPath, passkey uint32) *dbus.Error {
	return nil
}

// RequestAuthorization is the "just works" confirmation for devices
// with no input/output. Same auto-agree policy.
func (a *Agent) RequestAuthorization(device dbus.ObjectPath) *dbus.Error {
	return nil
}

// AuthorizeService gates a service UUID on an already-paired device.
// Trusted devices bypass this entirely. We auto-authorize so
// user-initiated connects don't hang waiting for consent.
func (a *Agent) AuthorizeService(device dbus.ObjectPath, uuid string) *dbus.Error {
	return nil
}

// Cancel is called when BlueZ aborts a pairing request before it
// completes (e.g. user pressed cancel on the other device).
func (a *Agent) Cancel() *dbus.Error { return nil }
