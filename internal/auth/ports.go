package auth

import "time"

// Clock abstracts wall-clock time so services and tokens are deterministic
// under test.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock backed by time.Now.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
