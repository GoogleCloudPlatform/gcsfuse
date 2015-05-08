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
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/lease/mock_lease"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestAutoRefreshingReadLease(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

const contents = "taco"

// A function that always successfully returns our contents constant.
func returnContents() (rc io.ReadCloser, err error) {
	rc = ioutil.NopCloser(strings.NewReader(contents))
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type AutoRefreshingReadLeaseTest struct {
	// A function that will be invoked for each call to the function given to
	// NewAutoRefreshingReadLease.
	f func() (io.ReadCloser, error)

	leaser mock_lease.MockFileLeaser
	lease  lease.ReadLease
}

var _ SetUpInterface = &AutoRefreshingReadLeaseTest{}

func init() { RegisterTestSuite(&AutoRefreshingReadLeaseTest{}) }

func (t *AutoRefreshingReadLeaseTest) SetUp(ti *TestInfo) {
	// Set up a function that defers to whatever is currently set as t.f.
	f := func() (rc io.ReadCloser, err error) {
		AssertNe(nil, t.f)
		rc, err = t.f()
		return
	}

	// Set up the leaser.
	t.leaser = mock_lease.NewMockFileLeaser(ti.MockController, "leaser")

	// Set up the lease.
	t.lease = lease.NewAutoRefreshingReadLease(
		t.leaser,
		int64(len(contents)),
		f)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *AutoRefreshingReadLeaseTest) Size() {
	ExpectEq(len(contents), t.lease.Size())
}

func (t *AutoRefreshingReadLeaseTest) LeaserReturnsError() {
	var err error

	// NewFile
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(nil, errors.New("taco")))

	// Attempt to read.
	_, err = t.lease.Read([]byte{})
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AutoRefreshingReadLeaseTest) CallsFunc() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) FuncReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) ContentsReturnReadError() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) ContentsReturnCloseError() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) ContentsAreWrongLength() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) DowngradesAfterRead() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) DowngradesAfterReadAt() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) DowngradesAfterSeek() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Upgrade_Error() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Upgrade_Success() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Upgrade_Failure() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) SecondRead_StillValid() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) SecondRead_Revoked_ErrorReading() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) SecondRead_Revoked_Successful() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Revoke() {
	AssertTrue(false, "TODO")
}
