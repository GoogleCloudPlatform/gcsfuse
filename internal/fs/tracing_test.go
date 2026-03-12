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

	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
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

type TracingTestSuite struct {
	suite.Suite
	globalExporter *tracetest.InMemoryExporter
}

func (s *TracingTestSuite) SetupSuite() {
	s.globalExporter = newInMemoryExporter()
}

func (s *TracingTestSuite) SetupSubTest() {
	s.globalExporter.Reset()
}

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
				EnableBufferedRead: false,
			},
			EnableNewReader: true,
			FileSystem: cfg.FileSystemConfig{
				IgnoreInterrupts: ignoreInterrupts,
			},
			Monitoring: cfg.MonitoringConfig{
				ExperimentalTracingMode:          []string{"stdout"},
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
		TraceHandle:          tracing.NewOTELTracer(),
	}
	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return bucket, server
}

func newInMemoryExporter() *tracetest.InMemoryExporter {
	ex := tracetest.NewInMemoryExporter()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(ex)))
	return ex
}

func (s *TracingTestSuite) TestTraceLookupInode() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup"}},
		{"disabled", false, []string{"fs.inode.lookup"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceStatFS() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.stat_fs"}},
		{"disabled", false, []string{"fs.stat_fs"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			statFsOp := &fuseops.StatFSOp{}

			err := m.StatFS(context.Background(), statFsOp)
			require.NoError(t, err)

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceGetInodeAttributes() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.get_attributes"}},
		{"disabled", false, []string{"fs.inode.get_attributes"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.GetInodeAttributesOp{
				Inode: fuseops.RootInodeID,
			}

			err := m.GetInodeAttributes(context.Background(), op)
			require.NoError(t, err)

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceSetInodeAttributes() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.inode.set_attributes"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.inode.set_attributes"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceForgetInode() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.inode.forget"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.inode.forget"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceMkDir() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.dir.mk"}},
		{"disabled", false, []string{"fs.dir.mk"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceMkNode() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.mknode"}},
		{"disabled", false, []string{"fs.mknode"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceCreateFile() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.file.create"}},
		{"disabled", false, []string{"fs.file.create"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceCreateLink() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.link.create"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.link.create"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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
			assert.Error(t, err) // The operation is not implemented, so we expect an error.

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceCreateSymlink() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.symlink.create"}},
		{"disabled", false, []string{"fs.symlink.create"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceRename() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.rename"}},
		{"disabled", false, []string{"fs.rename"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceRmDir() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.dir.mk", "fs.dir.rm"}},
		{"disabled", false, []string{"fs.dir.mk", "fs.dir.rm"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceUnlink() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.unlink"}},
		{"disabled", false, []string{"fs.unlink"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceOpenDir() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.dir.open"}},
		{"disabled", false, []string{"fs.dir.open"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
			ctx := context.Background()
			fileName := "test.txt"
			content := "test content"
			createWithContents(ctx, t, bucket, fileName, content)
			op := &fuseops.OpenDirOp{
				Inode: fuseops.RootInodeID,
			}

			err := m.OpenDir(ctx, op)
			require.NoError(t, err)

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceReadDir() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.dir.open", "fs.dir.read"}},
		{"disabled", false, []string{"fs.dir.open", "fs.dir.read"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceReadDirPlus() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.dir.open", "fs.dir.read_plus"}},
		{"disabled", false, []string{"fs.dir.open", "fs.dir.read_plus"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceReleaseDirHandle() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.dir.open", "fs.dir.release_handle"}},
		{"disabled", false, []string{"fs.dir.open", "fs.dir.release_handle"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceOpenFile() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.file.open"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.file.open"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceReadFile() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.file.open", "fs.file.read"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.file.open", "fs.file.read"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceWriteFile() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.file.open", "fs.file.write"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.file.open", "fs.file.write"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceSyncFile() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.file.sync"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.file.sync"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceFlushFile() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.file.open", "fs.file.flush"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.file.open", "fs.file.flush"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceReleaseFileHandle() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.file.open", "fs.file.release_handle"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.file.open", "fs.file.release_handle"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceReadSymlink() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.symlink.create", "fs.symlink.read"}},
		{"disabled", false, []string{"fs.symlink.create", "fs.symlink.read"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			_, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceRemoveXattr() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.xattr.remove"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.xattr.remove"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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
			assert.Error(t, err) // The operation is not implemented, so we expect an error.

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceGetXattr() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.xattr.get"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.xattr.get"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceListXattr() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.xattr.list"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.xattr.list"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceSetXattr() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.set_xattr"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.set_xattr"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceFallocate() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.file.open", "fs.fallocate"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.file.open", "fs.fallocate"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func (s *TracingTestSuite) TestTraceSyncFS() {
	ctx := context.Background()
	testCases := []struct {
		name             string
		ignoreInterrupts bool
		spans            []string
	}{
		{"enabled", true, []string{"fs.inode.lookup", "fs.sync_fs"}},
		{"disabled", false, []string{"fs.inode.lookup", "fs.sync_fs"}},
	}
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()
			bucket, server := createTestFileSystemWithTraces(ctx, t, tt.ignoreInterrupts)
			m := wrappers.WithTracing(server, tracing.NewOTELTracer())
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

			ss := s.globalExporter.GetSpans()
			require.Len(t, ss, len(tt.spans))
			for i, spanName := range tt.spans {
				assert.Equal(t, spanName, ss[i].Name)
				assert.Equal(t, trace.SpanKindServer, ss[i].SpanKind)
			}
		})
	}
}

func TestTracingTestSuite(t *testing.T) {
	suite.Run(t, new(TracingTestSuite))
}
