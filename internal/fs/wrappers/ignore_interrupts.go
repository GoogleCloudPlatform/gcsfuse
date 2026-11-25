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

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

// WithIgnoreInterrupts wraps a FileSystem so that cancellation and deadlines on
// incoming FUSE request contexts (for example, from system interrupt signals)
// do not abort in-flight operations, while preserving all context values.
func WithIgnoreInterrupts(wrapped fuseutil.FileSystem) fuseutil.FileSystem {
	return &ignoreInterrupts{
		wrapped: wrapped,
	}
}

type ignoreInterrupts struct {
	wrapped fuseutil.FileSystem
}

func (ii *ignoreInterrupts) Destroy() {
	ii.wrapped.Destroy()
}

func (ii *ignoreInterrupts) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	return ii.wrapped.StatFS(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	return ii.wrapped.LookUpInode(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	return ii.wrapped.GetInodeAttributes(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	return ii.wrapped.SetInodeAttributes(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	return ii.wrapped.ForgetInode(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) BatchForget(
	ctx context.Context,
	op *fuseops.BatchForgetOp) error {
	return ii.wrapped.BatchForget(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	return ii.wrapped.MkDir(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	return ii.wrapped.MkNode(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	return ii.wrapped.CreateFile(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	return ii.wrapped.CreateLink(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	return ii.wrapped.CreateSymlink(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	return ii.wrapped.Rename(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	return ii.wrapped.RmDir(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	return ii.wrapped.Unlink(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	return ii.wrapped.OpenDir(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	return ii.wrapped.ReadDir(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ReadDirPlus(
	ctx context.Context,
	op *fuseops.ReadDirPlusOp) error {
	return ii.wrapped.ReadDirPlus(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	return ii.wrapped.ReleaseDirHandle(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	return ii.wrapped.OpenFile(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	return ii.wrapped.ReadFile(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	return ii.wrapped.WriteFile(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	return ii.wrapped.SyncFile(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	return ii.wrapped.FlushFile(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	return ii.wrapped.ReleaseFileHandle(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	return ii.wrapped.ReadSymlink(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	return ii.wrapped.RemoveXattr(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	return ii.wrapped.GetXattr(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	return ii.wrapped.ListXattr(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	return ii.wrapped.SetXattr(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	return ii.wrapped.Fallocate(context.WithoutCancel(ctx), op)
}

func (ii *ignoreInterrupts) SyncFS(
	ctx context.Context,
	op *fuseops.SyncFSOp) error {
	return ii.wrapped.SyncFS(context.WithoutCancel(ctx), op)
}
