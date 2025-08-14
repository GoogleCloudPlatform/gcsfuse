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

package handle

import (
	"bytes"
	"context"
	crypto_rand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/read_manager"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const testDirName = "parentRoot"

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type fileTest struct {
	suite.Suite
	ctx    context.Context
	clock  timeutil.SimulatedClock
	bucket gcsx.SyncerBucket
}

func TestFileTestSuite(t *testing.T) {
	suite.Run(t, new(fileTest))
}

func (t *fileTest) SetupTest() {
	t.ctx = context.TODO()
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	t.bucket = gcsx.NewSyncerBucket(1, 10, ".gcsfuse_tmp/", fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{}))
}

func (t *fileTest) TearDownTest() {
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// createDirInode helps create the parent directory inode for the file inode
// which will be used for testing methods defined on the fileHandle.
func createDirInode(
	bucket *gcsx.SyncerBucket,
	clock *timeutil.SimulatedClock) inode.DirInode {
	return inode.NewDirInode(
		1,
		inode.NewDirName(inode.NewRootName(""), testDirName),
		fuseops.InodeAttributes{
			Uid:  0,
			Gid:  0,
			Mode: 0712,
		},
		false,
		false,
		true,
		0,
		bucket,
		clock,
		clock,
		4,
		false,
	)
}

// createFileInode is a helper to create a FileInode for testing.
func createFileInode(
	t *testing.T,
	bucket *gcsx.SyncerBucket,
	clock *timeutil.SimulatedClock,
	config *cfg.Config,
	parent inode.DirInode,
	objectName string,
	content []byte,
	localFileCache bool) *inode.FileInode {

	fullObjectName := parent.Name().GcsObjectName() + objectName
	obj := &gcs.MinObject{
		Name:           fullObjectName,
		Size:           uint64(len(content)),
		Generation:     1,
		MetaGeneration: 1,
		Updated:        clock.Now(),
	}

	// Create object in the fake bucket to simulate existing GCS object
	_, err := bucket.CreateObject(context.Background(), &gcs.CreateObjectRequest{
		Name:     fullObjectName,
		Contents: io.NopCloser(bytes.NewReader(content)),
	})
	if err != nil {
		t.Fatalf("Failed to create object in fake bucket: %v", err)
	}

	return inode.NewFileInode(
		fuseops.InodeID(2),
		inode.NewFileName(parent.Name(), objectName),
		obj,
		fuseops.InodeAttributes{},
		bucket,
		localFileCache,
		contentcache.New("", clock),
		clock,
		false,
		config,
		semaphore.NewWeighted(100),
	)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *fileTest) TestFileHandleWrite() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{Write: cfg.WriteConfig{EnableStreamingWrites: false}}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj", nil, false)
	fh := NewFileHandle(in, nil, false, nil, util.Write, &cfg.Config{}, nil, nil)
	data := []byte("hello")

	_, err := fh.Write(t.ctx, data, 0)

	assert.Nil(t.T(), err)
	// Validate that write is successful at inode.
	buf := make([]byte, len(data))
	n, err := in.Read(t.ctx, buf, 0)
	buf = buf[:n]
	// Ignore EOF.
	if err == io.EOF {
		err = nil
	}
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), data, buf)
}

