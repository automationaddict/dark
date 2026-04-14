package audio

import (
	"encoding/binary"
	"math"
	"sync"
	"testing"

	"github.com/jfreymuth/pulse/proto"
)

func TestAvgVolume(t *testing.T) {
	tests := []struct {
		name string
		cv   proto.ChannelVolumes
		want uint32
	}{
		{"empty", nil, 0},
		{"single", proto.ChannelVolumes{uint32(proto.VolumeNorm)}, uint32(proto.VolumeNorm)},
		{"stereo equal", proto.ChannelVolumes{1000, 1000}, 1000},
		{"stereo unequal", proto.ChannelVolumes{1000, 3000}, 2000},
		{"muted", proto.ChannelVolumes{0, 0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := avgVolume(tt.cv); got != tt.want {
				t.Errorf("avgVolume(%v) = %d, want %d", tt.cv, got, tt.want)
			}
		})
	}
}

func TestVolumeToPercent(t *testing.T) {
	tests := []struct {
		raw  uint32
		want int
	}{
		{0, 0},
		{uint32(proto.VolumeNorm), 100},
		{uint32(proto.VolumeNorm) / 2, 50},
		{uint32(proto.VolumeNorm) * 3 / 2, 150}, // over-amp
	}
	for _, tt := range tests {
		if got := volumeToPercent(tt.raw); got != tt.want {
			t.Errorf("volumeToPercent(%d) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}

func TestPercentToVolumes(t *testing.T) {
	tests := []struct {
		pct      int
		channels int
		wantLen  int
		wantVal  uint32
	}{
		{100, 2, 2, uint32(proto.VolumeNorm)},
		{0, 2, 2, 0},
		{50, 1, 1, uint32(proto.VolumeNorm) / 2},
		{-10, 2, 2, 0},                                  // clamped to 0
		{200, 2, 2, uint32(uint64(150) * uint64(proto.VolumeNorm) / 100)}, // clamped to 150
		{75, 0, 2, uint32(uint64(75) * uint64(proto.VolumeNorm) / 100)},   // 0 channels → default 2
	}
	for _, tt := range tests {
		cv := percentToVolumes(tt.pct, tt.channels)
		if len(cv) != tt.wantLen {
			t.Errorf("percentToVolumes(%d, %d) len = %d, want %d", tt.pct, tt.channels, len(cv), tt.wantLen)
			continue
		}
		if cv[0] != tt.wantVal {
			t.Errorf("percentToVolumes(%d, %d)[0] = %d, want %d", tt.pct, tt.channels, cv[0], tt.wantVal)
		}
	}
}

func TestSinkStateString(t *testing.T) {
	tests := []struct {
		state uint32
		want  string
	}{
		{0, "running"},
		{1, "idle"},
		{2, "suspended"},
		{3, "invalid"},
		{99, ""},
	}
	for _, tt := range tests {
		if got := sinkStateString(tt.state); got != tt.want {
			t.Errorf("sinkStateString(%d) = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// TestMeterConcurrentAccess runs Levels() and handleProtoMessage()
// concurrently under -race. The previous fix introduced metersMu; this
// is a regression guard so a future edit that drops the lock is caught
// immediately.
func TestMeterConcurrentAccess(t *testing.T) {
	b := &pulseBackend{
		meterStreams: map[uint32]meterEntry{
			1: {kind: meterKindSink, deviceIndex: 10},
			2: {kind: meterKindSource, deviceIndex: 20},
		},
		sinkLevels:   map[uint32][2]float32{10: {0.1, 0.2}},
		sourceLevels: map[uint32][2]float32{20: {0.3, 0.4}},
	}

	// Build a valid stereo peak-detect DataPacket payload (one pair).
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], math.Float32bits(0.5))
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(0.6))

	const iters = 500
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			_ = b.Levels()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			b.handleProtoMessage(&proto.DataPacket{StreamIndex: 1, Data: buf})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			b.handleProtoMessage(&proto.DataPacket{StreamIndex: 2, Data: buf})
		}
	}()
	wg.Wait()
}

func TestVolumeRoundTrip(t *testing.T) {
	for pct := 0; pct <= 150; pct += 5 {
		cv := percentToVolumes(pct, 2)
		got := volumeToPercent(avgVolume(cv))
		// Integer division truncation means ±1% loss is normal.
		// The UI rounds to whole percentages so this is acceptable.
		diff := pct - got
		if diff < 0 {
			diff = -diff
		}
		if diff > 1 {
			t.Errorf("round trip %d%% → %d%% (diff %d)", pct, got, diff)
		}
	}
}
