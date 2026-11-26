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

// Streaming write tests for parallel uploads: on a Rapid Cache Ultra bucket with
// rapid writes and parallel uploads enabled, new files are written through the
// unbuffered write handler.

package fs_test

import (
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v4/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v4/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StreamingWritesParallelUploadTest struct {
	suite.Suite
	fsTest
	// fileName is the name of t.f1. Each test uses a new name, so that it doesn't
	// meet the inode of an earlier test that the kernel hasn't forgotten yet.
	fileName string
}

func TestStreamingWritesParallelUploadTest(t *testing.T) {
	suite.Run(t, new(StreamingWritesParallelUploadTest))
}

func (t *StreamingWritesParallelUploadTest) SetupSuite() {
	t.serverCfg.NewConfig = &cfg.Config{
		Write: cfg.WriteConfig{
			BlockSizeMb:           1,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			// The unbuffered write handler doesn't use write.global-max-blocks. With
			// 0, the buffered write handler falls back to staged writes, so the tests
			// below also fail if writes don't go through the unbuffered write handler.
			GlobalMaxBlocks:  0,
			MaxBlocksPerFile: 10,
		},
		MetadataCache: cfg.MetadataCacheConfig{TtlSecs: 0},
	}
	t.mountCfg.DisableWritebackCaching = true
	bucketType = gcs.BucketType{RCU: true}.WithRapidWriteStrategy(cfg.RapidWriteStrategyEnabled).WithParallelUploadEnabled(true)
	t.SetUpTestSuite()
}

func (t *StreamingWritesParallelUploadTest) TearDownSuite() {
	t.TearDownTestSuite()
	// Don't leak the bucket type into the suites that run next.
	bucketType = gcs.BucketType{}
}

func (t *StreamingWritesParallelUploadTest) SetupTest() {
	t.fileName = "parallel_" + setup.GenerateRandomString(5)
	// CreateLocalFile creates a local file and validates that the object doesn't
	// exist in GCS.
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, t.fileName)
}

func (t *StreamingWritesParallelUploadTest) TearDownTest() {
	t.TearDown()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// closeF1 closes t.f1 and clears it, so that TearDown doesn't close it again.
func (t *StreamingWritesParallelUploadTest) closeF1() error {
	err := t.f1.Close()
	t.f1 = nil
	return err
}

func (t *StreamingWritesParallelUploadTest) readObject(name string) string {
	content, err := storageutil.ReadObject(ctx, bucket, name)
	require.NoError(t.T(), err)
	return string(content)
}

// readThroughNewHandle reads size bytes of t.f1 through a new O_DIRECT file
// handle, so that the read reaches GCSFuse instead of the page cache.
func (t *StreamingWritesParallelUploadTest) readThroughNewHandle(size int) (string, error) {
	reader, err := os.OpenFile(t.f1.Name(), os.O_RDONLY|syscall.O_DIRECT, filePerms)
	if err != nil {
		return "", err
	}
	buf := make([]byte, size)
	n, readErr := reader.ReadAt(buf, 0)
	if closeErr := reader.Close(); readErr == nil {
		readErr = closeErr
	}
	return string(buf[:n]), readErr
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StreamingWritesParallelUploadTest) TestSyncOnEmptyFileDoesNotCreateObject() {
	err := t.f1.Sync()

	require.NoError(t.T(), err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, t.fileName)
}

func (t *StreamingWritesParallelUploadTest) TestCloseOnEmptyFileCreatesEmptyObject() {
	err := t.closeF1()

	require.NoError(t.T(), err)
	assert.Empty(t.T(), t.readObject(t.fileName))
}

func (t *StreamingWritesParallelUploadTest) TestSyncDoesNotMakeWrittenDataVisible() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)

	err = t.f1.Sync()

	require.NoError(t.T(), err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, t.fileName)
}

func (t *StreamingWritesParallelUploadTest) TestCloseFinalizesWrittenData() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)
	require.NoError(t.T(), t.f1.Sync())
	_, err = t.f1.Write([]byte("bar"))
	require.NoError(t.T(), err)

	err = t.closeF1()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", t.readObject(t.fileName))
}

func (t *StreamingWritesParallelUploadTest) TestTruncateBeyondSizeIsZeroPaddedOnClose() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)
	require.NoError(t.T(), t.f1.Truncate(6))
	fileInfo, err := t.f1.Stat()
	require.NoError(t.T(), err)
	require.Equal(t.T(), int64(6), fileInfo.Size())

	err = t.closeF1()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "foo\x00\x00\x00", t.readObject(t.fileName))
}