func (t *fileTest) TestIsValidReadManager() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	const objectName = "test_obj"
	const objectContent = "some data"
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, objectName, []byte(objectContent), false)
	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)
	fh.inode.Lock()
	defer fh.inode.Unlock()

	// Mocks for test cases
	mockRMForMismatch := new(read_manager.MockReadManager)
	// Inode has generation 1. Let's set readManager's object generation to 2.
	mockRMForMismatch.On("Object").Return(&gcs.MinObject{Generation: 2})

	rmObjectForMatch := &gcs.MinObject{Generation: 1, Size: 0}
	mockRMForMatch := new(read_manager.MockReadManager)
	// Inode has generation 1.
	mockRMForMatch.On("Object").Return(rmObjectForMatch)

	testCases := []struct {
		name           string
		readManager    gcsx.ReadManager
		expectedResult bool
		postCheck      func()
	}{
		{
			name:           "NilReadManager",
			readManager:    nil,
			expectedResult: false,
		},
		{
			name:           "GenerationMismatch",
			readManager:    mockRMForMismatch,
			expectedResult: false,
			postCheck: func() {
				mockRMForMismatch.AssertExpectations(t.T())
			},
		},
		{
			name:           "GenerationMatch",
			readManager:    mockRMForMatch,
			expectedResult: true,
			postCheck: func() {
				assert.Equal(t.T(), in.SourceGeneration().Size, rmObjectForMatch.Size)
				mockRMForMatch.AssertExpectations(t.T())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			fh.readManager = tc.readManager
			result := fh.isValidReadManager()
			assert.Equal(t.T(), tc.expectedResult, result)
			if tc.postCheck != nil {
				tc.postCheck()
			}
		})
	}
}

func (t *fileTest) TestIsValidReader() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	const objectName = "test_obj"
	const objectContent = "some data"
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, objectName, []byte(objectContent), false)
	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)
	fh.inode.Lock()
	defer fh.inode.Unlock()

	// Mocks for test cases
	mockReaderForMismatch := new(gcsx.MockRandomReader)
	// Inode has generation 1. Let's set reader's object generation to 2.
	mockReaderForMismatch.On("Object").Return(&gcs.MinObject{Generation: 2})

	readerObjectForMatch := &gcs.MinObject{Generation: 1, Size: 0}
	mockReaderForMatch := new(gcsx.MockRandomReader)
	// Inode has generation 1.
	mockReaderForMatch.On("Object").Return(readerObjectForMatch)

	testCases := []struct {
		name           string
		reader         gcsx.RandomReader
		expectedResult bool
		postCheck      func()
	}{
		{
			name:           "NilReader",
			reader:         nil,
			expectedResult: false},
		{
			name:           "GenerationMismatch",
			reader:         mockReaderForMismatch,
			expectedResult: false,
			postCheck: func() {
				mockReaderForMismatch.AssertExpectations(t.T())
			},
		},
		{
			name:           "GenerationMatch",
			reader:         mockReaderForMatch,
			expectedResult: true,
			postCheck: func() {
				assert.Equal(t.T(), in.SourceGeneration().Size, readerObjectForMatch.Size)
				mockReaderForMatch.AssertExpectations(t.T())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			fh.reader = tc.reader
			result := fh.isValidReader()
			assert.Equal(t.T(), tc.expectedResult, result)
			if tc.postCheck != nil {
				tc.postCheck()
			}
		})
	}
}

// Test_Read_Success validates successful read behavior using the random reader.
func (t *fileTest) Test_Read_Success() {
	expectedData := []byte("hello from reader")
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, "test_obj_reader", expectedData, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)
	buf := make([]byte, len(expectedData))
	fh.inode.Lock()

	output, n, err := fh.Read(t.ctx, buf, 0, 200)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(expectedData), n)
	assert.Equal(t.T(), expectedData, output)
}

// Test_ReadWithReadManager_Success validates successful read behavior using the readManager.
func (t *fileTest) Test_ReadWithReadManager_Success() {
	expectedData := []byte("hello from readManager")
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, "test_obj_readManager", expectedData, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)
	buf := make([]byte, len(expectedData))
	fh.inode.Lock()

	output, n, err := fh.ReadWithReadManager(t.ctx, buf, 0, 200)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(expectedData), n)
	assert.Equal(t.T(), expectedData, output)
}

