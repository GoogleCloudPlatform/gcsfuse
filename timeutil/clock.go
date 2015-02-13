// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package timeutil

import "time"

type Clock interface {
	Now() time.Time
}

////////////////////////////////////////////////////////////////////////
// Real clock
////////////////////////////////////////////////////////////////////////

// Return a clock that follows the real time, according to the system.
func RealClock() Clock

////////////////////////////////////////////////////////////////////////
// Simulated clock
////////////////////////////////////////////////////////////////////////

// A clock that allows for manipulation of the time, which does not change
// unless AdvanceTime is called. The zero value is a clock initialized to the
// zero time.
type SimulatedClock struct {
	Clock
}

// Advance the current time according to the clock by the supplied duration.
func (sc *SimulatedClock) AdvanceTime(d time.Duration)
