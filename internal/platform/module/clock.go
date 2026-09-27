package module

import "time"

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// SystemClock is the production clock supplied to modules by the composition root.
func SystemClock() Clock { return systemClock{} }