// Test_ReadWithReadManager_Concurrent validates concurrent read behavior using the readManager.
func (t *fileTest) Test_ReadWithReadManager_Concurrent() {
	// Setup
	const (
		fileSize    = 1 * 1024 * 1024 // 1 MiB
		numReaders  = 20
		maxReadSize = 16 * 1024 // 16 KiB
	)
	// Create large content for the file.
	objectContent := make([]byte, fileSize)
	_, err := crypto_rand.Read(objectContent)
	assert.NoError(t.T(), err)

	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, &cfg.Config{}, parent, "concurrent_read_obj", objectContent, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)

	var wg sync.WaitGroup
	wg.Add(numReaders)

	// Run concurrent reads
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()

			// Each goroutine reads a random chunk.
			readSize := rand.Intn(maxReadSize-1) + 1 // Ensure readSize > 0
			offset := rand.Intn(fileSize - readSize)
			dst := make([]byte, readSize)

			fh.inode.Lock() // Lock required by ReadWithReadManager
			// The method is responsible for unlocking.
			_, n, err := fh.ReadWithReadManager(t.ctx, dst, int64(offset), 200)

			// Assertions
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readSize, n)
			assert.Equal(t.T(), objectContent[offset:offset+readSize], dst[:n])
		}()
	}

	// Wait for all goroutines to finish, with a timeout to detect deadlocks.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	// Test completed successfully.
	case <-time.After(10 * time.Second):
		t.T().Fatal("Test timed out, potential deadlock in ReadWithReadManager")
	}
}

// Test_Read_Concurrent validates concurrent read behavior using the random reader
func (t *fileTest) Test_Read_Concurrent() {
	// Setup
	const (
		fileSize    = 1 * 1024 * 1024 // 1 MiB
		numReaders  = 20
		maxReadSize = 16 * 1024 // 16 KiB
	)
	// Create large content for the file.
	objectContent := make([]byte, fileSize)
	_, err := crypto_rand.Read(objectContent)
	assert.NoError(t.T(), err)

	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, &cfg.Config{}, parent, "concurrent_read_obj", objectContent, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)

	var wg sync.WaitGroup
	wg.Add(numReaders)

	// Run concurrent reads
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()

			// Each goroutine reads a random chunk.
			readSize := rand.Intn(maxReadSize-1) + 1 // Ensure readSize > 0
			offset := rand.Intn(fileSize - readSize)
			dst := make([]byte, readSize)

			fh.inode.Lock() // Lock required by ReadWithReadManager
			// The method is responsible for unlocking.
			_, n, err := fh.Read(t.ctx, dst, int64(offset), 200)

			// Assertions
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readSize, n)
			assert.Equal(t.T(), objectContent[offset:offset+readSize], dst[:n])
		}()
	}

	// Wait for all goroutines to finish, with a timeout to detect deadlocks.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	// Test completed successfully.
	case <-time.After(10 * time.Second):
		t.T().Fatal("Test timed out, potential deadlock in ReadWithReadManager")
	}
}

// Test_ReadWithReadManager_ErrorScenarios verifies error handling in ReadWithReadManager.
func (t *fileTest) Test_ReadWithReadManager_ErrorScenarios() {
	type testCase struct {
		name      string
		returnErr error
	}

	object := gcs.MinObject{Name: "test_obj", Generation: 1}
	mockErr := fmt.Errorf("mock error")
	dst := make([]byte, 100)

	testCases := []testCase{
		{name: "EOF via readManager", returnErr: io.EOF},
		{name: "mock error via readManager", returnErr: mockErr},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			parent := createDirInode(&t.bucket, &t.clock)
			testInode := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, object.Name, []byte("data"), false)
			fh := NewFileHandle(testInode, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)
			fh.inode.Lock()
			mockRM := new(read_manager.MockReadManager)
			mockRM.On("ReadAt", t.ctx, dst, int64(0)).Return(gcsx.ReaderResponse{}, tc.returnErr)
			mockRM.On("Object").Return(&object)
			fh.readManager = mockRM

			output, n, err := fh.ReadWithReadManager(t.ctx, dst, 0, 200)

			assert.Zero(t.T(), n, "expected 0 bytes read")
			assert.Nil(t.T(), output, "expected output to be nil")
			assert.True(t.T(), errors.Is(err, tc.returnErr), "expected error to match")
			mockRM.AssertExpectations(t.T())
		})
	}
}

