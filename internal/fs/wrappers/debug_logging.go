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
	"log"

	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

// WithDebugLogging wraps a FileSystem, logging the debug messages for the
// file system's input and errors
func WithDebugLogging(wrapped fuseutil.FileSystem) fuseutil.FileSystem {
	return &debugLogging{
		wrapped: wrapped,
		logger:  logger.NewDebug("debug_fs: "),
	}
}

type debugLogging struct {
	wrapped fuseutil.FileSystem
	logger  *log.Logger
}

func (fs *debugLogging) Destroy() {
	fs.wrapped.Destroy()
}

func (fs *debugLogging) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	err := fs.wrapped.StatFS(ctx, op)
	fs.logger.Printf("StatFS(): %v", err)
	return err
}

func (fs *debugLogging) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	err := fs.wrapped.LookUpInode(ctx, op)
	fs.logger.Printf("LookUpInode(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	err := fs.wrapped.GetInodeAttributes(ctx, op)
	fs.logger.Printf("GetInodeAttributes(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	err := fs.wrapped.SetInodeAttributes(ctx, op)
	fs.logger.Printf("SetInodeAttributes(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	err := fs.wrapped.ForgetInode(ctx, op)
	fs.logger.Printf("ForgetInode(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	err := fs.wrapped.MkDir(ctx, op)
	fs.logger.Printf("MkDir(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	err := fs.wrapped.MkNode(ctx, op)
	fs.logger.Printf("MkNode(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	err := fs.wrapped.CreateFile(ctx, op)
	fs.logger.Printf("CreateFile(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	err := fs.wrapped.CreateLink(ctx, op)
	fs.logger.Printf("CreateLink(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	err := fs.wrapped.CreateSymlink(ctx, op)
	fs.logger.Printf("CreateSymlink(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	err := fs.wrapped.Rename(ctx, op)
	fs.logger.Printf("Rename(%v, %q): %v", op.OldParent, op.OldName, err)
	return err
}

func (fs *debugLogging) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	err := fs.wrapped.RmDir(ctx, op)
	fs.logger.Printf("RmDir(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	err := fs.wrapped.Unlink(ctx, op)
	fs.logger.Printf("Unlink(%v, %q): %v", op.Parent, op.Name, err)
	return err
}

func (fs *debugLogging) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	err := fs.wrapped.OpenDir(ctx, op)
	fs.logger.Printf("OpenDir(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	err := fs.wrapped.ReadDir(ctx, op)
	fs.logger.Printf("ReadDir(%v, %v): %v", op.Inode, op.Offset, err)
	return err
}

func (fs *debugLogging) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	err := fs.wrapped.ReleaseDirHandle(ctx, op)
	fs.logger.Printf("ReleaseDirHandle(%v): %v", op.Handle, err)
	return err
}

func (fs *debugLogging) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	err := fs.wrapped.OpenFile(ctx, op)
	fs.logger.Printf("OpenFile(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	err := fs.wrapped.ReadFile(ctx, op)
	fs.logger.Printf("ReadFile(%v, %v): %v", op.Inode, op.Offset, err)
	return err
}

func (fs *debugLogging) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	err := fs.wrapped.WriteFile(ctx, op)
	fs.logger.Printf("WriteFile(%v, %v): %v", op.Inode, op.Offset, err)
	return err
}

func (fs *debugLogging) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	err := fs.wrapped.SyncFile(ctx, op)
	fs.logger.Printf("SyncFile(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	err := fs.wrapped.FlushFile(ctx, op)
	fs.logger.Printf("FlushFile(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	err := fs.wrapped.ReleaseFileHandle(ctx, op)
	fs.logger.Printf("ReleaseFileHandle(%v): %v", op.Handle, err)
	return err
}

func (fs *debugLogging) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	err := fs.wrapped.ReadSymlink(ctx, op)
	fs.logger.Printf("ReadSymlink(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	err := fs.wrapped.RemoveXattr(ctx, op)
	fs.logger.Printf("RemoveXattr(%v, %v): %v", op.Inode, op.Name, err)
	return err
}

func (fs *debugLogging) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	err := fs.wrapped.GetXattr(ctx, op)
	fs.logger.Printf("GetXattr(%v, %v): %v", op.Inode, op.Name, err)
	return err
}

func (fs *debugLogging) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	err := fs.wrapped.ListXattr(ctx, op)
	fs.logger.Printf("ListXattr(%v): %v", op.Inode, err)
	return err
}

func (fs *debugLogging) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	err := fs.wrapped.SetXattr(ctx, op)
	fs.logger.Printf("GetXattr(%v, %v): %v", op.Inode, op.Name, err)
	return err
}

func (fs *debugLogging) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	err := fs.wrapped.Fallocate(ctx, op)
	fs.logger.Printf("GetXattr(%v, %v): %v", op.Inode, op.Offset, err)
	return err
}
