package appstore

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestBackend() *aurBackend {
	return &aurBackend{
		logger: silentLogger(),
		client: newAURClient(silentLogger()),
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"120", 120 * time.Second},
		{"0", 0},
		{"-5", 0},
		{"not-a-number", 0},
		{"  30  ", 30 * time.Second},
	}
	for _, tt := range tests {
		got := parseRetryAfter(tt.in)
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}

	// HTTP-date form — must produce a positive duration when the date
	// is in the future. net/http.TimeFormat is the IMF-fixdate form the
	// Retry-After header uses.
	future := time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat)
	d := parseRetryAfter(future)
	if d <= 0 || d > 3*time.Minute {
		t.Errorf("parseRetryAfter(future date) = %v, want ~2m", d)
	}

	// Past HTTP-date returns zero.
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	if parseRetryAfter(past) != 0 {
		t.Errorf("parseRetryAfter(past date) should return 0")
	}
}

func TestClassifyAURError(t *testing.T) {
	_, ok := classifyAURError(errors.New("plain error"))
	if ok {
		t.Error("plain error should not classify as throttled")
	}
	_, ok = classifyAURError(nil)
	if ok {
		t.Error("nil error should not classify as throttled")
	}

	throttle := &aurThrottleError{status: 429, retryAfter: 45 * time.Second}
	d, ok := classifyAURError(throttle)
	if !ok {
		t.Error("aurThrottleError should classify as throttled")
	}
	if d != 45*time.Second {
		t.Errorf("retryAfter = %v, want 45s", d)
	}

	// Wrapped throttle error.
	wrapped := fmt.Errorf("outer: %w", throttle)
	if _, ok := classifyAURError(wrapped); !ok {
		t.Error("wrapped aurThrottleError should still classify")
	}
}

func TestRecordErrorIgnoresNonThrottle(t *testing.T) {
	b := newTestBackend()
	b.recordError(errors.New("connection reset"))
	if b.lastLimit.Active {
		t.Error("non-throttle error should not activate rate limit")
	}
}

func TestRecordErrorWithServerProvidedRetryAfter(t *testing.T) {
	b := newTestBackend()
	b.recordError(&aurThrottleError{status: 429, retryAfter: 90 * time.Second})

	if !b.lastLimit.Active {
		t.Fatal("expected rate limit to be active")
	}
	window := time.Until(time.Unix(b.lastLimit.RetryAfterUnix, 0))
	if window < 80*time.Second || window > 95*time.Second {
		t.Errorf("retry window = %v, want ~90s", window)
	}
}

func TestRecordErrorExponentialBackoff(t *testing.T) {
	b := newTestBackend()

	// First throttle with no Retry-After starts at 10s.
	b.recordError(&aurThrottleError{status: 429})
	if !b.lastLimit.Active {
		t.Fatal("first throttle: limit not active")
	}
	first := time.Until(time.Unix(b.lastLimit.RetryAfterUnix, 0))
	if first < 8*time.Second || first > 12*time.Second {
		t.Errorf("first backoff = %v, want ~10s", first)
	}

	// Second throttle doubles the previous window.
	b.recordError(&aurThrottleError{status: 429})
	second := time.Until(time.Unix(b.lastLimit.RetryAfterUnix, 0))
	if second < 15*time.Second || second > 25*time.Second {
		t.Errorf("second backoff = %v, want ~20s", second)
	}

	// Backoff clamps at 5 minutes — simulate a large prior window.
	b.lastLimit.RetryAfterUnix = time.Now().Add(10 * time.Minute).Unix()
	b.recordError(&aurThrottleError{status: 429})
	clamped := time.Until(time.Unix(b.lastLimit.RetryAfterUnix, 0))
	if clamped > 5*time.Minute+5*time.Second {
		t.Errorf("clamped backoff = %v, want <= 5m", clamped)
	}
}

func TestCurrentLimitExpires(t *testing.T) {
	b := newTestBackend()
	b.lastLimit = RateLimitState{
		Active:         true,
		RetryAfterUnix: time.Now().Add(-1 * time.Second).Unix(),
		Message:        "expired",
	}
	got := b.currentLimit()
	if got.Active {
		t.Error("expired limit should present as inactive")
	}
}

func TestClearLimit(t *testing.T) {
	b := newTestBackend()
	b.lastLimit = RateLimitState{Active: true, RetryAfterUnix: time.Now().Add(time.Minute).Unix()}
	b.clearLimit()
	if b.lastLimit.Active {
		t.Error("clearLimit should deactivate limit")
	}
}

func TestSearchSkippedWhileRateLimited(t *testing.T) {
	b := newTestBackend()
	b.lastLimit = RateLimitState{
		Active:         true,
		RetryAfterUnix: time.Now().Add(2 * time.Minute).Unix(),
		Message:        "throttled",
	}
	// Search must not call the HTTP client when rate-limited — it should
	// return an empty result with the current limit attached.
	result, err := b.Search(SearchQuery{Text: "firefox"})
	if err != nil {
		t.Fatalf("Search returned error while rate-limited: %v", err)
	}
	if len(result.Packages) != 0 {
		t.Errorf("expected empty results while rate-limited, got %d", len(result.Packages))
	}
	if !result.AURLimit.Active {
		t.Error("result should carry the active rate limit")
	}
}

func TestSearchEmptyQueryIsNoop(t *testing.T) {
	b := newTestBackend()
	result, err := b.Search(SearchQuery{Text: "   "})
	if err != nil {
		t.Fatalf("empty search returned error: %v", err)
	}
	if len(result.Packages) != 0 {
		t.Errorf("empty search should return no packages, got %d", len(result.Packages))
	}
}
