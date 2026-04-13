package audio

import "github.com/jfreymuth/pulse/proto"

// avgVolume is the per-channel average of a ChannelVolumes vector.
// Mirrors libpulse's pa_cvolume_avg.
func avgVolume(cv proto.ChannelVolumes) uint32 {
	if len(cv) == 0 {
		return 0
	}
	var sum uint64
	for _, v := range cv {
		sum += uint64(v)
	}
	return uint32(sum / uint64(len(cv)))
}

// volumeToPercent converts a raw PulseAudio volume to a 0-100+ integer
// percentage using proto.VolumeNorm as the 100% reference.
func volumeToPercent(raw uint32) int {
	return int(uint64(raw) * 100 / uint64(proto.VolumeNorm))
}

// percentToVolumes builds a ChannelVolumes slice with the given
// percentage applied uniformly across `channels` channels. Clamps
// to the [0, 150] range PulseAudio considers "safe" — values above
// 100% enter software over-amplification territory which can clip.
func percentToVolumes(pct int, channels int) proto.ChannelVolumes {
	if channels <= 0 {
		channels = 2
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 150 {
		pct = 150
	}
	raw := uint32(uint64(pct) * uint64(proto.VolumeNorm) / 100)
	out := make(proto.ChannelVolumes, channels)
	for i := range out {
		out[i] = raw
	}
	return out
}

// computeBalance derives a -100..+100 balance from a stereo ChannelVolumes.
// -100 = full left, 0 = center, +100 = full right. Mono returns 0.
func computeBalance(cv proto.ChannelVolumes) int {
	if len(cv) < 2 {
		return 0
	}
	left := float64(cv[0])
	right := float64(cv[1])
	sum := left + right
	if sum == 0 {
		return 0
	}
	// balance = (right - left) / max(left, right) * 100
	max := left
	if right > max {
		max = right
	}
	return int((right - left) / max * 100)
}

// balanceToVolumes builds a ChannelVolumes with the given overall volume
// percentage and balance (-100..+100). The louder channel gets the full
// volume; the quieter channel is scaled down proportionally.
func balanceToVolumes(pct, balance, channels int) proto.ChannelVolumes {
	if channels <= 0 {
		channels = 2
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 150 {
		pct = 150
	}
	raw := uint32(uint64(pct) * uint64(proto.VolumeNorm) / 100)
	out := make(proto.ChannelVolumes, channels)

	if channels < 2 || balance == 0 {
		for i := range out {
			out[i] = raw
		}
		return out
	}

	var leftScale, rightScale float64
	if balance < 0 {
		leftScale = 1.0
		rightScale = 1.0 + float64(balance)/100.0
	} else {
		leftScale = 1.0 - float64(balance)/100.0
		rightScale = 1.0
	}

	out[0] = uint32(float64(raw) * leftScale)
	out[1] = uint32(float64(raw) * rightScale)
	for i := 2; i < channels; i++ {
		out[i] = raw
	}
	return out
}
