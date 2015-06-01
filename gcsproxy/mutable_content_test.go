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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease/mock_lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestMutableContent(t *testing.T) { RunTests(t) }

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

type mutableContentTest struct {
	ctx context.Context

	initialContent mock_lease.MockReadProxy
	clock          timeutil.SimulatedClock
	mc             checkingMutableContent
}

var _ SetUpInterface = &mutableContentTest{}

func (t *mutableContentTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the mock initial contents, including a default size.
	t.initialContent = mock_lease.NewMockReadProxy(
		ti.MockController,
		"initialContent")

	const initialContentSize = 11
	ExpectCall(t.initialContent, "Size")().
		WillRepeatedly(Return(initialContentSize))

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
	AssertTrue(false, "TODO")
}

func (t *CleanTest) ReadAt_ProxyFails() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) ReadAt_ProxySuceeds() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) Stat() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) WriteAt_CallsUpgrade() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) WriteAt_UpgradeFails() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) WriteAt_UpgradeSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) Truncate_CallsUpgrade() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) Truncate_UpgradeFails() {
	AssertTrue(false, "TODO")
}

func (t *CleanTest) Truncate_UpgradeSucceeds() {
	AssertTrue(false, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Dirty state
////////////////////////////////////////////////////////////////////////

type DirtyTest struct {
	mutableContentTest
}

func init() { RegisterTestSuite(&DirtyTest{}) }

func (t *DirtyTest) ReadAt_CallsLease() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) ReadAt_LeaseFails() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) ReadAt_LeaseSuceeds() {
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
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) Truncate_LeaseFails() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) Truncate_LeaseSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *DirtyTest) Truncate_DirtyThreshold() {
	AssertTrue(false, "TODO")
}
