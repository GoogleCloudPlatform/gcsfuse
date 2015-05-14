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
	"errors"
	"io"
	"io/ioutil"
	"math"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/lease"
	. "github.com/jacobsa/oglematchers"
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
	AssertThat(
		refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	AssertEq(4, len(refresherContents[0]))
	AssertEq(7, len(refresherContents[1]))
	AssertEq(9, len(refresherContents[2]))

	// Configure an error for the middle read lease.
	someErr := errors.New("foobar")
	t.refresherErrors[1] = someErr

	// Various ranges to read, the contents we expect to get back, and the
	// error we expect to see, if any.
	testCases := []struct {
		start            int64
		limit            int64
		expectedErr      error
		expectedContents string
	}{
		// First read lease only.
		{0, 0, nil, ""},
		{0, 1, nil, "t"},
		{0, 4, nil, "taco"},
		{1, 4, nil, "aco"},
		{4, 4, nil, ""},

		// First and second read leases.
		{0, 5, someErr, "taco"},
		{1, 11, someErr, "aco"},

		// All read leases.
		{0, 20, someErr, "taco"},
		{1, 20, someErr, "aco"},
		{1, 100, someErr, "aco"},

		// Second read lease only.
		{4, 4, nil, ""},
		{4, 5, someErr, ""},
		{4, 11, someErr, ""},

		// Second and third read leases.
		{4, 12, someErr, ""},
		{4, 20, someErr, ""},
		{5, 100, someErr, ""},

		// Third read lease only.
		{11, 20, nil, "enchilada"},
		{11, 100, io.EOF, "enchilada"},
		{12, 20, nil, "nchilada"},
		{19, 20, nil, "a"},
		{20, 20, nil, ""},

		// Past end.
		{21, 21, nil, ""},
		{21, 22, io.EOF, ""},
		{21, 100, io.EOF, ""},
		{100, 1000, io.EOF, ""},
	}

	AssertTrue(false, "TODO")
}

func (t *MultiReadProxyTest) ReadAt_AllSuccessful() {
	AssertThat(
		refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	AssertEq(4, len(refresherContents[0]))
	AssertEq(7, len(refresherContents[1]))
	AssertEq(9, len(refresherContents[2]))

	// Various ranges to read, the contents we expect to get back, and the
	// error we expect to see, if any.
	testCases := []struct {
		start            int64
		limit            int64
		expectedErr      error
		expectedContents string
	}{
		// First read lease only.
		{0, 0, nil, ""},
		{0, 1, nil, "t"},
		{0, 4, nil, "taco"},
		{1, 4, nil, "aco"},
		{4, 4, nil, ""},

		// First and second read leases.
		{0, 5, nil, "tacob"},
		{1, 11, nil, "acoburrito"},

		// All read leases.
		{0, 20, nil, "tacoburritoenchilada"},
		{1, 19, nil, "acoburritoenchilad"},
		{3, 17, nil, "oburritoenchil"},

		// Second read lease only.
		{4, 4, nil, ""},
		{4, 5, nil, "b"},
		{4, 11, nil, "burrito"},

		// Second and third read leases.
		{4, 12, nil, "burritoe"},
		{4, 20, nil, "burritoenchilada"},
		{5, 100, io.EOF, "urritoenchilada"},

		// Third read lease only.
		{11, 20, nil, "enchilada"},
		{11, 100, io.EOF, "enchilada"},
		{12, 20, nil, "nchilada"},
		{19, 20, nil, "a"},
		{20, 20, nil, ""},

		// Past end.
		{21, 21, nil, ""},
		{21, 22, io.EOF, ""},
		{21, 100, io.EOF, ""},
		{100, 1000, io.EOF, ""},
	}

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
