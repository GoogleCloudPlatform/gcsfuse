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

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	internal_trace "github.com/googlecloudplatform/gcsfuse/v3/tracing"
)

type tracing struct {
	wrapped fuseutil.FileSystem
	tracer  trace.Tracer
}

// WithTracing wraps a FileSystem and creates a root trace.
func WithTracing(wrapped fuseutil.FileSystem) fuseutil.FileSystem {
	return &tracing{
		wrapped: wrapped,
		tracer:  internal_trace.GCSFuseTracer,
	}
}

func (fs *tracing) Destroy() {
	fs.wrapped.Destroy()
}

func (fs *tracing) invokeWrapped(ctx context.Context, opName string, w wrappedCall) error {
	// Span's SpanKind is set to trace.SpanKindServer since GCSFuse is like a server for the requests that the Kernel sends.
	ctx, span := fs.tracer.Start(ctx, opName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	err := w(ctx)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (fs *tracing) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return fs.invokeWrapped(ctx, internal_trace.StatFS, func(ctx context.Context) error { return fs.wrapped.StatFS(ctx, op) })
}

func (fs *tracing) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	return fs.invokeWrapped(ctx, internal_trace.LookUpInode, func(ctx context.Context) error { return fs.wrapped.LookUpInode(ctx, op) })
}

func (fs *tracing) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, internal_trace.GetInodeAttributes, func(ctx context.Context) error { return fs.wrapped.GetInodeAttributes(ctx, op) })
}

func (fs *tracing) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, internal_trace.SetInodeAttributes, func(ctx context.Context) error { return fs.wrapped.SetInodeAttributes(ctx, op) })
}

func (fs *tracing) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ForgetInode, func(ctx context.Context) error { return fs.wrapped.ForgetInode(ctx, op) })
}

func (fs *tracing) BatchForget(ctx context.Context, op *fuseops.BatchForgetOp) error {
	return fs.invokeWrapped(ctx, internal_trace.BatchForget, func(ctx context.Context) error { return fs.wrapped.BatchForget(ctx, op) })
}

func (fs *tracing) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return fs.invokeWrapped(ctx, internal_trace.MkDir, func(ctx context.Context) error { return fs.wrapped.MkDir(ctx, op) })
}

func (fs *tracing) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	return fs.invokeWrapped(ctx, internal_trace.MkNode, func(ctx context.Context) error { return fs.wrapped.MkNode(ctx, op) })
}

func (fs *tracing) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) error {
	return fs.invokeWrapped(ctx, internal_trace.CreateFile, func(ctx context.Context) error { return fs.wrapped.CreateFile(ctx, op) })
}

func (fs *tracing) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return fs.invokeWrapped(ctx, internal_trace.CreateLink, func(ctx context.Context) error { return fs.wrapped.CreateLink(ctx, op) })
}

func (fs *tracing) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return fs.invokeWrapped(ctx, internal_trace.CreateSymlink, func(ctx context.Context) error { return fs.wrapped.CreateSymlink(ctx, op) })
}

func (fs *tracing) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	return fs.invokeWrapped(ctx, internal_trace.Rename, func(ctx context.Context) error { return fs.wrapped.Rename(ctx, op) })
}

func (fs *tracing) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return fs.invokeWrapped(ctx, internal_trace.RmDir, func(ctx context.Context) error { return fs.wrapped.RmDir(ctx, op) })
}

func (fs *tracing) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	return fs.invokeWrapped(ctx, internal_trace.Unlink, func(ctx context.Context) error { return fs.wrapped.Unlink(ctx, op) })
}

func (fs *tracing) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return fs.invokeWrapped(ctx, internal_trace.OpenDir, func(ctx context.Context) error { return fs.wrapped.OpenDir(ctx, op) })
}

func (fs *tracing) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ReadDir, func(ctx context.Context) error { return fs.wrapped.ReadDir(ctx, op) })
}

func (fs *tracing) ReadDirPlus(ctx context.Context, op *fuseops.ReadDirPlusOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ReadDirPlus, func(ctx context.Context) error { return fs.wrapped.ReadDirPlus(ctx, op) })
}

func (fs *tracing) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ReleaseDirHandle, func(ctx context.Context) error { return fs.wrapped.ReleaseDirHandle(ctx, op) })
}

func (fs *tracing) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return fs.invokeWrapped(ctx, internal_trace.OpenFile, func(ctx context.Context) error { return fs.wrapped.OpenFile(ctx, op) })
}

func (fs *tracing) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ReadFile, func(ctx context.Context) error { return fs.wrapped.ReadFile(ctx, op) })
}

func (fs *tracing) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return fs.invokeWrapped(ctx, internal_trace.WriteFile, func(ctx context.Context) error { return fs.wrapped.WriteFile(ctx, op) })
}

func (fs *tracing) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) error {
	return fs.invokeWrapped(ctx, internal_trace.SyncFile, func(ctx context.Context) error { return fs.wrapped.SyncFile(ctx, op) })
}

func (fs *tracing) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) error {
	return fs.invokeWrapped(ctx, internal_trace.FlushFile, func(ctx context.Context) error { return fs.wrapped.FlushFile(ctx, op) })
}

func (fs *tracing) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ReleaseFileHandle, func(ctx context.Context) error { return fs.wrapped.ReleaseFileHandle(ctx, op) })
}

func (fs *tracing) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ReadSymlink, func(ctx context.Context) error { return fs.wrapped.ReadSymlink(ctx, op) })
}

func (fs *tracing) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return fs.invokeWrapped(ctx, internal_trace.RemoveXattr, func(ctx context.Context) error { return fs.wrapped.RemoveXattr(ctx, op) })
}

func (fs *tracing) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	return fs.invokeWrapped(ctx, internal_trace.GetXattr, func(ctx context.Context) error { return fs.wrapped.GetXattr(ctx, op) })
}

func (fs *tracing) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	return fs.invokeWrapped(ctx, internal_trace.ListXattr, func(ctx context.Context) error { return fs.wrapped.ListXattr(ctx, op) })
}

func (fs *tracing) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return fs.invokeWrapped(ctx, internal_trace.SetXattr, func(ctx context.Context) error { return fs.wrapped.SetXattr(ctx, op) })
}

func (fs *tracing) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return fs.invokeWrapped(ctx, internal_trace.Fallocate, func(ctx context.Context) error { return fs.wrapped.Fallocate(ctx, op) })
}

func (fs *tracing) SyncFS(ctx context.Context, op *fuseops.SyncFSOp) error {
	return fs.invokeWrapped(ctx, internal_trace.SyncFS, func(ctx context.Context) error { return fs.wrapped.SyncFS(ctx, op) })
}
