// Copyright 2025 Google LLC
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

package gcsx

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/clock"
	"github.com/vipnydav/gcsfuse/v3/internal/storage"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/fake"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
)

type InactiveTimeoutReaderTestSuite struct {
	suite.Suite
	ctx        context.Context
	mockBucket *storage.TestifyMockBucket
	object     *gcs.MinObject
	reader     gcs.StorageReader // The reader under test

	// Fields to hold results from setup for individual tests
	initialData       []byte
	readHandle        []byte
	initialFakeReader *fake.FakeReader
	timeout           time.Duration
	simulatedClock    *clock.SimulatedClock
}

func TestInactiveTimeoutReaderTestSuite(t *testing.T) {
	suite.Run(t, new(InactiveTimeoutReaderTestSuite))
}

func (s *InactiveTimeoutReaderTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockBucket = new(storage.TestifyMockBucket)

	// Default values, can be overridden by tests before calling setupReader
	s.initialData = []byte("default data")
	s.timeout = 100 * time.Millisecond
	s.object = &gcs.MinObject{Name: "test-object", Generation: 123, Size: uint64(len(s.initialData))}
	s.reader = nil // Reset reader before each test
	s.initialFakeReader = nil
	s.simulatedClock = clock.NewSimulatedClock(time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC))
}

func (s *InactiveTimeoutReaderTestSuite) TearDownTest() {
	if s.reader == nil {
		return
	}

	// Close the wrapper reader, not the potentially nil internal one
	s.reader.Close()
	s.reader = nil
	s.mockBucket.AssertExpectations(s.T())
}

// setupReader is a helper within the suite to create the reader under test.
// Tests should call this after setting specific suite properties like initialData or timeout.
func (s *InactiveTimeoutReaderTestSuite) setupReader() {
	s.object.Size = uint64(len(s.initialData)) // Ensure object size matches data
	readCloser := getReadCloser(s.initialData)
	s.initialFakeReader = &fake.FakeReader{ReadCloser: readCloser, Handle: s.readHandle}

	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       s.object.Name,
		Generation: s.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: s.object.Size,
		},
		ReadCompressed: s.object.HasContentEncodingGzip(),
		ReadHandle:     s.readHandle,
	}
	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(s.initialFakeReader, nil).Times(1)

	var err error
	// Use NewInactiveTimeoutReader directly as NewStorageReaderWithInactiveTimeout is deprecated.
	s.reader, err = NewInactiveTimeoutReaderWithClock(s.ctx, s.mockBucket, s.object, s.readHandle, gcs.ByteRange{Start: uint64(0), Limit: s.object.Size}, s.timeout, s.simulatedClock)
	time.Sleep(5 * time.Millisecond) // Allow time to schedule and create a timer.
	s.Require().Nil(err)
	s.Require().NotNil(s.reader)
}

func (s *InactiveTimeoutReaderTestSuite) Test_NewInactiveTimeoutReader_InitialReadError() {
	s.initialData = make([]byte, 100) // Size doesn't matter here
	s.object = &gcs.MinObject{Name: "fail-object", Generation: 456, Size: 100}
	s.timeout = 100 * time.Millisecond
	initialErr := errors.New("initial connection failed")
	// Expect the initial NewReader call to fail
	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Name == s.object.Name &&
			req.Generation == s.object.Generation &&
			req.Range.Start == 0 &&
			req.Range.Limit == 100
	})).Return(nil, initialErr).Once()

	_, err := NewInactiveTimeoutReader(s.ctx, s.mockBucket, s.object, []byte{}, gcs.ByteRange{Start: 0, Limit: 100}, s.timeout)

	s.Error(err)
	s.ErrorIs(err, initialErr) // Should be the exact error from the bucket
}

func (s *InactiveTimeoutReaderTestSuite) Test_NewInactiveTimeoutReader_ZeroTimeoutError() {
	s.initialData = []byte("zero timeout")
	s.timeout = 0 // Zero timeout

	_, err := NewInactiveTimeoutReader(s.ctx, s.mockBucket, s.object, []byte{}, gcs.ByteRange{Start: 0, Limit: 100}, s.timeout)

	s.Error(err)
	s.ErrorIs(err, ErrZeroInactivityTimeout)
}

func (s *InactiveTimeoutReaderTestSuite) Test_Read_InitialReadNoError() {
	s.initialData = []byte("hello world")
	s.timeout = 100 * time.Millisecond
	s.setupReader()
	buf := make([]byte, 5)

	n, err := s.reader.Read(buf)

	s.NoError(err)
	s.Equal(5, n)
	s.Equal("hello", string(buf[:n]))
}

