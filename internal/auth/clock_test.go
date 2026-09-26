package auth_test

import "time"

// fakeClock is a deterministic auth.Clock for tests.
type fakeClock struct {
	now time.Time
}

func (clock *fakeClock) Now() time.Time { return clock.now }

func (clock *fakeClock) Advance(d time.Duration) { clock.now = clock.now.Add(d) }
