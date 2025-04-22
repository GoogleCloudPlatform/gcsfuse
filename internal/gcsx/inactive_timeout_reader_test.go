package gcsx_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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
}

func (s *InactiveTimeoutReaderTestSuite) TearDownTest() {
	if s.reader != nil {
		s.reader.Close()
		s.reader = nil
	}
	s.mockBucket.AssertExpectations(s.T())
}

// getReadCloser is a helper to create a ReadCloser for fake.NewFakeReader.
func getReadCloser(content []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(content))
}

// setupReader is a helper within the suite to create the reader under test.
// Tests should call this after setting specific suite properties like initialData or timeout.
func (s *InactiveTimeoutReaderTestSuite) setupReader(startOffset int64) {
	s.object.Size = uint64(len(s.initialData)) // Ensure object size matches data

	s.initialFakeReader = &fake.FakeReader{ReadCloser: getReadCloser(s.initialData), Handle: s.readHandle}

	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       s.object.Name,
		Generation: s.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(startOffset),
			Limit: s.object.Size,
		},
		ReadCompressed: s.object.HasContentEncodingGzip(),
		ReadHandle:     s.readHandle,
	}

	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(s.initialFakeReader, nil).Times(1)

	var err error
	// Use NewInactiveTimeoutReader directly as NewStorageReaderWithInactiveTimeout is deprecated.
	s.reader, err = gcsx.NewInactiveTimeoutReader(s.ctx, s.mockBucket, s.object, s.readHandle, startOffset, int64(s.object.Size), s.timeout)
	s.Require().NoError(err)
	s.Require().NotNil(s.reader)
}

func (s *InactiveTimeoutReaderTestSuite) Test_NewInactiveTimeoutReader_InitialReadError() {
	// Arrange
	s.initialData = make([]byte, 100) // Size doesn't matter much here
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

	// Act
	_, err := gcsx.NewInactiveTimeoutReader(s.ctx, s.mockBucket, s.object, []byte{}, 0, 100, s.timeout)

	// Assert
	s.Error(err)
	s.Equal(initialErr, err) // Should be the exact error from the bucket
}

func (s *InactiveTimeoutReaderTestSuite) Test_NewInactiveTimeoutReader_ZeroTimeoutError() {
	// Arrange
	s.initialData = []byte("zero timeout")
	s.timeout = 0 * time.Millisecond // Zero timeout

	// Act
	_, err := gcsx.NewInactiveTimeoutReader(s.ctx, s.mockBucket, s.object, []byte{}, 0, 100, s.timeout)

	// Assert
	s.Error(err)
	s.Equal(gcsx.ErrZeroInactivityTimeout, err)
}

func (s *InactiveTimeoutReaderTestSuite) Test_Read_SuccessfulWithinTimeout() {
	// Arrange
	s.initialData = []byte("hello world")
	s.timeout = 100 * time.Millisecond
	s.setupReader(0) // Initial read from offset 0
	buf1 := make([]byte, 5)
	buf2 := make([]byte, 6)
	buf3 := make([]byte, 6) // Buffer for EOF read

	// Act & Assert - First Read
	n1, err1 := s.reader.Read(buf1)
	s.NoError(err1)
	s.Equal(5, n1)
	s.Equal("hello", string(buf1[:n1]))

	// Arrange - Wait less than timeout
	time.Sleep(s.timeout / 2)

	// Act & Assert - Second Read
	n2, err2 := s.reader.Read(buf2)
	s.NoError(err2)
	s.Equal(6, n2)
	s.Equal(" world", string(buf2[:n2]))

	// Act & Assert - Read EOF
	n3, err3 := s.reader.Read(buf3)
	s.Same(io.EOF, err3)
	s.Equal(0, n3)
}

