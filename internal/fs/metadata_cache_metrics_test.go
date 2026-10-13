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

package fs_test

import (
	"context"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Shorthands for the metadata_cache/read_count attribute values.
const (
	statusNone     = metrics.EntryStatusAttr
	statusPositive = metrics.EntryStatusPositiveAttr
	statusNegative = metrics.EntryStatusNegativeAttr

	detailFound      = metrics.LookupDetailFoundAttr
	detailNotFound   = metrics.LookupDetailNotFoundAttr
	detailTTLExpired = metrics.LookupDetailTtlExpiredAttr
)

const metadataCacheReadCount = "metadata_cache/read_count"

func verifyMetadataCacheMetric(
	t *testing.T,
	ctx context.Context,
	reader *metric.ManualReader,
	cacheHit bool,
	entryStatus metrics.EntryStatus,
	lookupDetail metrics.LookupDetail,
	expected int64,
) {
	t.Helper()
	var attrs attribute.Set
	if entryStatus == metrics.EntryStatusAttr {
		attrs = attribute.NewSet(
			attribute.Bool("cache_hit", cacheHit),
			attribute.String("lookup_detail", string(lookupDetail)),
		)
	} else {
		attrs = attribute.NewSet(
			attribute.Bool("cache_hit", cacheHit),
			attribute.String("entry_status", string(entryStatus)),
			attribute.String("lookup_detail", string(lookupDetail)),
		)
	}
	metrics.VerifyCounterMetric(t, ctx, reader, metadataCacheReadCount, attrs, expected)
}

// totalMetadataCacheReads sums every metadata_cache/read_count series, returning 0
// when the instrument was never recorded.
func totalMetadataCacheReads(t *testing.T, ctx context.Context, reader *metric.ManualReader) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metadataCacheReadCount {
				continue
			}
			if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
				for _, dp := range sum.DataPoints {
					total += dp.Value
				}
			}
		}
	}
	return total
}

// wantRead is one expected metadata_cache/read_count series.
type wantRead struct {
	hit    bool
	status metrics.EntryStatus
	detail metrics.LookupDetail
	count  int64
}

// metadataCacheTestEnv is a self-contained fixture for a single scenario: its own
// bucket, file system, simulated clock and metric reader. Nothing leaks between
// subtests, so each is independently runnable and every assertion is an absolute
// count rather than a running total.
type metadataCacheTestEnv struct {
	bucket gcs.Bucket
	server fuseutil.FileSystem
	reader *metric.ManualReader
	clock  *timeutil.SimulatedClock
	ttl    time.Duration
}

func newMetadataCacheTestEnv(ctx context.Context, t *testing.T) *metadataCacheTestEnv {
	t.Helper()
	return newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{ttl: metadataCacheTTL, negativeTTL: metadataCacheTTL})
}

// putObject writes an object straight to the backing bucket, bypassing gcsfuse.
func (e *metadataCacheTestEnv) putObject(ctx context.Context, t *testing.T, name string) {
	t.Helper()
	require.NoError(t, storageutil.CreateObjects(ctx, e.bucket, map[string][]byte{name: []byte("contents")}))
}

func (e *metadataCacheTestEnv) lookUp(ctx context.Context, parent fuseops.InodeID, name string) (*fuseops.LookUpInodeOp, error) {
	op := &fuseops.LookUpInodeOp{Parent: parent, Name: name}
	return op, e.server.LookUpInode(ctx, op)
}

func (e *metadataCacheTestEnv) mkDir(ctx context.Context, t *testing.T, parent fuseops.InodeID, name string) fuseops.InodeID {
	t.Helper()
	op := &fuseops.MkDirOp{Parent: parent, Name: name, Mode: 0755}
	require.NoError(t, e.server.MkDir(ctx, op))
	return op.Entry.Child
}

// assertReads checks each expected series exactly and fails if any other
// metadata_cache/read_count event was emitted.
func (e *metadataCacheTestEnv) assertReads(ctx context.Context, t *testing.T, want ...wantRead) {
	t.Helper()
	waitForMetricsProcessing()

	var total int64
	for _, w := range want {
		verifyMetadataCacheMetric(t, ctx, e.reader, w.hit, w.status, w.detail, w.count)
		total += w.count
	}
	assert.Equal(t, total, totalMetadataCacheReads(t, ctx, e.reader),
		"unexpected extra metadata_cache/read_count events")
}

