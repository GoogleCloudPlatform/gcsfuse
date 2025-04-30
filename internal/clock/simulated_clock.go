// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clock

import (
	"sync"
	"time"
)

// afterRequest holds the information for a pending After call in SimulatedClock.
type afterRequest struct {
	targetTime time.Time
	ch         chan time.Time
}

// SimulatedClock is a clock that allows for manipulation of the time,
// which does not change unless AdvanceTime or SetTime is called.
// The zero value is a clock initialized to the zero time.
type SimulatedClock struct {
	mu      sync.RWMutex
	t       time.Time       // GUARDED_BY(mu)
	pending []*afterRequest // GUARDED_BY(mu)
}

func NewSimulatedClock(startTime time.Time) *SimulatedClock {
	return &SimulatedClock{
		t:       startTime,
		pending: nil,
	}
}

func (sc *SimulatedClock) Now() time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.t
}

// SetTime sets the current time according to the clock.
// It also processes any pending After calls that should fire.
func (sc *SimulatedClock) SetTime(t time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.t = t
	sc.processPending()
}

// AdvanceTime advances the current time according to the clock by the supplied duration.
// It also processes any pending After calls that should fire.
func (sc *SimulatedClock) AdvanceTime(d time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.t = sc.t.Add(d)
	sc.processPending()
}

// After returns a time channel and the expectedFired time is sent over the returned
// channel after the provided duration.
// If the duration is zero or negative, it sends the current simulated time immediately.
func (sc *SimulatedClock) After(d time.Duration) <-chan time.Time {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	ch := make(chan time.Time, 1)
	effectiveTargetTime := sc.t.Add(d)

	// If duration is non-positive, fire immediately with the current time.
	// Or if the target time is not after the current time (e.g. d <= 0)
	if !effectiveTargetTime.After(sc.t) {
		ch <- sc.t // Send current time as per time.After behavior for d <= 0
		return ch
	}

	// Otherwise, schedule it.
	ar := &afterRequest{
		targetTime: effectiveTargetTime,
		ch:         ch,
	}
	sc.pending = append(sc.pending, ar)

	return ch
}

// processPending checks all pending After requests and fires those
// whose target time has been reached or passed by the current simulated time.
// This method must be called with sc.mu held.
func (sc *SimulatedClock) processPending() {
	var stillPending []*afterRequest

	for _, ar := range sc.pending {
		// If current time sc.t is not before the targetTime (i.e., sc.t >= ar.targetTime)
		if !sc.t.Before(ar.targetTime) {
			ar.ch <- ar.targetTime // Send the time it was scheduled to fire
			// Do not close the channel, to mimic time.After behavior
		} else {
			stillPending = append(stillPending, ar)
		}
	}

	sc.pending = stillPending
}
