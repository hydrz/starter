package billing

import (
	"sync"
	"time"
)

// webhookRateLimitWindow and webhookRateLimitPerWindow bound how many
// webhook deliveries WebhookHTTPHandler accepts per source IP before doing
// any HMAC signature verification work, so an attacker cannot cheaply force
// repeated verification by hammering the unauthenticated endpoint.
//
// The limit is intentionally generous: Stripe's webhook endpoint is
// unauthenticated and unkeyed by any user/org (unlike the auth package's
// per-email/IP OTP limits), and Stripe redelivers in a burst after an
// outage or a period the endpoint was paused/unreachable. 300 requests per
// minute per source IP comfortably absorbs a large retry-storm burst from a
// single Stripe delivery IP while still bounding the verification work one
// caller can force.
const (
	webhookRateLimitWindow    = time.Minute
	webhookRateLimitPerWindow = 300
)

// ipWindowLimiter is a fixed-window, per-key request counter. It matches
// internal/auth/otp.go's rate-limiting approach (count requests for a key
// within a rolling window, reject once a fixed threshold is reached)
// rather than introducing a second, unrelated rate-limiting mechanism or
// third-party library; unlike otp.go's DB-backed per-email/IP counts, this
// limiter is in-memory because the webhook endpoint is unauthenticated,
// unkeyed by any stored user/org record, and does not need counts to
// survive a process restart.
type ipWindowLimiter struct {
	clock  Clock
	window time.Duration
	limit  int

	mu       sync.Mutex
	counters map[string]*windowCounter
}

type windowCounter struct {
	windowStart time.Time
	count       int
}

func newIPWindowLimiter(clock Clock, window time.Duration, limit int) *ipWindowLimiter {
	if clock == nil {
		clock = SystemClock{}
	}
	return &ipWindowLimiter{
		clock:    clock,
		window:   window,
		limit:    limit,
		counters: make(map[string]*windowCounter),
	}
}

// Allow reports whether a request keyed by ip is within the limit for the
// current window, incrementing that key's count as a side effect.
func (l *ipWindowLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()
	counter, ok := l.counters[ip]
	if !ok || now.Sub(counter.windowStart) >= l.window {
		l.counters[ip] = &windowCounter{windowStart: now, count: 1}
		l.sweepExpiredLocked(now)
		return true
	}
	if counter.count >= l.limit {
		return false
	}
	counter.count++
	return true
}

// webhookRateLimitSweepThreshold bounds how large l.counters is allowed to
// grow (across distinct source IPs) before an expired-entry sweep runs, so
// long-lived process uptime with many distinct callers cannot grow this map
// without bound.
const webhookRateLimitSweepThreshold = 10_000

// sweepExpiredLocked removes windows that have already elapsed. Callers
// must hold l.mu.
func (l *ipWindowLimiter) sweepExpiredLocked(now time.Time) {
	if len(l.counters) < webhookRateLimitSweepThreshold {
		return
	}
	for ip, counter := range l.counters {
		if now.Sub(counter.windowStart) >= l.window {
			delete(l.counters, ip)
		}
	}
}
