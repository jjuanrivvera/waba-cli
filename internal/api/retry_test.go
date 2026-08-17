package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsIdempotent(t *testing.T) {
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions} {
		assert.True(t, isIdempotent(m), "%s should be idempotent", m)
	}
	for _, m := range []string{http.MethodPost, http.MethodPatch} {
		assert.False(t, isIdempotent(m), "%s must never be auto-retried", m)
	}
	assert.True(t, isIdempotent("get"), "method matching is case-insensitive")
}

func TestShouldRetry(t *testing.T) {
	resp := func(code int) *http.Response { return &http.Response{StatusCode: code} }

	assert.True(t, shouldRetry(http.MethodGet, resp(http.StatusTooManyRequests), nil))
	assert.True(t, shouldRetry(http.MethodGet, resp(http.StatusInternalServerError), nil))
	assert.False(t, shouldRetry(http.MethodGet, resp(http.StatusOK), nil))
	assert.False(t, shouldRetry(http.MethodGet, resp(http.StatusNotFound), nil))
	assert.False(t, shouldRetry(http.MethodPost, resp(http.StatusInternalServerError), nil))
	assert.False(t, shouldRetry(http.MethodGet, nil, nil))
}

func TestIsTransientNetErr(t *testing.T) {
	assert.False(t, isTransientNetErr(nil))
	// A cancelled context means the user pressed Ctrl-C; retrying would ignore them.
	assert.False(t, isTransientNetErr(context.Canceled))
	assert.False(t, isTransientNetErr(context.DeadlineExceeded))
	assert.True(t, isTransientNetErr(errors.New("connection reset by peer")))
	assert.True(t, isTransientNetErr(errors.New("dial tcp: lookup x: no such host")))
	assert.True(t, isTransientNetErr(&net.DNSError{Err: "timeout", IsTimeout: true}))
}

func TestBackoff_FullJitterStaysInRange(t *testing.T) {
	// Full jitter draws uniformly from [0, base*2^n]; the randomness is the point, so the
	// assertion is on the bound rather than on a specific value.
	p := RetryPolicy{BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}
	for attempt := range 5 {
		upper := min(time.Duration(float64(p.BaseDelay)*pow2(attempt)), p.MaxDelay)
		for range 50 {
			got := backoff(p, attempt)
			assert.GreaterOrEqual(t, got, time.Duration(0))
			assert.LessOrEqual(t, got, upper, "attempt %d exceeded its ceiling", attempt)
		}
	}
}

func TestBackoff_SpreadsAcrossTheRange(t *testing.T) {
	// If jitter were accidentally removed, every draw would be identical and a thundering
	// herd would re-form after a 429. Seeing several distinct values proves it is live.
	p := RetryPolicy{BaseDelay: time.Second, MaxDelay: time.Minute}
	seen := map[time.Duration]bool{}
	for range 40 {
		seen[backoff(p, 3)] = true
	}
	assert.Greater(t, len(seen), 5, "backoff should be jittered, not constant")
}

func pow2(n int) float64 {
	out := 1.0
	for range n {
		out *= 2
	}
	return out
}

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)

	t.Run("delta seconds", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{"Retry-After": {"30"}}}
		d, ok := retryAfter(resp, now)
		require.True(t, ok)
		assert.Equal(t, 30*time.Second, d)
	})

	t.Run("fractional seconds", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{"Retry-After": {"1.5"}}}
		d, ok := retryAfter(resp, now)
		require.True(t, ok)
		assert.Equal(t, 1500*time.Millisecond, d)
	})

	t.Run("http date", func(t *testing.T) {
		future := now.Add(2 * time.Minute).Format(http.TimeFormat)
		resp := &http.Response{Header: http.Header{"Retry-After": {future}}}
		d, ok := retryAfter(resp, now)
		require.True(t, ok)
		assert.InDelta(t, float64(2*time.Minute), float64(d), float64(time.Second))
	})

	t.Run("past date clamps to zero", func(t *testing.T) {
		past := now.Add(-time.Hour).Format(http.TimeFormat)
		resp := &http.Response{Header: http.Header{"Retry-After": {past}}}
		d, ok := retryAfter(resp, now)
		require.True(t, ok)
		assert.Zero(t, d)
	})

	t.Run("absent and malformed", func(t *testing.T) {
		_, ok := retryAfter(&http.Response{Header: http.Header{}}, now)
		assert.False(t, ok)
		_, ok = retryAfter(&http.Response{Header: http.Header{"Retry-After": {"soon"}}}, now)
		assert.False(t, ok)
		_, ok = retryAfter(nil, now)
		assert.False(t, ok)
	})
}

func TestWaitFor_RespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Ctrl-C must interrupt a backoff wait rather than be swallowed by it.
	err := waitFor(ctx, time.Hour)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLimiter_HalvesOn429AndRestores(t *testing.T) {
	l := newLimiter(10)
	now := time.Now()

	l.observe(&http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}, now)
	assert.InDelta(t, 5.0, l.rate(), 0.01, "a 429 should halve the rate")

	l.observe(&http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}, now)
	assert.InDelta(t, 2.5, l.rate(), 0.01)

	// A quiet period walks the rate back up rather than leaving the client throttled forever.
	l.mu.Lock()
	l.lastRestore = now.Add(-30 * time.Second)
	l.restoreLocked(now)
	restored := l.currentRPS
	l.mu.Unlock()
	assert.Greater(t, restored, 2.5)
	assert.LessOrEqual(t, restored, 10.0, "restoration must not overshoot the configured rate")
}

func TestLimiter_SpreadsRemainingQuota(t *testing.T) {
	l := newLimiter(100) // 10ms between requests at the configured rate
	now := time.Now()

	// 2 requests left in a 10s window means one every 5s — far slower than the base rate.
	l.observe(&http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Ratelimit-Remaining": {"2"}, "X-Ratelimit-Reset": {"10"}},
	}, now)

	l.reserve(now) // first call sets the baseline
	got := l.reserve(now)
	assert.Greater(t, got, time.Second, "a nearly-exhausted quota should stretch the interval")
}

func TestLimiter_ZeroRateDisablesPacing(t *testing.T) {
	l := newLimiter(0)
	assert.Zero(t, l.reserve(time.Now()))
}

func TestHeaderReset(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)

	// Small values are seconds-until-reset; large ones are absolute epoch seconds.
	got, ok := headerReset(http.Header{"X-Ratelimit-Reset": {"60"}}, now)
	require.True(t, ok)
	assert.Equal(t, now.Add(time.Minute), got)

	epoch := now.Add(time.Hour).Unix()
	got, ok = headerReset(http.Header{"X-Ratelimit-Reset": {itoa(int(epoch))}}, now)
	require.True(t, ok)
	assert.Equal(t, epoch, got.Unix())

	_, ok = headerReset(http.Header{}, now)
	assert.False(t, ok)
}
