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
				EnableBufferedRead: true,
				GlobalMaxBlocks:    1,
				BlockSizeMb:        1,
				MaxBlocksPerHandle: 10,
			},
			EnableNewReader: true,
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
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookupOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(context.Background(), lookupOp)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceStatFS(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			statFsOp := &fuseops.StatFSOp{}
			err := m.StatFS(context.Background(), statFsOp)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "StatFS", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceGetInodeAttributes(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.GetInodeAttributesOp{
				Inode: fuseops.RootInodeID,
			}
			err := m.GetInodeAttributes(context.Background(), op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "GetInodeAttributes", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceSetInodeAttributes(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)

			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.SetInodeAttributesOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.SetInodeAttributes(context.Background(), op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "SetInodeAttributes", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceForgetInode(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			m := wrappers.WithTracing(server)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.ForgetInodeOp{
				Inode: lookUpOp.Entry.Child,
				N:     1,
			}
			err = m.ForgetInode(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "ForgetInode", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceMkDir(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.MkDirOp{
				Parent: fuseops.RootInodeID,
				Name:   "test",
			}
			err := m.MkDir(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "MkDir", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceMkNode(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.MkNodeOp{
				Parent: fuseops.RootInodeID,
				Name:   "test",
			}
			err := m.MkNode(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "MkNode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceCreateFile(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.CreateFileOp{
				Parent: fuseops.RootInodeID,
				Name:   "test",
			}
			err := m.CreateFile(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "CreateFile", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceCreateLink(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.CreateLinkOp{
				Parent: fuseops.RootInodeID,
				Name:   "link",
				Target: lookUpOp.Entry.Child,
			}
			err = m.CreateLink(ctx, op)
			assert.Error(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "CreateLink", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceCreateSymlink(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.CreateSymlinkOp{
				Parent: fuseops.RootInodeID,
				Name:   "test",
				Target: "target",
			}

			err := m.CreateSymlink(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "CreateSymlink", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceRename(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			oldName := "old"
			newName := "new"
			content := "test content"
			createWithContents(ctx, t, bucket, oldName, content)
			op := &fuseops.RenameOp{
				OldParent: fuseops.RootInodeID,
				OldName:   oldName,
				NewParent: fuseops.RootInodeID,
				NewName:   newName,
			}

			err := m.Rename(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "Rename", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceRmDir(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			dirName := "test"
			mkDirOp := &fuseops.MkDirOp{
				Parent: fuseops.RootInodeID,
				Name:   dirName,
			}
			err := m.MkDir(ctx, mkDirOp)
			require.NoError(t, err)
			op := &fuseops.RmDirOp{
				Parent: fuseops.RootInodeID,
				Name:   dirName,
			}
			err = m.RmDir(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "MkDir", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "RmDir", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceUnlink(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.UnlinkOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.Unlink(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "Unlink", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceOpenDir(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.OpenDirOp{
				Inode: fuseops.RootInodeID,
			}
			err := m.OpenDir(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 1)
			assert.Equal(t, "OpenDir", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
		})
	}
}

func TestTraceReadDir(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			openOp := &fuseops.OpenDirOp{
				Inode: fuseops.RootInodeID,
			}
			err := m.OpenDir(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.ReadDirOp{
				Inode:  fuseops.RootInodeID,
				Handle: openOp.Handle,
				Offset: 0,
				Dst:    make([]byte, 1024),
			}
			err = m.ReadDir(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "OpenDir", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "ReadDir", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceReadDirPlus(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			openOp := &fuseops.OpenDirOp{
				Inode: fuseops.RootInodeID,
			}
			err := m.OpenDir(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.ReadDirPlusOp{
				ReadDirOp: fuseops.ReadDirOp{
					Inode:  fuseops.RootInodeID,
					Handle: openOp.Handle,
					Offset: 0,
					Dst:    make([]byte, 1024),
				},
			}

			err = m.ReadDirPlus(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "OpenDir", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "ReadDirPlus", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceReleaseDirHandle(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			openOp := &fuseops.OpenDirOp{
				Inode: fuseops.RootInodeID,
			}
			err := m.OpenDir(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.ReleaseDirHandleOp{
				Handle: openOp.Handle,
			}
			err = m.ReleaseDirHandle(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "OpenDir", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "ReleaseDirHandle", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceOpenFile(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.OpenFileOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.OpenFile(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "OpenFile", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceReadFile(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			openOp := &fuseops.OpenFileOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.OpenFile(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.ReadFileOp{
				Inode:  lookUpOp.Entry.Child,
				Handle: openOp.Handle,
				Offset: 0,
			}
			err = m.ReadFile(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 3)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "OpenFile", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
			assert.Equal(t, "ReadFile", ss[2].Name)
			assert.Equal(t, trace.SpanKindServer, ss[2].SpanKind)
		})
	}
}

func TestTraceWriteFile(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := ""
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			openOp := &fuseops.OpenFileOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.OpenFile(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.WriteFileOp{
				Inode:  lookUpOp.Entry.Child,
				Handle: openOp.Handle,
				Offset: 0,
				Data:   []byte("test"),
			}

			err = m.WriteFile(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 3)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "OpenFile", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
			assert.Equal(t, "WriteFile", ss[2].Name)
			assert.Equal(t, trace.SpanKindServer, ss[2].SpanKind)
		})
	}
}

func TestTraceSyncFile(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			assert.NoError(t, err)
			op := &fuseops.SyncFileOp{
				Inode: lookUpOp.Entry.Child,
			}

			err = m.SyncFile(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "SyncFile", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceFlushFile(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			openOp := &fuseops.OpenFileOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.OpenFile(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.FlushFileOp{
				Inode:  lookUpOp.Entry.Child,
				Handle: openOp.Handle,
			}

			err = m.FlushFile(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 3)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "OpenFile", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
			assert.Equal(t, "FlushFile", ss[2].Name)
			assert.Equal(t, trace.SpanKindServer, ss[2].SpanKind)
		})
	}
}

func TestTraceReleaseFileHandle(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			openOp := &fuseops.OpenFileOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.OpenFile(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.ReleaseFileHandleOp{
				Handle: openOp.Handle,
			}
			err = m.ReleaseFileHandle(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 3)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "OpenFile", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
			assert.Equal(t, "ReleaseFileHandle", ss[2].Name)
			assert.Equal(t, trace.SpanKindServer, ss[2].SpanKind)
		})
	}
}

func TestTraceReadSymlink(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			_, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			symlinkName := "test"
			target := "target"
			createSymlinkOp := &fuseops.CreateSymlinkOp{
				Parent: fuseops.RootInodeID,
				Name:   symlinkName,
				Target: target,
			}
			err := m.CreateSymlink(ctx, createSymlinkOp)
			require.NoError(t, err)
			op := &fuseops.ReadSymlinkOp{
				Inode: createSymlinkOp.Entry.Child,
			}
			err = m.ReadSymlink(ctx, op)
			require.NoError(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "CreateSymlink", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "ReadSymlink", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceRemoveXattr(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.RemoveXattrOp{
				Inode: lookUpOp.Entry.Child,
				Name:  "user.test",
			}
			err = m.RemoveXattr(ctx, op)
			assert.Error(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "RemoveXattr", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceGetXattr(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.GetXattrOp{
				Inode: lookUpOp.Entry.Child,
				Name:  "user.test",
			}
			err = m.GetXattr(ctx, op)
			assert.NotNil(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "GetXattr", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceListXattr(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.ListXattrOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.ListXattr(ctx, op)
			assert.NotNil(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "ListXattr", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceSetXattr(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.SetXattrOp{
				Inode: lookUpOp.Entry.Child,
				Name:  "user.test",
				Value: []byte("test"),
			}
			err = m.SetXattr(ctx, op)
			assert.NotNil(t, err)

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "SetXattr", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}

func TestTraceFallocate(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			openOp := &fuseops.OpenFileOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.OpenFile(ctx, openOp)
			require.NoError(t, err)
			op := &fuseops.FallocateOp{
				Inode:  lookUpOp.Entry.Child,
				Handle: openOp.Handle,
				Offset: 0,
				Length: 10,
				Mode:   0,
			}
			err = m.Fallocate(ctx, op)
			assert.Error(t, err) // The operation is not implemented, so we expect an error.

			ss := ex.GetSpans()
			require.Len(t, ss, 3)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "OpenFile", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
			assert.Equal(t, "Fallocate", ss[2].Name)
			assert.Equal(t, trace.SpanKindServer, ss[2].SpanKind)
		})
	}
}

func TestTraceSyncFS(t *testing.T) {
	ctx := context.Background()
	ignoreInterruptTestCases := []struct {
		name             string
		ignoreInterrupts bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range ignoreInterruptTestCases {
		t.Run(tt.name, func(t *testing.T) {
			ex := newInMemoryExporter(t)
			t.Cleanup(func() {
				ex.Reset()
			})
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			lookUpOp := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   fileName,
			}
			err := m.LookUpInode(ctx, lookUpOp)
			require.NoError(t, err)
			op := &fuseops.SyncFSOp{
				Inode: lookUpOp.Entry.Child,
			}
			err = m.SyncFS(ctx, op)
			assert.Error(t, err) // The operation is not implemented, so we expect an error.

			ss := ex.GetSpans()
			require.Len(t, ss, 2)
			assert.Equal(t, "LookUpInode", ss[0].Name)
			assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
			assert.Equal(t, "SyncFS", ss[1].Name)
			assert.Equal(t, trace.SpanKindServer, ss[1].SpanKind)
		})
	}
}
