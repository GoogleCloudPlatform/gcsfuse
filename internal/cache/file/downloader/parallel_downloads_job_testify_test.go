package downloader

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type ParallelDownloaderJobTestifyTest struct {
	JobTestifyTest
}

func TestParallelDownloaderJobTestifyTestSuite(testSuite *testing.T) {
	suite.Run(testSuite, new(ParallelDownloaderJobTestifyTest))
}

func (t *ParallelDownloaderJobTestifyTest) SetupTest() {
	t.defaultFileCacheConfig = &cfg.FileCacheConfig{
		EnableParallelDownloads:  true,
		ParallelDownloadsPerFile: 3,
		DownloadChunkSizeMb:      3,
		EnableCrc:                true,
		WriteBufferSize:          4 * 1024 * 1024,
	}
	t.ctx, _ = context.WithCancel(context.Background())
	t.mockBucket = new(storage.TestifyMockBucket)
}

func (t *ParallelDownloaderJobTestifyTest) Test_ParallelDownloadObjectToFile_NewReaderWithReadHandle() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	t.initReadCacheTestifyTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	t.job.cancelCtx, t.job.cancelFunc = context.WithCancel(context.Background())
	// Add subscriber
	subscribedOffset := int64(1 * util.MiB)
	notificationC := t.job.subscribe(subscribedOffset)
	file, err := util.CreateFile(data.FileSpec{Path: t.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	assert.Equal(t.T(), nil, err)
	defer func() {
		_ = file.Close()
	}()
	// To download a file of 10mb using ParallelDownloadsPerFile = 3 and
	// DownloadChunkSizeMb = 3mb there will be one call to NewReaderWithReadHandle
	// with read handle.
	handle := []byte("opaque-handle")
	rc1 := io.NopCloser(strings.NewReader(string(objectContent[0 : 3*util.MiB])))
	rd1 := &fake.FakeReader{ReadCloser: rc1, Handle: handle}
	rc2 := io.NopCloser(strings.NewReader(string(objectContent[3*util.MiB : 6*util.MiB])))
	rd2 := &fake.FakeReader{ReadCloser: rc2, Handle: handle}
	rc3 := io.NopCloser(strings.NewReader(string(objectContent[6*util.MiB : 9*util.MiB])))
	rd3 := &fake.FakeReader{ReadCloser: rc3, Handle: handle}
	rc4 := io.NopCloser(strings.NewReader(string(objectContent[9*util.MiB : 10*util.MiB])))
	rd4 := &fake.FakeReader{ReadCloser: rc4, Handle: handle}
	t.mockBucket.On("Name").Return(storage.TestBucketName)
	readObjectReq := gcs.ReadObjectRequest{Name: objectName, Range: &gcs.ByteRange{Start: 0, Limit: 3 * util.MiB}, ReadHandle: nil}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, &readObjectReq).Return(rd1, nil).Times(1)
	readObjectReq2 := gcs.ReadObjectRequest{Name: objectName, Range: &gcs.ByteRange{Start: 3 * util.MiB, Limit: 6 * util.MiB}, ReadHandle: nil}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, &readObjectReq2).Return(rd2, nil).Times(1)
	readObjectReq3 := gcs.ReadObjectRequest{Name: objectName, Range: &gcs.ByteRange{Start: 6 * util.MiB, Limit: 9 * util.MiB}, ReadHandle: nil}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, &readObjectReq3).Return(rd3, nil).Times(1)
	readObjectReq4 := gcs.ReadObjectRequest{Name: objectName, Range: &gcs.ByteRange{Start: 9 * util.MiB, Limit: 10 * util.MiB}, ReadHandle: handle}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, &readObjectReq4).Return(rd4, nil).Times(1)

	// Start download
	err = t.job.parallelDownloadObjectToFile(file)

	t.mockBucket.AssertExpectations(t.T())
	assert.Equal(t.T(), nil, err)
	jobStatus, ok := <-notificationC
	assert.Equal(t.T(), true, ok)
	// Check the notification is sent after subscribed offset
	assert.GreaterOrEqual(t.T(), jobStatus.Offset, subscribedOffset)
	t.job.mu.Lock()
	defer t.job.mu.Unlock()
	// Verify file is downloaded
	verifyCompleteFile(t.T(), t.fileSpec, objectContent)
	// Verify fileInfoCache update
	verifyFileInfoEntry(t.T(), t.mockBucket, t.object, t.cache, uint64(objectSize))
}
