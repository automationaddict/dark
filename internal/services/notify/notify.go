// Package notify is a tiny client for the freedesktop desktop
// notification spec at org.freedesktop.Notifications. We use it to
// surface action results (especially errors) into whatever
// notification daemon the user has running — swaync, mako, dunst,
// gnome-shell, plasma-notify, etc. — without caring which one.
//
// Send is best-effort and intentionally swallows errors: a missing
// or crashed notification daemon should never break the rest of
// dark, and the user already gets the same information inline in
// the TUI. Notifications are an "also" channel, not the only one.
package notify

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	notifyBusName  = "org.freedesktop.Notifications"
	notifyObjPath  = dbus.ObjectPath("/org/freedesktop/Notifications")
	notifyIface    = "org.freedesktop.Notifications"
	notifyMethod   = notifyIface + ".Notify"
	defaultExpire  = int32(-1) // let the notification daemon pick
)

// Urgency is the freedesktop notification urgency hint. Critical
// notifications typically stay on screen until dismissed; low and
// normal time out. Daemons may also style by urgency.
type Urgency byte

const (
	UrgencyLow      Urgency = 0
	UrgencyNormal   Urgency = 1
	UrgencyCritical Urgency = 2
)

// Notifier is a connection to the user's session bus that knows
// how to fire freedesktop notifications. Construct one with New
// and reuse it across the lifetime of the program.
type Notifier struct {
	conn  *dbus.Conn
	obj   dbus.BusObject
	appID string
}

// New connects to the session bus and returns a Notifier ready to
// send. Returns an error only when the session bus itself is
// unreachable, which is unusual on a graphical session — every
// other failure mode (no notification daemon registered, daemon
// crashed, etc.) is handled silently by Send so the caller doesn't
// have to.
func New(appID string) (*Notifier, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("notify: connect session bus: %w", err)
	}
	return &Notifier{
		conn:  conn,
		obj:   conn.Object(notifyBusName, notifyObjPath),
		appID: appID,
	}, nil
}

// Close releases the session bus connection. Safe to call multiple
// times and on a nil receiver.
func (n *Notifier) Close() {
	if n == nil || n.conn == nil {
		return
	}
	_ = n.conn.Close()
	n.conn = nil
}

// Message is one notification. Summary is required (the title in
// most daemons), Body is optional (the longer text), Urgency
// controls how the daemon presents the popup, Icon is a freedesktop
// icon-naming-spec name like "dialog-error" or "network-error" —
// the daemon resolves it from the active icon theme.
type Message struct {
	Summary string
	Body    string
	Urgency Urgency
	Icon    string
}

// Send fires a notification. Errors are swallowed: if the daemon
// isn't running or the call fails for any reason, the user still
// gets the inline TUI message and the notification just doesn't
// pop. We never want a missing notification to break dark.
func (n *Notifier) Send(msg Message) {
	if n == nil || n.conn == nil {
		return
	}
	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(byte(msg.Urgency)),
	}
	var newID uint32
	_ = n.obj.Call(
		notifyMethod,
		0,
		n.appID,       // app_name
		uint32(0),     // replaces_id (0 = new notification)
		msg.Icon,      // app_icon
		msg.Summary,   // summary
		msg.Body,      // body
		[]string{},    // actions (none)
		hints,         // hints
		defaultExpire, // expire_timeout
	).Store(&newID)
}
