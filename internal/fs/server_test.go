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
	"context"
	"fmt"
	"syscall"
	"testing"

	newcfg "github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v4/metrics"
	"github.com/googlecloudplatform/gcsfuse/v4/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ctxCheckingBucket wraps a gcs.Bucket and returns ctx.Err() on StatObject when
// the incoming context is canceled, allowing tests to verify whether
// WithIgnoreInterrupts is wired in NewServer.
type ctxCheckingBucket struct {
	gcs.Bucket
}

func (b *ctxCheckingBucket) StatObject(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return b.Bucket.StatObject(ctx, req)
}

type serverTestBucketManager struct {
	buckets map[string]gcs.Bucket
}

func (bm *serverTestBucketManager) ShutDown() {}

func (bm *serverTestBucketManager) SetUpBucket(
	_ context.Context,
	name string,
	_ bool,
	_ metrics.MetricHandle) (gcsx.SyncerBucket, error) {
	bucket, ok := bm.buckets[name]
	if !ok {
		return gcsx.SyncerBucket{}, fmt.Errorf("bucket %q does not exist", name)
	}
	return gcsx.NewSyncerBucket(0, 0, 0, "", gcsx.NewContentTypeBucket(bucket), false), nil
}

func newTestServerConfig(t *testing.T, ignoreInterrupts bool) (*ServerConfig, gcs.Bucket) {
	t.Helper()
	const bucketName = "test-bucket"
	clock := &timeutil.SimulatedClock{}
	baseBucket := fake.NewFakeBucket(clock, bucketName, gcs.BucketType{})
	bucket := &ctxCheckingBucket{Bucket: baseBucket}
	return &ServerConfig{
		NewConfig: &newcfg.Config{
			FileSystem: newcfg.FileSystemConfig{
				IgnoreInterrupts: ignoreInterrupts,
			},
		},
		CacheClock:    clock,
		BucketName:    bucketName,
		BucketManager: &serverTestBucketManager{buckets: map[string]gcs.Bucket{bucketName: bucket}},
		MetricHandle:  metrics.NewNoopMetrics(),
		TraceHandle:   tracing.NewNoopTracer(),
	}, baseBucket
}

func TestNewServer_IgnoreInterrupts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		ignoreInterrupts bool
	}{
		{name: "enabled", ignoreInterrupts: true},
		{name: "disabled", ignoreInterrupts: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			serverCfg, _ := newTestServerConfig(t, tc.ignoreInterrupts)

			server, err := NewServer(context.Background(), serverCfg)

			require.NoError(t, err)
			assert.NotNil(t, server)
		})
	}
}

func TestNewWrappedFileSystem_IgnoreInterrupts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		ignoreInterrupts bool
		wantErr          error
	}{
		{name: "enabled detaches canceled context", ignoreInterrupts: true, wantErr: nil},
		{name: "disabled propagates cancellation as EINTR", ignoreInterrupts: false, wantErr: syscall.EINTR},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			serverCfg, baseBucket := newTestServerConfig(t, tc.ignoreInterrupts)
			_, err := storageutil.CreateObject(ctx, baseBucket, "file.txt", []byte("hello"))
			require.NoError(t, err)

			fs, err := newWrappedFileSystem(ctx, serverCfg)
			require.NoError(t, err)

			canceledCtx, cancel := context.WithCancel(ctx)
			cancel()

			err = fs.LookUpInode(canceledCtx, &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   "file.txt",
			})

			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}
