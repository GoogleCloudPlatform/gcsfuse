// Copyright 2024 Google LLC
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

package inode

import (
	"context"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v2/internal/storage/mock"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

type FileBufferedWritesTest struct {
	mockBucket *storagemock.TestifyMockBucket
	clock      timeutil.SimulatedClock
	in         *FileInode
	suite.Suite
}

func TestFileBufferedWritesTestSuite(t *testing.T) {
	suite.Run(t, new(FileBufferedWritesTest))
}

func (t *FileBufferedWritesTest) SetupTest() {
	t.mockBucket = new(storagemock.TestifyMockBucket)
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	t.in = &FileInode{
		bucket:         t.mockBucket,
		mtimeClock:     &t.clock,
		id:             1,
		name:           Name{bucketName: "bucketName", objectName: "objectName"},
		attrs:          fuseops.InodeAttributes{},
		contentCache:   contentcache.New("", &t.clock),
		localFileCache: false,
		mu:             syncutil.InvariantMutex{},
		lc:             lookupCount{},
		src:            gcs.MinObject{},
		content:        nil,
		destroyed:      false,
		local:          true,
		unlinked:       false,
		bwh:            nil,
		writeConfig: &cfg.WriteConfig{
			MaxBlocksPerFile:                  10,
			BlockSizeMb:                       10,
			ExperimentalEnableStreamingWrites: true,
		},
		globalMaxBlocksSem: semaphore.NewWeighted(math.MaxInt64),
	}
}

func (t *FileBufferedWritesTest) TestWriteToLocalFile_StreamingWritesEnabled_SignalNonRecoverableFailure() {
	assert.Nil(t.T(), t.in.bwh)
	err := t.in.Write(context.Background(), []byte("hello"), 0)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.in.bwh)

	// Close the channel to simulate a signal
	close(t.in.bwh.SignalNonRecoverableFailure())

	err = t.in.Write(context.Background(), []byte("hello"), 5)
	assert.NotNil(t.T(), err)
	assert.ErrorContains(t.T(), err, "buffered writes: non-recoverable failure while writing")
}

func (t *FileBufferedWritesTest) TestWriteToLocalFile_StreamingWritesEnabled_SignalUploadFailure() {
	assert.Nil(t.T(), t.in.bwh)
	err := t.in.Write(context.Background(), []byte("hello"), 0)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.in.bwh)
	t.mockBucket.On("NewReader", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stringToReader(""), nil)

	// Close the channel to simulate a signal
	close(t.in.bwh.SignalUploadFailure())

	// Write to file inode again. This time write will go to temp file.
	err = t.in.Write(context.Background(), []byte("hello"), 5)
	assert.Nil(t.T(), err)
	// Validate that temp file is passed to temp file channel.
	select {
	case tempFile := <-t.in.bwh.TempFileChannel():
		assert.Equal(t.T(), t.in.content, tempFile)
	case <-time.After(200 * time.Millisecond):
		t.T().Error("Timeout waiting for TempFile")
	}
	// Write again and validate that temp file is not passed again to temp file channel.
	err = t.in.Write(context.Background(), []byte("hello"), 10)
	assert.Nil(t.T(), err)
	select {
	case <-t.in.bwh.TempFileChannel():
		t.T().Error("Temp file received on channel again")
	case <-time.After(200 * time.Millisecond):
		break
	}
	data, err := io.ReadAll(t.in.content)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "hellohello", string(data[5:]))
}

func stringToReader(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}
