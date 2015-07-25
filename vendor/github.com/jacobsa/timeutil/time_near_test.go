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

package timeutil_test

import (
	"testing"
	"time"

	"github.com/jacobsa/timeutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestTimeNear(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type TimeNearTest struct {
}

func init() { RegisterTestSuite(&TimeNearTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TimeNearTest) Description() {
	expected := time.Now()
	matcher := timeutil.TimeNear(expected, 2*time.Second)

	desc := matcher.Description()
	ExpectThat(desc, HasSubstr("within 2s"))
	ExpectThat(desc, HasSubstr(expected.String()))
}

func (t *TimeNearTest) ActualIsNotATime() {
	matcher := timeutil.TimeNear(time.Now(), time.Second)
	var err error

	// nil
	err = matcher.Matches(nil)
	AssertNe(nil, err)
	ExpectEq("which is not a time", err.Error())

	// string
	err = matcher.Matches("foo")
	AssertNe(nil, err)
	ExpectEq("which is not a time", err.Error())
}

func (t *TimeNearTest) WithinRadius() {
	const radius = 100 * time.Millisecond
	expected := time.Now().Round(time.Second)
	matcher := timeutil.TimeNear(expected, radius)
	var err error

	// Left edge
	err = matcher.Matches(expected.Add(-radius + time.Nanosecond))
	ExpectEq(nil, err)

	// Left of center
	err = matcher.Matches(expected.Add(-time.Nanosecond))
	ExpectEq(nil, err)

	// Center
	err = matcher.Matches(expected)
	ExpectEq(nil, err)

	// Right of center
	err = matcher.Matches(expected.Add(time.Nanosecond))
	ExpectEq(nil, err)

	// Left edge
	err = matcher.Matches(expected.Add(radius - time.Nanosecond))
	ExpectEq(nil, err)
}

func (t *TimeNearTest) AtRadius() {
	const radius = 100 * time.Millisecond
	expected := time.Now().Round(time.Second)
	matcher := timeutil.TimeNear(expected, radius)
	var err error

	// Below
	err = matcher.Matches(expected.Add(-radius))

	AssertNe(nil, err)
	ExpectEq("which differs by 100ms", err.Error())

	// Above
	err = matcher.Matches(expected.Add(radius))

	AssertNe(nil, err)
	ExpectEq("which differs by 100ms", err.Error())
}

func (t *TimeNearTest) OutsideOfRadius() {
	const radius = 100 * time.Millisecond
	expected := time.Now().Round(time.Second)
	matcher := timeutil.TimeNear(expected, radius)
	var err error

	// Below
	err = matcher.Matches(expected.Add(-radius - time.Millisecond))

	AssertNe(nil, err)
	ExpectEq("which differs by 101ms", err.Error())

	// Above
	err = matcher.Matches(expected.Add(radius + time.Millisecond))

	AssertNe(nil, err)
	ExpectEq("which differs by 101ms", err.Error())
}
