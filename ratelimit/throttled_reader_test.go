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

package ratelimit_test

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestThrottledReader(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ThrottledReaderTest struct {
}

func init() { RegisterTestSuite(&ThrottledReaderTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ThrottledReaderTest) CallsThrottle() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) ThrottleSaysCancelled() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) CallsWrapped() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsEOF() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsFullRead() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_CallsAgain() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_SecondFails() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_SecondSuceeds() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) ReadSizeIsAboveThrottleCapacity() {
	AssertTrue(false, "TODO")
}
