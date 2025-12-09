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
	"os"
	"sync"
	"testing"

	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/read_manager"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const testDirName = "parentRoot"

var readMode = util.NewOpenMode(util.ReadOnly, 0)
var writeMode = util.NewOpenMode(util.WriteOnly, 0)

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
		true,
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
	fh := NewFileHandle(in, nil, false, nil, writeMode, &cfg.Config{}, nil, nil, 0)
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

func (t *fileTest) Test_IsValidReadManager_NilReadManager() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	const objectName = "test_obj"
	const objectContent = "some data"
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, objectName, []byte(objectContent), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)
	fh.inode.Lock()
	defer fh.inode.Unlock()
	fh.readManager = nil

	result := fh.isValidReadManager()

	assert.False(t.T(), result)
}

func (t *fileTest) Test_IsValidReadManager_GenerationValidation() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	const objectName = "test_obj"
	const objectContent = "some data"
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, objectName, []byte(objectContent), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)
	fh.inode.Lock()
	defer fh.inode.Unlock()

	testCases := []struct {
		name             string
		readerGeneration int64
		expectedIsValid  bool
	}{
		{
			name:             "Generation mismatch",
			readerGeneration: 2, // Inode has generation 1
			expectedIsValid:  false,
		},
		{
			name:             "Generation match",
			readerGeneration: 1, // Inode has generation 1
			expectedIsValid:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			minObj := in.Source()
			minObj.Generation = tc.readerGeneration
			fh.readManager = read_manager.NewReadManager(minObj, &t.bucket, &read_manager.ReadManagerConfig{Config: config})

			result := fh.isValidReadManager()

			assert.Equal(t.T(), tc.expectedIsValid, result)
		})
	}
}

func (t *fileTest) Test_IsValidReader_NilReader() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	const objectName = "test_obj"
	const objectContent = "some data"
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, objectName, []byte(objectContent), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)
	fh.inode.Lock()
	defer fh.inode.Unlock()
	fh.reader = nil

	result := fh.isValidReader()

	assert.False(t.T(), result)
}

func (t *fileTest) Test_IsValidReader_GenerationValidation() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	const objectName = "test_obj"
	const objectContent = "some data"
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, objectName, []byte(objectContent), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)
	fh.inode.Lock()
	defer fh.inode.Unlock()

	testCases := []struct {
		name             string
		readerGeneration int64
		expectedIsValid  bool
	}{
		{
			name:             "Generation mismatch",
			readerGeneration: 2, // Inode has generation 1
			expectedIsValid:  false,
		},
		{
			name:             "Generation match",
			readerGeneration: 1, // Inode has generation 1
			expectedIsValid:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			minObj := in.Source()
			minObj.Generation = tc.readerGeneration
			fh.reader = gcsx.NewRandomReader(minObj, &t.bucket, 200, nil, false, metrics.NewNoopMetrics(), &in.MRDWrapper, config, 0)

			result := fh.isValidReader()

			assert.Equal(t.T(), tc.expectedIsValid, result)
		})
	}
}

// Test_Read_Success validates successful read behavior using the random reader.
func (t *fileTest) Test_Read_Success() {
	expectedData := []byte("hello from reader")
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, "test_obj_reader", expectedData, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)
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
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)
	buf := make([]byte, len(expectedData))
	fh.inode.Lock()

	resp, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 0,
	}, 200)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(expectedData), resp.Size)
	assert.Equal(t.T(), expectedData, buf[:resp.Size])
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
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)

	var wg sync.WaitGroup
	wg.Add(numReaders)

	// Run concurrent reads
	for range numReaders {
		go func() {
			defer wg.Done()

			// Each goroutine reads a random chunk.
			readSize := rand.Intn(maxReadSize-1) + 1 // Ensure readSize > 0
			offset := rand.Intn(fileSize - readSize)
			dst := make([]byte, readSize)

			fh.inode.Lock() // Lock required by ReadWithReadManager
			// The method is responsible for unlocking.
			resp, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
				Buffer: dst,
				Offset: int64(offset),
			}, 200)

			// Assertions
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readSize, resp.Size)
			assert.Equal(t.T(), objectContent[offset:offset+readSize], dst[:resp.Size])
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
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)

	var wg sync.WaitGroup
	wg.Add(numReaders)

	// Run concurrent reads
	for range numReaders {
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
			fh := NewFileHandle(testInode, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)
			fh.inode.Lock()
			mockRM := new(read_manager.MockReadManager)
			mockRM.On("ReadAt", t.ctx, mock.AnythingOfType("*gcsx.ReadRequest")).Return(gcsx.ReadResponse{}, tc.returnErr)
			mockRM.On("Object").Return(&object)
			fh.readManager = mockRM

			resp, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
				Buffer: dst,
				Offset: 0,
			}, 200)

			assert.Zero(t.T(), resp.Size, "expected 0 bytes read")
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
			fh := NewFileHandle(testInode, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)
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
	object := gcs.MinObject{Name: "test_obj"}
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, object.Name, objectData, true)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)
	fh.inode.Lock()
	mockRM := new(read_manager.MockReadManager)
	fh.readManager = mockRM

	resp, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
		Buffer: dst,
		Offset: 0,
	}, 200)

	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), len(objectData), resp.Size)
	assert.Equal(t.T(), objectData, dst[:resp.Size])
	mockRM.AssertExpectations(t.T())
}

