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

package ratelimit_test

import (
	"errors"
	"io"
	"testing"

	"golang.org/x/net/context"

	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/ratelimit"
)

func TestThrottledReader(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// An io.Reader that defers to a function.
type funcReader struct {
	f func([]byte) (int, error)
}

func (fr *funcReader) Read(p []byte) (n int, err error) {
	n, err = fr.f(p)
	return
}

// A throttler that defers to a function.
type funcThrottle struct {
	f func(context.Context, uint64) error
}

func (ft *funcThrottle) Capacity() (c uint64) {
	return 1024
}

func (ft *funcThrottle) Wait(
	ctx context.Context,
	tokens uint64) (err error) {
	err = ft.f(ctx, tokens)
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ThrottledReaderTest struct {
	ctx context.Context

	wrapped  funcReader
	throttle funcThrottle

	reader io.Reader
}

var _ SetUpInterface = &ThrottledReaderTest{}

func init() { RegisterTestSuite(&ThrottledReaderTest{}) }

func (t *ThrottledReaderTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the default throttle function.
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		return
	}

	// Set up the reader.
	t.reader = ratelimit.ThrottledReader(t.ctx, &t.wrapped, &t.throttle)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ThrottledReaderTest) CallsThrottle() {
	const readSize = 17
	AssertLe(readSize, t.throttle.Capacity())

	// Throttle
	var throttleCalled bool
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		AssertFalse(throttleCalled)
		throttleCalled = true

		AssertEq(t.ctx, ctx)
		AssertEq(readSize, tokens)

		err = errors.New("")
		return
	}

	// Call
	t.reader.Read(make([]byte, readSize))

	ExpectTrue(throttleCalled)
}

func (t *ThrottledReaderTest) ThrottleReturnsError() {
	// Throttle
	expectedErr := errors.New("taco")
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		err = expectedErr
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 1))

	ExpectEq(0, n)
	ExpectEq(expectedErr, err)
}

func (t *ThrottledReaderTest) CallsWrapped() {
	buf := make([]byte, 16)
	AssertLe(len(buf), t.throttle.Capacity())

	// Wrapped
	var readCalled bool
	t.wrapped.f = func(p []byte) (n int, err error) {
		AssertFalse(readCalled)
		readCalled = true

		AssertEq(&buf[0], &p[0])
		AssertEq(len(buf), len(p))

		err = errors.New("")
		return
	}

	// Call
	t.reader.Read(buf)

	ExpectTrue(readCalled)
}

func (t *ThrottledReaderTest) WrappedReturnsError() {
	// Wrapped
	expectedErr := errors.New("taco")
	t.wrapped.f = func(p []byte) (n int, err error) {
		n = 11
		err = expectedErr
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 16))

	ExpectEq(11, n)
	ExpectEq(expectedErr, err)
}

func (t *ThrottledReaderTest) WrappedReturnsEOF() {
	// Wrapped
	t.wrapped.f = func(p []byte) (n int, err error) {
		n = 11
		err = io.EOF
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 16))

	ExpectEq(11, n)
	ExpectEq(io.EOF, err)
}

func (t *ThrottledReaderTest) WrappedReturnsFullRead() {
	const readSize = 17
	AssertLe(readSize, t.throttle.Capacity())

	// Wrapped
	t.wrapped.f = func(p []byte) (n int, err error) {
		n = len(p)
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, readSize))

	ExpectEq(readSize, n)
	ExpectEq(nil, err)
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_CallsAgain() {
	buf := make([]byte, 16)
	AssertLe(len(buf), t.throttle.Capacity())

	// Wrapped
	var callCount int
	t.wrapped.f = func(p []byte) (n int, err error) {
		AssertLt(callCount, 2)
		switch callCount {
		case 0:
			callCount++
			n = 2

		case 1:
			callCount++
			AssertEq(&buf[2], &p[0])
			AssertEq(len(buf)-2, len(p))
			err = errors.New("")
		}

		return
	}

	// Call
	t.reader.Read(buf)

	ExpectEq(2, callCount)
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_SecondReturnsError() {
	// Wrapped
	var callCount int
	expectedErr := errors.New("taco")

	t.wrapped.f = func(p []byte) (n int, err error) {
		AssertLt(callCount, 2)
		switch callCount {
		case 0:
			callCount++
			n = 2

		case 1:
			callCount++
			n = 11
			err = expectedErr
		}

		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 16))

	ExpectEq(2+11, n)
	ExpectEq(expectedErr, err)
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_SecondReturnsEOF() {
	// Wrapped
	var callCount int
	t.wrapped.f = func(p []byte) (n int, err error) {
		AssertLt(callCount, 2)
		switch callCount {
		case 0:
			callCount++
			n = 2

		case 1:
			callCount++
			n = 11
			err = io.EOF
		}

		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 16))

	ExpectEq(2+11, n)
	ExpectEq(io.EOF, err)
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_SecondSucceedsInFull() {
	// Wrapped
	var callCount int
	t.wrapped.f = func(p []byte) (n int, err error) {
		AssertLt(callCount, 2)
		switch callCount {
		case 0:
			callCount++
			n = 2

		case 1:
			callCount++
			n = len(p)
		}

		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 16))

	ExpectEq(16, n)
	ExpectEq(nil, err)
}

func (t *ThrottledReaderTest) ReadSizeIsAboveThrottleCapacity() {
	buf := make([]byte, 2048)
	AssertGt(len(buf), t.throttle.Capacity())

	// Wrapped
	var readCalled bool
	t.wrapped.f = func(p []byte) (n int, err error) {
		AssertFalse(readCalled)
		readCalled = true

		AssertEq(&buf[0], &p[0])
		ExpectEq(t.throttle.Capacity(), len(p))

		err = errors.New("")
		return
	}

	// Call
	t.reader.Read(buf)

	ExpectTrue(readCalled)
}