func (s *InactiveTimeoutReaderTestSuite) Test_NoReadCloserWithinTimeout() {
	s.initialData = []byte("hello world!")
	s.timeout = 100 * time.Millisecond
	s.setupReader()
	buf := make([]byte, 6)
	n1, err1 := s.reader.Read(buf)
	s.NoError(err1)
	s.Equal(6, n1)
	s.simulatedClock.AdvanceTime(s.timeout / 2)
	// Allow some time to routine incase timer fired in half timeout.
	time.Sleep(5 * time.Millisecond)

	n2, err2 := s.reader.Read(buf)

	inactiveReader := s.reader.(*InactiveTimeoutReader)
	s.True(inactiveReader.isActive)
	s.NoError(err2)
	s.Equal(6, n2)
	s.Equal("world!", string(buf[:n2]))
}

func (s *InactiveTimeoutReaderTestSuite) Test_ReadFull_Succeeds() {
	buf := make([]byte, 16)
	s.initialData = []byte("hello world!")
	s.timeout = 100 * time.Millisecond
	s.setupReader()

	n, err := s.reader.Read(buf)

	s.NoError(err)
	s.Equal(12, n)
}

func (s *InactiveTimeoutReaderTestSuite) Test_Read_ReconnectFails() {
	buf := make([]byte, 5)
	s.initialData = []byte("reconnect failure")
	s.timeout = 50 * time.Millisecond
	s.setupReader()
	n, err := s.reader.Read(buf)
	s.Require().NoError(err)
	s.Require().Equal(5, n)
	// First timeout fire will make the reader inactive.
	s.simulatedClock.AdvanceTime(s.timeout + time.Millisecond)
	// Wait for the monitor routine to make the read inactive.
	require.Eventually(s.T(), func() bool {
		rr := s.reader.(*InactiveTimeoutReader)
		rr.mu.Lock()
		defer rr.mu.Unlock()
		return !rr.isActive
	}, time.Second, 10*time.Millisecond, "Monitor did mark the reader inactive in time")
	// 2nd fire will close the inactive reader.
	s.simulatedClock.AdvanceTime(s.timeout + time.Millisecond)
	// Wait for the monitor routine to close the wrapped reader.
	require.Eventually(s.T(), func() bool {
		rr := s.reader.(*InactiveTimeoutReader)
		rr.mu.Lock()
		defer rr.mu.Unlock()
		return (rr.gcsReader == nil)
	}, time.Second, 10*time.Millisecond, "Monitor did not close the reader in time")
	reconnectErr := errors.New("failed to create new reader")
	expectedReadHandle := s.initialFakeReader.Handle
	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Name == s.object.Name &&
			req.Range.Start == 5 && // Expect reconnect from offset 5
			bytes.Equal(req.ReadHandle, expectedReadHandle)
	})).Return(nil, reconnectErr).Times(1) // Expect only one call for the first failed attempt

	nFail, errFail := s.reader.Read(buf)

	// First failed reconnect attempt
	s.ErrorIs(errFail, reconnectErr)
	s.Equal(0, nFail)
}

func (s *InactiveTimeoutReaderTestSuite) Test_Read_TimeoutAndSuccessfulReconnect() {
	s.initialData = []byte("abcdefghijklmnopqrstuvwxyz")
	s.timeout = 50 * time.Second
	s.setupReader()
	buf := make([]byte, 10)
	n, err := s.reader.Read(buf)
	s.Require().NoError(err)
	s.Require().Equal(10, n)
	s.Equal("abcdefghij", string(buf[:n]))
	// First timeout fire will make the reader inactive.
	s.simulatedClock.AdvanceTime(s.timeout + time.Millisecond)
	// Wait for the monitor routine to make the read inactive.
	require.Eventually(s.T(), func() bool {
		rr := s.reader.(*InactiveTimeoutReader)
		rr.mu.Lock()
		defer rr.mu.Unlock()
		return !rr.isActive
	}, time.Second, 10*time.Millisecond, "Monitor did mark the reader inactive in time")
	// 2nd fire will close the inactive reader.
	s.simulatedClock.AdvanceTime(s.timeout + time.Millisecond)
	// Wait for the monitor routine to close the wrapped reader.
	require.Eventually(s.T(), func() bool {
		rr := s.reader.(*InactiveTimeoutReader)
		rr.mu.Lock()
		defer rr.mu.Unlock()
		return (rr.gcsReader == nil)
	}, time.Second, 10*time.Millisecond, "Monitor did not close the reader in time")
	expectedReadHandleAfterClose := s.initialFakeReader.Handle // The handle that should be stored after close
	reconnectReadObjectRequest := &gcs.ReadObjectRequest{
		Name:       s.object.Name,
		Generation: s.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(10), // Expect reconnect from offset 10
			Limit: s.object.Size,
		},
		ReadCompressed: s.object.HasContentEncodingGzip(),
		ReadHandle:     expectedReadHandleAfterClose, // Expect the stored handle to be used for reconnect
	}
	// Use the same initialFakeReader for simplicity, as it tracks the read offset internally.
	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, reconnectReadObjectRequest).Return(s.initialFakeReader, nil).Times(1)
	bufReconnect := make([]byte, 5)

	// Read after timeout (should trigger reconnect)
	nReconnect, errReconnect := s.reader.Read(bufReconnect)

	s.Nil(errReconnect)
	s.Equal(5, nReconnect)
	s.Equal("klmno", string(bufReconnect[:nReconnect]))
}