func TestMetadataCache_EndToEndScenarios(t *testing.T) {
	ctx := context.Background()

	// Cold lookup of a name that does not exist: both candidate keys miss.
	t.Run("Scenario 1: Cold Nonexistent Stat", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 1})
	})

	// The cold lookup negatively caches both candidate keys, so the second lookup is
	// answered without contacting GCS.
	t.Run("Scenario 2: Warm Nonexistent Stat", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")
		require.Equal(t, fuse.ENOENT, err)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusNegative, detailFound, 1},
		)
	})

	// Writing straight to the bucket never touches the metadata cache.
	t.Run("Scenario 3: External GCS Create", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)

		// Act
		env.putObject(ctx, t, "existing_file.txt")

		// Assert
		env.assertReads(ctx, t)
	})

	t.Run("Scenario 4: Cold Stat on Existing File", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_file.txt")

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 1})
	})

	t.Run("Scenario 5: Warm Stat on Existing File", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_file.txt")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")
		require.NoError(t, err)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	// GetInodeAttributes' clobbered() check is a positive stat-cache hit.
	t.Run("Scenario 6: GetInodeAttributes records a cache hit", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_file.txt")
		op, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")
		require.NoError(t, err)

		// Act
		err = env.server.GetInodeAttributes(ctx, &fuseops.GetInodeAttributesOp{Inode: op.Entry.Child})

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	t.Run("Scenario 7: Stat Cache TTL Expiration", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_file.txt")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")
		require.NoError(t, err)
		env.clock.AdvanceTime(env.ttl + 10*time.Second)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{false, statusPositive, detailTTLExpired, 1},
		)
	})

	// MkDir caches the folder, so the follow-up lookup is a positive hit.
	t.Run("Scenario 8: Directory Creation & Lookup", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.mkDir(ctx, t, fuseops.RootInodeID, "new_dir")

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "new_dir")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t, wantRead{true, statusPositive, detailFound, 1})
	})

	// RmDir negatively caches only "new_dir/", never the sibling file key "new_dir",
	// so the follow-up lookup cannot be resolved from cache alone.
	t.Run("Scenario 9: Directory Deletion & Negative Lookup", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.mkDir(ctx, t, fuseops.RootInodeID, "new_dir")
		require.NoError(t, env.server.RmDir(ctx, &fuseops.RmDirOp{Parent: fuseops.RootInodeID, Name: "new_dir"}))

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "new_dir")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{true, statusPositive, detailFound, 1},
			wantRead{false, statusNone, detailNotFound, 1},
		)
	})

	// CreateFile's clobbered() check goes to GCS (miss); the later lookup is a hit.
	t.Run("Scenario 10: File Creation & Warm Lookup", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		createOp := &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: "new_file.txt", Mode: 0644}
		require.NoError(t, env.server.CreateFile(ctx, createOp))
		require.NoError(t, env.server.WriteFile(ctx, &fuseops.WriteFileOp{
			Inode: createOp.Entry.Child, Handle: createOp.Handle, Data: []byte("test content"),
		}))
		require.NoError(t, env.server.SyncFile(ctx, &fuseops.SyncFileOp{
			Inode: createOp.Entry.Child, Handle: createOp.Handle,
		}))

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "new_file.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	// The cold lookup negatively caches the directory key and Unlink negatively
	// caches the file key, so both candidates are negative afterwards.
	t.Run("Scenario 11: File Deletion & Negative Lookup", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "new_file.txt")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "new_file.txt")
		require.NoError(t, err)
		require.NoError(t, env.server.Unlink(ctx, &fuseops.UnlinkOp{Parent: fuseops.RootInodeID, Name: "new_file.txt"}))

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "new_file.txt")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusNegative, detailFound, 1},
		)
	})

	// Same as Scenario 6, via the SetInodeAttributes route into clobbered().
	t.Run("Scenario 12: SetInodeAttributes records a cache hit", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_file.txt")
		op, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")
		require.NoError(t, err)
		newSize := uint64(5)

		// Act
		err = env.server.SetInodeAttributes(ctx, &fuseops.SetInodeAttributesOp{Inode: op.Entry.Child, Size: &newSize})

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	t.Run("Scenario 13: Symlink Creation & Lookup", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		require.NoError(t, env.server.CreateSymlink(ctx, &fuseops.CreateSymlinkOp{
			Parent: fuseops.RootInodeID, Name: "symlink_target", Target: "existing_file.txt",
		}))

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "symlink_target")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t, wantRead{true, statusPositive, detailFound, 1})
	})

	t.Run("Scenario 14: Negative TTL Expiration", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")
		require.Equal(t, fuse.ENOENT, err)
		env.clock.AdvanceTime(env.ttl + 10*time.Second)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{false, statusNegative, detailTTLExpired, 1},
		)
	})

	// OpenFile forces a GCS stat, so it must not record a cache read.
	t.Run("Scenario 15: Open Existing File & Warm Lookup", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_file.txt")
		op, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")
		require.NoError(t, err)
		require.NoError(t, env.server.OpenFile(ctx, &fuseops.OpenFileOp{Inode: op.Entry.Child}))

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	// Listing populates the cache but performs no cache reads.
	t.Run("Scenario 16: ReadDir does not inflate counts", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_file.txt")
		openDirOp := &fuseops.OpenDirOp{Inode: fuseops.RootInodeID}
		require.NoError(t, env.server.OpenDir(ctx, openDirOp))

		// Act
		err := env.server.ReadDir(ctx, &fuseops.ReadDirOp{
			Inode: fuseops.RootInodeID, Handle: openDirOp.Handle, Dst: make([]byte, 4096),
		})

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t)
	})

	t.Run("Scenario 17: Nested File Cold Stat", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		subDir := env.mkDir(ctx, t, fuseops.RootInodeID, "nested_dir")
		env.putObject(ctx, t, "nested_dir/child.txt")

		// Act
		_, err := env.lookUp(ctx, subDir, "child.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 1})
	})

	t.Run("Scenario 18: Nested File Warm Stat", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		subDir := env.mkDir(ctx, t, fuseops.RootInodeID, "nested_dir")
		env.putObject(ctx, t, "nested_dir/child.txt")
		_, err := env.lookUp(ctx, subDir, "child.txt")
		require.NoError(t, err)

		// Act
		_, err = env.lookUp(ctx, subDir, "child.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	t.Run("Scenario 19: Nested Nonexistent Stat Cold", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		subDir := env.mkDir(ctx, t, fuseops.RootInodeID, "nested_dir")

		// Act
		_, err := env.lookUp(ctx, subDir, "nonexistent_child.txt")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 1})
	})

	t.Run("Scenario 20: Nested Nonexistent Stat Warm", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		subDir := env.mkDir(ctx, t, fuseops.RootInodeID, "nested_dir")
		_, err := env.lookUp(ctx, subDir, "nonexistent_child.txt")
		require.Equal(t, fuse.ENOENT, err)

		// Act
		_, err = env.lookUp(ctx, subDir, "nonexistent_child.txt")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusNegative, detailFound, 1},
		)
	})

	// The conflict-marker lookup checks "conflict/", a miss that goes to GCS.
	t.Run("Scenario 21: Conflicting Filename Lookup", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "conflict\n")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 1})
	})

	// CreateFile misses as in Scenario 10; GetInodeAttributes hits the negative entry.
	t.Run("Scenario 22: Unlinked local file Attributes", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		createOp := &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: "local_unlink.txt", Mode: 0644}
		require.NoError(t, env.server.CreateFile(ctx, createOp))
		require.NoError(t, env.server.Unlink(ctx, &fuseops.UnlinkOp{Parent: fuseops.RootInodeID, Name: "local_unlink.txt"}))
		getOp := &fuseops.GetInodeAttributesOp{Inode: createOp.Entry.Child}

		// Act
		err := env.server.GetInodeAttributes(ctx, getOp)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint32(0), getOp.Attributes.Nlink)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusNegative, detailFound, 1},
		)
	})

	t.Run("Scenario 23: Directory Stat Cache TTL Expiration", func(t *testing.T) {
		// Arrange
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "existing_dir/")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_dir")
		require.NoError(t, err)
		env.clock.AdvanceTime(env.ttl + 10*time.Second)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_dir")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{false, statusPositive, detailTTLExpired, 1},
		)
	})
}