func (t *StreamingWritesParallelUploadTest) TestTruncateBelowSizeFallsBackToStagedWrites() {
	_, err := t.f1.Write([]byte("foobar"))
	require.NoError(t.T(), err)
	require.NoError(t.T(), t.f1.Truncate(3))
	// The fallback finalizes the object written so far.
	require.Equal(t.T(), "foobar", t.readObject(t.fileName))

	err = t.closeF1()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "foo", t.readObject(t.fileName))
}

func (t *StreamingWritesParallelUploadTest) TestOutOfOrderWriteFallsBackToStagedWrites() {
	_, err := t.f1.Write([]byte("foobar"))
	require.NoError(t.T(), err)
	_, err = t.f1.WriteAt([]byte("foo"), 3)
	require.NoError(t.T(), err)
	// The fallback finalizes the object written before the out-of-order write.
	require.Equal(t.T(), "foobar", t.readObject(t.fileName))

	err = t.closeF1()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "foofoo", t.readObject(t.fileName))
}

func (t *StreamingWritesParallelUploadTest) TestUnlinkBeforeWriteCreatesNoObject() {
	require.NoError(t.T(), os.Remove(t.f1.Name()))

	_, writeErr := t.f1.Write([]byte("foo"))
	closeErr := t.closeF1()

	assert.NoError(t.T(), writeErr)
	assert.NoError(t.T(), closeErr)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, t.fileName)
}

func (t *StreamingWritesParallelUploadTest) TestWritesAfterUnlinkAreDiscarded() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)
	require.NoError(t.T(), os.Remove(t.f1.Name()))

	_, writeErr := t.f1.Write([]byte("bar"))
	closeErr := t.closeF1()

	assert.NoError(t.T(), writeErr)
	assert.NoError(t.T(), closeErr)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, t.fileName)
}

func (t *StreamingWritesParallelUploadTest) TestUnlinkAfterFallbackDeletesObject() {
	_, err := t.f1.Write([]byte("foobar"))
	require.NoError(t.T(), err)
	require.NoError(t.T(), t.f1.Truncate(3))

	err = os.Remove(t.f1.Name())

	require.NoError(t.T(), err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, t.fileName)
	assert.NoError(t.T(), t.closeF1())
}

func (t *StreamingWritesParallelUploadTest) TestRenameFinalizesPendingWrites() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)
	newFileName := "parallel_renamed_" + setup.GenerateRandomString(5)

	err = os.Rename(t.f1.Name(), path.Join(mntDir, newFileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "foo", t.readObject(newFileName))
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, t.fileName)
	assert.NoError(t.T(), t.closeF1())
}

func (t *StreamingWritesParallelUploadTest) TestReadDuringWriteFinalizesObject() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)

	content, err := t.readThroughNewHandle(3)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "foo", content)
	assert.Equal(t.T(), "foo", t.readObject(t.fileName))
}

func (t *StreamingWritesParallelUploadTest) TestWriteAfterReadUsesStagedWrites() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)
	_, err = t.readThroughNewHandle(3)
	require.NoError(t.T(), err)
	_, err = t.f1.Write([]byte("bar"))
	require.NoError(t.T(), err)

	err = t.closeF1()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", t.readObject(t.fileName))
}

func (t *StreamingWritesParallelUploadTest) TestAppendToExistingObjectUsesStagedWrites() {
	existingFileName := "parallel_existing_" + setup.GenerateRandomString(5)
	require.NoError(t.T(), t.createObjects(map[string]string{existingFileName: "taco"}))
	f, err := os.OpenFile(path.Join(mntDir, existingFileName), os.O_WRONLY|os.O_APPEND, filePerms)
	require.NoError(t.T(), err)
	_, err = f.Write([]byte("xyz"))
	require.NoError(t.T(), err)

	err = f.Close()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "tacoxyz", t.readObject(existingFileName))
}

func (t *StreamingWritesParallelUploadTest) TestObjectClobberedDuringWriteReturnsStaleFileHandleOnClose() {
	_, err := t.f1.Write([]byte("foo"))
	require.NoError(t.T(), err)
	_, err = storageutil.CreateObject(ctx, bucket, t.fileName, []byte("foobar"))
	require.NoError(t.T(), err)

	err = t.closeF1()

	operations.ValidateESTALEError(t.T(), err)
	assert.Equal(t.T(), "foobar", t.readObject(t.fileName))
}
