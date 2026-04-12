package wifi

import (
	"sync"

	"github.com/godbus/dbus/v5"
)

// Agent implements net.connman.iwd.Agent and answers iwd's credential
// prompts using passphrases the iwd backend stashed ahead of a connect.
// Only WPA-PSK is handled; enterprise methods return a Canceled error
// which tells iwd to abort that specific prompt without trying to
// escalate.
type Agent struct {
	backend *iwdBackend
	mu      sync.Mutex
	pending map[string]string // SSID → passphrase
}

func newAgent(backend *iwdBackend) *Agent {
	return &Agent{backend: backend, pending: map[string]string{}}
}

// SetPending stashes a passphrase under an SSID. The backend calls
// this before issuing Network.Connect on a network that needs credentials.
func (a *Agent) SetPending(ssid, passphrase string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pending[ssid] = passphrase
}

// ClearPending removes a stashed passphrase after the connect attempt
// finishes (success or failure).
func (a *Agent) ClearPending(ssid string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.pending, ssid)
}

// Release is called by iwd when it unregisters us or shuts down.
func (a *Agent) Release() *dbus.Error {
	return nil
}

// RequestPassphrase resolves the network object path to its SSID and
// returns the pending passphrase.
func (a *Agent) RequestPassphrase(network dbus.ObjectPath) (string, *dbus.Error) {
	ssid, err := a.backend.resolveNetworkSSID(network)
	if err != nil {
		return "", dbus.NewError("net.connman.iwd.Agent.Error.Canceled", nil)
	}
	a.mu.Lock()
	passphrase, ok := a.pending[ssid]
	a.mu.Unlock()
	if !ok {
		return "", dbus.NewError("net.connman.iwd.Agent.Error.Canceled", nil)
	}
	return passphrase, nil
}

func (a *Agent) RequestPrivateKeyPassphrase(network dbus.ObjectPath) (string, *dbus.Error) {
	return "", dbus.NewError("net.connman.iwd.Agent.Error.Canceled", nil)
}

func (a *Agent) RequestUserNameAndPassword(network dbus.ObjectPath) (string, string, *dbus.Error) {
	return "", "", dbus.NewError("net.connman.iwd.Agent.Error.Canceled", nil)
}

func (a *Agent) RequestUserPassword(network dbus.ObjectPath, user string) (string, *dbus.Error) {
	return "", dbus.NewError("net.connman.iwd.Agent.Error.Canceled", nil)
}

func (a *Agent) Cancel(reason string) *dbus.Error {
	return nil
}