func (s *InactiveTimeoutReaderTestSuite) Test_Read_TimeoutAndSuccessfulReconnect() {
	// Arrange
	s.initialData = []byte("abcdefghijklmnopqrstuvwxyz")
	s.timeout = 50 * time.Millisecond
	s.setupReader(0)
	buf := make([]byte, 10)

	// Act & Assert - First Read
	n, err := s.reader.Read(buf)
	s.Require().NoError(err)
	s.Require().Equal(10, n)
	s.Equal("abcdefghij", string(buf[:n]))

	// Arrange - Simulate Timeout and prepare for reconnect
	time.Sleep(2*s.timeout + time.Millisecond) // Wait long enough for the monitor goroutine to detect inactivity and close the reader.
	reconnectReadObjectRequest := &gcs.ReadObjectRequest{
		Name:       s.object.Name,
		Generation: s.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(10), // Expect reconnect from offset 10
			Limit: s.object.Size,
		},
		ReadCompressed: s.object.HasContentEncodingGzip(),
		ReadHandle:     s.readHandle, // Expect the stored handle to be used
	}
	// Use the same initialFakeReader for simplicity, as it tracks the read offset internally.
	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, reconnectReadObjectRequest).Return(s.initialFakeReader, nil).Times(1)
	bufReconnect := make([]byte, 5)

	// Act - Read After Timeout (should trigger reconnect)
	nReconnect, errReconnect := s.reader.Read(bufReconnect)

	// Assert - Reconnect and first read after reconnect
	s.Require().NoError(errReconnect, "Read after timeout failed")
	s.Require().Equal(5, nReconnect)
	s.Equal("klmno", string(bufReconnect[:nReconnect])) // Should read from the *reconnect* reader's data

	// Arrange - Prepare for reading remaining data
	bufRemaining := make([]byte, 20) // Buffer larger than remaining data

	// Act & Assert - Read remaining data from the new reader
	nRemaining, errRemaining := s.reader.Read(bufRemaining)
	s.Require().NoError(errRemaining)
	s.Equal(11, nRemaining, "Incorrect number of remaining bytes read") // 26 total - 10 initial - 5 reconnect = 11
	s.Equal("pqrstuvwxyz", string(bufRemaining[:nRemaining]))

	// Act & Assert - Read EOF from the new reader
	nEOF, errEOF := s.reader.Read(bufRemaining)
	s.Same(io.EOF, errEOF)
	s.Equal(0, nEOF)
}

func (s *InactiveTimeoutReaderTestSuite) Test_Close_ExplicitClose() {
	// Arrange
	s.initialData = []byte("close me")
	s.timeout = 100 * time.Millisecond
	s.setupReader(0)

	// Act
	err := s.reader.Close()
	s.reader = nil // Prevent TearDownTest from closing again

	// Assert
	s.NoError(err)
	// Mock expectations are asserted in TearDownTest
}

func (s *InactiveTimeoutReaderTestSuite) Test_Read_ReconnectFails() {
	// Arrange
	s.initialData = []byte("reconnect failure")
	s.timeout = 50 * time.Millisecond
	s.setupReader(0)
	buf := make([]byte, 5)

	// Act & Assert - First Read
	n, err := s.reader.Read(buf)
	s.Require().NoError(err)
	s.Require().Equal(5, n)

	// Arrange - Wait for timeout and set up mock for failed reconnect
	time.Sleep(2*s.timeout + time.Millisecond)
	reconnectErr := errors.New("failed to create new reader")
	expectedReadHandle := s.initialFakeReader.Handle // Handle stored from the initial reader
	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Name == s.object.Name &&
			req.Range.Start == 5 && // Expect reconnect from offset 5
			bytes.Equal(req.ReadHandle, expectedReadHandle)
	})).Return(nil, reconnectErr).Times(1) // Expect only one call for the first failed attempt

	// Act - Read After Timeout (should trigger the failed reconnect)
	nFail1, errFail1 := s.reader.Read(buf)

	// Assert - First failed reconnect attempt
	s.Error(errFail1)
	s.ErrorIs(errFail1, reconnectErr)
	s.Contains(errFail1.Error(), "NewReaderWithReadHandle:")
	s.Equal(0, nFail1)

	// Arrange - Set up mock for second failed reconnect attempt
	s.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Name == s.object.Name &&
			req.Range.Start == 5 &&
			bytes.Equal(req.ReadHandle, expectedReadHandle)
	})).Return(nil, reconnectErr).Times(1) // Expect another call

	// Act - Subsequent read should also fail trying to reconnect
	nFail2, errFail2 := s.reader.Read(buf)

	// Assert - Second failed reconnect attempt
	s.Error(errFail2)
	s.ErrorIs(errFail2, reconnectErr)
	s.Contains(errFail2.Error(), "NewReaderWithReadHandle:") // Check wrapping again
	s.Equal(0, nFail2)
}

// TODO: Add concurrent test
