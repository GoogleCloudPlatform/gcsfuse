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
	"io"
	"io/ioutil"
	"math"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/lease"
	. "github.com/jacobsa/ogletest"
)

func TestMultiReadProxy(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// A ReadProxy that wraps another, calling CheckInvariants before and after
// each action.
type checkingReadProxy struct {
	Ctx     context.Context
	Wrapped lease.ReadProxy
}

func (crp *checkingReadProxy) Size() (size int64) {
	crp.Wrapped.CheckInvariants()
	defer crp.Wrapped.CheckInvariants()

	size = crp.Wrapped.Size()
	return
}

func (crp *checkingReadProxy) Destroy() {
	crp.Wrapped.CheckInvariants()
	crp.Wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

// Canned contents returned by the refreshers.
var refresherContents = []string{
	"taco",
	"burrito",
	"enchilada",
}

type MultiReadProxyTest struct {
	ctx context.Context

	// Canned errors returned by the refreshers.
	refresherErrors []error

	leaser lease.FileLeaser
	proxy  *checkingReadProxy
}

var _ SetUpInterface = &MultiReadProxyTest{}
var _ TearDownInterface = &MultiReadProxyTest{}

func init() { RegisterTestSuite(&MultiReadProxyTest{}) }

func (t *MultiReadProxyTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.leaser = lease.NewFileLeaser("", math.MaxInt64)
	t.refresherErrors = make([]error, len(refresherContents))

	// Create the proxy.
	t.proxy = &checkingReadProxy{
		Ctx: t.ctx,
		Wrapped: lease.NewMultiReadProxy(
			t.leaser,
			t.makeRefreshers(),
			nil),
	}
}

func (t *MultiReadProxyTest) TearDown() {
	// Make sure nothing goes crazy.
	t.proxy.Destroy()
}

func (t *MultiReadProxyTest) makeRefreshers() (refreshers []lease.Refresher) {
	for i, contents := range refresherContents {
		iCopy := i
		r := &funcRefresher{
			N: int64(len(contents)),
			F: func(ctx context.Context) (rc io.ReadCloser, err error) {
				rc = ioutil.NopCloser(strings.NewReader(contents))
				err = t.refresherErrors[iCopy]
				return
			},
		}

		refreshers = append(refreshers, r)
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MultiReadProxyTest) NoRefreshers() {
	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) Size() {
	var expected int64
	for _, contents := range refresherContents {
		expected += int64(len(contents))
	}

	ExpectEq(expected, t.proxy.Size())
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