// Test_Read_FallbackToInode verifies that Read falls back to inode object data
// when reader is not valid.
func (t *fileTest) Test_Read_FallbackToInode() {
	dst := make([]byte, 100)
	objectData := []byte("fallback data")
	object := gcs.MinObject{Name: "test_obj"}
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, object.Name, objectData, true)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)
	fh.inode.Lock()
	mockR := new(gcsx.MockRandomReader)
	fh.reader = mockR

	output, n, err := fh.Read(t.ctx, dst, 0, 200)

	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), len(objectData), n)
	assert.Equal(t.T(), objectData, output[:n])
	mockR.AssertExpectations(t.T())
}

func (t *fileTest) Test_ReadWithReadManager_ReadManagerInvalidatedByGenerationChange() {
	content1 := []byte("content1")
	content2 := []byte("content2-larger")
	dst := make([]byte, len(content2))
	objectName := "test_obj_rm_gen_change"

	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, &cfg.Config{}, parent, objectName, content1, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)

	// First read, to create a readManager.
	fh.inode.Lock()
	_, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
		Buffer: make([]byte, len(content1)),
		Offset: 0,
	}, 200)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), fh.readManager)
	oldReadManager := fh.readManager

	// Now, update the object in GCS, which changes its generation.
	in.Lock()
	gcsSynced, err := in.Write(t.ctx, content2, 0, writeMode)
	assert.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)
	gcsSynced, err = in.Sync(t.ctx)
	assert.NoError(t.T(), err)
	assert.True(t.T(), gcsSynced)
	in.Unlock()

	// The existing readManager is now for an old generation.
	// The next ReadWithReadManager call should detect this, destroy the old one,
	// create a new one, and read the new content.
	fh.inode.Lock()
	resp, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
		Buffer: dst,
		Offset: 0,
	}, 200)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), fh.readManager)
	assert.NotEqual(t.T(), oldReadManager, fh.readManager)
	assert.Equal(t.T(), len(content2), resp.Size)
	assert.Equal(t.T(), content2, dst)
}

func (t *fileTest) Test_Read_ReaderInvalidatedByGenerationChange() {
	content1 := []byte("content1")
	content2 := []byte("content2-larger")
	dst := make([]byte, len(content2))
	objectName := "test_obj_rm_gen_change"

	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, &cfg.Config{}, parent, objectName, content1, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, &cfg.Config{}, nil, nil, 0)

	// First read, to create a reader.
	fh.inode.Lock()
	_, _, err := fh.Read(t.ctx, make([]byte, len(content1)), 0, 200)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), fh.reader)
	oldReader := fh.reader

	// Now, update the object in GCS, which changes its generation.
	in.Lock()
	gcsSynced, err := in.Write(t.ctx, content2, 0, writeMode)
	assert.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)
	gcsSynced, err = in.Sync(t.ctx)
	assert.NoError(t.T(), err)
	assert.True(t.T(), gcsSynced)
	in.Unlock()

	// The existing reader is now for an old generation.
	// The next Read call should detect this, destroy the old one,
	// create a new one, and read the new content.
	fh.inode.Lock()
	output, n, err := fh.Read(t.ctx, dst, 0, 200)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), fh.reader)
	assert.NotEqual(t.T(), oldReader, fh.reader)
	assert.Equal(t.T(), len(content2), n)
	assert.Equal(t.T(), content2, output)
}

