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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// A non-zero reference time for tests
	referenceTime    = time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC)
	shortTestTimeout = 10 * time.Millisecond // For non-blocking channel checks
	fireTestTimeout  = 50 * time.Millisecond // When expecting a channel to fire
)

func TestSimulatedClock_Now(t *testing.T) {
	testCases := []struct {
		name             string
		initialTimeSetup func(sc *SimulatedClock)
		expectedTime     time.Time
	}{
		{
			name:             "InitialState_IsZeroTime",
			initialTimeSetup: func(sc *SimulatedClock) {},
			expectedTime:     time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC), // referenceTime
		},
		{
			name: "AfterSetTime_ReturnsSetTime",
			initialTimeSetup: func(sc *SimulatedClock) {
				sc.SetTime(referenceTime)
			},
			expectedTime: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC), // referenceTime
		},
		{
			name: "AfterAdvanceTime_ReturnsAdvancedTime",
			initialTimeSetup: func(sc *SimulatedClock) {
				sc.SetTime(referenceTime)
				sc.AdvanceTime(time.Hour)
			},
			expectedTime: time.Date(2020, time.January, 1, 13, 0, 0, 0, time.UTC), // referenceTime + time.Hour
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(referenceTime)
			tc.initialTimeSetup(clock)

			now := clock.Now()

			assert.True(t, now.Equal(tc.expectedTime), "clock.Now() returned %v, expected %v", now, tc.expectedTime)
		})
	}
}

func TestSimulatedClock_SetTime(t *testing.T) {
	testCases := []struct {
		name             string
		initialTimeSetup func(sc *SimulatedClock) // To set a pre-existing time if needed
		timeToSet        time.Time
		expectedAfter    time.Time
	}{
		{
			name:             "SetFromZeroTime",
			initialTimeSetup: func(sc *SimulatedClock) {},
			timeToSet:        referenceTime,
			expectedAfter:    time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC), // referenceTime
		},
		{
			name: "OverwriteExistingTime",
			initialTimeSetup: func(sc *SimulatedClock) {
				sc.SetTime(referenceTime.Add(-time.Hour)) // Start with a different time
			},
			timeToSet:     referenceTime,
			expectedAfter: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC), // referenceTime - time.Hour
		},
		{
			name:             "SetToZeroTime",
			initialTimeSetup: func(sc *SimulatedClock) { sc.SetTime(referenceTime) },
			timeToSet:        time.Time{}, // Zero time
			expectedAfter:    time.Time{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(referenceTime)
			tc.initialTimeSetup(clock)

			clock.SetTime(tc.timeToSet)

			assert.True(t, clock.Now().Equal(tc.expectedAfter), "After SetTime, clock.Now() was %v, expected %v", clock.Now(), tc.expectedAfter)
		})
	}
}

func TestSimulatedClock_AdvanceTime(t *testing.T) {
	testCases := []struct {
		name         string
		initialTime  time.Time
		advanceBy    time.Duration
		expectedTime time.Time
	}{
		{
			name:         "AdvancePositiveDuration",
			initialTime:  referenceTime,
			advanceBy:    5 * time.Minute,
			expectedTime: time.Date(2020, time.January, 1, 12, 5, 0, 0, time.UTC), // refereceTime + 5 minutes
		},
		{
			name:         "AdvanceNegativeDuration",
			initialTime:  referenceTime,
			advanceBy:    -2 * time.Hour,
			expectedTime: time.Date(2020, time.January, 1, 10, 0, 0, 0, time.UTC), // referenceTime - 2 hours
		},
		{
			name:         "AdvanceByZeroDuration",
			initialTime:  referenceTime,
			advanceBy:    0,
			expectedTime: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC), // referenceTime
		},
		{
			name:         "AdvanceFromZeroTime",
			initialTime:  time.Time{},
			advanceBy:    time.Hour,
			expectedTime: time.Date(1, time.January, 1, 1, 0, 0, 0, time.UTC), // zeroTime + time.Hour
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(referenceTime)
			clock.SetTime(tc.initialTime) // Set initial time for the clock

			clock.AdvanceTime(tc.advanceBy)

			assert.True(t, clock.Now().Equal(tc.expectedTime), "After AdvanceTime, clock.Now() was %v, expected %v", clock.Now(), tc.expectedTime)
		})
	}
}

