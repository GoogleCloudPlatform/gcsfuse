// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package timeutil

import (
	"log"
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
}

func (sc *SimulatedClock) Now() time.Time {
	log.Fatal("TODO: Implement SimulatedClock.Now.")
	return time.Time{}
}

// Advance the current time according to the clock by the supplied duration.
func (sc *SimulatedClock) AdvanceTime(d time.Duration) {
	log.Fatal("TODO: Implement SimulatedClock.AdvanceTime.")
}
