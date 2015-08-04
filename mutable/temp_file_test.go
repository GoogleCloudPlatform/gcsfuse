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
	"errors"
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

type mutableTempFileTest struct {
	ctx   context.Context
	clock timeutil.SimulatedClock

	tf checkingTempFile
}

var _ SetUpInterface = &mutableTempFileTest{}

func (t *mutableTempFileTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the clock.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	// And the temp file.
	t.tf = mutable.NewTempFile(
		strings.NewReader(t.initialContent),
		&t.clock)
}

////////////////////////////////////////////////////////////////////////
// Clean state
////////////////////////////////////////////////////////////////////////

type CleanTest struct {
	mutableContentTest
}

func init() { RegisterTestSuite(&CleanTest{}) }

func (t *CleanTest) ReadAt_CallsProxy() {
	buf := make([]byte, 1)

	// Proxy
	ExpectCall(t.initialContent, "ReadAt")(t.ctx, bufferIs(buf), 17).
		WillOnce(Return(0, errors.New("")))

	// Call
	t.tf.ReadAt(buf, 17)
}

func (t *CleanTest) ReadAt_ProxyFails() {
	// Proxy
	ExpectCall(t.initialContent, "ReadAt")(Any(), Any(), Any()).
		WillOnce(Return(17, errors.New("taco")))

	// Call
	n, err := t.tf.ReadAt(make([]byte, 1), 0)

	ExpectEq(17, n)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CleanTest) ReadAt_ProxySuceeds() {
	// Proxy
	ExpectCall(t.initialContent, "ReadAt")(Any(), Any(), Any()).
		WillOnce(Return(17, nil))

	// Call
	n, err := t.tf.ReadAt(make([]byte, 1), 0)

	ExpectEq(17, n)
	ExpectEq(nil, err)
}

func (t *CleanTest) Stat() {
	sr, err := t.tf.Stat()

	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.Size)
	ExpectEq(initialContentSize, sr.DirtyThreshold)
	ExpectEq(nil, sr.Mtime)
}