func TestSimulatedClock_After_ShouldFireZeroOrNegativeDuration(t *testing.T) {
	testCases := []struct {
		name          string
		afterDuration time.Duration
		action        func(sc *SimulatedClock) // Action to manipulate the clock after After() is called
	}{
		{
			name:          "ZeroDuration_FiresImmediately",
			afterDuration: 0,
			action:        func(sc *SimulatedClock) { /* No action needed for immediate fire */ },
		},
		{
			name:          "NegativeDuration_FiresImmediately",
			afterDuration: -5 * time.Second,
			action:        func(sc *SimulatedClock) { /* No action needed for immediate fire */ },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(referenceTime)
			clockTimeAtAfterCall := clock.Now()

			ch := clock.After(tc.afterDuration)
			require.NotNil(t, ch, "Channel should not be nil")

			// Perform the action (if any) that might trigger the timer
			tc.action(clock)

			// Fires at the same time for zero/negative duration.
			expectedFireTimeOnChannel := clockTimeAtAfterCall

			select {
			case actualTime := <-ch:
				assert.True(t, expectedFireTimeOnChannel.Equal(actualTime), "Received time %v, expected %v", actualTime, expectedFireTimeOnChannel)

			case <-time.After(fireTestTimeout):
				t.Fatalf("Timeout waiting for time on channel. Expected after %v.", expectedFireTimeOnChannel)
			}
		})
	}
}

func TestSimulatedClock_After_ShouldFirePositiveDuration(t *testing.T) {
	testCases := []struct {
		name          string
		afterDuration time.Duration
		action        func(sc *SimulatedClock) // Action to manipulate the clock after After() is called
	}{
		{
			name:          "PositiveDuration_Fires_WhenTimeAdvancedPastDuration",
			afterDuration: 10 * time.Second,
			action: func(sc *SimulatedClock) {
				sc.AdvanceTime(15 * time.Second) // Advance well past the duration
			},
		},
		{
			name:          "PositiveDuration_Fires_WhenTimeSetPastDuration",
			afterDuration: 10 * time.Second,
			action: func(sc *SimulatedClock) {
				sc.SetTime(referenceTime.Add(15 * time.Second)) // Set time well past the duration
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(referenceTime)
			clockTimeAtAfterCall := clock.Now()

			ch := clock.After(tc.afterDuration)
			require.NotNil(t, ch, "Channel should not be nil")

			// Perform the action (if any) that might trigger the timer
			tc.action(clock)
			// Expected fire time, time + duration.
			expectedFireTimeOnChannel := clockTimeAtAfterCall.Add(tc.afterDuration)

			select {
			case actualTime := <-ch:
				assert.True(t, expectedFireTimeOnChannel.Equal(actualTime), "Received time %v, expected %v", actualTime, expectedFireTimeOnChannel)

			case <-time.After(fireTestTimeout):
				t.Fatalf("Timeout waiting for time on channel. Expected after %v.", expectedFireTimeOnChannel)
			}
		})
	}
}

func TestSimulatedClock_After_ShouldNotFire(t *testing.T) {
	testCases := []struct {
		name          string
		afterDuration time.Duration
		action        func(sc *SimulatedClock) // Action to manipulate the clock after After() is called
	}{
		{
			name:          "PositiveDuration_DoesNotFire_WhenTimeAdvancedBeforeDuration",
			afterDuration: 10 * time.Second,
			action: func(sc *SimulatedClock) {
				sc.AdvanceTime(5 * time.Second) // Advance, but not enough
			},
		},
		{
			name:          "PositiveDuration_DoesNotFire_WhenTimeSetBeforeDuration",
			afterDuration: 10 * time.Second,
			action: func(sc *SimulatedClock) {
				sc.SetTime(referenceTime.Add(5 * time.Second)) // Set time, but not enough
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(referenceTime)

			ch := clock.After(tc.afterDuration)
			require.NotNil(t, ch, "Channel should not be nil")

			// Perform the action (if any) that might trigger the timer
			tc.action(clock)

			select {
			case receivedTime := <-ch:
				t.Fatalf("Expected no time on channel, but received %v.", receivedTime)

			case <-time.After(shortTestTimeout):
				// Success, nothing received
			}
		})
	}
}
