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

// Helper to create a ReadCloser for fake.NewFakeReader
func getReadCloser(content []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(content))
}

// --- Test Suite Definition ---

type StorageReaderTimeoutSuite struct {
	suite.Suite
	ctx    context.Context
	mockB  *storage.TestifyMockBucket
	object *gcs.MinObject
	reader gcs.StorageReader // The reader under test

	// Fields to hold results from setup for individual tests
	initialData       []byte
	readHandle        []byte
	initialFakeReader *fake.FakeReader
	timeout           time.Duration
}

// --- Suite Setup and Teardown ---

func TestStorageReaderTimeoutTestSuite(t *testing.T) {
	suite.Run(t, new(StorageReaderTimeoutSuite))
}

// SetupTest runs before each test method in the suite.
func (s *StorageReaderTimeoutSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockB = new(storage.TestifyMockBucket)
	// Default values, can be overridden by tests before calling setupReader
	s.initialData = []byte("default data")
	s.timeout = 100 * time.Millisecond
	s.object = &gcs.MinObject{Name: "test-object", Generation: 123, Size: uint64(len(s.initialData))}
	s.reader = nil // Reset reader before each test
	s.initialFakeReader = nil
}

// TearDownTest runs after each test method.
func (s *StorageReaderTimeoutSuite) TearDownTest() {
	if s.reader != nil {
		// Attempt to close the reader if it was successfully created
		s.reader.Close()
	}
	// Verify mock expectations after each test
	s.mockB.AssertExpectations(s.T())
}

// setupReader is a helper within the suite to create the reader under test.
// Tests should call this after setting specific suite properties like initialData or timeout.
func (s *StorageReaderTimeoutSuite) setupReader(startOffset int64) {
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

	s.mockB.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(s.initialFakeReader, nil).Times(1)

	var err error
	s.reader, err = gcsx.NewStorageReaderWithInactiveTimeout(s.ctx, s.mockB, s.object, s.readHandle, startOffset, int64(s.object.Size), s.timeout)
	s.Require().NoError(err)
	s.Require().NotNil(s.reader)
}

func (s *StorageReaderTimeoutSuite) TestSuccessfulReadWithinTimeout() {
	s.initialData = []byte("hello world")
	s.timeout = 100 * time.Millisecond
	s.setupReader(0) // Initial read from offset 0

	buf := make([]byte, 5)
	n, err := s.reader.Read(buf)
	s.NoError(err)
	s.Equal(5, n)
	s.Equal("hello", string(buf[:n]))

	// Wait less than timeout
	time.Sleep(s.timeout / 2)

	buf = make([]byte, 6)
	n, err = s.reader.Read(buf)
	s.NoError(err)
	s.Equal(6, n)
	s.Equal(" world", string(buf[:n]))

	// Read EOF
	n, err = s.reader.Read(buf) // Use same buffer
	s.Same(io.EOF, err)
	s.Equal(0, n)
}

func (s *StorageReaderTimeoutSuite) TestTimeoutAndReconnect() {
	s.initialData = []byte("abcdefghijklmnopqrstuvwxyz")
	s.timeout = 50 * time.Millisecond
	s.setupReader(0)

	// Read first part
	buf := make([]byte, 10)
	n, err := s.reader.Read(buf)
	s.Require().NoError(err)
	s.Require().Equal(10, n)
	s.Equal("abcdefghij", string(buf[:n]))

	// --- Simulate Timeout ---
	time.Sleep(s.timeout + 20*time.Millisecond)

	// --- Read After Timeout ---
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       s.object.Name,
		Generation: s.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(10),
			Limit: s.object.Size,
		},
		ReadCompressed: s.object.HasContentEncodingGzip(),
		ReadHandle:     s.readHandle,
	}

	s.mockB.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(s.initialFakeReader, nil).Times(1)

	buf = make([]byte, 5)
	n, err = s.reader.Read(buf) // This read should trigger the reconnect
	s.Require().NoError(err, "Read after timeout failed")
	s.Require().Equal(5, n)
	s.Equal("klmno", string(buf[:n])) // Should read from the *new* reader's data

	// Read remaining data from the new reader
	buf = make([]byte, 20) // Buffer larger than remaining data
	n, err = s.reader.Read(buf)
	s.Require().NoError(err)
	// s.Require().Equal(len(reconnectData)-5, n) // 16 = 26 - 10 - 5
	s.Equal("pqrstuvwxyz", string(buf[:n]))

	// Read EOF from the new reader
	n, err = s.reader.Read(buf)
	s.Same(io.EOF, err)
	s.Equal(0, n)
}

