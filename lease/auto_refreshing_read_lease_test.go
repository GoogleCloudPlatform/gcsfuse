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
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Read
	ExpectCall(rwl, "Read")(Any()).
		WillOnce(Return(0, errors.New("taco")))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Attempt to read.
	buf := make([]byte, 1)
	_, err := t.lease.Read(buf)

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AutoRefreshingReadLeaseTest) Read_Successful() {
	const readLength = 3
	AssertLt(readLength, len(contents))

	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Read
	ExpectCall(rwl, "Read")(Any()).
		WillOnce(Invoke(func(p []byte) (n int, err error) {
		n = copy(p, []byte(contents[0:readLength]))
		return
	}))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Attempt to read.
	buf := make([]byte, readLength)
	n, err := t.lease.Read(buf)

	AssertEq(nil, err)
	AssertEq(readLength, n)
	ExpectEq(contents[0:n], string(buf[0:n]))
}

func (t *AutoRefreshingReadLeaseTest) Seek_CallsWrapped() {
	const offset = 17
	const whence = 2

	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Seek
	ExpectCall(rwl, "Seek")(offset, whence).
		WillOnce(Return(0, errors.New("")))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	t.lease.Seek(offset, whence)
}

func (t *AutoRefreshingReadLeaseTest) Seek_Error() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Seek
	ExpectCall(rwl, "Seek")(Any(), Any()).
		WillOnce(Return(0, errors.New("taco")))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	_, err := t.lease.Seek(0, 0)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AutoRefreshingReadLeaseTest) Seek_Successful() {
	const expected = 17

	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Seek
	ExpectCall(rwl, "Seek")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	off, err := t.lease.Seek(0, 0)
	AssertEq(nil, err)
	ExpectEq(expected, off)
}

func (t *AutoRefreshingReadLeaseTest) ReadAt_CallsWrapped() {
	const offset = 17

	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// ReadAt
	ExpectCall(rwl, "ReadAt")(Any(), 17).
		WillOnce(Return(0, errors.New("")))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	t.lease.ReadAt([]byte{}, offset)
}

func (t *AutoRefreshingReadLeaseTest) ReadAt_Error() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// ReadAt
	ExpectCall(rwl, "ReadAt")(Any(), Any()).
		WillOnce(Return(0, errors.New("taco")))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	_, err := t.lease.ReadAt([]byte{}, 0)

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AutoRefreshingReadLeaseTest) ReadAt_Successful() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// ReadAt
	ExpectCall(rwl, "ReadAt")(Any(), Any()).
		WillOnce(Return(0, nil))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	_, err := t.lease.ReadAt([]byte{}, 0)
	ExpectEq(nil, err)
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

func (t *AutoRefreshingReadLeaseTest) WrappedRevoked_Read() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) WrappedRevoked_Seek() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) WrappedRevoked_ReadAt() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) WrappedStillValid_Read() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) WrappedStillValid_Seek() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) WrappedStillValid_ReadAt() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Revoke() {
	var err error

	// Arrange a successful wrapped read lease.
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	ExpectCall(rwl, "ReadAt")(Any(), Any()).
		WillOnce(Return(0, errors.New("taco")))

	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl, nil))

	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	_, err = t.lease.ReadAt([]byte{}, 0)

	// Before revoking, Revoked should return false without needing to call
	// through.
	ExpectFalse(t.lease.Revoked())

	// When we revoke our lease, the wrapped should be revoked as well.
	ExpectCall(rl, "Revoke")()
	t.lease.Revoke()

	// Revoked should reflect this.
	ExpectTrue(t.lease.Revoked())

	// Further calls to all of our methods should return RevokedError.
	_, err = t.lease.Read([]byte{})
	ExpectThat(err, Error(HasSameTypeAs(&lease.RevokedError{})))

	_, err = t.lease.Seek(0, 0)
	ExpectThat(err, Error(HasSameTypeAs(&lease.RevokedError{})))

	_, err = t.lease.ReadAt([]byte{}, 0)
	ExpectThat(err, Error(HasSameTypeAs(&lease.RevokedError{})))

	_, err = t.lease.Upgrade()
	ExpectThat(err, Error(HasSameTypeAs(&lease.RevokedError{})))
}
