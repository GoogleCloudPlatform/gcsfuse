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

	"github.com/googlecloudplatform/gcsfuse/ratelimit"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
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
	f func(context.Context, uint64) bool
}

func (ft *funcThrottle) Capacity() (c uint64) {
	return 1024
}

func (ft *funcThrottle) Wait(
	ctx context.Context,
	tokens uint64) (ok bool) {
	ok = ft.f(ctx, tokens)
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
	t.throttle.f = func(ctx context.Context, tokens uint64) (ok bool) {
		ok = true
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
	t.throttle.f = func(ctx context.Context, tokens uint64) (ok bool) {
		AssertFalse(throttleCalled)
		throttleCalled = true

		AssertEq(t.ctx, ctx)
		AssertEq(readSize, tokens)

		return
	}

	// Call
	t.reader.Read(make([]byte, readSize))

	ExpectTrue(throttleCalled)
}

func (t *ThrottledReaderTest) ThrottleSaysCancelled() {
	// Throttle
	t.throttle.f = func(ctx context.Context, tokens uint64) (ok bool) {
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 1))

	ExpectEq(0, n)
	ExpectThat(err, Error(HasSubstr("throttle")))
	ExpectThat(err, Error(HasSubstr("cancel")))
}

func (t *ThrottledReaderTest) CallsWrapped() {
	buf := make([]byte, 16)
	AssertLe(len(buf), t.throttle.Capacity())

	// Wrapped
	var readCalled bool
	t.wrapped.f = func(p []byte) (n int, err error) {
		AssertFalse(readCalled)
		readCalled = true

		AssertEq(buf, p)

		err = errors.New("")
		return
	}

	// Call
	t.reader.Read(buf)

	ExpectTrue(readCalled)
}

func (t *ThrottledReaderTest) WrappedReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsEOF() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsFullRead() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_CallsAgain() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_SecondFails() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) WrappedReturnsShortRead_SecondSuceeds() {
	AssertTrue(false, "TODO")
}

func (t *ThrottledReaderTest) ReadSizeIsAboveThrottleCapacity() {
	AssertTrue(false, "TODO")
}
