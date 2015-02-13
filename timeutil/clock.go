// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package timeutil

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

////////////////////////////////////////////////////////////////////////
// Real clock
////////////////////////////////////////////////////////////////////////

type realClock struct{}

func (c realClock) Now() time.Time {
	return time.Now()
}

// Return a clock that follows the real time, according to the system.
func RealClock() Clock {
	return realClock{}
}

////////////////////////////////////////////////////////////////////////
// Simulated clock
////////////////////////////////////////////////////////////////////////

// A clock that allows for manipulation of the time, which does not change
// unless AdvanceTime is called. The zero value is a clock initialized to the
// zero time.
type SimulatedClock struct {
	Clock

	mu sync.RWMutex
	t  time.Time // GUARDED_BY(mu)
}

func (sc *SimulatedClock) Now() time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.t
}

// Advance the current time according to the clock by the supplied duration.
func (sc *SimulatedClock) AdvanceTime(d time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.t = sc.t.Add(d)
}
