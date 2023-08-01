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

	. "github.com/jacobsa/ogletest"
)

const MaxFloat64 = 0x1p1023 * (1 + (1 - 0x1p-52))

func TestChooseLimiterCapacity(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ChooseLimiterCapacityTest struct {
}

func init() { RegisterTestSuite(&ChooseLimiterCapacityTest{}) }
func (t *ChooseLimiterCapacityTest) TestRateLessThanOrEqualToZero() {
	var negativeRateHz float64 = -1
	var zeroRateHz float64 = 0

	_, err := ChooseLimiterCapacity(negativeRateHz, 30)

	expectedError := fmt.Errorf("Illegal rate: %f", negativeRateHz)
	AssertEq(expectedError.Error(), err.Error())

	_, err = ChooseLimiterCapacity(zeroRateHz, 30)

	expectedError = fmt.Errorf("Illegal rate: %f", zeroRateHz)
	AssertEq(expectedError.Error(), err.Error())
}

func (t *ChooseLimiterCapacityTest) TestRateEqualToInfinity() {
	var negativeRateHz float64 = MaxFloat64
	var zeroRateHz float64 = -MaxFloat64

	_, err := ChooseLimiterCapacity(negativeRateHz, 30)

	expectedError := fmt.Errorf("Illegal rate: %f", negativeRateHz)
	AssertEq(expectedError.Error(), err.Error())

	_, err = ChooseLimiterCapacity(zeroRateHz, 30)

	expectedError = fmt.Errorf("Illegal rate: %f", zeroRateHz)
	AssertEq(expectedError.Error(), err.Error())
}
