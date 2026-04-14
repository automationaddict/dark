package main

import (
	"log/slog"

	"github.com/nats-io/nats.go"
)

// respond sends data as a reply to the given NATS message. A failed
// reply means either the client timed out or the bus link dropped —
// both transient, so we log at Warn level and move on.
func respond(m *nats.Msg, data []byte) {
	if err := m.Respond(data); err != nil {
		slog.Warn("nats respond failed", "reply", m.Reply, "error", err)
	}
}

// publish fires data on subject. Failures are logged at Warn level;
// snapshot broadcasts are idempotent so dropping one is acceptable.
func publish(nc *nats.Conn, subject string, data []byte) {
	if err := nc.Publish(subject, data); err != nil {
		slog.Warn("nats publish failed", "subject", subject, "error", err)
	}
}
