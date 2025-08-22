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

import "time"

// Implements clock interface. It should be used during tests to mimic waiting.
type FakeClock struct {
	WaitTime time.Duration
}

// Now returns the current time. This implementation uses the real time, making
// this clock a hybrid.
func (mc *FakeClock) Now() time.Time {
	return time.Now()
}

// Notifies on the returned channel after the wait time specified during
// creation of FakeClock.
func (mc *FakeClock) After(time.Duration) <-chan time.Time {
	ch := make(chan time.Time)
	go func() {
		time.Sleep(mc.WaitTime)
		ch <- time.Now()
	}()
	return ch
}
