package audio

import "fmt"

// Backend abstracts the audio stack dark is talking to. The current
// implementation is a native Go client for the PulseAudio protocol,
// which works against pipewire-pulse. Mirrors wifi.Backend and
// bluetooth.Backend so the daemon/TUI patterns transfer 1:1.
type Backend interface {
	Name() string
	Snapshot() Snapshot
	Events() <-chan struct{}
	Levels() Levels

	SetSinkVolume(index uint32, pct int) error
	SetSinkMute(index uint32, mute bool) error
	SetSinkBalance(index uint32, balance int) error
	SetSourceVolume(index uint32, pct int) error
	SetSourceMute(index uint32, mute bool) error
	SetSourceBalance(index uint32, balance int) error
	SetDefaultSink(name string) error
	SetDefaultSource(name string) error
	SetCardProfile(cardIndex uint32, profile string) error
	SetSinkPort(sinkIndex uint32, port string) error
	SetSourcePort(sourceIndex uint32, port string) error

	SetSinkInputVolume(index uint32, pct int) error
	SetSinkInputMute(index uint32, mute bool) error
	MoveSinkInput(streamIndex, sinkIndex uint32) error
	KillSinkInput(streamIndex uint32) error
	SetSourceOutputVolume(index uint32, pct int) error
	SetSourceOutputMute(index uint32, mute bool) error
	MoveSourceOutput(streamIndex, sourceIndex uint32) error
	KillSourceOutput(streamIndex uint32) error

	SuspendSink(index uint32, suspend bool) error
	SuspendSource(index uint32, suspend bool) error

	Close()
}

// ErrBackendUnsupported is returned by backends that don't implement a
// particular operation.
var ErrBackendUnsupported = fmt.Errorf("operation not supported by this audio backend")
