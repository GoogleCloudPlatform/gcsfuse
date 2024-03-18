// Copyright 2015 Google Inc. All Rights Reserved.
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

package percentile_test

import (
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/benchmarks/internal/percentile"
	. "github.com/jacobsa/ogletest"
)

func TestDuration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type DurationTest struct {
}

func init() { RegisterTestSuite(&DurationTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DurationTest) OneObservation() {
	vals := []time.Duration{
		17,
	}

	testCases := []struct {
		p        int
		expected time.Duration
	}{
		{0, 17},
		{1, 17},
		{10, 17},
		{50, 17},
		{90, 17},
		{99, 17},
		{100, 17},
	}

	for _, tc := range testCases {
		ExpectEq(
			tc.expected,
			percentile.Duration(vals, tc.p),
			"p: %d", tc.p)
	}
}

func (t *DurationTest) TwoObservations() {
	vals := []time.Duration{
		100,
		200,
	}

	testCases := []struct {
		p        int
		expected time.Duration
	}{
		{0, 100},
		{1, 101},
		{10, 110},
		{50, 150},
		{90, 190},
		{99, 199},
		{100, 200},
	}

	for _, tc := range testCases {
		ExpectEq(
			tc.expected,
			percentile.Duration(vals, tc.p),
			"p: %d", tc.p)
	}
}

func (t *DurationTest) ThreeObservations() {
	vals := []time.Duration{
		100,
		200,
		300,
	}

	testCases := []struct {
		p        int
		expected time.Duration
	}{
		{0, 100},
		{1, 102},
		{10, 120},
		{50, 200},
		{90, 280},
		{99, 298},
		{100, 300},
	}

	for _, tc := range testCases {
		ExpectEq(
			tc.expected,
			percentile.Duration(vals, tc.p),
			"p: %d", tc.p)
	}
}

func (t *DurationTest) FiveObservations() {
	vals := []time.Duration{
		100,
		200,
		300,
		500,
		1000,
	}

	testCases := []struct {
		p        int
		expected time.Duration
	}{
		{0, 100},
		{5, 120},
		{10, 140},
		{15, 160},
		{20, 180},

		{25, 200},
		{30, 220},
		{35, 240},
		{40, 260},
		{45, 280},

		{50, 300},
		{55, 340},
		{60, 380},
		{65, 420},
		{70, 460},

		{75, 500},
		{80, 600},
		{85, 700},
		{90, 800},

		{100, 1000},
	}

	for _, tc := range testCases {
		ExpectEq(
			tc.expected,
			percentile.Duration(vals, tc.p),
			"p: %d", tc.p)
	}
}
