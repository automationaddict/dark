package network

import (
	"testing"
	"time"
)

func TestUpdateRateSample_FirstSampleSetsBaseline(t *testing.T) {
	s := &Service{ratePrev: map[string]rateSample{}}
	iface := &Interface{Name: "eth0", RxBytes: 1000, TxBytes: 500}
	now := time.Now()

	s.updateRateSample(iface, now)

	if iface.RxRateBps != 0 || iface.TxRateBps != 0 {
		t.Errorf("first sample should produce 0 rate, got rx=%d tx=%d",
			iface.RxRateBps, iface.TxRateBps)
	}
	if _, ok := s.ratePrev["eth0"]; !ok {
		t.Error("ratePrev not seeded")
	}
}

func TestUpdateRateSample_ComputesRate(t *testing.T) {
	s := &Service{ratePrev: map[string]rateSample{}}
	iface := &Interface{Name: "eth0", RxBytes: 1000, TxBytes: 500}
	t0 := time.Now()

	s.updateRateSample(iface, t0)
	iface.RxBytes = 11000
	iface.TxBytes = 2500
	s.updateRateSample(iface, t0.Add(10*time.Second))

	if iface.RxRateBps != 1000 {
		t.Errorf("RxRateBps: got %d, want 1000", iface.RxRateBps)
	}
	if iface.TxRateBps != 200 {
		t.Errorf("TxRateBps: got %d, want 200", iface.TxRateBps)
	}
}

func TestUpdateRateSample_CounterReset(t *testing.T) {
	s := &Service{ratePrev: map[string]rateSample{}}
	iface := &Interface{
		Name: "eth0", RxBytes: 100000, TxBytes: 100000,
		RxRateBps: 999, TxRateBps: 999,
	}
	t0 := time.Now()

	s.updateRateSample(iface, t0)
	iface.RxBytes = 50
	iface.TxBytes = 50
	s.updateRateSample(iface, t0.Add(time.Second))

	if iface.RxRateBps != 999 || iface.TxRateBps != 999 {
		t.Errorf("counter reset should leave rate unchanged, got rx=%d tx=%d",
			iface.RxRateBps, iface.TxRateBps)
	}
}
