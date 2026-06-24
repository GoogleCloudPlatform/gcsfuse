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

package inode

import (
	"context"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"golang.org/x/sync/semaphore"
)

func newBenchmarkDirInode(b *testing.B, implicitDirs bool) *dirInode {
	b.Helper()
	ctx := context.Background()
	var clock timeutil.SimulatedClock
	clock.SetTime(time.Date(2026, 6, 18, 12, 0, 0, 0, time.Local))
	bucket := fake.NewFakeBucket(&clock, "some_bucket", gcs.BucketType{Hierarchical: true})
	syncerBucket := gcsx.NewSyncerBucket(
		/*appendThreshold=*/ 1,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		bucket)

	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: true},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: 4},
		EnableHns:                    true,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   true,
	}

	in := NewDirInode(
		dirInodeID,
		NewDirName(NewRootName(""), dirInodeName),
		ctx,
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		implicitDirs,
		true, // enableNonexistentTypeCache
		typeCacheTTL,
		&syncerBucket,
		&clock,
		&clock,
		semaphore.NewWeighted(10),
		config,
	)

	return in.(*dirInode)
}

func runBenchmark(b *testing.B, cachedType metadata.Type, name string, setupBucket func(d *dirInode)) {
	b.Helper()
	d := newBenchmarkDirInode(b, true)
	if setupBucket != nil {
		setupBucket(d)
	}
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = d.fetchCoreEntity(ctx, name, cachedType)
	}
}

func Benchmark_ImplicitDirType(b *testing.B) {
	runBenchmark(b, metadata.ImplicitDirType, "subdir", nil)
}

func Benchmark_NonexistentType(b *testing.B) {
	runBenchmark(b, metadata.NonexistentType, "absent", nil)
}

func Benchmark_ExplicitDirType(b *testing.B) {
	runBenchmark(b, metadata.ExplicitDirType, "subdir", func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewDirName(d.Name(), "subdir").GcsObjectName(), []byte(""))
	})
}

func Benchmark_RegularFileType(b *testing.B) {
	runBenchmark(b, metadata.RegularFileType, "file", func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewFileName(d.Name(), "file").GcsObjectName(), []byte("taco"))
	})
}

func Benchmark_UnknownType_Nonexistent(b *testing.B) {
	runBenchmark(b, metadata.UnknownType, "absent", nil)
}

func Benchmark_UnknownType_ExistingFile(b *testing.B) {
	runBenchmark(b, metadata.UnknownType, "file", func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewFileName(d.Name(), "file").GcsObjectName(), []byte("taco"))
	})
}
