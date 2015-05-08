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
	"testing/iotest"

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

func successfulWrite(p []byte) (n int, err error) {
	n = len(p)
	return
}

// A ReadCloser that returns the supplied error when closing.
type closeErrorReader struct {
	Wrapped io.Reader
	Err     error
}

func (rc *closeErrorReader) Read(p []byte) (n int, err error) {
	n, err = rc.Wrapped.Read(p)
	return
}

func (rc *closeErrorReader) Close() (err error) {
	err = rc.Err
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type AutoRefreshingReadLeaseTest struct {
	// A function that will be invoked for each call to the function given to
	// NewAutoRefreshingReadLease.
	f func() (io.ReadCloser, error)

	mockController Controller
	leaser         mock_lease.MockFileLeaser
	lease          lease.ReadLease
}

var _ SetUpInterface = &AutoRefreshingReadLeaseTest{}

func init() { RegisterTestSuite(&AutoRefreshingReadLeaseTest{}) }

func (t *AutoRefreshingReadLeaseTest) SetUp(ti *TestInfo) {
	t.mockController = ti.MockController

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
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Downgrade
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(nil, errors.New("")))

	// Function
	var called bool
	t.f = func() (rc io.ReadCloser, err error) {
		AssertFalse(called)
		called = true

		err = errors.New("")
		return
	}

	// Attempt to read.
	t.lease.Read([]byte{})
	ExpectTrue(called)
}

func (t *AutoRefreshingReadLeaseTest) FuncReturnsError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		err = errors.New("taco")
		return
	}

	// Attempt to read.
	_, err := t.lease.Read([]byte{})
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AutoRefreshingReadLeaseTest) ContentsReturnReadError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(
			iotest.TimeoutReader(
				iotest.OneByteReader(
					strings.NewReader(contents))))

		return
	}

	// Attempt to read.
	_, err := t.lease.Read([]byte{})
	ExpectThat(err, Error(HasSubstr("Copy")))
	ExpectThat(err, Error(HasSubstr("timeout")))
}

func (t *AutoRefreshingReadLeaseTest) ContentsReturnCloseError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = &closeErrorReader{
			Wrapped: strings.NewReader(contents),
			Err:     errors.New("taco"),
		}

		return
	}

	// Attempt to read.
	_, err := t.lease.Read([]byte{})
	ExpectThat(err, Error(HasSubstr("Close")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AutoRefreshingReadLeaseTest) ContentsAreWrongLength() {
	AssertEq(4, len(contents))

	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents[:3]))
		return
	}

	// Attempt to read.
	_, err := t.lease.Read([]byte{})
	ExpectThat(err, Error(HasSubstr("Copied 3")))
	ExpectThat(err, Error(HasSubstr("expected 4")))
}

func (t *AutoRefreshingReadLeaseTest) WritesCorrectData() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	var written []byte
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Attempt to read.
	t.lease.Read([]byte{})
	ExpectEq(contents, string(written))
}

func (t *AutoRefreshingReadLeaseTest) WriteError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillOnce(Return(0, errors.New("taco")))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Attempt to read.
	_, err := t.lease.Read([]byte{})
	ExpectThat(err, Error(HasSubstr("Copy")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AutoRefreshingReadLeaseTest) Read_Error() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Read_Successful() {
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
