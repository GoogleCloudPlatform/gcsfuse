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

func TestTimeEq(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type TimeEqTest struct {
}

func init() { RegisterTestSuite(&TimeEqTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TimeEqTest) Description() {
	expected := time.Now()
	matcher := timeutil.TimeEq(expected)
	ExpectEq(expected.String(), matcher.Description())
}

func (t *TimeEqTest) ActualIsNotATime() {
	expected := time.Now()
	matcher := timeutil.TimeEq(expected)
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

func (t *TimeEqTest) ActualAndExpectedDontMatch() {
	expected := time.Now().Round(time.Second)
	matcher := timeutil.TimeEq(expected)
	var err error

	// actual before expected
	err = matcher.Matches(expected.Add(-2 * time.Second))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("off by -2s")))

	// actual after expected
	err = matcher.Matches(expected.Add(2 * time.Second))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("off by 2s")))
}

func (t *TimeEqTest) ActualAndExpectedMatch() {
	expected := time.Now().Round(time.Second)
	matcher := timeutil.TimeEq(expected)

	err := matcher.Matches(expected)
	ExpectEq(nil, err)
}
