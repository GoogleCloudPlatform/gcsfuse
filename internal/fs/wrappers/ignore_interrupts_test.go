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

package wrappers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v4/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v4/metrics"
	"github.com/googlecloudplatform/gcsfuse/v4/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCtxKey struct{}

// ctxRecordingFS records the name and context of every FUSE operation invoked on it.
type ctxRecordingFS struct {
	destroyCalled bool
	lastOp        string
	lastCtx       context.Context
	returnErr     error
	onCall        func(ctx context.Context)
}

var _ fuseutil.FileSystem = (*ctxRecordingFS)(nil)

func (f *ctxRecordingFS) record(op string, ctx context.Context) error {
	f.lastOp = op
	f.lastCtx = ctx
	if f.onCall != nil {
		f.onCall(ctx)
	}
	return f.returnErr
}

func (f *ctxRecordingFS) Destroy() {
	f.destroyCalled = true
}

func (f *ctxRecordingFS) StatFS(ctx context.Context, _ *fuseops.StatFSOp) error {
	return f.record("StatFS", ctx)
}

func (f *ctxRecordingFS) LookUpInode(ctx context.Context, _ *fuseops.LookUpInodeOp) error {
	return f.record("LookUpInode", ctx)
}

func (f *ctxRecordingFS) GetInodeAttributes(ctx context.Context, _ *fuseops.GetInodeAttributesOp) error {
	return f.record("GetInodeAttributes", ctx)
}

func (f *ctxRecordingFS) SetInodeAttributes(ctx context.Context, _ *fuseops.SetInodeAttributesOp) error {
	return f.record("SetInodeAttributes", ctx)
}

func (f *ctxRecordingFS) ForgetInode(ctx context.Context, _ *fuseops.ForgetInodeOp) error {
	return f.record("ForgetInode", ctx)
}

func (f *ctxRecordingFS) BatchForget(ctx context.Context, _ *fuseops.BatchForgetOp) error {
	return f.record("BatchForget", ctx)
}

func (f *ctxRecordingFS) MkDir(ctx context.Context, _ *fuseops.MkDirOp) error {
	return f.record("MkDir", ctx)
}

func (f *ctxRecordingFS) MkNode(ctx context.Context, _ *fuseops.MkNodeOp) error {
	return f.record("MkNode", ctx)
}

func (f *ctxRecordingFS) CreateFile(ctx context.Context, _ *fuseops.CreateFileOp) error {
	return f.record("CreateFile", ctx)
}

func (f *ctxRecordingFS) CreateLink(ctx context.Context, _ *fuseops.CreateLinkOp) error {
	return f.record("CreateLink", ctx)
}

func (f *ctxRecordingFS) CreateSymlink(ctx context.Context, _ *fuseops.CreateSymlinkOp) error {
	return f.record("CreateSymlink", ctx)
}

func (f *ctxRecordingFS) Rename(ctx context.Context, _ *fuseops.RenameOp) error {
	return f.record("Rename", ctx)
}

func (f *ctxRecordingFS) RmDir(ctx context.Context, _ *fuseops.RmDirOp) error {
	return f.record("RmDir", ctx)
}

func (f *ctxRecordingFS) Unlink(ctx context.Context, _ *fuseops.UnlinkOp) error {
	return f.record("Unlink", ctx)
}

func (f *ctxRecordingFS) OpenDir(ctx context.Context, _ *fuseops.OpenDirOp) error {
	return f.record("OpenDir", ctx)
}

func (f *ctxRecordingFS) ReadDir(ctx context.Context, _ *fuseops.ReadDirOp) error {
	return f.record("ReadDir", ctx)
}

func (f *ctxRecordingFS) ReadDirPlus(ctx context.Context, _ *fuseops.ReadDirPlusOp) error {
	return f.record("ReadDirPlus", ctx)
}

func (f *ctxRecordingFS) ReleaseDirHandle(ctx context.Context, _ *fuseops.ReleaseDirHandleOp) error {
	return f.record("ReleaseDirHandle", ctx)
}

func (f *ctxRecordingFS) OpenFile(ctx context.Context, _ *fuseops.OpenFileOp) error {
	return f.record("OpenFile", ctx)
}

func (f *ctxRecordingFS) ReadFile(ctx context.Context, _ *fuseops.ReadFileOp) error {
	return f.record("ReadFile", ctx)
}

