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

package lease_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/lease"
	. "github.com/jacobsa/ogletest"
)

func TestMultiReadProxy(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Information returned by our fake refreshers.
type refresherInfo struct {
	contents string
	err      error
}

// A ReadProxy that wraps another, calling CheckInvariants before and after
// each action.
type checkingReadProxy struct {
	ctx     context.Context
	wrapped lease.ReadProxy
}

func (crp *checkingReadProxy) Destroy() {
	crp.wrapped.CheckInvariants()
	defer crp.wrapped.CheckInvariants()

	crp.wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MultiReadProxyTest struct {
	ctx context.Context

	// Canned info returned by the refreshers.
	info []refresherInfo

	leaser lease.FileLeaser
	proxy  *checkingReadProxy
}

var _ SetUpInterface = &MultiReadProxyTest{}
var _ TearDownInterface = &MultiReadProxyTest{}

func init() { RegisterTestSuite(&MultiReadProxyTest{}) }

func (t *MultiReadProxyTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	panic("TODO")
}

func (t *MultiReadProxyTest) TearDown() {
	// Make sure nothing goes crazy.
	t.proxy.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MultiReadProxyTest) NoRefreshers() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) Size() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) ReadAt_OneRefresherReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) ReadAt_AllSuccessful() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) ReadAt_ContentAlreadyCached() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) Upgrade_OneRefresherReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) Upgrade_AllSuccessful() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) Upgrade_ContentAlreadyCached() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) InitialReadLeaseValid() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) InitialReadLeaseRevoked() {
	AssertTrue(false, "TODO")
}
