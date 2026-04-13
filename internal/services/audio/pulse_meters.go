package audio

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/jfreymuth/pulse/proto"
)

// peakDetectRate is how often per second the server sends peak chunks
// for our meter streams. 25 Hz is what pavucontrol uses — fast enough
// that the meter feels live, slow enough that the data volume across
// six or eight devices stays trivial (~1.2 KB/sec total even with
// stereo capture).
const peakDetectRate = 25

// peakDetectChannels is the channel count we ask PulseAudio for on
// every meter stream. Stereo so the TUI can show a center-anchored
// L/R meter; the server upmixes mono sources by duplicating the
// channel, which makes mono mics look symmetric — accurate.
const peakDetectChannels = 2

// peakDetectMaxLength caps the buffer the server allocates for our
// meter stream. One stereo peak pair = 2 channels × 4 bytes = 8 bytes.
const peakDetectMaxLength = 8

// handleProtoMessage is the *proto.Client.Callback. It runs on the
// protocol read goroutine, so it must be cheap and must not block on
// anything that could re-enter the proto layer (which would deadlock).
//
// SubscribeEvent → coalesce into eventCh (non-blocking — drop if full,
// the next event will trigger another snapshot anyway).
//
// DataPacket → look up the stream index, decode the stereo float32
// peak pair, and store it in the levels map. The data packet may
// carry multiple stereo pairs; we keep the loudest reading per channel
// so the meter doesn't undersell brief transients.
func (b *pulseBackend) handleProtoMessage(msg interface{}) {
	switch m := msg.(type) {
	case *proto.SubscribeEvent:
		select {
		case b.eventCh <- struct{}{}:
		default:
		}
	case *proto.DataPacket:
		b.metersMu.Lock()
		entry, ok := b.meterStreams[m.StreamIndex]
		b.metersMu.Unlock()
		if !ok {
			return
		}
		peaks := decodeStereoPeakBytes(m.Data)
		b.metersMu.Lock()
		switch entry.kind {
		case meterKindSink:
			b.sinkLevels[entry.deviceIndex] = peaks
		case meterKindSource:
			b.sourceLevels[entry.deviceIndex] = peaks
		}
		b.metersMu.Unlock()
	}
}

// decodeStereoPeakBytes interprets a peak-detect data packet as a
// sequence of interleaved little-endian float32 stereo pairs and
// returns the per-channel max. Mono streams that the server has
// upmixed for us will have left == right; that's fine, the meter
// will look symmetric.
func decodeStereoPeakBytes(data []byte) [2]float32 {
	var out [2]float32
	if len(data) < 8 {
		// Defensive: a misconfigured stream might send single-channel
		// chunks. Decode what we have, mirror to both channels.
		if len(data) >= 4 {
			v := absFloat32LE(data[0:4])
			out[0] = v
			out[1] = v
		}
		return out
	}
	for i := 0; i+8 <= len(data); i += 8 {
		l := absFloat32LE(data[i : i+4])
		r := absFloat32LE(data[i+4 : i+8])
		if l > out[0] {
			out[0] = l
		}
		if r > out[1] {
			out[1] = r
		}
	}
	return out
}

func absFloat32LE(b []byte) float32 {
	bits := binary.LittleEndian.Uint32(b)
	v := math.Float32frombits(bits)
	if v < 0 {
		v = -v
	}
	return v
}

// Events returns the buffered event channel for the daemon to drain.
func (b *pulseBackend) Events() <-chan struct{} { return b.eventCh }

// Levels returns a copy of the current sink and source peak readings.
// Mutex-protected so the publisher can read without racing the
// callback that's writing.
func (b *pulseBackend) Levels() Levels {
	b.metersMu.Lock()
	defer b.metersMu.Unlock()
	out := Levels{
		Sinks:   make(map[uint32][2]float32, len(b.sinkLevels)),
		Sources: make(map[uint32][2]float32, len(b.sourceLevels)),
	}
	for k, v := range b.sinkLevels {
		out.Sinks[k] = v
	}
	for k, v := range b.sourceLevels {
		out.Sources[k] = v
	}
	return out
}

// reconcileMeters opens or closes peak-detect record streams so that
// every sink and source in the snapshot has a meter, and any meter
// whose underlying device has gone away is torn down. Called from
// Snapshot after building the device lists.
//
// Lock ordering note: this function takes metersMu and may also call
// out to the proto layer (CreateRecordStream / DeleteRecordStream).
// proto.Client.Request is internally serialized via its own write
// mutex, so it's safe to issue from here without taking b.mu.
// meterTarget carries everything openMeterStream needs to pin a
// peak-detect record stream to the right PipeWire/PulseAudio source.
type meterTarget struct {
	sourceIndex uint32
	sourceName  string // used to defeat module-stream-restore
}