func (f *ctxRecordingFS) WriteFile(ctx context.Context, _ *fuseops.WriteFileOp) error {
	return f.record("WriteFile", ctx)
}

func (f *ctxRecordingFS) SyncFile(ctx context.Context, _ *fuseops.SyncFileOp) error {
	return f.record("SyncFile", ctx)
}

func (f *ctxRecordingFS) FlushFile(ctx context.Context, _ *fuseops.FlushFileOp) error {
	return f.record("FlushFile", ctx)
}

func (f *ctxRecordingFS) ReleaseFileHandle(ctx context.Context, _ *fuseops.ReleaseFileHandleOp) error {
	return f.record("ReleaseFileHandle", ctx)
}

func (f *ctxRecordingFS) ReadSymlink(ctx context.Context, _ *fuseops.ReadSymlinkOp) error {
	return f.record("ReadSymlink", ctx)
}

func (f *ctxRecordingFS) RemoveXattr(ctx context.Context, _ *fuseops.RemoveXattrOp) error {
	return f.record("RemoveXattr", ctx)
}

func (f *ctxRecordingFS) GetXattr(ctx context.Context, _ *fuseops.GetXattrOp) error {
	return f.record("GetXattr", ctx)
}

func (f *ctxRecordingFS) ListXattr(ctx context.Context, _ *fuseops.ListXattrOp) error {
	return f.record("ListXattr", ctx)
}

func (f *ctxRecordingFS) SetXattr(ctx context.Context, _ *fuseops.SetXattrOp) error {
	return f.record("SetXattr", ctx)
}

func (f *ctxRecordingFS) Fallocate(ctx context.Context, _ *fuseops.FallocateOp) error {
	return f.record("Fallocate", ctx)
}

func (f *ctxRecordingFS) SyncFS(ctx context.Context, _ *fuseops.SyncFSOp) error {
	return f.record("SyncFS", ctx)
}

