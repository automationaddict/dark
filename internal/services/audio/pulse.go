package audio

import (
	"fmt"
	"net"
	"os"
	"path"
	"sync"

	"github.com/jfreymuth/pulse/proto"
)

// pulseBackend talks to the host's audio server over the PulseAudio
// native protocol. PipeWire's pipewire-pulse shim implements the same
// protocol, so this backend works transparently on PipeWire systems
// (which is Omarchy's default).
//
// Beyond the request/reply control surface, the backend also runs two
// streaming pipelines off the client's Callback hook:
//
//   - Subscription events (sink/source/card added/removed/changed)
//     are coalesced into a buffered channel the daemon drains so it
//     can republish snapshots reactively instead of polling.
//
//   - Peak-detect record streams are opened against every sink's
//     monitor source and every source's index, producing one float32
//     per chunk that the callback decodes into a per-device level map
//     for the daemon's level meter publisher.
type pulseBackend struct {
	client *proto.Client
	conn   net.Conn
	mu     sync.Mutex

	eventCh chan struct{} // capacity 1, signals "republish snapshot"

	metersMu     sync.Mutex
	meterStreams map[uint32]meterEntry    // record stream index → device key
	sinkLevels   map[uint32][2]float32    // sink index → [left, right] peak
	sourceLevels map[uint32][2]float32    // source index → [left, right] peak
}

// meterEntry maps a record-stream index back to the device kind and
// PulseAudio device index it's measuring. We need this because the
// callback only knows the stream index — it has to translate that to
// "this is the level for sink 5" before updating the levels map.
type meterEntry struct {
	kind        meterKind
	deviceIndex uint32
}

type meterKind int

const (
	meterKindSink meterKind = iota
	meterKindSource
)

func newPulseBackend() (*pulseBackend, error) {
	client, conn, err := proto.Connect("")
	if err != nil {
		return nil, fmt.Errorf("pulse connect: %w", err)
	}
	// Register a client name so our process shows up nicely in
	// pactl list clients / PipeWire's graph view.
	props := proto.PropList{
		"application.name":           proto.PropListString("darkd"),
		"application.process.id":     proto.PropListString(fmt.Sprintf("%d", os.Getpid())),
		"application.process.binary": proto.PropListString(path.Base(os.Args[0])),
	}
	if err := client.Request(&proto.SetClientName{Props: props}, &proto.SetClientNameReply{}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pulse set client name: %w", err)
	}

	b := &pulseBackend{
		client:       client,
		conn:         conn,
		eventCh:      make(chan struct{}, 1),
		meterStreams: map[uint32]meterEntry{},
		sinkLevels:   map[uint32][2]float32{},
		sourceLevels: map[uint32][2]float32{},
	}

	// Install the callback BEFORE subscribing so we don't miss events
	// fired between the Subscribe call and the field assignment.
	client.Callback = b.handleProtoMessage

	if err := client.Request(&proto.Subscribe{Mask: proto.SubscriptionMaskAll}, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pulse subscribe: %w", err)
	}

	return b, nil
}

func (b *pulseBackend) Name() string { return BackendPulseProto }

func (b *pulseBackend) Close() {
	if b.client != nil {
		b.closeAllMeterStreams()
	}
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
	b.client = nil
}
