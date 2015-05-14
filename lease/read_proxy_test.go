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

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/lease/mock_lease"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestReadProxy(t *testing.T) { RunTests(t) }

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

// A refresher that defers to a function.
type funcRefresher struct {
	N int64
	F func(context.Context) (io.ReadCloser, error)
}

func (r *funcRefresher) Size() (size int64) {
	return r.N
}

func (r *funcRefresher) Refresh(
	ctx context.Context) (rc io.ReadCloser, err error) {
	rc, err = r.F(ctx)
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ReadProxyTest struct {
	// A function that will be invoked for each call to the refresher given to
	// NewReadProxy.
	f func() (io.ReadCloser, error)

	mockController Controller
	leaser         mock_lease.MockFileLeaser
	proxy          lease.ReadProxy
}

var _ SetUpInterface = &ReadProxyTest{}

func init() { RegisterTestSuite(&ReadProxyTest{}) }

func (t *ReadProxyTest) SetUp(ti *TestInfo) {
	t.mockController = ti.MockController

	// Set up the leaser.
	t.leaser = mock_lease.NewMockFileLeaser(ti.MockController, "leaser")

	// Set up the lease.
	t.proxy = lease.NewReadProxy(
		t.leaser,
		t.makeRefresher(),
		nil)
}

func (t *ReadProxyTest) makeRefresher() (r lease.Refresher) {
	r = &funcRefresher{
		N: int64(len(contents)),
		F: t.callF,
	}

	return
}

// Defer to whatever is currently set as t.f.
func (t *ReadProxyTest) callF(ctx context.Context) (io.ReadCloser, error) {
	AssertNe(nil, t.f)
	return t.f()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ReadProxyTest) Size() {
	ExpectEq(len(contents), t.proxy.Size())
}

func (t *ReadProxyTest) LeaserReturnsError() {
	var err error

	// NewFile
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(nil, errors.New("taco")))

	// Attempt to read.
	_, err = t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadProxyTest) CallsFunc() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))
	ExpectCall(rl, "Revoke")()

	// Function
	var called bool
	t.f = func() (rc io.ReadCloser, err error) {
		AssertFalse(called)
		called = true

		err = errors.New("")
		return
	}

	// Attempt to read.
	t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectTrue(called)
}

func (t *ReadProxyTest) FuncReturnsError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		err = errors.New("taco")
		return
	}

	// Attempt to read.
	_, err := t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadProxyTest) ContentsReturnReadError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))
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
	_, err := t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectThat(err, Error(HasSubstr("Copy")))
	ExpectThat(err, Error(HasSubstr("timeout")))
}

func (t *ReadProxyTest) ContentsReturnCloseError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))
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
	_, err := t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectThat(err, Error(HasSubstr("Close")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadProxyTest) ContentsAreWrongLength() {
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
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents[:3]))
		return
	}

	// Attempt to read.
	_, err := t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectThat(err, Error(HasSubstr("Copied 3")))
	ExpectThat(err, Error(HasSubstr("expected 4")))
}

func (t *ReadProxyTest) WritesCorrectData() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	var written []byte
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(func(p []byte) (n int, err error) {
		written = append(written, p...)
		n = len(p)
		return
	}))

	// Read
	ExpectCall(rwl, "Read")(Any()).
		WillRepeatedly(Return(0, errors.New("")))

	// Downgrade
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectEq(contents, string(written))
}

func (t *ReadProxyTest) WriteError() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillOnce(Return(0, errors.New("taco")))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Attempt to read.
	_, err := t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectThat(err, Error(HasSubstr("Copy")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadProxyTest) ReadAt_CallsWrapped() {
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
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	t.proxy.ReadAt(context.Background(), []byte{}, offset)
}

func (t *ReadProxyTest) ReadAt_Error() {
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
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	_, err := t.proxy.ReadAt(context.Background(), []byte{}, 0)

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadProxyTest) ReadAt_Successful() {
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
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	_, err := t.proxy.ReadAt(context.Background(), []byte{}, 0)
	ExpectEq(nil, err)
}

func (t *ReadProxyTest) Upgrade_Error() {
	// NewFile
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	// Write
	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Return(0, errors.New("taco")))

	// Downgrade and Revoke
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))
	ExpectCall(rl, "Revoke")()

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	_, err := t.proxy.Upgrade(context.Background())
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadProxyTest) Upgrade_Successful() {
	// NewFile
	expected := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(expected, nil))

	// Write
	ExpectCall(expected, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	// Function
	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	// Call.
	rwl, err := t.proxy.Upgrade(context.Background())
	AssertEq(nil, err)
	ExpectEq(expected, rwl)
}