// Test_Read_ErrorScenarios verifies error handling in Read (random reader).
func (t *fileTest) Test_Read_ErrorScenarios() {
	type testCase struct {
		name      string
		returnErr error
	}

	object := gcs.MinObject{Name: "test_obj", Generation: 1}
	mockErr := fmt.Errorf("mock error")
	dst := make([]byte, 100)

	testCases := []testCase{
		{name: "EOF via random reader", returnErr: io.EOF},
		{name: "mock error via random reader", returnErr: mockErr},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			parent := createDirInode(&t.bucket, &t.clock)
			testInode := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, object.Name, []byte("data"), false)
			fh := NewFileHandle(testInode, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)
			fh.inode.Lock()
			mockReader := new(gcsx.MockRandomReader)
			mockReader.On("ReadAt", t.ctx, dst, int64(0)).Return(gcsx.ObjectData{}, tc.returnErr)
			mockReader.On("Object").Return(&object)
			fh.reader = mockReader

			output, n, err := fh.Read(t.ctx, dst, 0, 200)

			assert.Zero(t.T(), n, "expected 0 bytes read")
			assert.Nil(t.T(), output, "expected output to be nil")
			assert.True(t.T(), errors.Is(err, tc.returnErr), "expected error to match")
			mockReader.AssertExpectations(t.T())
		})
	}
}

// Test_ReadWithReadManager_FallbackToInode verifies that ReadWithReadManager
// falls back to inode object data when readManager is not valid.
func (t *fileTest) Test_ReadWithReadManager_FallbackToInode() {
	dst := make([]byte, 100)
	objectData := []byte("fallback data")
	object := gcs.MinObject{Name: "test_obj", Generation: 0}
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, object.Name, objectData, true)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)
	fh.inode.Lock()
	mockRM := new(read_manager.MockReadManager)
	fh.readManager = mockRM

	output, n, err := fh.ReadWithReadManager(t.ctx, dst, 0, 200)

	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), len(objectData), n)
	assert.Equal(t.T(), objectData, output[:n])
	mockRM.AssertExpectations(t.T())
}

// Test_Read_FallbackToInode verifies that Read falls back to inode object data
// when reader is not valid.
func (t *fileTest) Test_Read_FallbackToInode() {
	dst := make([]byte, 100)
	objectData := []byte("fallback data")
	object := gcs.MinObject{Name: "test_obj", Generation: 0}
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, object.Name, objectData, true)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)
	fh.inode.Lock()
	mockR := new(gcsx.MockRandomReader)
	fh.reader = mockR

	output, n, err := fh.Read(t.ctx, dst, 0, 200)

	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), len(objectData), n)
	assert.Equal(t.T(), objectData, output[:n])
	mockR.AssertExpectations(t.T())
}

func (t *fileTest) TestOpenMode() {
	testCases := []struct {
		name     string
		openMode util.OpenMode
	}{
		{
			name:     "OpenModeRead",
			openMode: util.Read,
		},
		{
			name:     "OpenModeWrite",
			openMode: util.Write,
		},
		{
			name:     "OpenModeAppend",
			openMode: util.Append,
		},
	}
	for _, tc := range testCases {
		parent := createDirInode(&t.bucket, &t.clock)
		config := &cfg.Config{Write: cfg.WriteConfig{EnableStreamingWrites: false}}
		in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj", nil, false)
		fh := NewFileHandle(in, nil, false, nil, tc.openMode, &cfg.Config{}, nil, nil)

		openMode := fh.OpenMode()

		assert.Equal(t.T(), tc.openMode, openMode)
	}
}

