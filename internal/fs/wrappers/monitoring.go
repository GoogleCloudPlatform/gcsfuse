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
	"fmt"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/internal/monitor/tags"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	opCount = stats.Int64("fs/op_count", "The number of ops processed by the file system.", stats.UnitDimensionless)
)

// Initialize the metrics.
func init() {
	if err := view.Register(
		&view.View{
			Name:        "fs/op_count",
			Measure:     opCount,
			Description: "The cumulative number of ops processed by the file system.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.FSOp, tags.FSError},
		},
	); err != nil {
		fmt.Printf("Failed to register metrics for the file system: %v\n", err)
	}
}

// fsErrStr maps an error to a error string. Uncommon errors are aggregated to
// reduce the cardinality of the fs error to save the monitoring cost.
func fsErrStr(err error) string {
	if err == nil {
		return ""
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno.Error()
	}
	return DefaultFSError.Error()
}

func recordOp(ctx context.Context, method string, fsErr error) {

	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.FSOp, method),
			tag.Upsert(tags.FSError, fsErrStr(fsErr)),
		},
		opCount.M(1),
	); err != nil {
		fmt.Printf("Cannot record file system op: %v", err)
	}
}

// WithMonitoring takes a FileSystem, returns a FileSystem with monitoring
// on the counts of requests per API.
func WithMonitoring(fs fuseutil.FileSystem) fuseutil.FileSystem {
	return &monitoring{
		wrapped: fs,
	}
}

type monitoring struct {
	wrapped fuseutil.FileSystem
}

func (fs *monitoring) Destroy() {
	fs.wrapped.Destroy()
}

func (fs *monitoring) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	err := fs.wrapped.StatFS(ctx, op)
	recordOp(ctx, "StatFS", err)
	return err
}

func (fs *monitoring) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	err := fs.wrapped.LookUpInode(ctx, op)
	recordOp(ctx, "LookUpInode", err)
	return err
}

func (fs *monitoring) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	err := fs.wrapped.GetInodeAttributes(ctx, op)
	recordOp(ctx, "GetInodeAttributes", err)
	return err
}

func (fs *monitoring) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	err := fs.wrapped.SetInodeAttributes(ctx, op)
	recordOp(ctx, "SetInodeAttributes", err)
	return err
}

func (fs *monitoring) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	err := fs.wrapped.ForgetInode(ctx, op)
	recordOp(ctx, "ForgetInode", err)
	return err
}

func (fs *monitoring) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	err := fs.wrapped.MkDir(ctx, op)
	recordOp(ctx, "MkDir", err)
	return err
}

func (fs *monitoring) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	err := fs.wrapped.MkNode(ctx, op)
	recordOp(ctx, "MkNode", err)
	return err
}

func (fs *monitoring) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	err := fs.wrapped.CreateFile(ctx, op)
	recordOp(ctx, "CreateFile", err)
	return err
}

func (fs *monitoring) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	err := fs.wrapped.CreateLink(ctx, op)
	recordOp(ctx, "CreateLink", err)
	return err
}

func (fs *monitoring) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	err := fs.wrapped.CreateSymlink(ctx, op)
	recordOp(ctx, "CreateSymlink", err)
	return err
}

func (fs *monitoring) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	err := fs.wrapped.Rename(ctx, op)
	recordOp(ctx, "Rename", err)
	return err
}

func (fs *monitoring) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	err := fs.wrapped.RmDir(ctx, op)
	recordOp(ctx, "RmDir", err)
	return err
}

func (fs *monitoring) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	err := fs.wrapped.Unlink(ctx, op)
	recordOp(ctx, "Unlink", err)
	return err
}

func (fs *monitoring) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	err := fs.wrapped.OpenDir(ctx, op)
	recordOp(ctx, "OpenDir", err)
	return err
}

func (fs *monitoring) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	err := fs.wrapped.ReadDir(ctx, op)
	recordOp(ctx, "ReadDir", err)
	return err
}

func (fs *monitoring) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	err := fs.wrapped.ReleaseDirHandle(ctx, op)
	recordOp(ctx, "ReleaseDirHandle", err)
	return err
}

func (fs *monitoring) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	err := fs.wrapped.OpenFile(ctx, op)
	recordOp(ctx, "OpenFile", err)
	return err
}

func (fs *monitoring) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	err := fs.wrapped.ReadFile(ctx, op)
	recordOp(ctx, "ReadFile", err)
	return err
}

func (fs *monitoring) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	err := fs.wrapped.WriteFile(ctx, op)
	recordOp(ctx, "WriteFile", err)
	return err
}

func (fs *monitoring) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	err := fs.wrapped.SyncFile(ctx, op)
	recordOp(ctx, "SyncFile", err)
	return err
}

func (fs *monitoring) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	err := fs.wrapped.FlushFile(ctx, op)
	recordOp(ctx, "FlushFile", err)
	return err
}

func (fs *monitoring) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	err := fs.wrapped.ReleaseFileHandle(ctx, op)
	recordOp(ctx, "ReleaseFileHandle", err)
	return err
}

func (fs *monitoring) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	err := fs.wrapped.ReadSymlink(ctx, op)
	recordOp(ctx, "ReadSymlink", err)
	return err
}

func (fs *monitoring) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	err := fs.wrapped.RemoveXattr(ctx, op)
	recordOp(ctx, "RemoveXattr", err)
	return err
}

func (fs *monitoring) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	err := fs.wrapped.GetXattr(ctx, op)
	recordOp(ctx, "GetXattr", err)
	return err
}

func (fs *monitoring) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	err := fs.wrapped.ListXattr(ctx, op)
	recordOp(ctx, "ListXattr", err)
	return err
}

func (fs *monitoring) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	err := fs.wrapped.SetXattr(ctx, op)
	recordOp(ctx, "SetXattr", err)
	return err
}

func (fs *monitoring) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	err := fs.wrapped.Fallocate(ctx, op)
	recordOp(ctx, "Fallocate", err)
	return err
}