func (t *ReadProxyTest) WrappedRevoked() {
	// Arrange a successful wrapped read lease.
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	ExpectCall(rwl, "ReadAt")(Any(), Any()).
		WillOnce(Return(0, errors.New("taco")))

	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))

	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	t.proxy.ReadAt(context.Background(), []byte{}, 0)

	// Simulate it being revoked for all methods.
	ExpectCall(rl, "ReadAt")(Any(), Any()).
		WillOnce(Return(0, &lease.RevokedError{}))

	ExpectCall(rl, "Upgrade")().
		WillOnce(Return(nil, &lease.RevokedError{}))

	ExpectCall(t.leaser, "NewFile")().
		Times(4).
		WillRepeatedly(Return(nil, errors.New("")))

	t.proxy.ReadAt(context.Background(), []byte{}, 0)
	t.proxy.Upgrade(context.Background())
}

func (t *ReadProxyTest) WrappedStillValid() {
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
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))

	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	t.proxy.ReadAt(context.Background(), []byte{}, 0)

	// ReadAt
	ExpectCall(rl, "ReadAt")(Any(), 11).
		WillOnce(Return(0, errors.New("taco"))).
		WillOnce(Return(17, nil))

	_, err = t.proxy.ReadAt(context.Background(), []byte{}, 11)
	ExpectThat(err, Error(HasSubstr("taco")))

	n, err := t.proxy.ReadAt(context.Background(), []byte{}, 11)
	ExpectEq(17, n)

	// Upgrade
	ExpectCall(rl, "Revoke")()
	ExpectCall(rl, "Upgrade")().
		WillOnce(Return(nil, errors.New("taco"))).
		WillOnce(Return(rwl, nil))

	_, err = t.proxy.Upgrade(context.Background())
	ExpectThat(err, Error(HasSubstr("taco")))

	tmp, _ := t.proxy.Upgrade(context.Background())
	ExpectEq(rwl, tmp)
}

func (t *ReadProxyTest) InitialReadLease_Revoked() {
	// Set up an initial lease.
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	t.proxy = lease.NewReadProxy(
		t.leaser,
		t.makeRefresher(),
		rl)

	// Simulate it being revoked for all methods.
	ExpectCall(rl, "ReadAt")(Any(), Any()).
		WillOnce(Return(0, &lease.RevokedError{}))

	ExpectCall(rl, "Upgrade")().
		WillOnce(Return(nil, &lease.RevokedError{}))

	ExpectCall(t.leaser, "NewFile")().
		Times(4).
		WillRepeatedly(Return(nil, errors.New("")))

	t.proxy.ReadAt(context.Background(), []byte{}, 0)
	t.proxy.Upgrade(context.Background())
}

func (t *ReadProxyTest) InitialReadLease_Valid() {
	var err error

	// Set up an initial lease.
	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	t.proxy = lease.NewReadProxy(
		t.leaser,
		t.makeRefresher(),
		rl)

	// ReadAt
	ExpectCall(rl, "ReadAt")(Any(), 11).
		WillOnce(Return(0, errors.New("taco"))).
		WillOnce(Return(17, nil))

	_, err = t.proxy.ReadAt(context.Background(), []byte{}, 11)
	ExpectThat(err, Error(HasSubstr("taco")))

	n, err := t.proxy.ReadAt(context.Background(), []byte{}, 11)
	ExpectEq(17, n)

	// Upgrade
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")

	ExpectCall(rl, "Revoke")()
	ExpectCall(rl, "Upgrade")().
		WillOnce(Return(nil, errors.New("taco"))).
		WillOnce(Return(rwl, nil))

	_, err = t.proxy.Upgrade(context.Background())
	ExpectThat(err, Error(HasSubstr("taco")))

	tmp, _ := t.proxy.Upgrade(context.Background())
	ExpectEq(rwl, tmp)
}

func (t *ReadProxyTest) Destroy() {
	// Arrange a successful wrapped read lease.
	rwl := mock_lease.NewMockReadWriteLease(t.mockController, "rwl")
	ExpectCall(t.leaser, "NewFile")().
		WillOnce(Return(rwl, nil))

	ExpectCall(rwl, "Write")(Any()).
		WillRepeatedly(Invoke(successfulWrite))

	ExpectCall(rwl, "ReadAt")(Any(), Any()).
		WillOnce(Return(0, errors.New("taco")))

	rl := mock_lease.NewMockReadLease(t.mockController, "rl")
	ExpectCall(rwl, "Downgrade")().WillOnce(Return(rl))

	t.f = func() (rc io.ReadCloser, err error) {
		rc = ioutil.NopCloser(strings.NewReader(contents))
		return
	}

	t.proxy.ReadAt(context.Background(), []byte{}, 0)

	// When we destroy our lease, the wrapped should be revoked.
	ExpectCall(rl, "Revoke")()
	t.proxy.Destroy()
}
