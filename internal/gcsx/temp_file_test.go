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

package gcsx_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestTempFile(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func readAll(rs io.ReadSeeker) (content []byte, err error) {
	_, err = rs.Seek(0, 0)
	if err != nil {
		err = fmt.Errorf("Seek: %w", err)
		return
	}

	content, err = ioutil.ReadAll(rs)
	if err != nil {
		err = fmt.Errorf("ReadFull: %w", err)
		return
	}

	return
}

type dummyReadCloser struct {
	io.Reader
}

func (rc dummyReadCloser) Close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////
// Invariant-checking temp file
////////////////////////////////////////////////////////////////////////

// A wrapper around a TempFile that calls CheckInvariants whenever invariants
// should hold. For catching logic errors early in the test.
type checkingTempFile struct {
	wrapped gcsx.TempFile
}

func (tf *checkingTempFile) Stat() (gcsx.StatResult, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Stat()
}

func (tf *checkingTempFile) Read(b []byte) (int, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Read(b)
}

func (tf *checkingTempFile) Seek(offset int64, whence int) (int64, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Seek(offset, whence)
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

func (tf *checkingTempFile) SetMtime(mtime time.Time) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	tf.wrapped.SetMtime(mtime)
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
	var err error
	t.ctx = ti.Ctx

	// Set up the clock.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	// And the temp file.
	t.tf.wrapped, err = gcsx.NewTempFile(
		dummyReadCloser{strings.NewReader(initialContent)},
		"",
		&t.clock)

	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TempFileTest) Stat_InitialState() {
	sr, err := t.tf.Stat()

	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.Size)
	ExpectEq(initialContentSize, sr.DirtyThreshold)
	ExpectEq(nil, sr.Mtime)
}

func (t *TempFileTest) ReadAt() {
	// Call
	var buf [2]byte
	n, err := t.tf.ReadAt(buf[:], 1)

	ExpectEq(2, n)
	ExpectEq(nil, err)
	ExpectEq(initialContent[1:3], string(buf[:]))

	n, err = t.tf.ReadAt(buf[:], int64(initialContentSize)-1)
	ExpectEq(1, n)
	ExpectEq(io.EOF, err)
	ExpectEq(
		initialContent[initialContentSize-1:initialContentSize],
		string(buf[0:n]),
	)

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
	expected := []byte(initialContent)
	expected[1] = 'f'
	expected[2] = 'o'

	actual, err := readAll(&t.tf)
	AssertEq(nil, err)
	ExpectEq(string(expected), string(actual))
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

	actual, err := readAll(&t.tf)
	AssertEq(nil, err)
	ExpectEq(expected, string(actual))
}

func (t *TempFileTest) SetMtime() {
	mtime := time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local)
	AssertThat(mtime, Not(timeutil.TimeEq(t.clock.Now())))

	// Set.
	t.tf.SetMtime(mtime)

	// Check.
	sr, err := t.tf.Stat()

	AssertEq(nil, err)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(mtime)))
}
