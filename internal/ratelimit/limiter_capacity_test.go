// Copyright 2023 Google Inc. All Rights Reserved.
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

package ratelimit

import (
	"fmt"
	"testing"
	"time"

	. "github.com/jacobsa/ogletest"
)

func TestLimiterCapacity(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type LimiterCapacityTest struct {
}

func init() { RegisterTestSuite(&LimiterCapacityTest{}) }

func rateLessThanOrEqualToZero(rate float64) {
	_, err := ChooseLimiterCapacity(rate, 30)

	expectedError := fmt.Errorf("Illegal rate: %f", rate)

	AssertEq(expectedError.Error(), err.Error())
}

func (t *LimiterCapacityTest) TestRateLessThanZero() {
	var negativeRateHz float64 = -1

	rateLessThanOrEqualToZero(negativeRateHz)
}

func (t *LimiterCapacityTest) TestRateEqualToZero() {
	var zeroRateHz float64 = 0

	rateLessThanOrEqualToZero(zeroRateHz)
}

func windowLessThanOrEqualToZero(window time.Duration) {
	_, err := ChooseLimiterCapacity(1, window)

	expectedError := fmt.Errorf("Illegal window: %v", window)

	AssertEq(expectedError.Error(), err.Error())
}

func (t *LimiterCapacityTest) TestWindowLessThanZero() {
	var negativeWindow time.Duration = -1

	windowLessThanOrEqualToZero(negativeWindow)
}

func (t *LimiterCapacityTest) TestWindowEqualToZero() {
	var zeroWindow time.Duration = 0

	windowLessThanOrEqualToZero(zeroWindow)
}

func (t *LimiterCapacityTest) TestCapacityEqualToZero() {
	var rate = 0.5
	var window time.Duration = 1

	capacity, err := ChooseLimiterCapacity(rate, window)

	expectedError := fmt.Errorf(
		"Can't use a token bucket to limit to %f Hz over a window of %v (result is a capacity of %f)", rate, window, float64(capacity))
	AssertEq(expectedError.Error(), err.Error())
}

func (t *LimiterCapacityTest) TestExpectedCapacity() {
	var rate float64 = 20
	var window = 10 * time.Second

	capacity, err := ChooseLimiterCapacity(rate, window)
	// capacity = floor((20.0 * 10)/50) = floor(4.0) = 4

	ExpectEq(nil, err)
	ExpectEq(4, capacity)
}