func (t *fileTest) TestOpenMode() {
	testCases := []struct {
		name     string
		openMode util.OpenMode
	}{
		{
			name:     "OpenModeRead",
			openMode: util.NewOpenMode(util.ReadOnly, 0),
		},
		{
			name:     "OpenModeWrite",
			openMode: util.NewOpenMode(util.WriteOnly, 0),
		},
		{
			name:     "OpenModeAppend",
			openMode: util.NewOpenMode(util.WriteOnly, util.O_APPEND),
		},
		{
			name:     "OpenModeReadWithODirect",
			openMode: util.NewOpenMode(util.ReadOnly, util.O_DIRECT),
		},
		{
			name:     "OpenModeWriteWithODirect",
			openMode: util.NewOpenMode(util.WriteOnly, util.O_DIRECT),
		},
		{
			name:     "OpenModeAppendWithODirect",
			openMode: util.NewOpenMode(util.WriteOnly, util.O_APPEND|util.O_DIRECT),
		},
	}
	for _, tc := range testCases {
		parent := createDirInode(&t.bucket, &t.clock)
		config := &cfg.Config{Write: cfg.WriteConfig{EnableStreamingWrites: false}}
		in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj", nil, false)
		fh := NewFileHandle(in, nil, false, nil, tc.openMode, &cfg.Config{}, nil, nil, 0)

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
	fh := NewFileHandle(fileInode, nil, false, nil, readMode, config, nil, nil, 0)
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
	fh := NewFileHandle(fileInode, nil, false, nil, readMode, config, nil, nil, 0)
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
	fh := NewFileHandle(fileInode, nil, false, nil, readMode, config, nil, nil, 0)
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

	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)

	// Should not panic even if both are nil
	assert.NotPanics(t.T(), func() {
		fh.checkInvariants()
	})
}

func (t *fileTest) Test_LockHandleAndRelockInode_Lock_NoDeadlockWithContention() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)
	var wg sync.WaitGroup
	const numContenders = 10
	wg.Add(2 * numContenders)
	done := make(chan struct{})

	// Simulate the flow that uses lockHandleAndRelockInode
	for range numContenders {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.lockHandleAndRelockInode(false) // This should not deadlock
			fh.inode.Unlock()
			fh.mu.Unlock()
		}()
	}

	// Simulate conflicting lock acquisition order
	for range numContenders {
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

func (t *fileTest) Test_LockHandleAndRelockInode_RLock_NoDeadlockWithContention() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)
	var wg sync.WaitGroup
	const numContenders = 10
	wg.Add(2 * numContenders)
	done := make(chan struct{})

	// Simulate the flow that uses lockHandleAndRelockInode
	for range numContenders {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.lockHandleAndRelockInode(true) // This should not deadlock
			fh.inode.Unlock()
			fh.mu.RUnlock()
		}()
	}

	// Simulate conflicting lock acquisition order
	for range numContenders {
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

func (t *fileTest) Test_LockHandleAndRelockInode_Mixed_NoDeadlockWithContention() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)
	var wg sync.WaitGroup
	const numRContenders = 10
	const numWContenders = 10
	wg.Add(numRContenders + numWContenders)
	done := make(chan struct{})

	// Simulate the flow that uses lockHandleAndRelockInode(false)
	for range numWContenders {
		go func() {
			defer wg.Done()
			fh.inode.Lock()
			fh.lockHandleAndRelockInode(false) // This should not deadlock
			fh.mu.Unlock()
			fh.inode.Unlock()
		}()
	}

	// Simulate the flow that uses lockHandleAndRelockInode(true)
	for range numRContenders {
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

func (t *fileTest) Test_UnlockHandleAndInode() {
	parent := createDirInode(&t.bucket, &t.clock)
	config := &cfg.Config{}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_deadlock", []byte("content"), false)
	fh := NewFileHandle(in, nil, false, nil, readMode, config, nil, nil, 0)

	var wg sync.WaitGroup
	const numContenders = 10
	wg.Add(3 * numContenders)
	done := make(chan struct{})

	for range numContenders {
		go func() {
			defer wg.Done()
			fh.mu.Lock()
			fh.inode.Lock()
			fh.unlockHandleAndInode(false)
		}()
	}

	for range numContenders {
		go func() {
			defer wg.Done()
			fh.mu.RLock()
			fh.inode.Lock()
			fh.unlockHandleAndInode(true)
		}()
	}

	for range numContenders {
		go func() {
			defer wg.Done()
			fh.mu.Lock()
			defer fh.mu.Unlock()
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
	// Success: locks were re-acquired without blocking.
	case <-time.After(2 * time.Second):
		t.T().Fatal("Potential deadlock detected: locks were not released for write lock.")
	}
}

func (t *fileTest) Test_ReadWithReadManager_FullReadSuccessWithBufferedRead() {
	const (
		fileSize = 1 * 1024 * 1024 // 1 MiB
	)
	expectedData := util.GenerateRandomBytes(fileSize)
	// Setup for Buffered Read test case
	config := &cfg.Config{
		Read: cfg.ReadConfig{
			EnableBufferedRead:   true,
			MaxBlocksPerHandle:   10,
			BlockSizeMb:          1,
			StartBlocksPerHandle: 2,
		},
	}
	workerPool, err := workerpool.NewStaticWorkerPoolForCurrentCPU(20)
	require.NoError(t.T(), err)
	defer workerPool.Stop()
	globalSemaphore := semaphore.NewWeighted(20) // Sufficient blocks for the test
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "read_obj", expectedData, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, config, workerPool, globalSemaphore, 0)
	fh.inode.Lock()
	buf := make([]byte, fileSize)

	// ReadWithReadManager will unlock the inode.
	resp, err := fh.ReadWithReadManager(context.Background(), &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 0,
	}, 200)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), fileSize, resp.Size)
	assert.Equal(t.T(), expectedData, util.ConvertReadResponseToBytes(resp.Data, resp.Size))
}

