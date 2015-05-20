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
	"fmt"
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

// A range to read, the contents we expect to get back, and a matcher for the
// returned error. Special case: a nil matcher means Equals(nil).
type readAtTestCase struct {
	start            int64
	limit            int64
	errMatcher       Matcher
	expectedContents string
}

func runReadAtTestCases(
	rp lease.ReadProxy,
	cases []readAtTestCase) {
	for i, tc := range cases {
		desc := fmt.Sprintf("Test case %d: [%d, %d)", i, tc.start, tc.limit)

		AssertLe(tc.start, tc.limit)
		buf := make([]byte, tc.limit-tc.start)

		n, err := rp.ReadAt(context.Background(), buf, tc.start)
		AssertEq(tc.expectedContents, string(buf[:n]), "%s", desc)

		if tc.errMatcher == nil {
			AssertEq(nil, err, "%s", desc)
		} else {
			ExpectThat(err, tc.errMatcher, desc)
		}
	}
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MultiReadProxyTest struct {
	ctx context.Context

	// Canned content and errors returned by the refreshers.
	refresherContents []string
	refresherErrors   []error

	leaser       lease.FileLeaser
	initialLease lease.ReadLease
	proxy        *checkingReadProxy
}

var _ SetUpInterface = &MultiReadProxyTest{}
var _ TearDownInterface = &MultiReadProxyTest{}

func init() { RegisterTestSuite(&MultiReadProxyTest{}) }

func (t *MultiReadProxyTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.leaser = lease.NewFileLeaser("", math.MaxInt32, math.MaxInt64)

	// Set up default refresher contents and nil errors.
	t.refresherContents = []string{
		"taco",
		"burrito",
		"enchilada",
	}
	t.refresherErrors = make([]error, len(t.refresherContents))

	// Create the proxy.
	t.resetProxy()
}

func (t *MultiReadProxyTest) TearDown() {
	// Make sure nothing goes crazy.
	if t.proxy != nil {
		t.proxy.Destroy()
	}
}

// Recreate refreshers using makeRefreshers and reset the proxy to use them and
// t.initialLease.
func (t *MultiReadProxyTest) resetProxy() {
	t.proxy = &checkingReadProxy{
		Wrapped: lease.NewMultiReadProxy(
			t.leaser,
			t.makeRefreshers(),
			t.initialLease),
	}
}

// Create refreshers based on the current contents of t.refresherContents.
// t.refresherErrors will be inspected only when Refresh is called.
func (t *MultiReadProxyTest) makeRefreshers() (refreshers []lease.Refresher) {
	for i := range t.refresherContents {
		iCopy := i
		contents := t.refresherContents[i]

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

func (t *MultiReadProxyTest) SizeZero_NoRefreshers() {
	t.refresherContents = []string{}
	t.refresherErrors = []error{}
	t.resetProxy()

	// Size
	ExpectEq(0, t.proxy.Size())

	// ReadAt
	eofMatcher := Equals(io.EOF)
	testCases := []readAtTestCase{
		readAtTestCase{0, 0, eofMatcher, ""},
		readAtTestCase{0, 10, eofMatcher, ""},
		readAtTestCase{5, 10, eofMatcher, ""},
	}

	runReadAtTestCases(t.proxy, testCases)
}

func (t *MultiReadProxyTest) SizeZero_WithRefreshers() {
	t.refresherContents = []string{"", "", "", ""}
	t.refresherErrors = make([]error, len(t.refresherContents))
	t.resetProxy()

	// Size
	ExpectEq(0, t.proxy.Size())

	// ReadAt
	eofMatcher := Equals(io.EOF)
	testCases := []readAtTestCase{
		readAtTestCase{0, 0, eofMatcher, ""},
		readAtTestCase{0, 10, eofMatcher, ""},
		readAtTestCase{5, 10, eofMatcher, ""},
	}

	runReadAtTestCases(t.proxy, testCases)
}

func (t *MultiReadProxyTest) Size() {
	var expected int64
	for _, contents := range t.refresherContents {
		expected += int64(len(contents))
	}

	ExpectEq(expected, t.proxy.Size())
}

func (t *MultiReadProxyTest) ReadAt_NegativeOffset() {
	// Test cases.
	m := Error(HasSubstr("Invalid offset"))
	testCases := []readAtTestCase{
		readAtTestCase{-1, 0, m, ""},
		readAtTestCase{-1, 1, m, ""},
	}

	runReadAtTestCases(t.proxy, testCases)
}

func (t *MultiReadProxyTest) ReadAt_OneRefresherReturnsError() {
	AssertThat(
		t.refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	AssertEq(4, len(t.refresherContents[0]))
	AssertEq(7, len(t.refresherContents[1]))
	AssertEq(9, len(t.refresherContents[2]))

	// Configure an error for the middle read lease.
	someErr := errors.New("foobar")
	t.refresherErrors[1] = someErr

	// Test cases.
	someErrMatcher := Error(HasSubstr(someErr.Error()))
	eofMatcher := Equals(io.EOF)

	testCases := []readAtTestCase{
		// First read lease only.
		readAtTestCase{0, 0, nil, ""},
		readAtTestCase{0, 1, nil, "t"},
		readAtTestCase{0, 4, nil, "taco"},
		readAtTestCase{1, 4, nil, "aco"},
		readAtTestCase{4, 4, nil, ""},

		// First and second read leases.
		readAtTestCase{0, 5, someErrMatcher, "taco"},
		readAtTestCase{1, 11, someErrMatcher, "aco"},

		// All read leases.
		readAtTestCase{0, 20, someErrMatcher, "taco"},
		readAtTestCase{1, 20, someErrMatcher, "aco"},
		readAtTestCase{1, 100, someErrMatcher, "aco"},

		// Second read lease only.
		readAtTestCase{4, 4, nil, ""},
		readAtTestCase{4, 5, someErrMatcher, ""},
		readAtTestCase{4, 11, someErrMatcher, ""},

		// Second and third read leases.
		readAtTestCase{4, 12, someErrMatcher, ""},
		readAtTestCase{4, 20, someErrMatcher, ""},
		readAtTestCase{5, 100, someErrMatcher, ""},

		// Third read lease only.
		readAtTestCase{11, 20, nil, "enchilada"},
		readAtTestCase{11, 100, eofMatcher, "enchilada"},
		readAtTestCase{12, 20, nil, "nchilada"},
		readAtTestCase{19, 20, nil, "a"},
		readAtTestCase{20, 20, eofMatcher, ""},

		// Past end.
		readAtTestCase{21, 21, eofMatcher, ""},
		readAtTestCase{21, 22, eofMatcher, ""},
		readAtTestCase{21, 100, eofMatcher, ""},
		readAtTestCase{100, 1000, eofMatcher, ""},
	}

	runReadAtTestCases(t.proxy, testCases)
}

func (t *MultiReadProxyTest) ReadAt_AllSuccessful() {
	AssertThat(
		t.refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	AssertEq(4, len(t.refresherContents[0]))
	AssertEq(7, len(t.refresherContents[1]))
	AssertEq(9, len(t.refresherContents[2]))

	// Test cases.
	eofMatcher := Equals(io.EOF)
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
		readAtTestCase{5, 100, eofMatcher, "urritoenchilada"},

		// Third read lease only.
		readAtTestCase{11, 20, nil, "enchilada"},
		readAtTestCase{11, 100, eofMatcher, "enchilada"},
		readAtTestCase{12, 20, nil, "nchilada"},
		readAtTestCase{19, 20, nil, "a"},
		readAtTestCase{20, 20, eofMatcher, ""},

		// Past end.
		readAtTestCase{21, 21, eofMatcher, ""},
		readAtTestCase{21, 22, eofMatcher, ""},
		readAtTestCase{21, 100, eofMatcher, ""},
		readAtTestCase{100, 1000, eofMatcher, ""},
	}

	runReadAtTestCases(t.proxy, testCases)
}

func (t *MultiReadProxyTest) ReadAt_ContentAlreadyCached() {
	AssertThat(
		t.refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	// Read the entire contents, causing read leases to be issued for each
	// sub-proxy.
	buf := make([]byte, 1024)
	n, err := t.proxy.ReadAt(context.Background(), buf, 0)

	AssertThat(err, AnyOf(nil, io.EOF))
	AssertEq("tacoburritoenchilada", string(buf[:n]))

	// Set up all refreshers to return errors when invoked.
	for i, _ := range t.refresherErrors {
		t.refresherErrors[i] = errors.New("foo")
	}

	// Despite this, the content should still be available.
	n, err = t.proxy.ReadAt(context.Background(), buf, 0)

	AssertThat(err, AnyOf(nil, io.EOF))
	AssertEq("tacoburritoenchilada", string(buf[:n]))
}

func (t *MultiReadProxyTest) Upgrade_OneRefresherReturnsError() {
	AssertEq(3, len(t.refresherErrors))

	// Configure an error for the middle read lease.
	t.refresherErrors[1] = errors.New("foobar")

	// Upgrade
	_, err := t.proxy.Upgrade(context.Background())
	t.proxy = nil

	ExpectThat(err, Error(HasSubstr("foobar")))
}

func (t *MultiReadProxyTest) Upgrade_AllSuccessful() {
	AssertThat(
		t.refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	// Upgrade
	rwl, err := t.proxy.Upgrade(context.Background())
	t.proxy = nil
	AssertEq(nil, err)

	defer func() { rwl.Downgrade().Revoke() }()

	// Check the contents of the read/write lease.
	_, err = rwl.Seek(0, 0)
	AssertEq(nil, err)

	contents, err := ioutil.ReadAll(rwl)
	AssertEq(nil, err)
	ExpectEq("tacoburritoenchilada", string(contents))
}

func (t *MultiReadProxyTest) Upgrade_ContentAlreadyCached() {
	AssertThat(
		t.refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	// Read the entire contents, causing read leases to be issued for each
	// sub-proxy.
	buf := make([]byte, 1024)
	n, err := t.proxy.ReadAt(context.Background(), buf, 0)

	AssertThat(err, AnyOf(nil, io.EOF))
	AssertEq("tacoburritoenchilada", string(buf[:n]))

	// Set up all refreshers to return errors when invoked.
	for i, _ := range t.refresherErrors {
		t.refresherErrors[i] = errors.New("foo")
	}

	// Despite this, the content should still be available.
	rwl, err := t.proxy.Upgrade(context.Background())
	t.proxy = nil
	AssertEq(nil, err)

	defer func() { rwl.Downgrade().Revoke() }()

	_, err = rwl.Seek(0, 0)
	AssertEq(nil, err)

	contents, err := ioutil.ReadAll(rwl)
	AssertEq(nil, err)
	ExpectEq("tacoburritoenchilada", string(contents))
}

func (t *MultiReadProxyTest) InitialReadLeaseValid() {
	AssertThat(
		t.refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	// Set up an initial read lease.
	rwl, err := t.leaser.NewFile()
	AssertEq(nil, err)

	_, err = rwl.Write([]byte("tacoburritoenchilada"))
	AssertEq(nil, err)

	t.initialLease = rwl.Downgrade()
	rwl = nil

	// Recreate the proxy using that lease.
	t.resetProxy()

	// Set up all refreshers to return errors when invoked.
	for i, _ := range t.refresherErrors {
		t.refresherErrors[i] = errors.New("foo")
	}

	// Despite this, the content should still be available for reading.
	buf := make([]byte, 1024)
	n, err := t.proxy.ReadAt(context.Background(), buf, 0)

	AssertThat(err, AnyOf(nil, io.EOF))
	AssertEq("tacoburritoenchilada", string(buf[:n]))

	// And for upgrading.
	rwl, err = t.proxy.Upgrade(context.Background())
	t.proxy = nil
	AssertEq(nil, err)

	defer func() { rwl.Downgrade().Revoke() }()

	_, err = rwl.Seek(0, 0)
	AssertEq(nil, err)

	contents, err := ioutil.ReadAll(rwl)
	AssertEq(nil, err)
	ExpectEq("tacoburritoenchilada", string(contents))
}

func (t *MultiReadProxyTest) InitialReadLeaseRevoked() {
	AssertThat(
		t.refresherContents,
		ElementsAre(
			"taco",
			"burrito",
			"enchilada",
		))

	// Set up an initial read lease with the correct length that has been revoked.
	rwl, err := t.leaser.NewFile()
	AssertEq(nil, err)

	_, err = rwl.Write([]byte("tacoburritoenchilada"))
	AssertEq(nil, err)

	t.initialLease = rwl.Downgrade()
	rwl = nil

	t.leaser.RevokeReadLeases()

	// Recreate the proxy using that lease.
	t.resetProxy()

	// Set up all refreshers to return errors when invoked.
	for i, _ := range t.refresherErrors {
		t.refresherErrors[i] = errors.New("foobar")
	}

	// Reading should fall through to the refreshers, and fail.
	buf := make([]byte, 1024)
	_, err = t.proxy.ReadAt(context.Background(), buf, 0)

	ExpectThat(err, Error(HasSubstr("foobar")))

	// Ditto upgrading.
	_, err = t.proxy.Upgrade(context.Background())
	t.proxy = nil

	ExpectThat(err, Error(HasSubstr("foobar")))
}
