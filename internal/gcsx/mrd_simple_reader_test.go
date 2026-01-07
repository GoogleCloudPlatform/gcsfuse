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

package gcsx

import (
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MrdSimpleReaderTest struct {
	suite.Suite
	object      *gcs.MinObject
	bucket      *storage.TestifyMockBucket
	cache       *lru.Cache
	inodeID     fuseops.InodeID
	mrdConfig   cfg.MrdConfig
	mrdInstance *MrdInstance
	reader      *MrdSimpleReader
}

func TestMrdSimpleReaderTestSuite(t *testing.T) {
	suite.Run(t, new(MrdSimpleReaderTest))
}

func (t *MrdSimpleReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       1024 * MiB,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.cache = lru.NewCache(2)
	t.inodeID = 100
	t.mrdConfig = cfg.MrdConfig{PoolSize: 1}

	t.mrdInstance = NewMrdInstance(t.object, t.bucket, t.cache, t.inodeID, t.mrdConfig)
	t.reader = NewMrdSimpleReader(t.mrdInstance)
}

func (t *MrdSimpleReaderTest) TestNewMrdSimpleReader() {
	assert.NotNil(t.T(), t.reader)
	assert.Equal(t.T(), t.mrdInstance, t.reader.mrdInstance)
}

func (t *MrdSimpleReaderTest) TestReadAt_EmptyBuffer() {
	req := &ReadRequest{
		Buffer: []byte{},
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_Success() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}
	// Verify initial refCount
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "hello", string(req.Buffer))
	// Verify refCount incremented
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
}

func (t *MrdSimpleReaderTest) TestReadAt_MultipleCalls() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	// Expect NewMultiRangeDownloader only once because subsequent reads reuse the instance/pool
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	// First call
	_, err := t.reader.ReadAt(context.Background(), req)
	assert.NoError(t.T(), err)

	// Second call
	_, err = t.reader.ReadAt(context.Background(), req)
	assert.NoError(t.T(), err)

	// Verify refCount is still 1
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
}

func (t *MrdSimpleReaderTest) TestReadAt_NilMrdInstance() {
	t.reader.mrdInstance = nil
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "mrdInstance is nil")
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestDestroy() {
	// Setup state where refCount is incremented
	t.reader.mrdInstanceInUse.Store(true)
	t.mrdInstance.refCount = 1

	t.reader.Destroy()

	assert.Nil(t.T(), t.reader.mrdInstance)
	assert.False(t.T(), t.reader.mrdInstanceInUse.Load())
	// Verify refCount decremented
	t.mrdInstance.refCountMu.Lock()
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	t.mrdInstance.refCountMu.Unlock()
	// Verify that calling Destroy again doesn't panic
	t.reader.Destroy()
}