func (t *fileTest) Test_ShouldSkipSizeChecks() {
	const objectSize = 100
	unfinalizedObject := &gcs.MinObject{Name: "unfinalized", Size: objectSize}
	finalizedObject := &gcs.MinObject{Name: "finalized", Size: objectSize, Finalized: time.Now()}
	directIOReadMode := util.NewOpenMode(util.ReadOnly, util.O_DIRECT)
	readOnlyMode := util.NewOpenMode(util.ReadOnly, 0)

	testCases := []struct {
		name              string
		object            *gcs.MinObject
		openMode          util.OpenMode
		offset            int64
		bufferSize        int
		expectedToSkip    bool
		useNilReadManager bool
	}{
		{
			name:           "All conditions met: unfinalized, direct I/O, positive offset, extends beyond size",
			object:         unfinalizedObject,
			openMode:       directIOReadMode,
			offset:         50,
			bufferSize:     60, // 50 + 60 > 100
			expectedToSkip: true,
		},
		{
			name:           "Finalized object: should not skip",
			object:         finalizedObject,
			openMode:       directIOReadMode,
			offset:         50,
			bufferSize:     60,
			expectedToSkip: false,
		},
		{
			name:           "Not direct I/O: should not skip",
			object:         unfinalizedObject,
			openMode:       readOnlyMode,
			offset:         50,
			bufferSize:     60,
			expectedToSkip: false,
		},
		{
			name:           "Negative offset: should not skip",
			object:         unfinalizedObject,
			openMode:       directIOReadMode,
			offset:         -10,
			bufferSize:     20,
			expectedToSkip: false,
		},
		{
			name:           "Read within size: should not skip",
			object:         unfinalizedObject,
			openMode:       directIOReadMode,
			offset:         50,
			bufferSize:     50, // 50 + 50 <= 100
			expectedToSkip: false,
		},
		{
			name:           "Read exactly at size boundary: should not skip",
			object:         unfinalizedObject,
			openMode:       directIOReadMode,
			offset:         100,
			bufferSize:     0,
			expectedToSkip: false,
		},
		{
			name:           "Read starts at size and extends: should skip",
			object:         unfinalizedObject,
			openMode:       directIOReadMode,
			offset:         100,
			bufferSize:     10, // 100 + 10 > 100
			expectedToSkip: true,
		},
		{
			name:              "Nil read manager: should panic",
			object:            unfinalizedObject,
			openMode:          directIOReadMode,
			offset:            100,
			bufferSize:        10,
			useNilReadManager: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			parent := createDirInode(&t.bucket, &t.clock)
			config := &cfg.Config{}
			in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, tc.object.Name, nil, false)
			fh := NewFileHandle(in, nil, false, nil, tc.openMode, config, nil, nil, 0)

			if tc.useNilReadManager {
				fh.readManager = nil
				req := &gcsx.ReadRequest{Offset: tc.offset, Buffer: make([]byte, tc.bufferSize)}
				assert.Panics(t.T(), func() { fh.shouldSkipSizeChecks(req) })
				return
			}

			rmConfig := &read_manager.ReadManagerConfig{Config: config}
			fh.readManager = read_manager.NewReadManager(tc.object, &t.bucket, rmConfig)

			req := &gcsx.ReadRequest{Offset: tc.offset, Buffer: make([]byte, tc.bufferSize)}

			skip := fh.shouldSkipSizeChecks(req)

			assert.Equal(t.T(), tc.expectedToSkip, skip)
		})
	}
}
func (t *fileTest) Test_ReadWithReadManager_ConcurrentReadsWithBufferedReader() {
	const (
		fileSize      = 9 * 1024 * 1024 // 9 MiB
		numGoroutines = 3
	)
	// Create expected data for the file.
	expectedData := util.GenerateRandomBytes(fileSize)
	// Setup configuration for buffered read.
	config := &cfg.Config{
		Read: cfg.ReadConfig{
			EnableBufferedRead:   true,
			MaxBlocksPerHandle:   10,
			StartBlocksPerHandle: 2,
			BlockSizeMb:          1,
			RandomSeekThreshold:  3,
		},
	}
	workerPool, err := workerpool.NewStaticWorkerPoolForCurrentCPU(20)
	require.NoError(t.T(), err)
	defer workerPool.Stop()
	globalSemaphore := semaphore.NewWeighted(20)
	// Create mock inode and file handle.
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "read_obj", expectedData, false)
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, config, workerPool, globalSemaphore, 0)
	// Use a WaitGroup to synchronize goroutines.
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	readSize := fileSize / numGoroutines
	results := make([][]byte, numGoroutines)
	for i := range numGoroutines {
		go func(index int) {
			defer wg.Done()
			offset := int64(index * readSize)
			readBuf := make([]byte, readSize)
			fh.inode.Lock()

			// Each goroutine use same file handle.
			resp, err := fh.ReadWithReadManager(context.Background(), &gcsx.ReadRequest{
				Buffer: readBuf,
				Offset: offset,
			}, int32(readSize))

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readSize, resp.Size)
			results[index] = util.ConvertReadResponseToBytes(resp.Data, resp.Size)
		}(i)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	// Combine the results from all goroutines.
	combinedResult := make([]byte, 0, fileSize)
	for _, res := range results {
		combinedResult = append(combinedResult, res...)
	}
	// Final assertion: compare the combined result with the original expected data.
	assert.Equal(t.T(), expectedData, combinedResult, "Combined result should match expected data.")
	// Clean up the original file handle.
	fh.Destroy()
}

