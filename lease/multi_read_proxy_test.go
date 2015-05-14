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
	Wrapped lease.ReadProxy
}

func (crp *checkingReadProxy) Size() (size int64) {
	crp.Wrapped.CheckInvariants()
	defer crp.Wrapped.CheckInvariants()

	size = crp.Wrapped.Size()
	return
}

func (crp *checkingReadProxy) ReadAt(
	ctx context.Context,
	p []byte,
	off int64) (n int, err error) {
	crp.Wrapped.CheckInvariants()
	defer crp.Wrapped.CheckInvariants()

	n, err = crp.Wrapped.ReadAt(ctx, p, off)
	return
}

func (crp *checkingReadProxy) Upgrade(
	ctx context.Context) (rwl lease.ReadWriteLease, err error) {
	crp.Wrapped.CheckInvariants()
	defer crp.Wrapped.CheckInvariants()

	rwl, err = crp.Wrapped.Upgrade(ctx)
	return
}

func (crp *checkingReadProxy) Destroy() {
	crp.Wrapped.CheckInvariants()
	crp.Wrapped.Destroy()
}

func (crp *checkingReadProxy) CheckInvariants() {
	crp.Wrapped.CheckInvariants()
}

// A range to read, the contents we expect to get back, and the error we expect
// to see, if any.
type readAtTestCase struct {
	start            int64
	limit            int64
	expectedErr      error
	expectedContents string
}

func runReadAtTestCases(
	rp lease.ReadProxy,
	cases []readAtTestCase) {
	for _, tc := range cases {
		AssertLe(tc.start, tc.limit)
		buf := make([]byte, tc.limit-tc.start)

		n, err := rp.ReadAt(context.Background(), buf, tc.start)
		AssertEq(tc.expectedErr, err)
		AssertEq(tc.expectedContents, string(buf[:n]))
	}
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

	// Test cases.
	// error we expect to see, if any.
	testCases := []readAtTestCase{
		// First read lease only.
		readAtTestCase{0, 0, nil, ""},
		readAtTestCase{0, 1, nil, "t"},
		readAtTestCase{0, 4, nil, "taco"},
		readAtTestCase{1, 4, nil, "aco"},
		readAtTestCase{4, 4, nil, ""},

		// First and second read leases.
		readAtTestCase{0, 5, someErr, "taco"},
		readAtTestCase{1, 11, someErr, "aco"},

		// All read leases.
		readAtTestCase{0, 20, someErr, "taco"},
		readAtTestCase{1, 20, someErr, "aco"},
		readAtTestCase{1, 100, someErr, "aco"},

		// Second read lease only.
		readAtTestCase{4, 4, nil, ""},
		readAtTestCase{4, 5, someErr, ""},
		readAtTestCase{4, 11, someErr, ""},

		// Second and third read leases.
		readAtTestCase{4, 12, someErr, ""},
		readAtTestCase{4, 20, someErr, ""},
		readAtTestCase{5, 100, someErr, ""},

		// Third read lease only.
		readAtTestCase{11, 20, nil, "enchilada"},
		readAtTestCase{11, 100, io.EOF, "enchilada"},
		readAtTestCase{12, 20, nil, "nchilada"},
		readAtTestCase{19, 20, nil, "a"},
		readAtTestCase{20, 20, nil, ""},

		// Past end.
		readAtTestCase{21, 21, nil, ""},
		readAtTestCase{21, 22, io.EOF, ""},
		readAtTestCase{21, 100, io.EOF, ""},
		readAtTestCase{100, 1000, io.EOF, ""},
	}

	runReadAtTestCases(t.proxy, testCases)
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
	testCases := []readAtTestCase{
		// First read lease only.
		readAtTestCase{0, 0, nil, ""},
		readAtTestCase{0, 1, nil, "t"},
		readAtTestCase{0, 4, nil, "taco"},
		readAtTestCase{1, 4, nil, "aco"},
		readAtTestCase{4, 4, nil, ""},

		// First and second read leases.
		readAtTestCase{0, 5, nil, "tacob"},
		readAtTestCase{1, 11, nil, "acoburrito"},

		// All read leases.
		readAtTestCase{0, 20, nil, "tacoburritoenchilada"},
		readAtTestCase{1, 19, nil, "acoburritoenchilad"},
		readAtTestCase{3, 17, nil, "oburritoenchil"},

		// Second read lease only.
		readAtTestCase{4, 4, nil, ""},
		readAtTestCase{4, 5, nil, "b"},
		readAtTestCase{4, 11, nil, "burrito"},

		// Second and third read leases.
		readAtTestCase{4, 12, nil, "burritoe"},
		readAtTestCase{4, 20, nil, "burritoenchilada"},
		readAtTestCase{5, 100, io.EOF, "urritoenchilada"},

		// Third read lease only.
		readAtTestCase{11, 20, nil, "enchilada"},
		readAtTestCase{11, 100, io.EOF, "enchilada"},
		readAtTestCase{12, 20, nil, "nchilada"},
		readAtTestCase{19, 20, nil, "a"},
		readAtTestCase{20, 20, nil, ""},

		// Past end.
		readAtTestCase{21, 21, nil, ""},
		readAtTestCase{21, 22, io.EOF, ""},
		readAtTestCase{21, 100, io.EOF, ""},
		readAtTestCase{100, 1000, io.EOF, ""},
	}

	runReadAtTestCases(t.proxy, testCases)
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
