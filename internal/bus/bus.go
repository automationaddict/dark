package bus

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// DiscoveryPath returns the filesystem location where darkd writes its NATS
// URL on startup. Clients read this file to find the running daemon. Lives
// under $XDG_RUNTIME_DIR which is per-user and wiped on logout, so stale
// files don't linger across reboots.
func DiscoveryPath() string {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = filepath.Join(os.TempDir(), fmt.Sprintf("dark-%d", os.Getuid()))
	}
	return filepath.Join(base, "dark", "daemon.url")
}

// StartServer stands up an in-process NATS server bound to an ephemeral
// loopback port and writes its client URL to the discovery file. Returns
// the server handle and a connected client. Callers must Shutdown the
// server and Close the client on exit.
func StartServer() (*server.Server, *nats.Conn, error) {
	opts := &server.Options{
		Host:           "127.0.0.1",
		Port:           -1, // ephemeral
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 4096,
	}
	srv, err := server.NewServer(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("nats: new server: %w", err)
	}
	go srv.Start()

	if !srv.ReadyForConnections(3 * time.Second) {
		srv.Shutdown()
		return nil, nil, fmt.Errorf("nats: server not ready after 3s")
	}

	if err := writeDiscoveryFile(srv.ClientURL()); err != nil {
		srv.Shutdown()
		return nil, nil, fmt.Errorf("discovery file: %w", err)
	}

	nc, err := nats.Connect(srv.ClientURL(),
		nats.Name("darkd"),
		nats.InProcessServer(srv),
	)
	if err != nil {
		srv.Shutdown()
		return nil, nil, fmt.Errorf("nats: in-process connect: %w", err)
	}
	return srv, nc, nil
}

// ClientCallbacks lets callers observe connection lifecycle events. Any field
// may be nil. Handlers fire on the NATS client's internal goroutine, so they
// must be safe to call from there (typically just push a message into a
// channel or call tea.Program.Send).
type ClientCallbacks struct {
	OnDisconnect func(error)
	OnReconnect  func()
	OnClosed     func()
}

// ConnectClient reads the discovery file and opens a NATS connection to
// the running daemon. Returns a helpful error if the daemon isn't running.
// Pass a non-nil callbacks struct to be notified of connection lifecycle
// events while the client is running.
func ConnectClient(name string, cb *ClientCallbacks) (*nats.Conn, error) {
	url, err := os.ReadFile(DiscoveryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("darkd is not running — start it with `task dev:up`")
		}
		return nil, fmt.Errorf("read discovery file: %w", err)
	}
	opts := []nats.Option{
		nats.Name(name),
		nats.RetryOnFailedConnect(false),
		nats.MaxReconnects(-1), // keep retrying forever
		nats.ReconnectWait(500 * time.Millisecond),
		nats.Timeout(2 * time.Second),
	}
	if cb != nil {
		if cb.OnDisconnect != nil {
			opts = append(opts, nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
				cb.OnDisconnect(err)
			}))
		}
		if cb.OnReconnect != nil {
			opts = append(opts, nats.ReconnectHandler(func(_ *nats.Conn) {
				cb.OnReconnect()
			}))
		}
		if cb.OnClosed != nil {
			opts = append(opts, nats.ClosedHandler(func(_ *nats.Conn) {
				cb.OnClosed()
			}))
		}
	}
	nc, err := nats.Connect(string(url), opts...)
	if err != nil {
		return nil, fmt.Errorf("connect to darkd: %w", err)
	}
	return nc, nil
}

// CleanupDiscoveryFile removes the stale URL file when the daemon shuts
// down so clients get a clear "not running" error instead of trying to
// connect to a dead port.
func CleanupDiscoveryFile() {
	_ = os.Remove(DiscoveryPath())
}

func writeDiscoveryFile(url string) error {
	path := DiscoveryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(url), 0600)
}
