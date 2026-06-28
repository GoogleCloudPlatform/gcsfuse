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

package fs

import (
	"sync"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quotaTestName(objectName string) inode.Name {
	return inode.NewDescendantName(inode.NewRootName(""), objectName)
}

func TestLogicalQuotaDisabledNeverRejects(t *testing.T) {
	var q *logicalQuota
	name := quotaTestName("file")

	commit, rollback, err := q.tryReserveGrowth(name, 0, 1<<30)

	require.NoError(t, err)
	rollback()
	commit()
}

func TestLogicalQuotaFileCountRejectsNextFile(t *testing.T) {
	q, err := newLogicalQuota(1, 0)
	require.NoError(t, err)

	rollback, err := q.tryReserveFile(quotaTestName("a"))
	require.NoError(t, err)
	defer rollback()

	_, err = q.tryReserveFile(quotaTestName("b"))

	assert.ErrorIs(t, err, syscall.ENOSPC)
}

func TestLogicalQuotaSizeRejectsGrowthBeyondLimit(t *testing.T) {
	q, err := newLogicalQuota(0, 1)
	require.NoError(t, err)

	_, _, err = q.tryReserveGrowth(quotaTestName("file"), 0, 1<<20+1)

	assert.ErrorIs(t, err, syscall.ENOSPC)
}

func TestLogicalQuotaOverwriteWithinExistingSize(t *testing.T) {
	q, err := newLogicalQuota(0, 1)
	require.NoError(t, err)
	name := quotaTestName("file")

	commit, _, err := q.tryReserveGrowth(name, 0, 512)
	require.NoError(t, err)
	commit()

	commit, rollback, err := q.tryReserveGrowth(name, 512, 512)

	require.NoError(t, err)
	rollback()
	commit()
	assert.Equal(t, uint64(512), q.usedBytes)
}

func TestLogicalQuotaShrinkReleasesBytes(t *testing.T) {
	q, err := newLogicalQuota(0, 1)
	require.NoError(t, err)
	name := quotaTestName("file")
	commit, _, err := q.tryReserveGrowth(name, 0, 1<<20)
	require.NoError(t, err)
	commit()

	q.applyShrink(name, 128)

	assert.Equal(t, uint64(128), q.usedBytes)
}

func TestLogicalQuotaRollbackReleasesReservedGrowth(t *testing.T) {
	q, err := newLogicalQuota(0, 1)
	require.NoError(t, err)
	name := quotaTestName("file")

	_, rollback, err := q.tryReserveGrowth(name, 0, 512)
	require.NoError(t, err)
	rollback()

	assert.Equal(t, uint64(0), q.usedBytes)
	_, ok := q.byName[name]
	assert.False(t, ok)
}

func TestLogicalQuotaFileOnlyDoesNotTrackBytes(t *testing.T) {
	q, err := newLogicalQuota(2, 0)
	require.NoError(t, err)
	name := quotaTestName("file")

	commit, _, err := q.tryReserveGrowth(name, 0, 1<<20)
	require.NoError(t, err)
	commit()

	assert.Equal(t, uint64(1), q.usedFiles)
	assert.Equal(t, uint64(0), q.usedBytes)
	assert.Equal(t, uint64(0), q.byName[name].size)
}

func TestLogicalQuotaSizeOnlyDoesNotTrackFiles(t *testing.T) {
	q, err := newLogicalQuota(0, 1)
	require.NoError(t, err)

	commit, _, err := q.tryReserveGrowth(quotaTestName("file"), 0, 512)
	require.NoError(t, err)
	commit()

	assert.Equal(t, uint64(0), q.usedFiles)
	assert.Equal(t, uint64(512), q.usedBytes)
}

func TestLogicalQuotaUntrackedExistingFileChargesFullSize(t *testing.T) {
	q, err := newLogicalQuota(0, 1)
	require.NoError(t, err)
	name := quotaTestName("file")

	commit, rollback, err := q.tryReserveGrowth(name, 512, 1024)
	require.NoError(t, err)
	commit()

	assert.Equal(t, uint64(1024), q.usedBytes)
	assert.Equal(t, uint64(1024), q.byName[name].size)

	rollback()
	assert.Equal(t, uint64(1024), q.usedBytes)
}

func TestLogicalQuotaStatFSReportsBoundedCapacity(t *testing.T) {
	q, err := newLogicalQuota(10, 1)
	require.NoError(t, err)
	commit, _, err := q.tryReserveGrowth(quotaTestName("file"), 0, 512*1024)
	require.NoError(t, err)
	commit()

	blocks, free, available, inodes, inodesFree := q.statFS(128 * 1024)

	assert.Equal(t, uint64(8), blocks)
	assert.Equal(t, uint64(4), free)
	assert.Equal(t, free, available)
	assert.Equal(t, uint64(10), inodes)
	assert.Equal(t, uint64(9), inodesFree)
}

func TestLogicalQuotaApplyRenameOverExistingReleasesDestination(t *testing.T) {
	q, err := newLogicalQuota(10, 1)
	require.NoError(t, err)
	src := quotaTestName("src")
	dst := quotaTestName("dst")
	q.addInitialEntryLocked(src, 128)
	q.addInitialEntryLocked(dst, 512)

	q.applyRename(src, dst, 128)

	assert.Equal(t, uint64(1), q.usedFiles)
	assert.Equal(t, uint64(128), q.usedBytes)
	_, ok := q.byName[src]
	assert.False(t, ok)
	assert.Equal(t, uint64(128), q.byName[dst].size)
}

func TestLogicalQuotaConcurrentReservationsCannotOversubscribe(t *testing.T) {
	q, err := newLogicalQuota(0, 1)
	require.NoError(t, err)

	const goroutines = 16
	const chunkSize = 256 * 1024

	var wg sync.WaitGroup
	successes := make(chan struct{}, goroutines)
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := quotaTestName(string(rune('a' + i)))
			commit, _, err := q.tryReserveGrowth(name, 0, chunkSize)
			if err == nil {
				commit()
				successes <- struct{}{}
			}
		}(i)
	}
	wg.Wait()
	close(successes)

	assert.Len(t, successes, 4)
	assert.Equal(t, uint64(1<<20), q.usedBytes)
}

func TestLogicalQuotaRejectsAllBucketsMount(t *testing.T) {
	_, err := newLogicalQuotaForServerConfig(&ServerConfig{
		BucketName: "",
		NewConfig: &cfg.Config{
			FileSystem: cfg.FileSystemConfig{
				ExperimentalMaxSizeMb: 1,
			},
		},
	})

	assert.ErrorContains(t, err, "single mounted bucket or --only-dir prefix")
	assert.ErrorContains(t, err, "all-buckets mounts are not supported")
}

func TestLogicalQuotaAllowsSingleBucketMount(t *testing.T) {
	q, err := newLogicalQuotaForServerConfig(&ServerConfig{
		BucketName: "bucket",
		NewConfig: &cfg.Config{
			FileSystem: cfg.FileSystemConfig{
				ExperimentalMaxSizeMb: 1,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, q)
	assert.True(t, q.hasSizeLimit())
}