func (t *fileTest) TestFileHandle_Destroy_WithReaderAndReadManager() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	fileInode := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "destroy_test_obj", nil, false)
	// Create mocks
	mockReader := new(gcsx.MockRandomReader)
	mockReadManager := new(read_manager.MockReadManager)
	// Expect Destroy to be called on both
	mockReader.On("Destroy").Once()
	mockReadManager.On("Destroy").Once()
	// Construct file handle with mocks
	fh := NewFileHandle(fileInode, nil, false, nil, util.Read, config, nil, nil)
	fh.reader = mockReader
	fh.readManager = mockReadManager

	fh.Destroy()

	// Assert expectations
	mockReader.AssertExpectations(t.T())
	mockReadManager.AssertExpectations(t.T())
}

func (t *fileTest) TestFileHandle_Destroy_WithNilReaderAndReadManager() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	fileInode := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "destroy_test_nil_obj", nil, false)
	// Construct file handle with nils
	fh := NewFileHandle(fileInode, nil, false, nil, util.Read, config, nil, nil)
	fh.reader = nil
	fh.readManager = nil

	// Should not panic
	assert.NotPanics(t.T(), func() {
		fh.Destroy()
	})
}

func (t *fileTest) TestFileHandle_CheckInvariants_WithNonNilReaderAndManager() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	fileInode := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "destroy_test_obj", nil, false)
	// Create mocks
	mockReader := new(gcsx.MockRandomReader)
	mockRM := new(read_manager.MockReadManager)
	// Expectations
	mockReader.On("CheckInvariants").Once()
	mockRM.On("CheckInvariants").Once()
	fh := NewFileHandle(fileInode, nil, false, nil, util.Read, config, nil, nil)
	fh.reader = mockReader
	fh.readManager = mockRM

	assert.NotPanics(t.T(), func() {
		fh.checkInvariants()
	})

	mockReader.AssertExpectations(t.T())
	mockRM.AssertExpectations(t.T())
}

func (t *fileTest) TestFileHandle_CheckInvariants_WithNilReaderAndManager() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_check_invariants_nil", nil, false)

	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)

	// Should not panic even if both are nil
	assert.NotPanics(t.T(), func() {
		fh.checkInvariants()
	})
}

func (t *fileTest) Test_ReadWithReadManager_ReadManagerInvalidatedByGenerationChange() {
	content1 := []byte("content1")
	content2 := []byte("content2-larger")
	dst := make([]byte, len(content2))
	objectName := "test_obj_rm_gen_change"

	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, &cfg.Config{}, parent, objectName, content1, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), util.Read, &cfg.Config{}, nil, nil)

	// First read, to create a readManager.
	fh.inode.Lock()
	_, _, err := fh.ReadWithReadManager(t.ctx, make([]byte, len(content1)), 0, 200)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), fh.readManager)
	oldReadManager := fh.readManager

	// Now, update the object in GCS, which changes its generation.
	in.Lock()
	gcsSynced, err := in.Write(t.ctx, content2, 0, util.Write)
	assert.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)
	gcsSynced, err = in.Sync(t.ctx)
	assert.NoError(t.T(), err)
	assert.True(t.T(), gcsSynced)
	in.Unlock()

	// The existing readManager is now for an old generation.
	// The next ReadWithReadManager should detect this, destroy the old one, create a new one, and read the new content.
	fh.inode.Lock()
	output, n, err := fh.ReadWithReadManager(t.ctx, dst, 0, 200)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), fh.readManager)
	assert.NotEqual(t.T(), oldReadManager, fh.readManager)
	assert.Equal(t.T(), len(content2), n)
	assert.Equal(t.T(), content2, output)
}

func (t *fileTest) TestLockHandleAndRelockInode_Lock_NoDeadlockWithContention() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)
	var wg sync.WaitGroup
	const numContenders = 4
	wg.Add(2 * numContenders)
	done := make(chan struct{})

	// Simulate the flow that uses lockHandleAndRelockInode
	for i := 0; i < numContenders; i++ {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.lockHandleAndRelockInode(false) // This should not deadlock
			fh.inode.Unlock()
			fh.mu.Unlock()
		}()
	}

	// Simulate conflicting lock acquisition order
	for i := 0; i < numContenders; i++ {
		go func() {
			defer wg.Done()
			fh.mu.Lock()
			defer fh.mu.Unlock()
			// Now try to get the inode lock
			fh.inode.Lock()
			defer fh.inode.Unlock()
		}()
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	// Success, no deadlock
	case <-time.After(2 * time.Second):
		t.T().Fatal("Potential deadlock detected with Lock")
	}
}

