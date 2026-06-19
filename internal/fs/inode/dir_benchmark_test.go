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
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// fetchCoreEntityOld contains the old, unoptimized implementation of fetchCoreEntity.
func (d *dirInode) fetchCoreEntityOld(ctx context.Context, name string, cachedType metadata.Type) (*Core, error) {
	group, ctx := errgroup.WithContext(ctx)

	var fileResult *Core
	var dirResult *Core
	var err error

	lookUpFile := func() (err error) {
		fileResult, err = findExplicitInode(ctx, d.Bucket(), NewFileName(d.Name(), name), false)
		return
	}
	lookUpExplicitDir := func() (err error) {
		dirResult, err = findExplicitInode(ctx, d.Bucket(), NewDirName(d.Name(), name), false)
		return
	}
	lookUpImplicitOrExplicitDir := func() (err error) {
		dirResult, err = findDirInode(ctx, d.Bucket(), NewDirName(d.Name(), name))
		return
	}
	lookUpHNSDir := func() (err error) {
		dirResult, err = findExplicitFolder(ctx, d.Bucket(), NewDirName(d.Name(), name), false)
		return
	}

	switch cachedType {
	case metadata.ImplicitDirType:
		return &Core{
			Bucket:    d.Bucket(),
			FullName:  NewDirName(d.Name(), name),
			MinObject: nil,
		}, nil
	case metadata.ExplicitDirType:
		if d.isBucketHierarchical() {
			group.Go(lookUpHNSDir)
		} else {
			group.Go(lookUpExplicitDir)
		}
	case metadata.RegularFileType, metadata.SymlinkType:
		group.Go(lookUpFile)

	case metadata.NonexistentType:
		return nil, nil
	case metadata.UnknownType:
		// Entry not present in cache.
		// Trigger prefetcher
		if d.prefetcher != nil {
			d.prefetcher.Run(NewFileName(d.Name(), name).GcsObjectName())
		}

		if d.isBucketHierarchical() {
			return d.lookUpHNSRace(ctx, name)
		}

		group.Go(lookUpFile)
		if d.implicitDirs {
			group.Go(lookUpImplicitOrExplicitDir)
		} else {
			group.Go(lookUpExplicitDir)
		}
	}

	if err = group.Wait(); err != nil {
		return nil, err
	}

	if dirResult != nil {
		return dirResult, nil
	}
	return fileResult, nil
}

func newBenchmarkDirInode(tb testing.TB, implicitDirs bool) *dirInode {
	ctx := context.Background()
	var clock timeutil.SimulatedClock
	clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := fake.NewFakeBucket(&clock, "some_bucket", gcs.BucketType{})
	syncerBucket := gcsx.NewSyncerBucket(
		/*appendThreshold=*/ 1,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		bucket)

	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: true},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: 4},
		EnableHns:                    false,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   false,
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

func runBenchmark(b *testing.B, cachedType metadata.Type, name string, useOld bool, setupBucket func(d *dirInode)) {
	d := newBenchmarkDirInode(b, true)
	if setupBucket != nil {
		setupBucket(d)
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if useOld {
			_, _ = d.fetchCoreEntityOld(ctx, name, cachedType)
		} else {
			_, _ = d.fetchCoreEntity(ctx, name, cachedType)
		}
	}
}

func Benchmark_ImplicitDirType_Old(b *testing.B) {
	runBenchmark(b, metadata.ImplicitDirType, "subdir", true, nil)
}

func Benchmark_ImplicitDirType_New(b *testing.B) {
	runBenchmark(b, metadata.ImplicitDirType, "subdir", false, nil)
}

func Benchmark_NonexistentType_Old(b *testing.B) {
	runBenchmark(b, metadata.NonexistentType, "absent", true, nil)
}

func Benchmark_NonexistentType_New(b *testing.B) {
	runBenchmark(b, metadata.NonexistentType, "absent", false, nil)
}

func Benchmark_ExplicitDirType_Old(b *testing.B) {
	runBenchmark(b, metadata.ExplicitDirType, "subdir", true, func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewDirName(d.Name(), "subdir").GcsObjectName(), []byte(""))
	})
}

func Benchmark_ExplicitDirType_New(b *testing.B) {
	runBenchmark(b, metadata.ExplicitDirType, "subdir", false, func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewDirName(d.Name(), "subdir").GcsObjectName(), []byte(""))
	})
}

func Benchmark_RegularFileType_Old(b *testing.B) {
	runBenchmark(b, metadata.RegularFileType, "file", true, func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewFileName(d.Name(), "file").GcsObjectName(), []byte("taco"))
	})
}

func Benchmark_RegularFileType_New(b *testing.B) {
	runBenchmark(b, metadata.RegularFileType, "file", false, func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewFileName(d.Name(), "file").GcsObjectName(), []byte("taco"))
	})
}

func Benchmark_UnknownType_Nonexistent_Old(b *testing.B) {
	runBenchmark(b, metadata.UnknownType, "absent", true, nil)
}

func Benchmark_UnknownType_Nonexistent_New(b *testing.B) {
	runBenchmark(b, metadata.UnknownType, "absent", false, nil)
}

func Benchmark_UnknownType_ExistingFile_Old(b *testing.B) {
	runBenchmark(b, metadata.UnknownType, "file", true, func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewFileName(d.Name(), "file").GcsObjectName(), []byte("taco"))
	})
}

func Benchmark_UnknownType_ExistingFile_New(b *testing.B) {
	runBenchmark(b, metadata.UnknownType, "file", false, func(d *dirInode) {
		_, _ = storageutil.CreateObject(context.Background(), d.Bucket(), NewFileName(d.Name(), "file").GcsObjectName(), []byte("taco"))
	})
}
