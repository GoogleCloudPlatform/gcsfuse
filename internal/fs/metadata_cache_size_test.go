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

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const metadataCacheSize = "metadata_cache/size"

// metadataCacheSizes returns the metadata_cache/size series of the positive
// and of the negative entries.
func metadataCacheSizes(ctx context.Context, t *testing.T, reader *metric.ManualReader) (positive, negative int64) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metadataCacheSize {
				continue
			}
			found = true
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is a %T, not a Sum[int64]", metadataCacheSize, m.Data)
			require.False(t, sum.IsMonotonic, "%s must be an up-down counter", metadataCacheSize)
			for _, dp := range sum.DataPoints {
				status, _ := dp.Attributes.Value("entry_status")
				switch status.AsString() {
				case "positive":
					positive = dp.Value
				case "negative":
					negative = dp.Value
				default:
					t.Errorf("unexpected entry_status %q", status.AsString())
				}
			}
		}
	}
	require.True(t, found, "metric %s not found", metadataCacheSize)
	return positive, negative
}

func TestMetadataCacheSize_EndToEndScenarios(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty cache", func(t *testing.T) {
		env := newMetadataCacheTestEnv(ctx, t)

		positive, negative := metadataCacheSizes(ctx, t, env.reader)

		assert.Zero(t, positive)
		assert.Zero(t, negative)
	})

	t.Run("Lookup of a nonexistent name caches only negative entries", func(t *testing.T) {
		env := newMetadataCacheTestEnv(ctx, t)

		_, err := env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")

		require.Equal(t, fuse.ENOENT, err)
		positive, negative := metadataCacheSizes(ctx, t, env.reader)
		assert.Zero(t, positive)
		assert.Positive(t, negative)
	})

	t.Run("Lookup of an existing file caches a positive entry", func(t *testing.T) {
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "file.txt")

		_, err := env.lookUp(ctx, fuseops.RootInodeID, "file.txt")

		require.NoError(t, err)
		positive, _ := metadataCacheSizes(ctx, t, env.reader)
		assert.Positive(t, positive)
	})

	t.Run("Unlink moves the file's bytes from positive to negative", func(t *testing.T) {
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "file.txt")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "file.txt")
		require.NoError(t, err)
		_, negativeBefore := metadataCacheSizes(ctx, t, env.reader)

		err = env.server.Unlink(ctx, &fuseops.UnlinkOp{Parent: fuseops.RootInodeID, Name: "file.txt"})

		require.NoError(t, err)
		positive, negative := metadataCacheSizes(ctx, t, env.reader)
		assert.Zero(t, positive)
		assert.Greater(t, negative, negativeBefore)
	})

	t.Run("Expired entries are replaced without drift", func(t *testing.T) {
		env := newMetadataCacheTestEnv(ctx, t)
		env.putObject(ctx, t, "file.txt")
		_, err := env.lookUp(ctx, fuseops.RootInodeID, "file.txt")
		require.NoError(t, err)
		_, err = env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")
		require.Equal(t, fuse.ENOENT, err)
		positiveBefore, negativeBefore := metadataCacheSizes(ctx, t, env.reader)

		for range 3 {
			env.clock.AdvanceTime(env.ttl + time.Second)
			_, err = env.lookUp(ctx, fuseops.RootInodeID, "file.txt")
			require.NoError(t, err)
			_, err = env.lookUp(ctx, fuseops.RootInodeID, "nonexistent.txt")
			require.Equal(t, fuse.ENOENT, err)
		}

		positive, negative := metadataCacheSizes(ctx, t, env.reader)
		assert.Equal(t, positiveBefore, positive)
		assert.Equal(t, negativeBefore, negative)
	})
}

func TestMetadataCacheSize_HNSBucketScenarios(t *testing.T) {
	ctx := context.Background()
	newHNSEnv := func(t *testing.T) *metadataCacheTestEnv {
		t.Helper()
		return newProductionLikeMetadataCacheFS(ctx, t, metadataCacheFSOptions{
			ttl: metadataCacheTTL, negativeTTL: metadataCacheTTL, hierarchical: true,
		})
	}

	t.Run("MkDir caches a positive folder entry", func(t *testing.T) {
		env := newHNSEnv(t)

		env.mkDir(ctx, t, fuseops.RootInodeID, "new_dir")

		positive, negative := metadataCacheSizes(ctx, t, env.reader)
		assert.Positive(t, positive)
		assert.Zero(t, negative)
	})

	t.Run("RmDir moves the folder's bytes from positive to negative", func(t *testing.T) {
		env := newHNSEnv(t)
		env.mkDir(ctx, t, fuseops.RootInodeID, "new_dir")

		err := env.server.RmDir(ctx, &fuseops.RmDirOp{Parent: fuseops.RootInodeID, Name: "new_dir"})

		require.NoError(t, err)
		positive, negative := metadataCacheSizes(ctx, t, env.reader)
		assert.Zero(t, positive)
		assert.Positive(t, negative)
	})
}
