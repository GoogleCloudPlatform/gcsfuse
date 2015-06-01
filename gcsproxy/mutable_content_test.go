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

package gcsproxy_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease/mock_lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestMutableContent(t *testing.T) { RunTests(t) }

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
// Invariant-checking mutable content
////////////////////////////////////////////////////////////////////////

// A wrapper around MutableContent that calls CheckInvariants whenever
// invariants should hold. For catching logic errors early in the test.
type checkingMutableContent struct {
	ctx     context.Context
	wrapped *gcsproxy.MutableContent
}

func (mc *checkingMutableContent) Stat() (gcsproxy.StatResult, error) {
	mc.wrapped.CheckInvariants()
	defer mc.wrapped.CheckInvariants()
	return mc.wrapped.Stat(mc.ctx)
}

func (mc *checkingMutableContent) ReadAt(b []byte, o int64) (int, error) {
	mc.wrapped.CheckInvariants()
	defer mc.wrapped.CheckInvariants()
	return mc.wrapped.ReadAt(mc.ctx, b, o)
}

func (mc *checkingMutableContent) WriteAt(b []byte, o int64) (int, error) {
	mc.wrapped.CheckInvariants()
	defer mc.wrapped.CheckInvariants()
	return mc.wrapped.WriteAt(mc.ctx, b, o)
}

func (mc *checkingMutableContent) Truncate(n int64) error {
	mc.wrapped.CheckInvariants()
	defer mc.wrapped.CheckInvariants()
	return mc.wrapped.Truncate(mc.ctx, n)
}

func (mc *checkingMutableContent) Destroy() {
	mc.wrapped.CheckInvariants()
	defer mc.wrapped.CheckInvariants()
	mc.wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const initialContentSize = 11

type mutableContentTest struct {
	ctx context.Context

	initialContent mock_lease.MockReadProxy
	rwl            mock_lease.MockReadWriteLease
	clock          timeutil.SimulatedClock

	mc checkingMutableContent
}

var _ SetUpInterface = &mutableContentTest{}

func (t *mutableContentTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the mock initial contents, including a default size.
	t.initialContent = mock_lease.NewMockReadProxy(
		ti.MockController,
		"initialContent")

	ExpectCall(t.initialContent, "Size")().
		WillRepeatedly(Return(initialContentSize))

	// Set up a mock read/write lease.
	t.rwl = mock_lease.NewMockReadWriteLease(
		ti.MockController,
		"rwl")

	// Ignore uninteresting calls.
	ExpectCall(t.initialContent, "CheckInvariants")().
		WillRepeatedly(Return())

	// Set up the clock.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	// And the mutable content.
	t.mc.ctx = ti.Ctx
	t.mc.wrapped = gcsproxy.NewMutableContent(
		t.initialContent,
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
	t.mc.ReadAt(buf, 17)
}

func (t *CleanTest) ReadAt_ProxyFails() {
	// Proxy
	ExpectCall(t.initialContent, "ReadAt")(Any(), Any(), Any()).
		WillOnce(Return(17, errors.New("taco")))

	// Call
	n, err := t.mc.ReadAt(make([]byte, 1), 0)

	ExpectEq(17, n)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CleanTest) ReadAt_ProxySuceeds() {
	// Proxy
	ExpectCall(t.initialContent, "ReadAt")(Any(), Any(), Any()).
		WillOnce(Return(17, nil))

	// Call
	n, err := t.mc.ReadAt(make([]byte, 1), 0)

	ExpectEq(17, n)
	ExpectEq(nil, err)
}

func (t *CleanTest) Stat() {
	sr, err := t.mc.Stat()

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
	_, err := t.mc.WriteAt(make([]byte, 1), 0)

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
	t.mc.WriteAt(make([]byte, 1), 17)

	// A further call should go right through to the read/write lease again.
	ExpectCall(t.rwl, "WriteAt")(Any(), 19).
		WillOnce(Return(0, errors.New("")))

	t.mc.WriteAt(make([]byte, 1), 19)
}

func (t *CleanTest) Truncate_UpgradeFails() {
	// Upgrade
	ExpectCall(t.initialContent, "Upgrade")(Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	err := t.mc.Truncate(0)

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
	t.mc.Truncate(17)

	// A further call should go right through to the read/write lease again.
	ExpectCall(t.rwl, "Truncate")(19).
		WillOnce(Return(errors.New("")))

	t.mc.Truncate(19)
}

////////////////////////////////////////////////////////////////////////
// Dirty state
////////////////////////////////////////////////////////////////////////

type DirtyTest struct {
	mutableContentTest
}

func init() { RegisterTestSuite(&DirtyTest{}) }

func (t *DirtyTest) SetUp(ti *TestInfo) {
	t.mutableContentTest.SetUp(ti)

	// Simulate a successful upgrade.
	ExpectCall(t.initialContent, "Upgrade")(Any()).
		WillOnce(Return(t.rwl, nil))

	ExpectCall(t.rwl, "Truncate")(Any()).
		WillOnce(Return(nil))

	err := t.mc.Truncate(initialContentSize)
	AssertEq(nil, err)
}

func (t *DirtyTest) ReadAt_CallsLease() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) ReadAt_LeaseFails() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) ReadAt_LeaseSuceeds() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) Stat_CallsLease() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) Stat_LeaseFails() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) Stat_LeaseSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) WriteAt_CallsLease() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) WriteAt_LeaseFails() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) WriteAt_LeaseSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) WriteAt_DirtyThreshold() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) Truncate_CallsLease() {
	// Lease
	ExpectCall(t.rwl, "Truncate")(17).
		WillOnce(Return(errors.New("")))

	// Call
	t.mc.Truncate(17)
}

func (t *DirtyTest) Truncate_LeaseFails() {
	// Lease
	ExpectCall(t.rwl, "Truncate")(Any()).
		WillOnce(Return(errors.New("taco")))

	// Call
	err := t.mc.Truncate(0)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *DirtyTest) Truncate_LeaseSucceeds() {
	// Lease
	ExpectCall(t.rwl, "Truncate")(Any()).
		WillOnce(Return(nil))

	// Call
	err := t.mc.Truncate(0)
	ExpectEq(nil, err)
}

func (t *DirtyTest) Truncate_DirtyThreshold() {
	AssertTrue(false, "TODO")
}