func TestWithIgnoreInterrupts_AllOpsDetachCancelAndPreserveValues(t *testing.T) {
	t.Parallel()

	ops := []struct {
		name string
		call func(fs fuseutil.FileSystem, ctx context.Context) error
	}{
		{"StatFS", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.StatFS(ctx, &fuseops.StatFSOp{}) }},
		{"LookUpInode", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.LookUpInode(ctx, &fuseops.LookUpInodeOp{})
		}},
		{"GetInodeAttributes", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.GetInodeAttributes(ctx, &fuseops.GetInodeAttributesOp{})
		}},
		{"SetInodeAttributes", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.SetInodeAttributes(ctx, &fuseops.SetInodeAttributesOp{})
		}},
		{"ForgetInode", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.ForgetInode(ctx, &fuseops.ForgetInodeOp{})
		}},
		{"BatchForget", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.BatchForget(ctx, &fuseops.BatchForgetOp{})
		}},
		{"MkDir", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.MkDir(ctx, &fuseops.MkDirOp{}) }},
		{"MkNode", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.MkNode(ctx, &fuseops.MkNodeOp{}) }},
		{"CreateFile", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.CreateFile(ctx, &fuseops.CreateFileOp{})
		}},
		{"CreateLink", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.CreateLink(ctx, &fuseops.CreateLinkOp{})
		}},
		{"CreateSymlink", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.CreateSymlink(ctx, &fuseops.CreateSymlinkOp{})
		}},
		{"Rename", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.Rename(ctx, &fuseops.RenameOp{}) }},
		{"RmDir", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.RmDir(ctx, &fuseops.RmDirOp{}) }},
		{"Unlink", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.Unlink(ctx, &fuseops.UnlinkOp{}) }},
		{"OpenDir", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.OpenDir(ctx, &fuseops.OpenDirOp{}) }},
		{"ReadDir", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.ReadDir(ctx, &fuseops.ReadDirOp{}) }},
		{"ReadDirPlus", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.ReadDirPlus(ctx, &fuseops.ReadDirPlusOp{})
		}},
		{"ReleaseDirHandle", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.ReleaseDirHandle(ctx, &fuseops.ReleaseDirHandleOp{})
		}},
		{"OpenFile", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.OpenFile(ctx, &fuseops.OpenFileOp{})
		}},
		{"ReadFile", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.ReadFile(ctx, &fuseops.ReadFileOp{})
		}},
		{"WriteFile", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.WriteFile(ctx, &fuseops.WriteFileOp{})
		}},
		{"SyncFile", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.SyncFile(ctx, &fuseops.SyncFileOp{})
		}},
		{"FlushFile", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.FlushFile(ctx, &fuseops.FlushFileOp{})
		}},
		{"ReleaseFileHandle", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{})
		}},
		{"ReadSymlink", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.ReadSymlink(ctx, &fuseops.ReadSymlinkOp{})
		}},
		{"RemoveXattr", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.RemoveXattr(ctx, &fuseops.RemoveXattrOp{})
		}},
		{"GetXattr", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.GetXattr(ctx, &fuseops.GetXattrOp{})
		}},
		{"ListXattr", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.ListXattr(ctx, &fuseops.ListXattrOp{})
		}},
		{"SetXattr", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.SetXattr(ctx, &fuseops.SetXattrOp{})
		}},
		{"Fallocate", func(fs fuseutil.FileSystem, ctx context.Context) error {
			return fs.Fallocate(ctx, &fuseops.FallocateOp{})
		}},
		{"SyncFS", func(fs fuseutil.FileSystem, ctx context.Context) error { return fs.SyncFS(ctx, &fuseops.SyncFSOp{}) }},
	}

	for _, tc := range ops {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			expectedErr := errors.New("wrapped error")
			inner := &ctxRecordingFS{returnErr: expectedErr}
			wrapped := WithIgnoreInterrupts(inner)

			ctxWithVal := context.WithValue(context.Background(), testCtxKey{}, "preserved-value")
			deadlineCtx, cancelDeadline := context.WithDeadline(ctxWithVal, time.Now().Add(time.Hour))
			defer cancelDeadline()
			canceledCtx, cancel := context.WithCancel(deadlineCtx)
			cancel()
			require.ErrorIs(t, canceledCtx.Err(), context.Canceled)

			err := tc.call(wrapped, canceledCtx)

			assert.ErrorIs(t, err, expectedErr)
			assert.Equal(t, tc.name, inner.lastOp)
			require.NotNil(t, inner.lastCtx)
			assert.NoError(t, inner.lastCtx.Err())
			assert.Nil(t, inner.lastCtx.Done())
			_, hasDeadline := inner.lastCtx.Deadline()
			assert.False(t, hasDeadline)
			assert.Equal(t, "preserved-value", inner.lastCtx.Value(testCtxKey{}))
		})
	}
}

func TestWithIgnoreInterrupts_Destroy(t *testing.T) {
	t.Parallel()
	inner := &ctxRecordingFS{}
	wrapped := WithIgnoreInterrupts(inner)

	wrapped.Destroy()

	assert.True(t, inner.destroyCalled)
}

func TestWithIgnoreInterrupts_PreservesTracingAndMonitoringContext(t *testing.T) {
	ex := newInMemoryExporter(t)
	tracer := tracing.NewOTELTracer()
	recorder := &readCountRecorder{MetricHandle: metrics.NewNoopMetrics()}

	inner := &ctxRecordingFS{
		onCall: func(ctx context.Context) {
			assert.NoError(t, ctx.Err())
			_, childSpan := tracer.StartSpan(ctx, "inner.op")
			tracer.EndSpan(childSpan)
			metadata.RecordCacheRead(ctx, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
		},
	}

	// Match the production wrapper stack in NewServer:
	// WithMonitoring -> WithTracing -> WithIgnoreInterrupts -> inner FS.
	fs := WithIgnoreInterrupts(inner)
	fs = WithTracing(fs, tracer)
	fs = WithMonitoring(fs, recorder)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, fs.LookUpInode(canceledCtx, &fuseops.LookUpInodeOp{}))

	// Metadata cache read recorded inside the wrapped FS is emitted by WithMonitoring.
	assert.Equal(t, []readCountEvent{
		{true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr},
	}, recorder.events)

	// Child span created inside the wrapped FS shares the trace ID with WithTracing's root span.
	spans := ex.GetSpans()
	require.Len(t, spans, 2)
	assert.Equal(t, spans[0].SpanContext.TraceID(), spans[1].SpanContext.TraceID())
	assert.Equal(t, spans[1].SpanContext.SpanID(), spans[0].Parent.SpanID())
}