// Test_ReadWithReadManager_WorkloadInsightVisual validates that when
// Workload Insight visualization is enabled, the output file is created
// after performing reads with ReadWithReadManager.
func (t *fileTest) Test_ReadWithReadManager_WorkloadInsightVisual() {
	config := &cfg.Config{
		WorkloadInsight: cfg.WorkloadInsightConfig{
			Visualize:  true,
			OutputFile: "test.txt",
		},
	}
	const (
		fileSize = 9 * 1024 * 1024 // 9 MiB
		MiB      = 1024 * 1024
	)
	content := util.GenerateRandomBytes(fileSize)
	// Create a new file handle with the updated config.
	parent := createDirInode(&t.bucket, &t.clock)
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj_visual", content, false)
	in.Lock()
	fh := NewFileHandle(in, nil, false, metrics.NewNoopMetrics(), readMode, config, nil, nil, 0)
	in.Unlock()

	// Perform multiple reads and destroy the file-handle.
	fh.inode.Lock()
	dst := make([]byte, MiB)
	// First read.
	resp1, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
		Buffer: dst,
		Offset: MiB,
	}, 200)
	require.NoError(t.T(), err)
	require.Equal(t.T(), MiB, resp1.Size)
	require.Equal(t.T(), content[MiB:2*MiB], dst[:resp1.Size])
	clear(dst)
	// Second read.
	fh.inode.Lock()
	resp2, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{
		Buffer: dst,
		Offset: 0,
	}, 200)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), MiB, resp2.Size)
	assert.Equal(t.T(), content[0:MiB], dst[:resp2.Size])
	clear(dst)
	// Third read.
	fh.inode.Lock()
	resp3, err := fh.ReadWithReadManager(t.ctx, &gcsx.ReadRequest{Buffer: dst, Offset: 2 * MiB}, 200)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), MiB, resp3.Size)
	assert.Equal(t.T(), content[2*MiB:3*MiB], dst[:resp3.Size])
	fh.Destroy()

	// Validate the output file creation for workload insight.
	assert.FileExists(t.T(), "test.txt")
	require.NoError(t.T(), os.Remove("test.txt")) // Clean up the file after test.
}
