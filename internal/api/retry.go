package api

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RetryPolicy controls automatic re-sending of failed requests.
type RetryPolicy struct {
	MaxAttempts int           // total attempts, including the first
	BaseDelay   time.Duration // first backoff window
	MaxDelay    time.Duration // ceiling for a single wait
}

// DefaultRetryPolicy is tuned for Atlassian: its 429s carry Retry-After (which is honoured
// verbatim and can legitimately be tens of seconds), and its 5xx are usually brief.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{MaxAttempts: 4, BaseDelay: 500 * time.Millisecond, MaxDelay: 30 * time.Second}
}

// idempotentMethods are the HTTP methods that may be retried automatically.
//
// POST and PATCH are deliberately absent: re-sending a POST after a timeout can create a
// second issue, comment or page, and the client cannot tell a lost request from a lost
// response. Silent duplication is worse than a visible error.
var idempotentMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodOptions: true,
}

func isIdempotent(method string) bool { return idempotentMethods[strings.ToUpper(method)] }

// shouldRetry reports whether a completed request is worth re-sending. Only idempotent
// methods are ever eligible.
func shouldRetry(method string, resp *http.Response, err error) bool {
	if !isIdempotent(method) {
		return false
	}
	if err != nil {
		return isTransientNetErr(err)
	}
	if resp == nil {
		return false
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
}

// isTransientNetErr distinguishes "the network hiccuped" from "this will never work".
// A DNS failure or refused connection is retried; a context cancellation is not, because the
// user pressed Ctrl-C.
func isTransientNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// The stdlib wraps several connection-level failures in errors that satisfy no
	// interface worth matching, so fall back to the message for the well-known ones.
	msg := err.Error()
	for _, s := range []string{"connection reset", "connection refused", "broken pipe", "EOF", "unexpected EOF", "no such host", "server closed"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// backoff returns the wait before attempt n (0-based), using AWS-style **full jitter**:
// a uniform draw from [0, base·2^n] capped at MaxDelay.
//
// The randomness is intentional and load-bearing, not a placeholder: with a fixed or
// merely-perturbed delay, every client throttled by the same 429 retries at the same instant
// and re-creates the burst that caused it. Full jitter spreads them out. It is not used for
// anything security-sensitive, hence math/rand.
func backoff(p RetryPolicy, attempt int) time.Duration {
	if p.BaseDelay <= 0 {
		p.BaseDelay = 500 * time.Millisecond
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = 30 * time.Second
	}
	exp := float64(p.BaseDelay) * math.Pow(2, float64(attempt))
	if exp > float64(p.MaxDelay) {
		exp = float64(p.MaxDelay)
	}
	if exp <= 0 {
		return 0
	}
	// #nosec G404 -- jitter for retry spreading, not a security decision
	return time.Duration(rand.Int64N(int64(exp) + 1))
}

// retryAfter reads the Retry-After header, which Atlassian sets on 429 and some 503s.
// Both documented forms are supported: delta-seconds and an HTTP-date. The header always
// wins over computed backoff — it is the server telling us exactly when it will accept work.
func retryAfter(resp *http.Response, now time.Time) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs * float64(time.Second)), true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// waitFor sleeps for d, or returns the context error if the user cancels first. Every wait
// in the client goes through here so Ctrl-C interrupts backoff instead of being swallowed
// by an unbreakable time.Sleep.
func waitFor(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// sortDetails keeps map-derived error detail lists in a stable order across runs.
func sortDetails(d []string) {
	sort.Strings(d)
}
