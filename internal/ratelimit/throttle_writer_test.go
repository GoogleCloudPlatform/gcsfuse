// Copyright 2026 Google LLC
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
	"bytes"
	"errors"
	"io"
	"testing"

	"context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// An io.Writer that defers to a function.
type funcWriter struct {
	f func([]byte) (int, error)
}

func (fw *funcWriter) Write(p []byte) (n int, err error) {
	n, err = fw.f(p)
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ThrottledWriterTest struct {
	suite.Suite
	ctx context.Context

	wrapped  funcWriter
	throttle funcThrottle

	writer io.Writer
}

func TestThrottledWriterSuite(t *testing.T) {
	suite.Run(t, new(ThrottledWriterTest))
}

func (t *ThrottledWriterTest) SetupTest() {
	t.ctx = context.Background()

	// Set up the default throttle and writer functions.
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		return
	}
	t.wrapped.f = func(p []byte) (n int, err error) {
		return len(p), nil
	}

	// Set up the writer.
	t.writer = ThrottledWriter(t.ctx, &t.wrapped, &t.throttle)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ThrottledWriterTest) TestWriteWithinCapacity() {
	const writeSize = 17
	assert.LessOrEqual(t.T(), uint64(writeSize), t.throttle.Capacity())
	input := bytes.Repeat([]byte{'a'}, writeSize)
	var waits []uint64
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		assert.Equal(t.T(), t.ctx.Done(), ctx.Done())
		waits = append(waits, tokens)
		return
	}
	var got bytes.Buffer
	t.wrapped.f = got.Write

	n, err := t.writer.Write(input)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), writeSize, n)
	assert.Equal(t.T(), []uint64{writeSize}, waits)
	assert.Equal(t.T(), input, got.Bytes())
}

func (t *ThrottledWriterTest) TestWriteLargerThanCapacity() {
	capacity := t.throttle.Capacity()
	writeSize := int(2*capacity + 5)
	input := make([]byte, writeSize)
	for i := range input {
		input[i] = byte(i)
	}
	var waits []uint64
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		waits = append(waits, tokens)
		return
	}
	var got bytes.Buffer
	t.wrapped.f = func(p []byte) (int, error) {
		assert.LessOrEqual(t.T(), uint64(len(p)), capacity)
		return got.Write(p)
	}

	n, err := t.writer.Write(input)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), writeSize, n)
	assert.Equal(t.T(), []uint64{capacity, capacity, 5}, waits)
	assert.Equal(t.T(), input, got.Bytes())
}

func (t *ThrottledWriterTest) TestThrottleReturnsError() {
	capacity := t.throttle.Capacity()
	expectedErr := errors.New("taco")
	var waitCount int
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		waitCount++
		if waitCount == 2 {
			return expectedErr
		}
		return
	}
	var writeCount int
	t.wrapped.f = func(p []byte) (int, error) {
		writeCount++
		return len(p), nil
	}

	n, err := t.writer.Write(make([]byte, capacity+3))

	assert.EqualError(t.T(), err, expectedErr.Error())
	assert.Equal(t.T(), int(capacity), n)
	assert.Equal(t.T(), 1, writeCount)
}

func (t *ThrottledWriterTest) TestWrappedReturnsError() {
	expectedErr := errors.New("burrito")
	t.wrapped.f = func(p []byte) (int, error) {
		return 11, expectedErr
	}

	n, err := t.writer.Write(make([]byte, 16))

	assert.Equal(t.T(), 11, n)
	assert.EqualError(t.T(), err, expectedErr.Error())
}

func (t *ThrottledWriterTest) TestWrappedShortWrite() {
	var writeCount int
	t.wrapped.f = func(p []byte) (int, error) {
		writeCount++
		return 7, nil
	}

	n, err := t.writer.Write(make([]byte, 16))

	assert.Equal(t.T(), 7, n)
	assert.ErrorIs(t.T(), err, io.ErrShortWrite)
	assert.Equal(t.T(), 1, writeCount)
}

func (t *ThrottledWriterTest) TestZeroLengthWrite() {
	var waitCalled, writeCalled bool
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		waitCalled = true
		return
	}
	t.wrapped.f = func(p []byte) (int, error) {
		writeCalled = true
		return 0, nil
	}

	n, err := t.writer.Write(nil)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, n)
	assert.False(t.T(), waitCalled)
	assert.False(t.T(), writeCalled)
}

////////////////////////////////////////////////////////////////////////
// throttledUploadPayloadReader
////////////////////////////////////////////////////////////////////////

type ThrottledUploadPayloadReaderTest struct {
	suite.Suite
	ctx context.Context

	wrapped  funcReader
	throttle funcThrottle

	reader io.Reader
}

func TestThrottledUploadPayloadReaderSuite(t *testing.T) {
	suite.Run(t, new(ThrottledUploadPayloadReaderTest))
}

func (t *ThrottledUploadPayloadReaderTest) SetupTest() {
	t.ctx = context.Background()
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		return
	}
	t.reader = throttledUploadPayloadReader(t.ctx, &t.wrapped, &t.throttle)
}

func (t *ThrottledUploadPayloadReaderTest) TestChargesBytesActuallyRead() {
	const readSize = 100
	buf := make([]byte, 32*1024)
	assert.Greater(t.T(), uint64(len(buf)), t.throttle.Capacity())
	t.wrapped.f = func(p []byte) (int, error) {
		return readSize, nil
	}
	var waits []uint64
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		assert.Equal(t.T(), t.ctx.Done(), ctx.Done())
		waits = append(waits, tokens)
		return
	}

	n, err := t.reader.Read(buf)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readSize, n)
	assert.Equal(t.T(), []uint64{readSize}, waits)
}

func (t *ThrottledUploadPayloadReaderTest) TestEmptyReaderDoesNotWait() {
	t.wrapped.f = func(p []byte) (int, error) {
		return 0, io.EOF
	}
	var waitCalled bool
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		waitCalled = true
		return
	}

	n, err := t.reader.Read(make([]byte, 16))

	assert.Equal(t.T(), 0, n)
	assert.ErrorIs(t.T(), err, io.EOF)
	assert.False(t.T(), waitCalled)
}

func (t *ThrottledUploadPayloadReaderTest) TestThrottleErrorReturnedWithBytesRead() {
	expectedErr := errors.New("taco")
	t.wrapped.f = func(p []byte) (int, error) {
		return 11, nil
	}
	t.throttle.f = func(ctx context.Context, tokens uint64) (err error) {
		return expectedErr
	}

	n, err := t.reader.Read(make([]byte, 16))

	assert.Equal(t.T(), 11, n)
	assert.EqualError(t.T(), err, expectedErr.Error())
}

func (t *ThrottledUploadPayloadReaderTest) TestBufferClampedToCapacity() {
	buf := make([]byte, 2048)
	assert.Greater(t.T(), uint64(len(buf)), t.throttle.Capacity())
	var readLen int
	t.wrapped.f = func(p []byte) (int, error) {
		assert.Equal(t.T(), &buf[0], &p[0])
		readLen = len(p)
		return len(p), nil
	}

	n, err := t.reader.Read(buf)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), t.throttle.Capacity(), uint64(readLen))
	assert.Equal(t.T(), readLen, n)
}
