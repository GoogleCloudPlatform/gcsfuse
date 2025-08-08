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
			expectedTime:     time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "AfterSetTime_ReturnsSetTime",
			initialTimeSetup: func(sc *SimulatedClock) {
				sc.SetTime(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
			},
			expectedTime: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "AfterAdvanceTime_ReturnsAdvancedTime",
			initialTimeSetup: func(sc *SimulatedClock) {
				sc.SetTime(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
				sc.AdvanceTime(time.Hour)
			},
			expectedTime: time.Date(2020, time.January, 1, 13, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
			tc.initialTimeSetup(clock)

			now := clock.Now()

			assert.Equal(t, tc.expectedTime, now)
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
			timeToSet:        time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
			expectedAfter:    time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "OverwriteExistingTime",
			initialTimeSetup: func(sc *SimulatedClock) {
				sc.SetTime(time.Date(2020, time.January, 1, 11, 0, 0, 0, time.UTC)) // Start with a different time
			},
			timeToSet:     time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
			expectedAfter: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name:             "SetToZeroTime",
			initialTimeSetup: func(sc *SimulatedClock) { sc.SetTime(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC)) },
			timeToSet:        time.Time{}, // Zero time
			expectedAfter:    time.Time{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
			tc.initialTimeSetup(clock)

			clock.SetTime(tc.timeToSet)

			assert.Equal(t, tc.expectedAfter, clock.Now())
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
			initialTime:  time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
			advanceBy:    5 * time.Minute,
			expectedTime: time.Date(2020, time.January, 1, 12, 5, 0, 0, time.UTC),
		},
		{
			name:         "AdvanceNegativeDuration",
			initialTime:  time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
			advanceBy:    -2 * time.Hour,
			expectedTime: time.Date(2020, time.January, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			name:         "AdvanceByZeroDuration",
			initialTime:  time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
			advanceBy:    0,
			expectedTime: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
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
			clock := NewSimulatedClock(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
			clock.SetTime(tc.initialTime) // Set initial time for the clock

			clock.AdvanceTime(tc.advanceBy)

			assert.Equal(t, tc.expectedTime, clock.Now())
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
			clock := NewSimulatedClock(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
			clockTimeAtAfterCall := clock.Now()

			ch := clock.After(tc.afterDuration)
			require.NotNil(t, ch, "Channel should not be nil")

			// Perform the action (if any) that might trigger the timer
			tc.action(clock)

			// Fires at the same time for zero/negative duration.
			expectedFireTimeOnChannel := clockTimeAtAfterCall

			select {
			case actualTime := <-ch:
				assert.Equal(t, expectedFireTimeOnChannel, actualTime)

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
				sc.SetTime(time.Date(2020, time.January, 1, 12, 0, 15, 0, time.UTC)) // Set time well past the duration
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
			clockTimeAtAfterCall := clock.Now()

			ch := clock.After(tc.afterDuration)
			require.NotNil(t, ch, "Channel should not be nil")

			// Perform the action (if any) that might trigger the timer
			tc.action(clock)
			// Expected fire time, time + duration.
			expectedFireTimeOnChannel := clockTimeAtAfterCall.Add(tc.afterDuration)

			select {
			case actualTime := <-ch:
				assert.Equal(t, expectedFireTimeOnChannel, actualTime)

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
				sc.SetTime(time.Date(2020, time.January, 1, 12, 0, 5, 0, time.UTC)) // Set time, but not enough
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := NewSimulatedClock(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))

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
