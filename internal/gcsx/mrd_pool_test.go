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
	"fmt"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mrdPoolTest struct {
	suite.Suite
	object     *gcs.MinObject
	bucket     *storage.TestifyMockBucket
	poolConfig *MRDPoolConfig
}

func TestMRDPoolTestSuite(t *testing.T) {
	suite.Run(t, new(mrdPoolTest))
}

func (t *mrdPoolTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       1024 * MiB,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.poolConfig = &MRDPoolConfig{
		PoolSize: 4,
		object:   t.object,
		bucket:   t.bucket,
	}
}

func (t *mrdPoolTest) TestNewMRDPool_SmallFile() {
	t.object.Size = 100 * MiB
	t.poolConfig.PoolSize = 4

	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1, pool.poolConfig.PoolSize)
	assert.Len(t.T(), pool.entries, 1)
	assert.NotNil(t.T(), pool.entries[0].mrd)
	pool.Close()
}

func (t *mrdPoolTest) TestNewMRDPool_LargeFile() {
	t.object.Size = 1024 * MiB
	t.poolConfig.PoolSize = 2

	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// Expect calls for initial + async creation
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Times(2)

	pool, err := NewMRDPool(t.poolConfig, nil)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 2, pool.poolConfig.PoolSize)

	// Wait for async creation to finish
	pool.creationWg.Wait()

	assert.Equal(t.T(), uint64(2), pool.currentSize.Load())
	assert.NotNil(t.T(), pool.entries[0].mrd)
	assert.NotNil(t.T(), pool.entries[1].mrd)
	pool.Close()
}

func (t *mrdPoolTest) TestNewMRDPool_AsyncCreationFailure() {
	t.object.Size = 1024 * MiB
	t.poolConfig.PoolSize = 2

	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// First succeeds
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	// Second fails
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("async error")).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)

	assert.NoError(t.T(), err)
	pool.creationWg.Wait()

	assert.Equal(t.T(), uint64(2), pool.currentSize.Load())
	assert.NotNil(t.T(), pool.entries[0].mrd)
	assert.Nil(t.T(), pool.entries[1].mrd)
	pool.Close()
}

func (t *mrdPoolTest) TestNewMRDPool_Error() {
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error")).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)

	assert.Error(t.T(), err)
	assert.Nil(t.T(), pool)
}

func (t *mrdPoolTest) TestNext() {
	t.poolConfig.PoolSize = 3
	// Return a new downloader for each call to ensure we get different instances.
	// We set up three separate expectations to return distinct mock instances.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, nil), nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, nil), nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, nil), nil).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)
	pool.creationWg.Wait()

	// Verify round robin
	e1 := pool.Next()
	e2 := pool.Next()
	e3 := pool.Next()
	e4 := pool.Next()

	// The MRDEntry pointers should be different.
	assert.NotSame(t.T(), e1, e2)
	assert.NotSame(t.T(), e2, e3)
	assert.NotSame(t.T(), e3, e1)
	assert.Same(t.T(), e1, e4) // Wraps around

	// The underlying mrd instances should also be distinct.
	assert.NotNil(t.T(), e1.mrd)
	assert.NotNil(t.T(), e2.mrd)
	assert.NotNil(t.T(), e3.mrd)
	assert.NotSame(t.T(), e1.mrd, e2.mrd)
	assert.NotSame(t.T(), e2.mrd, e3.mrd)
	assert.NotSame(t.T(), e1.mrd, e3.mrd)

	pool.Close()
}

func (t *mrdPoolTest) TestRecreateMRD() {
	t.poolConfig.PoolSize = 1
	mrd1 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	mrd2 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd2, nil).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)

	entry := pool.Next()
	oldMRD := entry.mrd

	err = pool.RecreateMRD(entry, nil)
	assert.NoError(t.T(), err)
	assert.NotSame(t.T(), oldMRD, entry.mrd)

	pool.Close()
}

func (t *mrdPoolTest) TestRecreateMRD_Error() {
	t.poolConfig.PoolSize = 1
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)

	entry := pool.Next()
	oldMRD := entry.mrd

	// Fail the recreation
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("recreate error")).Once()

	err = pool.RecreateMRD(entry, nil)
	assert.Error(t.T(), err)
	assert.Equal(t.T(), oldMRD, entry.mrd) // Should remain unchanged on error

	pool.Close()
}

func (t *mrdPoolTest) TestClose() {
	t.poolConfig.PoolSize = 2
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Times(2)

	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)
	// pool.creationWg.Wait()

	pool.Close()

	// Verify entries are cleared
	for i := 0; i < len(pool.entries); i++ {
		assert.Nil(t.T(), pool.entries[i].mrd)
	}
}