func (s *InactiveTimeoutReaderTestSuite) Test_Close_ExplicitClose() {
	s.initialData = []byte("close me")
	s.timeout = 100 * time.Millisecond
	s.setupReader()

	err := s.reader.Close()

	s.NoError(err)
	s.Nil(s.reader.(*InactiveTimeoutReader).gcsReader)
	s.reader = nil // Prevent TearDownTest from closing again
}

func (s *InactiveTimeoutReaderTestSuite) Test_handleTimeout_InactiveClose() {
	s.initialData = []byte("simple close test")
	s.timeout = 50 * time.Millisecond
	s.readHandle = []byte("handle-before-close")
	s.setupReader() // Sets up s.reader and s.initialFakeReader
	expectedHandleAfterClose := []byte("handle-after-close")
	s.initialFakeReader.Handle = expectedHandleAfterClose
	itr := s.reader.(*InactiveTimeoutReader)
	itr.isActive = false // Simulate inactivity

	itr.handleTimeout()

	s.Nil(itr.gcsReader)
	s.False(itr.isActive, "isActive should remain false")
	s.Equal(expectedHandleAfterClose, itr.readHandle, "readHandle should be updated from closed reader")
}

func (s *InactiveTimeoutReaderTestSuite) Test_handleTimeout_ActiveBecomeInactive() {
	s.initialData = []byte("simple close test")
	s.timeout = 50 * time.Millisecond
	s.readHandle = []byte("handle-before-close")
	s.setupReader() // Sets up s.reader and s.initialFakeReader
	expectedHandleAfterClose := []byte("handle-after-close")
	s.initialFakeReader.Handle = expectedHandleAfterClose
	itr := s.reader.(*InactiveTimeoutReader)
	itr.isActive = true

	itr.handleTimeout()

	s.NotNil(itr.gcsReader)
	s.False(itr.isActive, "isActive become false")
}

func (s *InactiveTimeoutReaderTestSuite) Test_closeGCSReader_NilReader() {
	s.initialData = []byte("simple close test")
	s.timeout = 50 * time.Millisecond
	s.readHandle = []byte("handle-before-close")
	s.setupReader() // Sets up s.reader and s.initialFakeReader
	itr := s.reader.(*InactiveTimeoutReader)
	itr.gcsReader = nil

	itr.closeGCSReader()

	s.Nil(itr.gcsReader)
	s.Equal(s.readHandle, itr.readHandle)
}

func (s *InactiveTimeoutReaderTestSuite) Test_closeGCSReader_NonNilReader() {
	s.initialData = []byte("simple close test")
	s.timeout = 50 * time.Millisecond
	s.readHandle = []byte("handle-before-close")
	s.setupReader() // Sets up s.reader and s.initialFakeReader
	expectedHandleAfterClose := []byte("handle-after-close")
	s.initialFakeReader.Handle = expectedHandleAfterClose
	itr := s.reader.(*InactiveTimeoutReader)

	itr.closeGCSReader()

	s.Nil(itr.gcsReader)
	s.Equal(expectedHandleAfterClose, itr.readHandle, "readHandle should be updated from closed reader")
}

func (s *InactiveTimeoutReaderTestSuite) TestRaceCondition() {
	var wg sync.WaitGroup
	wg.Add(2)
	s.initialData = []byte(strings.Repeat("abc", 1000))
	s.timeout = time.Second
	s.readHandle = []byte("handle-before-close")
	s.setupReader() // Sets up s.reader and s.initialFakeReader

	// Read()
	go func() {
		defer wg.Done()
		// Read the complete object with buffer size 100.
		for offset := 0; offset < int(s.object.Size); offset += 100 {
			rc := &fake.FakeReader{ReadCloser: getReadCloser(s.initialData[offset:])}
			s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(rc, nil).Maybe()
			buf := make([]byte, 100)

			n, err := s.reader.Read(buf)

			s.NoError(err)
			s.Equal(100, n)
		}
	}()

	// Concurrent handleTimeout.
	go func() {
		defer wg.Done()
		for range 1000 {
			s.reader.(*InactiveTimeoutReader).handleTimeout()
		}
	}()

	wg.Wait()
}
