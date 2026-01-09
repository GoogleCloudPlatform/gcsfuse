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

	tracing "github.com/googlecloudplatform/gcsfuse/v3/tracing"
)

type tracedFS struct {
	wrapped     fuseutil.FileSystem
	traceHandle tracing.TraceHandle
}

// WithTracing wraps a FileSystem and creates a root trace.
func WithTracing(wrapped fuseutil.FileSystem, traceHandle tracing.TraceHandle) fuseutil.FileSystem {
	return &tracedFS{
		wrapped:     wrapped,
		traceHandle: traceHandle,
	}
}

func (fs *tracedFS) Destroy() {
	fs.wrapped.Destroy()
}

func (fs *tracedFS) invokeWrapped(ctx context.Context, opName string, w wrappedCall) error {
	// Span's SpanKind is set to trace.SpanKindServer since GCSFuse is like a server for the requests that the Kernel sends.
	ctx, span := fs.traceHandle.StartServerSpan(ctx, opName)
	defer fs.traceHandle.EndSpan(span)
	err := w(ctx)

	if err != nil {
		fs.traceHandle.RecordError(span, err)
	}

	return err
}

func (fs *tracedFS) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return fs.invokeWrapped(ctx, tracing.StatFS, func(ctx context.Context) error { return fs.wrapped.StatFS(ctx, op) })
}

func (fs *tracedFS) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	return fs.invokeWrapped(ctx, tracing.LookUpInode, func(ctx context.Context) error { return fs.wrapped.LookUpInode(ctx, op) })
}

func (fs *tracedFS) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, tracing.GetInodeAttributes, func(ctx context.Context) error { return fs.wrapped.GetInodeAttributes(ctx, op) })
}

func (fs *tracedFS) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, tracing.SetInodeAttributes, func(ctx context.Context) error { return fs.wrapped.SetInodeAttributes(ctx, op) })
}

func (fs *tracedFS) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	return fs.invokeWrapped(ctx, tracing.ForgetInode, func(ctx context.Context) error { return fs.wrapped.ForgetInode(ctx, op) })
}

func (fs *tracedFS) BatchForget(ctx context.Context, op *fuseops.BatchForgetOp) error {
	return fs.invokeWrapped(ctx, tracing.BatchForget, func(ctx context.Context) error { return fs.wrapped.BatchForget(ctx, op) })
}

func (fs *tracedFS) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return fs.invokeWrapped(ctx, tracing.MkDir, func(ctx context.Context) error { return fs.wrapped.MkDir(ctx, op) })
}

func (fs *tracedFS) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	return fs.invokeWrapped(ctx, tracing.MkNode, func(ctx context.Context) error { return fs.wrapped.MkNode(ctx, op) })
}

func (fs *tracedFS) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) error {
	return fs.invokeWrapped(ctx, tracing.CreateFile, func(ctx context.Context) error { return fs.wrapped.CreateFile(ctx, op) })
}

func (fs *tracedFS) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return fs.invokeWrapped(ctx, tracing.CreateLink, func(ctx context.Context) error { return fs.wrapped.CreateLink(ctx, op) })
}

func (fs *tracedFS) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return fs.invokeWrapped(ctx, tracing.CreateSymlink, func(ctx context.Context) error { return fs.wrapped.CreateSymlink(ctx, op) })
}

func (fs *tracedFS) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	return fs.invokeWrapped(ctx, tracing.Rename, func(ctx context.Context) error { return fs.wrapped.Rename(ctx, op) })
}

func (fs *tracedFS) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return fs.invokeWrapped(ctx, tracing.RmDir, func(ctx context.Context) error { return fs.wrapped.RmDir(ctx, op) })
}

func (fs *tracedFS) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	return fs.invokeWrapped(ctx, tracing.Unlink, func(ctx context.Context) error { return fs.wrapped.Unlink(ctx, op) })
}

func (fs *tracedFS) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return fs.invokeWrapped(ctx, tracing.OpenDir, func(ctx context.Context) error { return fs.wrapped.OpenDir(ctx, op) })
}

func (fs *tracedFS) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	return fs.invokeWrapped(ctx, tracing.ReadDir, func(ctx context.Context) error { return fs.wrapped.ReadDir(ctx, op) })
}

func (fs *tracedFS) ReadDirPlus(ctx context.Context, op *fuseops.ReadDirPlusOp) error {
	return fs.invokeWrapped(ctx, tracing.ReadDirPlus, func(ctx context.Context) error { return fs.wrapped.ReadDirPlus(ctx, op) })
}

func (fs *tracedFS) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	return fs.invokeWrapped(ctx, tracing.ReleaseDirHandle, func(ctx context.Context) error { return fs.wrapped.ReleaseDirHandle(ctx, op) })
}

func (fs *tracedFS) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return fs.invokeWrapped(ctx, tracing.OpenFile, func(ctx context.Context) error { return fs.wrapped.OpenFile(ctx, op) })
}

func (fs *tracedFS) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	return fs.invokeWrapped(ctx, tracing.ReadFile, func(ctx context.Context) error { return fs.wrapped.ReadFile(ctx, op) })
}

func (fs *tracedFS) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return fs.invokeWrapped(ctx, tracing.WriteFile, func(ctx context.Context) error { return fs.wrapped.WriteFile(ctx, op) })
}

func (fs *tracedFS) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) error {
	return fs.invokeWrapped(ctx, tracing.SyncFile, func(ctx context.Context) error { return fs.wrapped.SyncFile(ctx, op) })
}

func (fs *tracedFS) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) error {
	return fs.invokeWrapped(ctx, tracing.FlushFile, func(ctx context.Context) error { return fs.wrapped.FlushFile(ctx, op) })
}

func (fs *tracedFS) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	return fs.invokeWrapped(ctx, tracing.ReleaseFileHandle, func(ctx context.Context) error { return fs.wrapped.ReleaseFileHandle(ctx, op) })
}

func (fs *tracedFS) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return fs.invokeWrapped(ctx, tracing.ReadSymlink, func(ctx context.Context) error { return fs.wrapped.ReadSymlink(ctx, op) })
}

func (fs *tracedFS) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return fs.invokeWrapped(ctx, tracing.RemoveXattr, func(ctx context.Context) error { return fs.wrapped.RemoveXattr(ctx, op) })
}

func (fs *tracedFS) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	return fs.invokeWrapped(ctx, tracing.GetXattr, func(ctx context.Context) error { return fs.wrapped.GetXattr(ctx, op) })
}

func (fs *tracedFS) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	return fs.invokeWrapped(ctx, tracing.ListXattr, func(ctx context.Context) error { return fs.wrapped.ListXattr(ctx, op) })
}

func (fs *tracedFS) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return fs.invokeWrapped(ctx, tracing.SetXattr, func(ctx context.Context) error { return fs.wrapped.SetXattr(ctx, op) })
}

func (fs *tracedFS) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return fs.invokeWrapped(ctx, tracing.Fallocate, func(ctx context.Context) error { return fs.wrapped.Fallocate(ctx, op) })
}

func (fs *tracedFS) SyncFS(ctx context.Context, op *fuseops.SyncFSOp) error {
	return fs.invokeWrapped(ctx, tracing.SyncFS, func(ctx context.Context) error { return fs.wrapped.SyncFS(ctx, op) })
}
