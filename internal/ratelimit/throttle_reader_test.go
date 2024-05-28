// Copyright 2023 Google Inc. All Rights Reserved.
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

package ratelimit

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

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
	suite.Suite
	ctx context.Context

	wrapped  funcReader
	throttle funcThrottle

	reader io.Reader
}

func TestThrottledReaderSuite(t *testing.T) {
	suite.Run(t, new(ThrottledReaderTest))
}

func (t *ThrottledReaderTest) SetupTest() {
	t.ctx = context.Background()

	// Set up the default throttle function.
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		return
	}

	// Set up the reader.
	t.reader = ThrottledReader(t.ctx, &t.wrapped, &t.throttle)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ThrottledReaderTest) TestCallsThrottle() {
	const readSize = 17
	assert.LessOrEqual(t.T(), uint64(readSize), t.throttle.Capacity())

	// Throttle
	var throttleCalled bool
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		assert.False(t.T(), throttleCalled)
		throttleCalled = true

		assert.Equal(t.T(), t.ctx.Err(), ctx.Err())
		assert.Equal(t.T(), t.ctx.Done(), ctx.Done())
		assert.Equal(t.T(), uint64(readSize), tokens)

		err = errors.New("")
		return
	}

	// Call
	_, err := t.reader.Read(make([]byte, readSize))

	assert.Equal(t.T(), "", err.Error())
	assert.True(t.T(), throttleCalled)
}

func (t *ThrottledReaderTest) TestThrottleReturnsError() {
	// Throttle
	expectedErr := errors.New("taco")
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		err = expectedErr
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 1))

	assert.Equal(t.T(), 0, n)
	assert.EqualError(t.T(), err, expectedErr.Error())
}

func (t *ThrottledReaderTest) TestCallsWrapped() {
	buf := make([]byte, 16)
	assert.LessOrEqual(t.T(), uint64(len(buf)), t.throttle.Capacity())

	// Wrapped
	var readCalled bool
	t.wrapped.f = func(p []byte) (n int, err error) {
		assert.False(t.T(), readCalled)
		readCalled = true

		assert.Equal(t.T(), &buf[0], &p[0])
		assert.Equal(t.T(), len(buf), len(p))

		err = errors.New("")
		return
	}

	// Call
	_, err := t.reader.Read(buf)

	assert.Equal(t.T(), "", err.Error())
	assert.True(t.T(), readCalled)
}

func (t *ThrottledReaderTest) TestWrappedReturnsError() {
	// Wrapped
	expectedErr := errors.New("taco")
	t.wrapped.f = func(p []byte) (n int, err error) {
		n = 11
		err = expectedErr
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 16))

	assert.Equal(t.T(), 11, n)
	assert.EqualError(t.T(), err, expectedErr.Error())
}

func (t *ThrottledReaderTest) TestWrappedReturnsEOF() {
	// Wrapped
	t.wrapped.f = func(p []byte) (n int, err error) {
		n = 11
		err = io.EOF
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, 16))

	assert.Equal(t.T(), 11, n)
	assert.EqualError(t.T(), err, io.EOF.Error())
}

func (t *ThrottledReaderTest) TestWrappedReturnsFullRead() {
	const readSize = 17
	assert.LessOrEqual(t.T(), uint64(readSize), t.throttle.Capacity())

	// Wrapped
	t.wrapped.f = func(p []byte) (n int, err error) {
		n = len(p)
		return
	}

	// Call
	n, err := t.reader.Read(make([]byte, readSize))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readSize, n)
}

func (t *ThrottledReaderTest) TestWrappedReturnsShortRead_CallsAgain() {
	buf := make([]byte, 16)
	assert.LessOrEqual(t.T(), uint64(len(buf)), t.throttle.Capacity())

	// Wrapped
	var callCount int
	t.wrapped.f = func(p []byte) (n int, err error) {
		assert.Less(t.T(), callCount, 2)
		switch callCount {
		case 0:
			callCount++
			n = 2

		case 1:
			callCount++
			assert.Equal(t.T(), &buf[2], &p[0])
			assert.Equal(t.T(), len(buf)-2, len(p))
			err = errors.New("")
		}

		return
	}

	// Call
	_, err := t.reader.Read(buf)

	assert.Equal(t.T(), "", err.Error())
	assert.Equal(t.T(), 2, callCount)
}

func (t *ThrottledReaderTest) TestWrappedReturnsShortRead_SecondReturnsError() {
	// Wrapped
	var callCount int
	expectedErr := errors.New("taco")

	t.wrapped.f = func(p []byte) (n int, err error) {
		assert.Less(t.T(), callCount, 2)
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

	assert.Equal(t.T(), 2+11, n)
	assert.EqualError(t.T(), err, expectedErr.Error())
}

func (t *ThrottledReaderTest) TestWrappedReturnsShortRead_SecondReturnsEOF() {
	// Wrapped
	var callCount int
	t.wrapped.f = func(p []byte) (n int, err error) {
		assert.Less(t.T(), callCount, 2)
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

	assert.Equal(t.T(), 2+11, n)
	assert.EqualError(t.T(), err, io.EOF.Error())
}

func (t *ThrottledReaderTest) TestWrappedReturnsShortRead_SecondSucceedsInFull() {
	// Wrapped
	var callCount int
	t.wrapped.f = func(p []byte) (n int, err error) {
		assert.Less(t.T(), callCount, 2)
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

	assert.Equal(t.T(), 16, n)
	assert.NoError(t.T(), err)
}

func (t *ThrottledReaderTest) TestReadSizeIsAboveThrottleCapacity() {
	buf := make([]byte, 2048)
	assert.Greater(t.T(), uint64(len(buf)), t.throttle.Capacity())

	// Wrapped
	var readCalled bool
	t.wrapped.f = func(p []byte) (n int, err error) {
		assert.False(t.T(), readCalled)
		readCalled = true

		assert.Equal(t.T(), &buf[0], &p[0])
		assert.Equal(t.T(), t.throttle.Capacity(), uint64(len(p)))

		err = errors.New("")
		return
	}

	// Call
	_, err := t.reader.Read(buf)

	assert.Equal(t.T(), "", err.Error())
	assert.True(t.T(), readCalled)
}
