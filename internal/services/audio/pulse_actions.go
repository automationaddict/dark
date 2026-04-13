package audio

import (
	"fmt"

	"github.com/jfreymuth/pulse/proto"
)

func (b *pulseBackend) SetSinkVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sinkChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSinkVolume{
		SinkIndex:      index,
		ChannelVolumes: percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkBalance(index uint32, balance int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	reply := &proto.GetSinkInfoReply{}
	if err := b.client.Request(&proto.GetSinkInfo{SinkIndex: index}, reply); err != nil {
		return fmt.Errorf("get sink info: %w", err)
	}
	pct := volumeToPercent(avgVolume(reply.ChannelVolumes))
	channels := len(reply.ChannelVolumes)
	req := &proto.SetSinkVolume{
		SinkIndex:      index,
		ChannelVolumes: balanceToVolumes(pct, balance, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink balance: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSinkMute{SinkIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sourceChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSourceVolume{
		SourceIndex:    index,
		ChannelVolumes: percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSourceMute{SourceIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceBalance(index uint32, balance int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	reply := &proto.GetSourceInfoReply{}
	if err := b.client.Request(&proto.GetSourceInfo{SourceIndex: index}, reply); err != nil {
		return fmt.Errorf("get source info: %w", err)
	}
	pct := volumeToPercent(avgVolume(reply.ChannelVolumes))
	channels := len(reply.ChannelVolumes)
	req := &proto.SetSourceVolume{
		SourceIndex:    index,
		ChannelVolumes: balanceToVolumes(pct, balance, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source balance: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetDefaultSink(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.SetDefaultSink{SinkName: name}, nil); err != nil {
		return fmt.Errorf("set default sink: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetDefaultSource(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.SetDefaultSource{SourceName: name}, nil); err != nil {
		return fmt.Errorf("set default source: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetCardProfile(cardIndex uint32, profile string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetCardProfile{CardIndex: cardIndex, ProfileName: profile}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set card profile: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkPort(sinkIndex uint32, port string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSinkPort{SinkIndex: sinkIndex, Port: port}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink port: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourcePort(sourceIndex uint32, port string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSourcePort{SourceIndex: sourceIndex, Port: port}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source port: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkInputVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sinkInputChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSinkInputVolume{
		SinkInputIndex: index,
		ChannelVolumes: percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink input volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkInputMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSinkInputMute{SinkInputIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink input mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) MoveSinkInput(streamIndex, sinkIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.MoveSinkInput{SinkInputIndex: streamIndex, DeviceIndex: sinkIndex}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("move sink input: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceOutputVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sourceOutputChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSourceOutputVolume{
		SourceOutputIndex: index,
		ChannelVolumes:    percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source output volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceOutputMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSourceOutputMute{SourceOutputIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source output mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) MoveSourceOutput(streamIndex, sourceIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.MoveSourceOutput{SourceOutputIndex: streamIndex, DeviceIndex: sourceIndex}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("move source output: %w", err)
	}
	return nil
}

func (b *pulseBackend) KillSinkInput(streamIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.KillSinkInput{SinkInputIndex: streamIndex}, nil); err != nil {
		return fmt.Errorf("kill sink input: %w", err)
	}
	return nil
}

func (b *pulseBackend) KillSourceOutput(streamIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.KillSourceOutput{SourceOutputIndex: streamIndex}, nil); err != nil {
		return fmt.Errorf("kill source output: %w", err)
	}
	return nil
}

func (b *pulseBackend) SuspendSink(index uint32, suspend bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SuspendSink{SinkIndex: index, Suspend: suspend}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("suspend sink: %w", err)
	}
	return nil
}

func (b *pulseBackend) SuspendSource(index uint32, suspend bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SuspendSource{SourceIndex: index, Suspend: suspend}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("suspend source: %w", err)
	}
	return nil
}

// sinkInputChannelCount re-reads the live channel count for a sink
// input so the ChannelVolumes vector we send matches what the server
// expects.
func (b *pulseBackend) sinkInputChannelCount(index uint32) (int, error) {
	var reply proto.GetSinkInputInfoReply
	if err := b.client.Request(&proto.GetSinkInputInfo{SinkInputIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get sink input info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}

func (b *pulseBackend) sourceOutputChannelCount(index uint32) (int, error) {
	var reply proto.GetSourceOutputInfoReply
	if err := b.client.Request(&proto.GetSourceOutputInfo{SourceOutpuIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get source output info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}

// sinkChannelCount re-reads the live channel count for a sink so the
// ChannelVolumes vector we send matches what the server expects. The
// snapshot cached in the TUI may be stale by a few ticks.
func (b *pulseBackend) sinkChannelCount(index uint32) (int, error) {
	var reply proto.GetSinkInfoReply
	if err := b.client.Request(&proto.GetSinkInfo{SinkIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get sink info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}

func (b *pulseBackend) sourceChannelCount(index uint32) (int, error) {
	var reply proto.GetSourceInfoReply
	if err := b.client.Request(&proto.GetSourceInfo{SourceIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get source info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}