func TestMetadataCache_HNSBucketScenarios(t *testing.T) {
	ctx := context.Background()
	newHNSEnv := func(t *testing.T) *metadataCacheTestEnv {
		t.Helper()
		return newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{
			ttl: metadataCacheTTL, negativeTTL: metadataCacheTTL, hierarchical: true,
		})
	}

	t.Run("Cold Folder Lookup", func(t *testing.T) {
		// Arrange
		env := newHNSEnv(t)
		_, err := env.bucket.CreateFolder(ctx, "existing_dir/")
		require.NoError(t, err)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_dir")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 1})
	})

	t.Run("Warm Folder Lookup", func(t *testing.T) {
		// Arrange
		env := newHNSEnv(t)
		_, err := env.bucket.CreateFolder(ctx, "existing_dir/")
		require.NoError(t, err)
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_dir")
		require.NoError(t, err)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_dir")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})

	t.Run("Folder TTL Expiration", func(t *testing.T) {
		// Arrange
		env := newHNSEnv(t)
		_, err := env.bucket.CreateFolder(ctx, "existing_dir/")
		require.NoError(t, err)
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_dir")
		require.NoError(t, err)
		env.clock.AdvanceTime(env.ttl + 10*time.Second)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_dir")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{false, statusPositive, detailTTLExpired, 1},
		)
	})

	// MkDir caches the folder, so the follow-up lookup is a positive hit.
	t.Run("Folder Creation & Lookup", func(t *testing.T) {
		// Arrange
		env := newHNSEnv(t)
		env.mkDir(ctx, t, fuseops.RootInodeID, "new_dir")

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "new_dir")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t, wantRead{true, statusPositive, detailFound, 1})
	})

	// RmDir negatively caches only the folder, never the sibling file key, so
	// the follow-up lookup cannot be resolved from cache alone.
	t.Run("Folder Deletion & Negative Lookup", func(t *testing.T) {
		// Arrange
		env := newHNSEnv(t)
		env.mkDir(ctx, t, fuseops.RootInodeID, "new_dir")
		require.NoError(t, env.server.RmDir(ctx, &fuseops.RmDirOp{Parent: fuseops.RootInodeID, Name: "new_dir"}))

		// Act
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "new_dir")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{true, statusPositive, detailFound, 1},
			wantRead{false, statusNone, detailNotFound, 1},
		)
	})

	// The cold lookup negatively caches the folder and the file, so the second
	// lookup is answered without contacting GCS.
	t.Run("Warm Nonexistent Stat", func(t *testing.T) {
		// Arrange
		env := newHNSEnv(t)
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "nonexistent")
		require.Equal(t, fuse.ENOENT, err)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "nonexistent")

		// Assert
		assert.Equal(t, fuse.ENOENT, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusNegative, detailFound, 1},
		)
	})

	t.Run("Warm Stat on Existing File", func(t *testing.T) {
		// Arrange
		env := newHNSEnv(t)
		env.putObject(ctx, t, "existing_file.txt")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")
		require.NoError(t, err)

		// Act
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "existing_file.txt")

		// Assert
		require.NoError(t, err)
		env.assertReads(ctx, t,
			wantRead{false, statusNone, detailNotFound, 1},
			wantRead{true, statusPositive, detailFound, 1},
		)
	})
}

