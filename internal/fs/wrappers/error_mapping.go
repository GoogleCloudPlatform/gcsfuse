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
	"errors"
	"net/http"
	"strings"
	"syscall"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"google.golang.org/api/googleapi"
)

var (
	DefaultFSError = syscall.EIO
)

func errno(err error) error {
	if err == nil {
		return nil
	}

	// Use existing em errno
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno
	}

	// The fuse op is interrupted
	if errors.Is(err, context.Canceled) {
		return syscall.EINTR
	}
	if errors.Is(err, storage.ErrObjectNotExist) {
		return syscall.ENOENT
	}

	// The HTTP request is canceled
	if strings.Contains(err.Error(), "net/http: request canceled") {
		return syscall.ECANCELED
	}

	// Cannot authenticate
	if strings.Contains(err.Error(), "oauth2: cannot fetch token") {
		return syscall.EACCES
	}

	// Translate API errors into an em errno
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case http.StatusForbidden:
			return syscall.EACCES
		case http.StatusNotFound:
			return syscall.ENOENT
		}
	}

	return DefaultFSError
}

// WithErrorMapping wraps a FileSystem, processing the returned errors, and
// mapping them into syscall.Errno that can be understood by FUSE.
func WithErrorMapping(wrapped fuseutil.FileSystem) fuseutil.FileSystem {
	return &errorMapping{
		wrapped: wrapped,
	}
}

type errorMapping struct {
	wrapped fuseutil.FileSystem
}

func (em *errorMapping) handlePanic() {
	// detect if panic occurred or not
	a := recover()
	if a != nil {
		logger.Fatal("Panic: %v", a)
	}
}

func (em *errorMapping) mapError(op string, err error) error {
	fsErr := errno(err)
	if err != nil && fsErr != nil && err != fsErr {
		logger.Errorf("%s: %v, %v", op, fsErr, err)
	}
	return fsErr
}

func (em *errorMapping) Destroy() {
	defer em.handlePanic()

	em.wrapped.Destroy()
}

func (em *errorMapping) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	defer em.handlePanic()

	err := em.wrapped.StatFS(ctx, op)
	return em.mapError("StatFS", err)
}

func (em *errorMapping) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	defer em.handlePanic()

	err := em.wrapped.LookUpInode(ctx, op)
	return em.mapError("LookUpInode", err)
}

func (em *errorMapping) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	defer em.handlePanic()

	err := em.wrapped.GetInodeAttributes(ctx, op)
	return em.mapError("GetInodeAttributes", err)
}

func (em *errorMapping) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	defer em.handlePanic()

	err := em.wrapped.SetInodeAttributes(ctx, op)
	return em.mapError("SetInodeAttributes", err)
}

func (em *errorMapping) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	defer em.handlePanic()

	err := em.wrapped.ForgetInode(ctx, op)
	return em.mapError("ForgetInode", err)
}

func (em *errorMapping) BatchForget(
	ctx context.Context,
	op *fuseops.BatchForgetOp) error {
	defer em.handlePanic()

	err := em.wrapped.BatchForget(ctx, op)
	return em.mapError("BatchForget", err)
}

func (em *errorMapping) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	defer em.handlePanic()

	err := em.wrapped.MkDir(ctx, op)
	return em.mapError("MkDir", err)
}

func (em *errorMapping) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	defer em.handlePanic()

	err := em.wrapped.MkNode(ctx, op)
	return em.mapError("MkNode", err)
}

func (em *errorMapping) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	defer em.handlePanic()

	err := em.wrapped.CreateFile(ctx, op)
	return em.mapError("CreateFile", err)
}

func (em *errorMapping) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	defer em.handlePanic()

	err := em.wrapped.CreateLink(ctx, op)
	return em.mapError("CreateLink", err)
}

func (em *errorMapping) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	defer em.handlePanic()

	err := em.wrapped.CreateSymlink(ctx, op)
	return em.mapError("CreateSymlink", err)
}

func (em *errorMapping) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	defer em.handlePanic()

	err := em.wrapped.Rename(ctx, op)
	return em.mapError("Rename", err)
}

func (em *errorMapping) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	defer em.handlePanic()

	err := em.wrapped.RmDir(ctx, op)
	return em.mapError("RmDir", err)
}

func (em *errorMapping) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	defer em.handlePanic()

	err := em.wrapped.Unlink(ctx, op)
	return em.mapError("Unlink", err)
}

func (em *errorMapping) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	defer em.handlePanic()

	err := em.wrapped.OpenDir(ctx, op)
	return em.mapError("OpenDir", err)
}

func (em *errorMapping) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	defer em.handlePanic()

	err := em.wrapped.ReadDir(ctx, op)
	return em.mapError("ReadDir", err)
}

func (em *errorMapping) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	defer em.handlePanic()

	err := em.wrapped.ReleaseDirHandle(ctx, op)
	return em.mapError("ReleaseDirHandle", err)
}

func (em *errorMapping) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	defer em.handlePanic()

	err := em.wrapped.OpenFile(ctx, op)
	return em.mapError("OpenFile", err)
}

func (em *errorMapping) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	defer em.handlePanic()

	err := em.wrapped.ReadFile(ctx, op)
	return em.mapError("ReadFile", err)
}

func (em *errorMapping) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	defer em.handlePanic()

	err := em.wrapped.WriteFile(ctx, op)
	return em.mapError("WriteFile", err)
}

func (em *errorMapping) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	defer em.handlePanic()

	err := em.wrapped.SyncFile(ctx, op)
	return em.mapError("SyncFile", err)
}

func (em *errorMapping) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	defer em.handlePanic()

	err := em.wrapped.FlushFile(ctx, op)
	return em.mapError("FlushFile", err)
}

func (em *errorMapping) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	defer em.handlePanic()

	err := em.wrapped.ReleaseFileHandle(ctx, op)
	return em.mapError("ReleaseFileHandle", err)
}

func (em *errorMapping) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	defer em.handlePanic()

	err := em.wrapped.ReadSymlink(ctx, op)
	return em.mapError("ReadSymlink", err)
}

func (em *errorMapping) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	defer em.handlePanic()

	err := em.wrapped.RemoveXattr(ctx, op)
	return em.mapError("RemoveXattr", err)
}

func (em *errorMapping) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	defer em.handlePanic()

	err := em.wrapped.GetXattr(ctx, op)
	return em.mapError("GetXattr", err)
}

func (em *errorMapping) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	defer em.handlePanic()

	err := em.wrapped.ListXattr(ctx, op)
	return em.mapError("ListXattr", err)
}

func (em *errorMapping) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	defer em.handlePanic()

	err := em.wrapped.SetXattr(ctx, op)
	return em.mapError("SetXattr", err)
}

func (em *errorMapping) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	defer em.handlePanic()

	err := em.wrapped.Fallocate(ctx, op)
	return em.mapError("Fallocate", err)
}
