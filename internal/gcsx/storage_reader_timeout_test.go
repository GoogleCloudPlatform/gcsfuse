package gcsx_test

import (
	"bytes"
	"context"
	// "errors"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	// "github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
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

	readHandle := []byte("fake-handle")
	s.initialFakeReader = &fake.FakeReader{ReadCloser: getReadCloser(s.initialData), Handle: readHandle}

	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       s.object.Name,
		Generation: s.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(startOffset),
			Limit: s.object.Size,
		},
		ReadCompressed: s.object.HasContentEncodingGzip(),
		ReadHandle:     readHandle,
	}

	s.mockB.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(s.initialFakeReader, nil).Times(1)

	var err error
	s.reader, err = gcsx.NewStorageReaderWithInactiveTimeout(s.ctx, s.mockB, s.object, readHandle, startOffset, int64(s.object.Size), s.timeout)
	s.Require().NoError(err)
	s.Require().NotNil(s.reader)
}

// --- Test Cases (Methods of the Suite) ---

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

	// s.setupReader(10)

	// // --- Read After Timeout ---
	// buf = make([]byte, 5)
	// n, err = s.reader.Read(buf) // This read should trigger the reconnect
	// s.Require().NoError(err, "Read after timeout failed")
	// s.Require().Equal(5, n)
	// s.Equal("klmno", string(buf[:n])) // Should read from the *new* reader's data

	// // Verify the old reader was closed
	// // s.True(s.initialFakeReader.IsClosed(), "Initial fake reader IsClosed should return true after timeout")

	// // Read remaining data from the new reader
	// buf = make([]byte, 20) // Buffer larger than remaining data
	// n, err = s.reader.Read(buf)
	// s.Require().NoError(err)
	// // s.Require().Equal(len(reconnectData)-5, n) // 16 = 26 - 10 - 5
	// s.Equal("pqrstuvwxyz", string(buf[:n]))

	// // Read EOF from the new reader
	// n, err = s.reader.Read(buf)
	// s.Same(io.EOF, err)
	// s.Equal(0, n)
}

// func (s *StorageReaderTimeoutSuite) TestExplicitClose() {
// 	s.initialData = []byte("close me")
// 	s.timeout = 100 * time.Millisecond
// 	s.setupReader(0)

// 	s.False(s.initialFakeReader.IsClosed(), "Initial reader should not be closed yet")

// 	// Close explicitly
// 	err := s.reader.Close()
// 	s.NoError(err)

// 	// Verify the underlying reader was closed
// 	s.True(s.initialFakeReader.IsClosed(), "Initial reader should be closed after explicit Close()")

// 	// Subsequent reads should fail
// 	buf := make([]byte, 1)
// 	n, err := s.reader.Read(buf)
// 	s.ErrorIs(err, io.ErrClosedPipe) // Expecting the error from the closed fakeReader
// 	s.Equal(0, n)

// 	// Closing again should be idempotent
// 	err = s.reader.Close()
// 	s.NoError(err) // No error expected, Close() in wrapper handles this
// }

// func (s *StorageReaderTimeoutSuite) TestCloseWithError() {
// 	s.initialData = []byte("close error")
// 	s.timeout = 100 * time.Millisecond
// 	s.setupReader(0)

// 	// Configure the fake reader to return an error on Close
// 	closeErr := errors.New("mock close error")
// 	s.initialFakeReader.SetCloseError(closeErr)

// 	err := s.reader.Close()
// 	s.Error(err)
// 	s.ErrorIs(err, closeErr)                 // Check if the specific error is wrapped
// 	s.Contains(err.Error(), "close reader:") // Check for wrapping message
// }

// func (s *StorageReaderTimeoutSuite) TestReconnectFails() {
// 	s.initialData = []byte("reconnect failure")
// 	s.timeout = 50 * time.Millisecond
// 	s.setupReader(0)

// 	// Read first part
// 	buf := make([]byte, 5)
// 	n, err := s.reader.Read(buf)
// 	s.Require().NoError(err)
// 	s.Require().Equal(5, n)

// 	// Wait for timeout
// 	time.Sleep(s.timeout + 20*time.Millisecond)

// 	// --- Expect Reconnect Attempt ---
// 	reconnectErr := errors.New("failed to create new reader")
// 	s.mockB.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
// 		return req.Name == s.object.Name &&
// 			req.Range.Start == 5 &&
// 			bytes.Equal(req.ReadHandle, s.initialFakeReader.Handle)
// 	})).Return(nil, reconnectErr).Once()

// 	// --- Read After Timeout ---
// 	buf = make([]byte, 5)
// 	n, err = s.reader.Read(buf) // This read should trigger the failed reconnect
// 	s.Error(err)
// 	s.ErrorIs(err, reconnectErr)
// 	s.Contains(err.Error(), "NewReaderWithReadHandle:")
// 	s.Equal(0, n)

// 	// Verify the old reader was still closed due to timeout
// 	s.True(s.initialFakeReader.IsClosed(), "Close should have been called on the old reader after timeout")

// 	// Subsequent reads should also fail trying to reconnect
// 	s.mockB.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
// 		return req.Name == s.object.Name &&
// 			req.Range.Start == 5 &&
// 			bytes.Equal(req.ReadHandle, s.initialFakeReader.Handle)
// 	})).Return(nil, reconnectErr).Once()

