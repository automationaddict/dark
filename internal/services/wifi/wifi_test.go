package wifi

import (
	"testing"
	"time"
)

func TestUpdateRateSample_FirstSampleSetsBaseline(t *testing.T) {
	s := &Service{ratePrev: map[string]rateSample{}}
	a := &Adapter{Name: "wlan0"}
	now := time.Now()

	s.updateRateSample(a, 1000, 500, now)

	if a.RxRateBps != 0 || a.TxRateBps != 0 {
		t.Errorf("first sample should produce 0 rate, got rx=%d tx=%d", a.RxRateBps, a.TxRateBps)
	}
	if _, ok := s.ratePrev["wlan0"]; !ok {
		t.Error("ratePrev not seeded")
	}
}

func TestUpdateRateSample_ComputesRate(t *testing.T) {
	s := &Service{ratePrev: map[string]rateSample{}}
	a := &Adapter{Name: "wlan0"}
	t0 := time.Now()

	s.updateRateSample(a, 1000, 500, t0)
	s.updateRateSample(a, 11000, 2500, t0.Add(10*time.Second))

	if a.RxRateBps != 1000 {
		t.Errorf("RxRateBps: got %d, want 1000", a.RxRateBps)
	}
	if a.TxRateBps != 200 {
		t.Errorf("TxRateBps: got %d, want 200", a.TxRateBps)
	}
}

func TestUpdateRateSample_CounterReset(t *testing.T) {
	s := &Service{ratePrev: map[string]rateSample{}}
	a := &Adapter{Name: "wlan0", RxRateBps: 999, TxRateBps: 999}
	t0 := time.Now()

	s.updateRateSample(a, 100000, 100000, t0)
	// Counter reset to smaller value (e.g. interface down/up).
	s.updateRateSample(a, 50, 50, t0.Add(time.Second))

	// Rate should not underflow — should be left at prior value.
	if a.RxRateBps != 999 || a.TxRateBps != 999 {
		t.Errorf("counter reset should leave rate unchanged, got rx=%d tx=%d", a.RxRateBps, a.TxRateBps)
	}
}

func TestUpdateRateSample_ZeroElapsed(t *testing.T) {
	s := &Service{ratePrev: map[string]rateSample{}}
	a := &Adapter{Name: "wlan0", RxRateBps: 42}
	t0 := time.Now()

	s.updateRateSample(a, 1000, 500, t0)
	s.updateRateSample(a, 5000, 2500, t0) // same timestamp

	if a.RxRateBps != 42 {
		t.Errorf("zero elapsed should leave rate unchanged, got %d", a.RxRateBps)
	}
}