func (t *CleanTest) WriteAt_UpgradeFails() {
	// Upgrade
	ExpectCall(t.initialContent, "Upgrade")(Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.tf.WriteAt(make([]byte, 1), 0)

	ExpectThat(err, Error(HasSubstr("Upgrade")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CleanTest) WriteAt_UpgradeSucceeds() {
	// Upgrade -- succeed.
	ExpectCall(t.initialContent, "Upgrade")(Any()).
		WillOnce(Return(t.rwl, nil))

	// The read/write lease should be called.
	ExpectCall(t.rwl, "WriteAt")(Any(), 17).
		WillOnce(Return(0, errors.New("")))

	// Call.
	t.tf.WriteAt(make([]byte, 1), 17)

	// A further call should go right through to the read/write lease again.
	ExpectCall(t.rwl, "WriteAt")(Any(), 19).
		WillOnce(Return(0, errors.New("")))

	t.tf.WriteAt(make([]byte, 1), 19)
}

func (t *CleanTest) Truncate_UpgradeFails() {
	// Upgrade
	ExpectCall(t.initialContent, "Upgrade")(Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	err := t.tf.Truncate(0)

	ExpectThat(err, Error(HasSubstr("Upgrade")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CleanTest) Truncate_UpgradeSucceeds() {
	// Upgrade -- succeed.
	ExpectCall(t.initialContent, "Upgrade")(Any()).
		WillOnce(Return(t.rwl, nil))

	// The read/write lease should be called.
	ExpectCall(t.rwl, "Truncate")(17).
		WillOnce(Return(errors.New("")))

	// Call.
	t.tf.Truncate(17)

	// A further call should go right through to the read/write lease again.
	ExpectCall(t.rwl, "Truncate")(19).
		WillOnce(Return(errors.New("")))

	t.tf.Truncate(19)
}

func (t *CleanTest) Release() {
	rwl := t.tf.Release()
	ExpectEq(nil, rwl)
}

////////////////////////////////////////////////////////////////////////
// Dirty state
////////////////////////////////////////////////////////////////////////

type DirtyTest struct {
	mutableContentTest

	setUpTime time.Time
}

func init() { RegisterTestSuite(&DirtyTest{}) }

func (t *DirtyTest) SetUp(ti *TestInfo) {
	t.mutableContentTest.SetUp(ti)
	t.setUpTime = t.clock.Now()

	// Simulate a successful upgrade.
	ExpectCall(t.initialContent, "Upgrade")(Any()).
		WillOnce(Return(t.rwl, nil))

	ExpectCall(t.rwl, "Truncate")(Any()).
		WillOnce(Return(nil))

	err := t.tf.Truncate(initialContentSize)
	AssertEq(nil, err)

	// Change the time.
	t.clock.AdvanceTime(time.Second)
}

func (t *DirtyTest) ReadAt_CallsLease() {
	buf := make([]byte, 4)
	const offset = 17

	// Lease
	ExpectCall(t.rwl, "ReadAt")(bufferIs(buf), offset).
		WillOnce(Return(0, errors.New("")))

	// Call
	t.tf.ReadAt(buf, offset)
}

func (t *DirtyTest) ReadAt_LeaseFails() {
	// Lease
	ExpectCall(t.rwl, "ReadAt")(Any(), Any()).
		WillOnce(Return(13, errors.New("taco")))

	// Call
	n, err := t.tf.ReadAt([]byte{}, 0)

	ExpectEq(13, n)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *DirtyTest) ReadAt_LeaseSuceeds() {
	// Lease
	ExpectCall(t.rwl, "ReadAt")(Any(), Any()).
		WillOnce(Return(13, nil))

	// Call
	n, err := t.tf.ReadAt([]byte{}, 0)

	ExpectEq(13, n)
	ExpectEq(nil, err)
}

func (t *DirtyTest) Stat_LeaseFails() {
	// Lease
	ExpectCall(t.rwl, "Size")().
		WillOnce(Return(0, errors.New("taco")))

	// Call
	_, err := t.tf.Stat()
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *DirtyTest) Stat_LeaseSucceeds() {
	// Lease
	ExpectCall(t.rwl, "Size")().
		WillOnce(Return(17, nil))

	// Call
	sr, err := t.tf.Stat()
	AssertEq(nil, err)

	// Check the initial state.
	ExpectEq(17, sr.Size)
	ExpectEq(initialContentSize, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(t.setUpTime)))
}

func (t *DirtyTest) WriteAt_CallsLease() {
	buf := make([]byte, 4)
	const offset = 17

	// Lease
	ExpectCall(t.rwl, "WriteAt")(bufferIs(buf), offset).
		WillOnce(Return(0, errors.New("")))

	// Call
	t.tf.WriteAt(buf, offset)
}

func (t *DirtyTest) WriteAt_LeaseFails() {
	const offset = initialContentSize - 2

	// Lease
	ExpectCall(t.rwl, "WriteAt")(Any(), Any()).
		WillOnce(Return(13, errors.New("taco")))

	// Call
	n, err := t.tf.WriteAt([]byte{}, offset)

	ExpectEq(13, n)
	ExpectThat(err, Error(HasSubstr("taco")))

	// The dirty threshold and mtime should have been updated.
	ExpectCall(t.rwl, "Size")().
		WillRepeatedly(Return(initialContentSize, nil))

	sr, err := t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(offset, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(t.clock.Now())))
}

func (t *DirtyTest) WriteAt_LeaseSucceeds() {
	const offset = initialContentSize - 2

	// Lease
	ExpectCall(t.rwl, "WriteAt")(Any(), Any()).
		WillOnce(Return(13, nil))

	// Call
	n, err := t.tf.WriteAt([]byte{}, offset)

	ExpectEq(13, n)
	ExpectEq(nil, err)

	// The dirty threshold and mtime should have been updated.
	ExpectCall(t.rwl, "Size")().
		WillRepeatedly(Return(initialContentSize, nil))

	sr, err := t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(offset, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(t.clock.Now())))
}

func (t *DirtyTest) WriteAt_DirtyThreshold() {
	var sr mutable.StatResult
	var err error

	// Simulate successful writes and size requests.
	ExpectCall(t.rwl, "WriteAt")(Any(), Any()).
		WillRepeatedly(Return(0, nil))

	ExpectCall(t.rwl, "Size")().
		WillRepeatedly(Return(100, nil))

	// Writing at the end of the initial content should not affect the dirty
	// threshold.
	_, err = t.tf.WriteAt([]byte{}, initialContentSize)
	AssertEq(nil, err)

	sr, err = t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.DirtyThreshold)

	// Nor should writing past the end.
	_, err = t.tf.WriteAt([]byte{}, initialContentSize+100)
	AssertEq(nil, err)

	sr, err = t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.DirtyThreshold)

	// But writing before the end should.
	_, err = t.tf.WriteAt([]byte{}, initialContentSize-1)
	AssertEq(nil, err)

	sr, err = t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(initialContentSize-1, sr.DirtyThreshold)
}

func (t *DirtyTest) Truncate_CallsLease() {
	// Lease
	ExpectCall(t.rwl, "Truncate")(17).
		WillOnce(Return(errors.New("")))

	// Call
	t.tf.Truncate(17)
}

func (t *DirtyTest) Truncate_LeaseFails() {
	// Lease
	ExpectCall(t.rwl, "Truncate")(Any()).
		WillOnce(Return(errors.New("taco")))

	// Call
	err := t.tf.Truncate(1)
	ExpectThat(err, Error(HasSubstr("taco")))

	// The dirty threshold and mtime should have been updated.
	ExpectCall(t.rwl, "Size")().
		WillRepeatedly(Return(0, nil))

	sr, err := t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(1, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(t.clock.Now())))
}

func (t *DirtyTest) Truncate_LeaseSucceeds() {
	// Lease
	ExpectCall(t.rwl, "Truncate")(Any()).
		WillOnce(Return(nil))

	// Call
	err := t.tf.Truncate(1)
	ExpectEq(nil, err)

	// The dirty threshold and mtime should have been updated.
	ExpectCall(t.rwl, "Size")().
		WillRepeatedly(Return(0, nil))

	sr, err := t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(1, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(t.clock.Now())))
}

func (t *DirtyTest) Truncate_DirtyThreshold() {
	var sr mutable.StatResult
	var err error

	// Simulate successful truncations and size requests.
	ExpectCall(t.rwl, "Truncate")(Any()).
		WillRepeatedly(Return(nil))

	ExpectCall(t.rwl, "Size")().
		WillRepeatedly(Return(100, nil))

	// Truncating to the same size should not affect the dirty threshold.
	err = t.tf.Truncate(initialContentSize)
	AssertEq(nil, err)

	sr, err = t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.DirtyThreshold)

	// Nor should truncating upward.
	err = t.tf.Truncate(initialContentSize + 100)
	AssertEq(nil, err)

	sr, err = t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(initialContentSize, sr.DirtyThreshold)

	// But truncating downward should.
	err = t.tf.Truncate(initialContentSize - 1)
	AssertEq(nil, err)

	sr, err = t.tf.Stat()
	AssertEq(nil, err)
	ExpectEq(initialContentSize-1, sr.DirtyThreshold)
}

func (t *DirtyTest) Release() {
	rwl := t.tf.Release()
	ExpectEq(t.rwl, rwl)
}
