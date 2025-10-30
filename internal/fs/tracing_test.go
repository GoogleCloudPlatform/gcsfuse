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

package fs_test

import (
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func createTestFileSystemWithTraces(ctx context.Context, t *testing.T, ignoreInterrupts bool) (gcs.Bucket, fuseutil.FileSystem) {
	t.Helper()

	bucketName := "test-bucket"
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Hierarchical: false})
	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			Read: cfg.ReadConfig{
				GlobalMaxBlocks:    1,
				BlockSizeMb:        1,
				MaxBlocksPerHandle: 10,
			},
			FileSystem: cfg.FileSystemConfig{
				IgnoreInterrupts: ignoreInterrupts,
			},
			Monitoring: cfg.MonitoringConfig{
				ExperimentalTracingMode:          "stdout",
				ExperimentalTracingSamplingRatio: 1.0,
			},
		},
		CacheClock: &timeutil.SimulatedClock{},
		BucketName: bucketName,
		BucketManager: &fakeBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
		},
		SequentialReadSizeMb: 200,
	}
	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return bucket, server
}

func newInMemoryExporter(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	ex := tracetest.NewInMemoryExporter()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(ex)))
	return ex
}

func TestTraceLookupInode(t *testing.T) {
	ctx := context.Background()
	var ignoreInterruptTestCases = []struct {
		caseName string
		value    bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.caseName, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.value)

			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)

			lookupOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}

			m := wrappers.WithTracing(server)
			err := m.LookUpInode(context.Background(), lookupOp)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}
