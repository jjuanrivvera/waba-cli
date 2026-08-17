package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// limiter paces outbound requests.
//
// Atlassian's limits are cost-based and vary by endpoint, plan and instance size, so there is
// no single safe constant. It therefore runs in two modes at once: a fixed token interval as
// the floor, and — when the response carries quota headers — an adaptive slowdown as the
// budget depletes. A 429 halves the rate immediately and it recovers gradually, so a burst
// that trips the limit does not simply trip it again.
type limiter struct {
	mu sync.Mutex

	baseRPS    float64   // configured steady-state rate
	currentRPS float64   // possibly reduced after a 429
	last       time.Time // when the previous request was released

	// Quota state, populated only when the server reports it.
	remaining   int
	reset       time.Time
	haveQuota   bool
	lastRestore time.Time
}

// newLimiter creates a limiter at rps requests/second. rps <= 0 disables pacing.
func newLimiter(rps float64) *limiter {
	return &limiter{baseRPS: rps, currentRPS: rps}
}

// wait blocks until the next request may be sent, or the context is cancelled.
func (l *limiter) wait(ctx context.Context) error {
	d := l.reserve(time.Now())
	if d <= 0 {
		return ctx.Err()
	}
	return waitFor(ctx, d)
}

// reserve computes (and records) the delay before the next release. Split out from wait so
// the pacing logic is testable without real sleeping.
func (l *limiter) reserve(now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.restoreLocked(now)

	rps := l.currentRPS
	if rps <= 0 {
		l.last = now
		return 0
	}

	interval := time.Duration(float64(time.Second) / rps)

	// With quota headers, stretch the interval so the remaining budget lasts until the reset
	// instant. Spending the last few tokens instantly is what turns a warning into a 429.
	if l.haveQuota && l.remaining > 0 && l.reset.After(now) {
		if spread := l.reset.Sub(now) / time.Duration(l.remaining); spread > interval {
			interval = spread
		}
	}

	next := l.last.Add(interval)
	if next.Before(now) || l.last.IsZero() {
		l.last = now
		return 0
	}
	l.last = next
	return next.Sub(now)
}

// observe folds a response's rate-limit signals into the limiter's state.
func (l *limiter) observe(resp *http.Response, now time.Time) {
	if resp == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if remaining, ok := headerInt(resp.Header, "X-RateLimit-Remaining", "X-Ratelimit-Remaining"); ok {
		l.remaining = remaining
		l.haveQuota = true
	}
	if reset, ok := headerReset(resp.Header, now); ok {
		l.reset = reset
		l.haveQuota = true
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		// Halve, with a floor so the client still makes progress rather than stalling.
		l.currentRPS /= 2
		if min := l.baseRPS / 16; l.currentRPS < min {
			l.currentRPS = min
		}
		if l.currentRPS < 0.1 {
			l.currentRPS = 0.1
		}
		l.lastRestore = now
	}
}

// restoreLocked walks a reduced rate back toward the configured one, +25% per 10s of calm.
// Recovering gradually avoids oscillating between full speed and throttled.
func (l *limiter) restoreLocked(now time.Time) {
	if l.currentRPS >= l.baseRPS || l.lastRestore.IsZero() {
		return
	}
	const step = 10 * time.Second
	for now.Sub(l.lastRestore) >= step && l.currentRPS < l.baseRPS {
		l.currentRPS *= 1.25
		l.lastRestore = l.lastRestore.Add(step)
	}
	if l.currentRPS > l.baseRPS {
		l.currentRPS = l.baseRPS
	}
}

// rate reports the current effective requests/second (used by `doctor` and -v output).
func (l *limiter) rate() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.currentRPS
}

func headerInt(h http.Header, names ...string) (int, bool) {
	for _, n := range names {
		if v := strings.TrimSpace(h.Get(n)); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i, true
			}
		}
	}
	return 0, false
}

// headerReset accepts the two forms seen in the wild: seconds-until-reset and an absolute
// epoch second.
func headerReset(h http.Header, now time.Time) (time.Time, bool) {
	v := strings.TrimSpace(h.Get("X-RateLimit-Reset"))
	if v == "" {
		v = strings.TrimSpace(h.Get("X-Ratelimit-Reset"))
	}
	if v == "" {
		return time.Time{}, false
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		// Anything below ~1e9 cannot be a Unix timestamp, so treat it as a duration.
		if n < 1_000_000_000 {
			return now.Add(time.Duration(n) * time.Second), true
		}
		return time.Unix(n, 0), true
	}
	if t, err := http.ParseTime(v); err == nil {
		return t, true
	}
	return time.Time{}, false
}