func (b *pulseBackend) reconcileMeters(snap Snapshot) {
	wantedSinks := map[uint32]meterTarget{} // sink index → monitor target
	for _, s := range snap.Sinks {
		if s.MonitorIndex == proto.Undefined {
			continue
		}
		wantedSinks[s.Index] = meterTarget{
			sourceIndex: s.MonitorIndex,
			sourceName:  s.MonitorName,
		}
	}
	wantedSources := map[uint32]meterTarget{}
	for _, s := range snap.Sources {
		wantedSources[s.Index] = meterTarget{
			sourceIndex: s.Index,
			sourceName:  s.Name,
		}
	}

	b.metersMu.Lock()
	haveSinks := map[uint32]uint32{}
	haveSources := map[uint32]uint32{}
	for streamIdx, entry := range b.meterStreams {
		switch entry.kind {
		case meterKindSink:
			haveSinks[entry.deviceIndex] = streamIdx
		case meterKindSource:
			haveSources[entry.deviceIndex] = streamIdx
		}
	}
	var toDelete []uint32
	for streamIdx, entry := range b.meterStreams {
		switch entry.kind {
		case meterKindSink:
			if _, ok := wantedSinks[entry.deviceIndex]; !ok {
				toDelete = append(toDelete, streamIdx)
				delete(b.sinkLevels, entry.deviceIndex)
			}
		case meterKindSource:
			if _, ok := wantedSources[entry.deviceIndex]; !ok {
				toDelete = append(toDelete, streamIdx)
				delete(b.sourceLevels, entry.deviceIndex)
			}
		}
	}
	for _, idx := range toDelete {
		delete(b.meterStreams, idx)
	}
	createSinks := map[uint32]meterTarget{}
	for sinkIdx, target := range wantedSinks {
		if _, ok := haveSinks[sinkIdx]; !ok {
			createSinks[sinkIdx] = target
		}
	}
	createSources := map[uint32]meterTarget{}
	for sourceIdx, target := range wantedSources {
		if _, ok := haveSources[sourceIdx]; !ok {
			createSources[sourceIdx] = target
		}
	}
	b.metersMu.Unlock()

	for _, idx := range toDelete {
		_ = b.client.Request(&proto.DeleteRecordStream{StreamIndex: idx}, nil)
	}

	for sinkIdx, target := range createSinks {
		streamIdx, ok := b.openMeterStream(target)
		if !ok {
			continue
		}
		b.metersMu.Lock()
		b.meterStreams[streamIdx] = meterEntry{kind: meterKindSink, deviceIndex: sinkIdx}
		b.metersMu.Unlock()
	}
	for sourceIdx, target := range createSources {
		streamIdx, ok := b.openMeterStream(target)
		if !ok {
			continue
		}
		b.metersMu.Lock()
		b.meterStreams[streamIdx] = meterEntry{kind: meterKindSource, deviceIndex: sourceIdx}
		b.metersMu.Unlock()
	}
}

// openMeterStream opens a peak-detect record stream against the given
// PipeWire/PulseAudio source. We target the source by name rather than
// index because PipeWire's module-stream-restore remembers per-app
// routing by application.name and will redirect index-based requests to
// the last-used source. Name-based targeting bypasses this.
func (b *pulseBackend) openMeterStream(target meterTarget) (uint32, bool) {
	req := &proto.CreateRecordStream{
		SampleSpec: proto.SampleSpec{
			Format:   proto.FormatFloat32LE,
			Channels: peakDetectChannels,
			Rate:     peakDetectRate,
		},
		ChannelMap:             proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight},
		SourceIndex:            proto.Undefined,
		SourceName:             target.sourceName,
		BufferMaxLength:        peakDetectMaxLength,
		BufferFragSize:         peakDetectMaxLength,
		Corked:                 false,
		NoMove:                 true,
		PeakDetect:             true,
		AdjustLatency:          true,
		DontInhibitAutoSuspend: true,
		Properties: proto.PropList{
			"media.name": proto.PropListString(fmt.Sprintf("dark-peak-%d", target.sourceIndex)),
		},
	}
	var reply proto.CreateRecordStreamReply
	if err := b.client.Request(req, &reply); err != nil {
		return 0, false
	}
	return reply.StreamIndex, true
}

// closeAllMeterStreams tears down every open meter on shutdown.
func (b *pulseBackend) closeAllMeterStreams() {
	b.metersMu.Lock()
	indices := make([]uint32, 0, len(b.meterStreams))
	for idx := range b.meterStreams {
		indices = append(indices, idx)
	}
	b.meterStreams = map[uint32]meterEntry{}
	b.sinkLevels = map[uint32][2]float32{}
	b.sourceLevels = map[uint32][2]float32{}
	b.metersMu.Unlock()

	for _, idx := range indices {
		_ = b.client.Request(&proto.DeleteRecordStream{StreamIndex: idx}, nil)
	}
}
