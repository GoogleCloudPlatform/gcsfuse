// Copyright 2020 Google Inc. All Rights Reserved.
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
	"net/http"
	"syscall"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"google.golang.org/api/googleapi"
)

func errno(err error) error {
	if err == nil {
		return nil
	}

	// Use existing FS errno
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno
	}

	// Translate API errors into an FS errno
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case http.StatusForbidden:
			return syscall.EACCES
		}
	}

	// Unknown errors
	return syscall.EIO
}

// WithErrorMapping wraps a FileSystem, processing the returned errors, and 
// mapping them into syscall.Errno that can be understood by FUSE.
func WithErrorMapping(wrapped fuseutil.FileSystem) fuseutil.FileSystem {
	return &errorMapping{wrapped: wrapped}
}

type errorMapping struct {
	wrapped fuseutil.FileSystem
}

func (fs *errorMapping) Destroy() {
	fs.wrapped.Destroy()
}

func (fs *errorMapping) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	err := fs.wrapped.StatFS(ctx, op)
	return errno(err)
}

func (fs *errorMapping) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	err := fs.wrapped.LookUpInode(ctx, op)
	return errno(err)
}

func (fs *errorMapping) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	err := fs.wrapped.GetInodeAttributes(ctx, op)
	return errno(err)
}

func (fs *errorMapping) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	err := fs.wrapped.SetInodeAttributes(ctx, op)
	return errno(err)
}

func (fs *errorMapping) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	err := fs.wrapped.ForgetInode(ctx, op)
	return errno(err)
}

func (fs *errorMapping) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	err := fs.wrapped.MkDir(ctx, op)
	return errno(err)
}

func (fs *errorMapping) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	err := fs.wrapped.MkNode(ctx, op)
	return errno(err)
}

func (fs *errorMapping) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	err := fs.wrapped.CreateFile(ctx, op)
	return errno(err)
}

func (fs *errorMapping) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	err := fs.wrapped.CreateLink(ctx, op)
	return errno(err)
}

func (fs *errorMapping) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	err := fs.wrapped.CreateSymlink(ctx, op)
	return errno(err)
}

func (fs *errorMapping) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	err := fs.wrapped.Rename(ctx, op)
	return errno(err)
}

func (fs *errorMapping) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	err := fs.wrapped.RmDir(ctx, op)
	return errno(err)
}

func (fs *errorMapping) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	err := fs.wrapped.Unlink(ctx, op)
	return errno(err)
}

func (fs *errorMapping) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	err := fs.wrapped.OpenDir(ctx, op)
	return errno(err)
}

func (fs *errorMapping) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	err := fs.wrapped.ReadDir(ctx, op)
	return errno(err)
}

func (fs *errorMapping) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	err := fs.wrapped.ReleaseDirHandle(ctx, op)
	return errno(err)
}

func (fs *errorMapping) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	err := fs.wrapped.OpenFile(ctx, op)
	return errno(err)
}

func (fs *errorMapping) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	err := fs.wrapped.ReadFile(ctx, op)
	return errno(err)
}

func (fs *errorMapping) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	err := fs.wrapped.WriteFile(ctx, op)
	return errno(err)
}

func (fs *errorMapping) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	err := fs.wrapped.SyncFile(ctx, op)
	return errno(err)
}

func (fs *errorMapping) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	err := fs.wrapped.FlushFile(ctx, op)
	return errno(err)
}

func (fs *errorMapping) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	err := fs.wrapped.ReleaseFileHandle(ctx, op)
	return errno(err)
}

func (fs *errorMapping) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	err := fs.wrapped.ReadSymlink(ctx, op)
	return errno(err)
}

func (fs *errorMapping) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	err := fs.wrapped.RemoveXattr(ctx, op)
	return errno(err)
}

func (fs *errorMapping) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	err := fs.wrapped.GetXattr(ctx, op)
	return errno(err)
}

func (fs *errorMapping) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	err := fs.wrapped.ListXattr(ctx, op)
	return errno(err)
}

func (fs *errorMapping) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	err := fs.wrapped.SetXattr(ctx, op)
	return errno(err)
}

func (fs *errorMapping) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	err := fs.wrapped.Fallocate(ctx, op)
	return errno(err)
}
