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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type LimiterCapacityTest struct {
	suite.Suite
}

func TestLimiterCapacitySuite(t *testing.T) {
	suite.Run(t, new(LimiterCapacityTest))
}

func rateLessThanOrEqualToZero(t *testing.T, rate float64) {
	_, err := ChooseLimiterCapacity(rate, 30)

	expectedError := fmt.Sprintf("Illegal rate: %f", rate)

	assert.EqualError(t, err, expectedError)
}

func (t *LimiterCapacityTest) TestRateLessThanZero() {
	var negativeRateHz float64 = -1

	rateLessThanOrEqualToZero(t.T(), negativeRateHz)
}

func (t *LimiterCapacityTest) TestRateEqualToZero() {
	var zeroRateHz float64 = 0

	rateLessThanOrEqualToZero(t.T(), zeroRateHz)
}

func windowLessThanOrEqualToZero(t *testing.T, window time.Duration) {
	_, err := ChooseLimiterCapacity(1, window)

	expectedError := fmt.Sprintf("Illegal window: %v", window)

	assert.EqualError(t, err, expectedError)
}

func (t *LimiterCapacityTest) TestWindowLessThanZero() {
	var negativeWindow time.Duration = -1

	windowLessThanOrEqualToZero(t.T(), negativeWindow)
}

func (t *LimiterCapacityTest) TestWindowEqualToZero() {
	var zeroWindow time.Duration = 0

	windowLessThanOrEqualToZero(t.T(), zeroWindow)
}

func (t *LimiterCapacityTest) TestCapacityEqualToZero() {
	var rate = 0.5
	var window time.Duration = 1

	capacity, err := ChooseLimiterCapacity(rate, window)

	expectedError := fmt.Sprintf(
		"Can't use a token bucket to limit to %f Hz over a window of %v (result is a capacity of %f)", rate, window, float64(capacity))
	assert.EqualError(t.T(), err, expectedError)
}

func (t *LimiterCapacityTest) TestExpectedCapacity() {
	var rate float64 = 20
	var window = 10 * time.Second

	capacity, err := ChooseLimiterCapacity(rate, window)
	// capacity = floor((20.0 * 10)/50) = floor(4.0) = 4

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(4), capacity)
}
