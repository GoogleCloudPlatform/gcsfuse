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

package mutable_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/mutable"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestTempFile(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func bufferIs(buf []byte) Matcher {
	return NewMatcher(
		func(candidate interface{}) error {
			p := candidate.([]byte)

			// Compare.
			if &buf[0] != &p[0] {
				return fmt.Errorf(
					"Differing first bytes: %p vs. %p",
					&buf[0],
					&p[0])
			}

			if len(buf) != len(p) {
				return fmt.Errorf(
					"Differing lengths: %d vs. %d",
					len(buf),
					len(p))
			}

			return nil
		},
		fmt.Sprintf("Buffer matches"))
}

////////////////////////////////////////////////////////////////////////
// Invariant-checking temp file
////////////////////////////////////////////////////////////////////////

// A wrapper around a TempFile that calls CheckInvariants whenever invariants
// should hold. For catching logic errors early in the test.
type checkingTempFile struct {
	wrapped mutable.TempFile
}

func (tf *checkingTempFile) Stat() (mutable.StatResult, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Stat()
}

func (tf *checkingTempFile) ReadAt(b []byte, o int64) (int, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.ReadAt(b, o)
}

func (tf *checkingTempFile) WriteAt(b []byte, o int64) (int, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.WriteAt(b, o)
}

func (tf *checkingTempFile) Truncate(n int64) error {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Truncate(n)
}

func (tf *checkingTempFile) Destroy() {
	tf.wrapped.CheckInvariants()
	tf.wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const initialContent = "tacoburrito"

const initialContentSize = len(initialContent)

type TempFileTest struct {
	ctx   context.Context
	clock timeutil.SimulatedClock

	tf checkingTempFile
}

func init() { RegisterTestSuite(&TempFileTest{}) }

var _ SetUpInterface = &TempFileTest{}

func (t *TempFileTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the clock.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	// And the temp file.
	t.tf = mutable.NewTempFile(
		strings.NewReader(t.initialContent),
		&t.clock)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TempFileTest) Stat() {
	sr, err := t.tf.Stat()

	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.Size)
	ExpectEq(initialContentSize, sr.DirtyThreshold)
	ExpectEq(nil, sr.Mtime)
}

func (t *TempFileTest) ReadAt() {
	// Call
	var buf [2]byte
	n, err := t.tf.ReadAt(buf, 1)

	ExpectEq(2, n)
	ExpectEq(nil, err)
	ExpectEq(initialContent[1:3], string(buf))

	// Check Stat.
	sr, err := t.tf.Stat()

	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.Size)
	ExpectEq(initialContentSize, sr.DirtyThreshold)
	ExpectEq(nil, sr.Mtime)
}

func (t *TempFileTest) WriteAt() {
	// Call
	p := []byte("fo")
	n, err := t.tf.WriteAt(p, 1)

	ExpectEq(2, n)
	ExpectEq(nil, err)

	// Check Stat.
	sr, err := t.tf.Stat()

	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.Size)
	ExpectEq(1, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(t.clock.Now())))

	// Read back.
	expected := initialContent
	expected[1] = 'f'
	expected[2] = 'o'

	actual, err := readAll(t.tf)
	AssertEq(nil, err)
	ExpectEq(expected, string(actual))
}

func (t *TempFileTest) Truncate() {
	// Call
	err := t.tf.Truncate(2)
	ExpectEq(nil, err)

	// Check Stat.
	sr, err := t.tf.Stat()

	AssertEq(nil, err)
	ExpectEq(2, sr.Size)
	ExpectEq(2, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(t.clock.Now())))

	// Read back.
	expected := initialContent[0:2]

	actual, err := readAll(t.tf)
	AssertEq(nil, err)
	ExpectEq(expected, string(actual))
}
