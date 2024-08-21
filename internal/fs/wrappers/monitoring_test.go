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
		expectedStr      string
		expectedCategory string
	}{
		{
			fsErr:            fmt.Errorf("some random error"),
			expectedStr:      "input/output error",
			expectedCategory: "input/output error",
		},
		{
			fsErr:            syscall.ENOTEMPTY,
			expectedStr:      "directory not empty",
			expectedCategory: "directory not empty",
		},
		{
			fsErr:            syscall.EEXIST,
			expectedStr:      "file exists",
			expectedCategory: "file exists",
		},
		{
			fsErr:            syscall.EINVAL,
			expectedStr:      "invalid argument",
			expectedCategory: "invalid argument",
		},
		{
			fsErr:            syscall.EINTR,
			expectedStr:      "interrupted system call",
			expectedCategory: "interrupt errors",
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedStr:      "function not implemented",
			expectedCategory: "function not implemented",
		},
		{
			fsErr:            syscall.ENOSPC,
			expectedStr:      "no space left on device",
			expectedCategory: "process/resource management errors",
		},
		{
			fsErr:            syscall.E2BIG,
			expectedStr:      "argument list too long",
			expectedCategory: "invalid operation",
		},
		{
			fsErr:            syscall.EHOSTDOWN,
			expectedStr:      "host is down",
			expectedCategory: "network errors",
		},
		{
			fsErr:            syscall.ENODATA,
			expectedStr:      "no data available",
			expectedCategory: "miscellaneous errors",
		},
		{
			fsErr:            syscall.ENODEV,
			expectedStr:      "no such device",
			expectedCategory: "device errors",
		},
		{
			fsErr:            syscall.EISDIR,
			expectedStr:      "is a directory",
			expectedCategory: "file/directory errors",
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedStr:      "function not implemented",
			expectedCategory: "function not implemented",
		},
		{
			fsErr:            syscall.ENFILE,
			expectedStr:      "too many open files in system",
			expectedCategory: "too many open files",
		},
		{
			fsErr:            syscall.EPERM,
			expectedStr:      "operation not permitted",
			expectedCategory: "permission errors",
		},
	}

	for idx, tc := range tests {
		t.Run(fmt.Sprintf("fsErrStrAndCategor_case_%d", idx), func(t *testing.T) {
			t.Parallel()

			actualErrStr, actualErrGrp := errStrAndCategory(tc.fsErr)

			assert.Equal(t, tc.expectedStr, actualErrStr)
			assert.Equal(t, tc.expectedCategory, actualErrGrp)
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

func TestSpan(t *testing.T) {
	ex := newInMemoryExporter(t)
	m := monitoring{
		wrapped: dummyFS{},
	}

	err := m.StatFS(context.Background(), nil)
	require.NoError(t, err)

	ss := ex.GetSpans()
	assert.Len(t, ss, 1)
	assert.Equal(t, "StatFS", ss[0].Name)
	assert.Equal(t, trace.SpanKindServer, ss[0].SpanKind)
}