/*
*
This brings the behavior of Read() after close, I think client shouldn't call.
*/
func (s *StorageReaderTimeoutSuite) TestClose() {
	s.initialData = []byte("close me")
	s.timeout = 100 * time.Millisecond
	s.setupReader(0)

	// Close explicitly
	err := s.reader.Close()
	s.NoError(err)
}

func (s *StorageReaderTimeoutSuite) TestReconnectFails() {
	s.initialData = []byte("reconnect failure")
	s.timeout = 50 * time.Millisecond
	s.setupReader(0)

	// Read first part
	buf := make([]byte, 5)
	n, err := s.reader.Read(buf)
	s.Require().NoError(err)
	s.Require().Equal(5, n)

	// Wait for timeout
	time.Sleep(s.timeout + 20*time.Millisecond)

	// --- Expect Reconnect Attempt ---
	reconnectErr := errors.New("failed to create new reader")
	s.mockB.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Name == s.object.Name &&
			req.Range.Start == 5 &&
			bytes.Equal(req.ReadHandle, s.initialFakeReader.Handle)
	})).Return(nil, reconnectErr).Once()

	// --- Read After Timeout ---
	buf = make([]byte, 5)
	n, err = s.reader.Read(buf) // This read should trigger the failed reconnect
	s.Error(err)
	s.ErrorIs(err, reconnectErr)
	s.Contains(err.Error(), "NewReaderWithReadHandle:")
	s.Equal(0, n)

	// Subsequent reads should also fail trying to reconnect
	s.mockB.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Name == s.object.Name &&
			req.Range.Start == 5 &&
			bytes.Equal(req.ReadHandle, s.initialFakeReader.Handle)
	})).Return(nil, reconnectErr).Once()

	n, err = s.reader.Read(buf)
	s.Error(err)
	s.ErrorIs(err, reconnectErr)
	s.Equal(0, n)
}

func (s *StorageReaderTimeoutSuite) TestInitialReadError() {
	// Override setup defaults for this specific test
	s.initialData = make([]byte, 100) // Size doesn't matter much here
	s.object = &gcs.MinObject{Name: "fail-object", Generation: 456, Size: 100}
	s.timeout = 100 * time.Millisecond
	initialErr := errors.New("initial connection failed")

	// Expect the initial NewReader call to fail
	s.mockB.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		return req.Name == s.object.Name &&
			req.Generation == s.object.Generation &&
			req.Range.Start == 0 &&
			req.Range.Limit == 100
	})).Return(nil, initialErr).Once()

	// Attempt to create the reader (don't use s.setupReader helper here)
	_, err := gcsx.NewStorageReaderWithInactiveTimeout(s.ctx, s.mockB, s.object, []byte{}, 0, 100, s.timeout)
	s.Error(err)
	s.Equal(initialErr, err) // Should be the exact error from the bucket
}

func (s *StorageReaderTimeoutSuite) TestZeroTimeout() {
	s.initialData = []byte("zero timeout")
	s.timeout = 0 * time.Millisecond // Effectively disables the timer's auto-close

	_, err := gcsx.NewStorageReaderWithInactiveTimeout(s.ctx, s.mockB, s.object, []byte{}, 0, 100, s.timeout)
	s.Error(err)
	s.Equal(err, gcsx.DontUseErr) // Should be the exact error from the bucket
}
