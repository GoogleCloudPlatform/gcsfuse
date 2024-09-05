// Copyright 2024 Google LLC
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
	"fmt"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestFsErrStrAndCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fsErr            error
		expectedCategory string
	}{
		{
			fsErr:            fmt.Errorf("some random error"),
			expectedCategory: errIO,
		},
		{
			fsErr:            syscall.ENOTEMPTY,
			expectedCategory: errDirNotEmpty,
		},
		{
			fsErr:            syscall.EEXIST,
			expectedCategory: errFileExists,
		},
		{
			fsErr:            syscall.EINVAL,
			expectedCategory: errInvalidArg,
		},
		{
			fsErr:            syscall.EINTR,
			expectedCategory: errInterrupt,
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedCategory: errNotImplemented,
		},
		{
			fsErr:            syscall.ENOSPC,
			expectedCategory: errProcessMgmt,
		},
		{
			fsErr:            syscall.E2BIG,
			expectedCategory: errInvalidOp,
		},
		{
			fsErr:            syscall.EHOSTDOWN,
			expectedCategory: errNetwork,
		},
		{
			fsErr:            syscall.ENODATA,
			expectedCategory: errMisc,
		},
		{
			fsErr:            syscall.ENODEV,
			expectedCategory: errDevice,
		},
		{
			fsErr:            syscall.EISDIR,
			expectedCategory: errFileDir,
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedCategory: errNotImplemented,
		},
		{
			fsErr:            syscall.ENFILE,
			expectedCategory: errTooManyFiles,
		},
		{
			fsErr:            syscall.EPERM,
			expectedCategory: errPerm,
		},
	}

	for idx, tc := range tests {
		t.Run(fmt.Sprintf("fsErrStrAndCategor_case_%d", idx), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expectedCategory, categorize(tc.fsErr))
		})
	}
}

func newInMemoryExporter(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	ex := tracetest.NewInMemoryExporter()
	t.Cleanup(func() {
		ex.Reset()
	})
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(ex)))
	monitor.InitializeTracer()
	return ex
}

type dummyFS struct{}

func (d dummyFS) StatFS(_ context.Context, _ *fuseops.StatFSOp) error {
	return nil
}

func (d dummyFS) LookUpInode(_ context.Context, _ *fuseops.LookUpInodeOp) error {
	return nil
}

func (d dummyFS) GetInodeAttributes(_ context.Context, _ *fuseops.GetInodeAttributesOp) error {
	return nil
}

func (d dummyFS) SetInodeAttributes(_ context.Context, _ *fuseops.SetInodeAttributesOp) error {
	return nil
}

func (d dummyFS) ForgetInode(_ context.Context, _ *fuseops.ForgetInodeOp) error {
	return nil
}

func (d dummyFS) BatchForget(_ context.Context, _ *fuseops.BatchForgetOp) error {
	return nil
}

func (d dummyFS) MkDir(_ context.Context, _ *fuseops.MkDirOp) error {
	return nil
}

func (d dummyFS) MkNode(_ context.Context, _ *fuseops.MkNodeOp) error {
	return nil
}

func (d dummyFS) CreateFile(_ context.Context, _ *fuseops.CreateFileOp) error {
	return nil
}

func (d dummyFS) CreateLink(_ context.Context, _ *fuseops.CreateLinkOp) error {
	return nil
}

func (d dummyFS) CreateSymlink(_ context.Context, _ *fuseops.CreateSymlinkOp) error {
	return nil
}

func (d dummyFS) Rename(_ context.Context, _ *fuseops.RenameOp) error {
	return nil
}

func (d dummyFS) RmDir(_ context.Context, _ *fuseops.RmDirOp) error {
	return nil
}

func (d dummyFS) Unlink(_ context.Context, _ *fuseops.UnlinkOp) error {
	return nil
}

func (d dummyFS) OpenDir(_ context.Context, _ *fuseops.OpenDirOp) error {
	return nil
}

func (d dummyFS) ReadDir(_ context.Context, _ *fuseops.ReadDirOp) error {
	return nil
}

func (d dummyFS) ReleaseDirHandle(_ context.Context, _ *fuseops.ReleaseDirHandleOp) error {
	return nil
}

func (d dummyFS) OpenFile(_ context.Context, _ *fuseops.OpenFileOp) error {
	return nil
}

func (d dummyFS) ReadFile(_ context.Context, _ *fuseops.ReadFileOp) error {
	return nil
}

func (d dummyFS) WriteFile(_ context.Context, _ *fuseops.WriteFileOp) error {
	return nil
}

func (d dummyFS) SyncFile(_ context.Context, _ *fuseops.SyncFileOp) error {
	return nil
}

func (d dummyFS) FlushFile(_ context.Context, _ *fuseops.FlushFileOp) error {
	return nil
}

func (d dummyFS) ReleaseFileHandle(_ context.Context, _ *fuseops.ReleaseFileHandleOp) error {
	return nil
}

func (d dummyFS) ReadSymlink(_ context.Context, _ *fuseops.ReadSymlinkOp) error {
	return nil
}

func (d dummyFS) RemoveXattr(_ context.Context, _ *fuseops.RemoveXattrOp) error {
	return nil
}

func (d dummyFS) GetXattr(_ context.Context, _ *fuseops.GetXattrOp) error {
	return nil
}

func (d dummyFS) ListXattr(_ context.Context, _ *fuseops.ListXattrOp) error {
	return nil
}

func (d dummyFS) SetXattr(_ context.Context, _ *fuseops.SetXattrOp) error {
	return nil
}

func (d dummyFS) Fallocate(_ context.Context, _ *fuseops.FallocateOp) error {
	return nil
}

func (d dummyFS) Destroy() {}

func TestSpanCreation(t *testing.T) {
	ex := newInMemoryExporter(t)
	t.Cleanup(func() {
		ex.Reset()
	})
	m := monitoring{
		wrapped: dummyFS{},
	}

	err := m.StatFS(context.Background(), nil)
	require.NoError(t, err)

	ss := ex.GetSpans()
	require.Len(t, ss, 1)
	assert.Equal(t, "StatFS", ss[0].Name)
	assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
}