func (t *fileTest) TestLockHandleAndRelockInode_RLock_NoDeadlockWithContention() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)
	var wg sync.WaitGroup
	const numContenders = 4
	wg.Add(2 * numContenders)
	done := make(chan struct{})

	// Simulate the flow that uses lockHandleAndRelockInode
	for i := 0; i < numContenders; i++ {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.lockHandleAndRelockInode(true) // This should not deadlock
			fh.inode.Unlock()
			fh.mu.RUnlock()
		}()
	}

	// Simulate conflicting lock acquisition order
	for i := 0; i < numContenders; i++ {
		go func() {
			defer wg.Done()
			fh.mu.Lock()
			defer fh.mu.Unlock()
			// Now try to get the inode lock
			fh.inode.Lock()
			defer fh.inode.Unlock()
		}()
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	// Success, no deadlock
	case <-time.After(2 * time.Second):
		t.T().Fatal("Potential deadlock detected with RLock")
	}
}

func (t *fileTest) TestLockHandleAndRelockInode_Mixed_NoDeadlockWithContention() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)
	var wg sync.WaitGroup
	const numRContenders = 4
	const numWContenders = 4
	wg.Add(numRContenders + numWContenders)
	done := make(chan struct{})

	// Simulate the flow that uses lockHandleAndRelockInode(false)
	for i := 0; i < numWContenders; i++ {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.lockHandleAndRelockInode(false) // This should not deadlock
			fh.mu.Unlock()
			fh.inode.Unlock()
		}()
	}

	// Simulate the flow that uses lockHandleAndRelockInode(true)
	for i := 0; i < numRContenders; i++ {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.lockHandleAndRelockInode(true) // This should not deadlock
			fh.mu.RUnlock()
			fh.inode.Unlock()
		}()
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	// Success, no deadlock
	case <-time.After(2 * time.Second):
		t.T().Fatal("Potential deadlock detected with mixed Lock/RLock")
	}
}

func (t *fileTest) TestUnlockHandleAndInode_Unlock() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)

	var wg sync.WaitGroup
	const numContenders = 4
	wg.Add(3 * numContenders)
	done := make(chan struct{})

	for i := 0; i < numContenders; i++ {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.mu.Lock()
			fh.unlockHandleAndInode(false)
		}()
	}

	for i := 0; i < numContenders; i++ {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.mu.RLock()
			fh.unlockHandleAndInode(true)
		}()
	}

	for i := 0; i < numContenders; i++ {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.mu.Lock()
			fh.inode.Unlock()
			fh.mu.Unlock()
		}()
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	// Success: locks were re-acquired without blocking.
	case <-time.After(1 * time.Second):
		t.T().Fatal("Potential deadlock detected: locks were not released for write lock.")
	}
}

func (t *fileTest) TestUnlockHandleAndInode_RUnlock() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, util.Read, config, nil, nil)

	// Lock in the required order.
	fh.inode.Lock()
	fh.mu.RLock()

	// Unlock using the function under test.
	fh.unlockHandleAndInode(true)

	// Verify both locks are released by trying to lock them again.
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Use a write lock to ensure the read lock is fully released.
		fh.mu.Lock()
		defer fh.mu.Unlock()
		fh.inode.Lock()
		defer fh.inode.Unlock()
	}()

	select {
	case <-done:
	// Success: locks were re-acquired without blocking.
	case <-time.After(1 * time.Second):
		t.T().Fatal("Potential deadlock detected: locks were not released for read lock.")
	}
}