// 	n, err = s.reader.Read(buf)
// 	s.Error(err)
// 	s.ErrorIs(err, reconnectErr)
// 	s.Equal(0, n)

// 	// We expect three calls: initial, first failed attempt, second failed attempt
// 	// Note: AssertExpectations in TearDownTest handles the count verification implicitly.
// 	// If specific count needed here: s.mockB.AssertNumberOfCalls(s.T(), "NewReaderWithReadHandle", 3)
// }

// func (s *StorageReaderTimeoutSuite) TestReadErrorPropagated() {
// 	s.initialData = []byte("read error test")
// 	s.timeout = 100 * time.Millisecond
// 	s.setupReader(0)

// 	// Configure the mock reader to return an error on Read
// 	readErr := errors.New("mock read error")
// 	s.initialFakeReader.SetReader(io.NopCloser(&errorReader{err: readErr}))

// 	buf := make([]byte, 5)
// 	n, err := s.reader.Read(buf)
// 	s.Error(err)
// 	s.Equal(readErr, err) // Should be the exact error
// 	s.Equal(0, n)

// 	// Verify the reader isn't closed by timeout shortly after the error.
// 	time.Sleep(s.timeout / 2)
// 	s.False(s.initialFakeReader.IsClosed(), "Reader should not be closed by timeout after read error")
// }

// func (s *StorageReaderTimeoutSuite) TestReadHandle() {
// 	s.initialData = []byte("handle test")
// 	s.timeout = 50 * time.Millisecond
// 	s.setupReader(0)

// 	initialReadHandle := s.initialFakeReader.ReadHandle()

// 	// 1. Get handle before any timeout
// 	h1 := s.reader.ReadHandle()
// 	s.Equal(initialReadHandle, h1, "Handle before timeout should match initial")

// 	// 2. Read some data
// 	buf := make([]byte, 4)
// 	_, _ = s.reader.Read(buf) // "hand"

// 	// 3. Wait for timeout
// 	time.Sleep(s.timeout + 20*time.Millisecond)

// 	// 4. Setup expectation for reconnect reader with a *different* handle
// 	reconnectReadHandle := []byte("different-handle-content")
// 	reconnectData := s.initialData[4:] // "le test"
// 	reconnectFakeReader := fake.NewFakeReader(getReadCloser(reconnectData), reconnectReadHandle)
// 	s.mockB.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
// 		return req.Range.Start == 4 &&
// 			bytes.Equal(req.ReadHandle, initialReadHandle)
// 	})).Return(reconnectFakeReader, nil).Once()

// 	// 5. Read again to trigger reconnect
// 	_, err := s.reader.Read(buf) // Read 4 bytes: "le t"
// 	s.Require().NoError(err)

// 	// 6. Get handle *after* timeout and reconnect
// 	h2 := s.reader.ReadHandle()
// 	s.Equal(reconnectReadHandle, h2, "Handle after timeout should match the new reader's handle")
// }

// func (s *StorageReaderTimeoutSuite) TestInitialReadError() {
// 	// Override setup defaults for this specific test
// 	s.initialData = make([]byte, 100) // Size doesn't matter much here
// 	s.object = &gcs.MinObject{Name: "fail-object", Generation: 456, Size: 100}
// 	s.timeout = 100 * time.Millisecond
// 	initialReadHandle := storageutil.ConvertObjectToReadHandle(s.object, false)
// 	initialErr := errors.New("initial connection failed")

// 	// Expect the initial NewReader call to fail
// 	s.mockB.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
// 		return req.Name == s.object.Name &&
// 			req.Generation == s.object.Generation &&
// 			req.Range.Start == 0 &&
// 			req.Range.Limit == 100 &&
// 			bytes.Equal(req.ReadHandle, initialReadHandle)
// 	})).Return(nil, initialErr).Once()

// 	// Attempt to create the reader (don't use s.setupReader helper here)
// 	_, err := gcsx.NewStorageReaderWithInactiveTimeout(s.ctx, s.mockB, s.object, initialReadHandle, 0, 100, s.timeout)
// 	s.Error(err)
// 	s.Equal(initialErr, err) // Should be the exact error from the bucket
// }

// func (s *StorageReaderTimeoutSuite) TestZeroTimeout() {
// 	s.initialData = []byte("zero timeout")
// 	s.timeout = 0 * time.Millisecond // Effectively disables the timer's auto-close
// 	s.setupReader(0)

// 	buf := make([]byte, 4)
// 	n, err := s.reader.Read(buf)
// 	s.NoError(err)
// 	s.Equal(4, n)
// 	s.Equal("zero", string(buf[:n]))

// 	// Wait a bit, timer shouldn't fire
// 	time.Sleep(50 * time.Millisecond)
// 	s.False(s.initialFakeReader.IsClosed(), "Reader should not be closed with zero timeout")

// 	n, err = s.reader.Read(buf) // Read " tim"
// 	s.NoError(err)
// 	s.Equal(4, n)
// 	s.Equal(" tim", string(buf[:n]))
// 	s.False(s.initialFakeReader.IsClosed(), "Reader should still not be closed")
// }
