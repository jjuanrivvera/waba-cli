package api

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests here cover the failures that cost money or reach a human twice: a send that gets
// retried, a rate limit answered with a storm. They exist so an operator does not have to run
// them by hand before pointing the CLI at a production System User token (issue #5).

// A dropped connection is not a status code, so it takes a different branch from a 500 — and
// it is the more dangerous one: the request may well have been delivered, and the client has
// no way to know. A retry here would send the same WhatsApp message to a human twice.
func TestClient_NeverRetriesPOST_OnTransportFailure(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
		panic(http.ErrAbortHandler) // drop the connection mid-flight
	})

	err := c.PostJSON(context.Background(), "123/messages", map[string]string{"text": "hi"}, nil)
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load(),
		"a POST that failed in transit may already have been delivered; retrying double-sends it")
}

// The same guarantee when the request times out rather than being dropped. The deadline is a
// PER-ATTEMPT one on the transport, not a cancellation of the caller's context: a cancelled
// caller would stop a retry by itself, and the test would pass even if the client had started
// retrying sends. With the caller's context alive, the attempt count is the real evidence.
func TestClient_NeverRetriesPOST_OnTimeout(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(400 * time.Millisecond):
		}
	}, WithHTTPClient(&http.Client{Timeout: 100 * time.Millisecond}))

	err := c.PostJSON(context.Background(), "123/messages", map[string]string{"text": "hi"}, nil)
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load(),
		"a timed-out POST must not be replayed — the message may already be on its way")
}

// A 429 answered with an immediate retry, and that retry answered with another, is how a
// rate limit turns into a ban. The client must give up after its bounded attempts.
func TestClient_RateLimitStormIsBounded(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0.01")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":4,"message":"(#4) Application request limit reached"}}`))
	})

	start := time.Now()
	err := c.GetJSON(context.Background(), "x", nil, nil)

	require.Error(t, err, "a permanent 429 must surface, not loop")
	assert.Equal(t, int32(3), calls.Load(), "attempts are capped by the retry policy (MaxAttempts)")
	assert.Less(t, time.Since(start), 2*time.Second, "bounded attempts must not add up to a hang")
}

// Note on what is deliberately NOT tested here: the growth of the backoff window. The policy
// uses full jitter — a uniform draw from [0, base·2^n] — so a single observed gap can be
// nearly zero without anything being wrong, which makes a timing assertion flaky by
// construction. The bound and the spread are tested directly against the policy in
// retry_test.go (TestBackoff_FullJitterStaysInRange / TestBackoff_SpreadsAcrossTheRange),
// which is where that property can be checked without a stopwatch.