func TestMetadataCache_NegativeTTLScenarios(t *testing.T) {
	ctx := context.Background()
	const ttl = 60 * time.Second
	testCases := []struct {
		name        string
		negativeTTL time.Duration
		fileExists  bool
		// advance is the time between the cold lookup and the measured one.
		advance time.Duration
		// want is what the measured lookup records.
		want wantRead
	}{
		{
			name:        "Nonexistent file within the negative TTL",
			negativeTTL: 5 * time.Second,
			advance:     3 * time.Second,
			want:        wantRead{true, statusNegative, detailFound, 1},
		},
		{
			name:        "Nonexistent file after the negative TTL",
			negativeTTL: 5 * time.Second,
			advance:     6 * time.Second,
			want:        wantRead{false, statusNegative, detailTTLExpired, 1},
		},
		{
			// The negative entry for "file.txt/" expired, but the positive
			// entry for "file.txt" answers the lookup without GCS.
			name:        "Existing file after the negative TTL",
			negativeTTL: 5 * time.Second,
			fileExists:  true,
			advance:     6 * time.Second,
			want:        wantRead{true, statusPositive, detailFound, 1},
		},
		{
			name:        "Existing file after the TTL",
			negativeTTL: 5 * time.Second,
			fileExists:  true,
			advance:     ttl + time.Second,
			want:        wantRead{false, statusPositive, detailTTLExpired, 1},
		},
		{
			// Negative entries are still cached, with a zero TTL, so the next
			// lookup finds them expired.
			name:    "Nonexistent file without negative caching",
			advance: time.Second,
			want:    wantRead{false, statusNegative, detailTTLExpired, 1},
		},
		{
			name:       "Existing file without negative caching",
			fileExists: true,
			advance:    time.Second,
			want:       wantRead{true, statusPositive, detailFound, 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			env := newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{ttl: ttl, negativeTTL: tc.negativeTTL})
			var wantErr error = fuse.ENOENT
			if tc.fileExists {
				env.putObject(ctx, t, "file.txt")
				wantErr = nil
			}
			_, err := env.lookUp(ctx, fuseops.RootInodeID, "file.txt")
			require.Equal(t, wantErr, err)
			env.clock.AdvanceTime(tc.advance)

			// Act
			_, err = env.lookUp(ctx, fuseops.RootInodeID, "file.txt")

			// Assert
			assert.Equal(t, wantErr, err)
			env.assertReads(ctx, t, wantRead{false, statusNone, detailNotFound, 1}, tc.want)
		})
	}
}
