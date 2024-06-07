// Copyright 2021 Google Inc. All Rights Reserved.
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

type ExtraInfo struct {
	Inode     fuseops.InodeID
	OpContext fuseops.OpContext
	Handle    fuseops.HandleID
}

func WithExtraLogInfo(wrapped fuseutil.FileSystem) fuseutil.FileSystem {
	return &extraLogInfo{
		wrapped: wrapped,
	}
}

type extraLogInfo struct {
	wrapped fuseutil.FileSystem
}

func (em *extraLogInfo) Destroy() {

	em.wrapped.Destroy()
}

func (em *extraLogInfo) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	myExtraInfo := ExtraInfo{Inode: fuseops.InodeID(op.Inodes)}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.StatFS(ctx, op)
	return err
}

func (em *extraLogInfo) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.LookUpInode(ctx, op)
	return err
}

func (em *extraLogInfo) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.GetInodeAttributes(ctx, op)
	return err
}

func (em *extraLogInfo) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.SetInodeAttributes(ctx, op)
	return err
}

func (em *extraLogInfo) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.ForgetInode(ctx, op)
	return err
}

func (em *extraLogInfo) BatchForget(
	ctx context.Context,
	op *fuseops.BatchForgetOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.BatchForget(ctx, op)
	return err
}

func (em *extraLogInfo) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.MkDir(ctx, op)
	return err
}

func (em *extraLogInfo) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.MkNode(ctx, op)
	return err
}

func (em *extraLogInfo) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.CreateFile(ctx, op)
	return err
}

func (em *extraLogInfo) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.CreateLink(ctx, op)
	return err
}

func (em *extraLogInfo) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.CreateSymlink(ctx, op)
	return err
}

func (em *extraLogInfo) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.Rename(ctx, op)
	return err
}

func (em *extraLogInfo) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.RmDir(ctx, op)
	return err
}

func (em *extraLogInfo) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	myExtraInfo := ExtraInfo{OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.Unlink(ctx, op)
	return err
}

func (em *extraLogInfo) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.OpenDir(ctx, op)
	return err
}

func (em *extraLogInfo) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.ReadDir(ctx, op)
	return err
}

func (em *extraLogInfo) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.ReleaseDirHandle(ctx, op)
	return err
}

func (em *extraLogInfo) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.OpenFile(ctx, op)
	return err
}

func (em *extraLogInfo) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, Inode: op.Inode, OpContext: op.OpContext}
	//logger.Infof("Writing to Read context %v", myExtraInfo)

	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.ReadFile(ctx, op)
	return err
}

func (em *extraLogInfo) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.WriteFile(ctx, op)
	return err
}

func (em *extraLogInfo) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, Handle: op.Handle, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.SyncFile(ctx, op)
	return err
}

func (em *extraLogInfo) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.FlushFile(ctx, op)
	return err
}

func (em *extraLogInfo) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	myExtraInfo := ExtraInfo{Handle: op.Handle, OpContext: op.OpContext}
	//logger.Tracef("%v -> %v", op.OpContext.FuseID, op.Handle)
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.ReleaseFileHandle(ctx, op)
	return err
}

func (em *extraLogInfo) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.ReadSymlink(ctx, op)
	return err
}

func (em *extraLogInfo) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.RemoveXattr(ctx, op)
	return err
}

func (em *extraLogInfo) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.GetXattr(ctx, op)
	return err
}

func (em *extraLogInfo) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.ListXattr(ctx, op)
	return err
}

func (em *extraLogInfo) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.SetXattr(ctx, op)
	return err
}

func (em *extraLogInfo) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	myExtraInfo := ExtraInfo{Inode: op.Inode, Handle: op.Handle, OpContext: op.OpContext}
	ctx = context.WithValue(ctx, "extraInfo", myExtraInfo)
	err := em.wrapped.Fallocate(ctx, op)
	return err
}
