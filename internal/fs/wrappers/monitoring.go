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
	"fmt"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	methodName = tag.MustNewKey("method_name")
)

var (
	requestCount = stats.Int64(
		"fs_requests",
		"Number of file system requests.",
		stats.UnitDimensionless)
	errorCount = stats.Int64(
		"fs_errors",
		"Number of file system errors.",
		stats.UnitDimensionless)
)

// Initialize the metrics.
func init() {
	if err := view.Register(
		&view.View{
			Name:        requestCount.Name(),
			Measure:     requestCount,
			Description: requestCount.Description(),
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{methodName},
		},
		&view.View{
			Name:        errorCount.Name(),
			Measure:     errorCount,
			Description: errorCount.Description(),
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{methodName},
		}); err != nil {
		fmt.Printf("Failed to register metrics in the monitoring bucket\n")
	}
}

func incrementCounterFsRequests(method string) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(methodName, method),
		},
		requestCount.M(1),
	)
}
func incrementCounterFsErrors(method string, err error) {
	if err == nil {
		return
	}
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(methodName, method),
		},
		errorCount.M(1),
	)
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
	incrementCounterFsRequests("Destroy")
	fs.wrapped.Destroy()
}

func (fs *monitoring) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	incrementCounterFsRequests("StatFS")
	err := fs.wrapped.StatFS(ctx, op)
	incrementCounterFsErrors("StatFS", err)
	return err
}

func (fs *monitoring) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	incrementCounterFsRequests("LookUpInode")
	err := fs.wrapped.LookUpInode(ctx, op)
	incrementCounterFsErrors("LookUpInode", err)
	return err
}

func (fs *monitoring) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	incrementCounterFsRequests("GetInodeAttributes")
	err := fs.wrapped.GetInodeAttributes(ctx, op)
	incrementCounterFsErrors("GetInodeAttributes", err)
	return err
}

func (fs *monitoring) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	incrementCounterFsRequests("SetInodeAttributes")
	err := fs.wrapped.SetInodeAttributes(ctx, op)
	incrementCounterFsErrors("SetInodeAttributes", err)
	return err
}

func (fs *monitoring) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	incrementCounterFsRequests("ForgetInode")
	err := fs.wrapped.ForgetInode(ctx, op)
	incrementCounterFsErrors("ForgetInode", err)
	return err
}

func (fs *monitoring) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	incrementCounterFsRequests("MkDir")
	err := fs.wrapped.MkDir(ctx, op)
	incrementCounterFsErrors("MkDir", err)
	return err
}

func (fs *monitoring) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	incrementCounterFsRequests("MkNode")
	err := fs.wrapped.MkNode(ctx, op)
	incrementCounterFsErrors("MkNode", err)
	return err
}

func (fs *monitoring) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	incrementCounterFsRequests("CreateFile")
	err := fs.wrapped.CreateFile(ctx, op)
	incrementCounterFsErrors("CreateFile", err)
	return err
}

func (fs *monitoring) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) error {
	incrementCounterFsRequests("CreateLink")
	err := fs.wrapped.CreateLink(ctx, op)
	incrementCounterFsErrors("CreateLink", err)
	return err
}

func (fs *monitoring) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	incrementCounterFsRequests("CreateSymlink")
	err := fs.wrapped.CreateSymlink(ctx, op)
	incrementCounterFsErrors("CreateSymlink", err)
	return err
}

func (fs *monitoring) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	incrementCounterFsRequests("Rename")
	err := fs.wrapped.Rename(ctx, op)
	incrementCounterFsErrors("Rename", err)
	return err
}

func (fs *monitoring) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	incrementCounterFsRequests("RmDir")
	err := fs.wrapped.RmDir(ctx, op)
	incrementCounterFsErrors("RmDir", err)
	return err
}

func (fs *monitoring) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	incrementCounterFsRequests("Unlink")
	err := fs.wrapped.Unlink(ctx, op)
	incrementCounterFsErrors("Unlink", err)
	return err
}

func (fs *monitoring) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	incrementCounterFsRequests("OpenDir")
	err := fs.wrapped.OpenDir(ctx, op)
	incrementCounterFsErrors("OpenDir", err)
	return err
}

func (fs *monitoring) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	incrementCounterFsRequests("ReadDir")
	err := fs.wrapped.ReadDir(ctx, op)
	incrementCounterFsErrors("ReadDir", err)
	return err
}

func (fs *monitoring) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	incrementCounterFsRequests("ReleaseDirHandle")
	err := fs.wrapped.ReleaseDirHandle(ctx, op)
	incrementCounterFsErrors("ReleaseDirHandle", err)
	return err
}

func (fs *monitoring) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	incrementCounterFsRequests("OpenFile")
	err := fs.wrapped.OpenFile(ctx, op)
	incrementCounterFsErrors("OpenFile", err)
	return err
}

func (fs *monitoring) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	incrementCounterFsRequests("ReadFile")
	err := fs.wrapped.ReadFile(ctx, op)
	incrementCounterFsErrors("ReadFile", err)
	return err
}

func (fs *monitoring) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	incrementCounterFsRequests("WriteFile")
	err := fs.wrapped.WriteFile(ctx, op)
	incrementCounterFsErrors("WriteFile", err)
	return err
}

func (fs *monitoring) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	incrementCounterFsRequests("SyncFile")
	err := fs.wrapped.SyncFile(ctx, op)
	incrementCounterFsErrors("SyncFile", err)
	return err
}

func (fs *monitoring) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	incrementCounterFsRequests("FlushFile")
	err := fs.wrapped.FlushFile(ctx, op)
	incrementCounterFsErrors("FlushFile", err)
	return err
}

func (fs *monitoring) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	incrementCounterFsRequests("ReleaseFileHandle")
	err := fs.wrapped.ReleaseFileHandle(ctx, op)
	incrementCounterFsErrors("ReleaseFileHandle", err)
	return err
}

func (fs *monitoring) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	incrementCounterFsRequests("ReadSymlink")
	err := fs.wrapped.ReadSymlink(ctx, op)
	incrementCounterFsErrors("ReadSymlink", err)
	return err
}

func (fs *monitoring) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	incrementCounterFsRequests("RemoveXattr")
	err := fs.wrapped.RemoveXattr(ctx, op)
	incrementCounterFsErrors("RemoveXattr", err)
	return err
}

func (fs *monitoring) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	incrementCounterFsRequests("GetXattr")
	err := fs.wrapped.GetXattr(ctx, op)
	incrementCounterFsErrors("GetXattr", err)
	return err
}

func (fs *monitoring) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	incrementCounterFsRequests("ListXattr")
	err := fs.wrapped.ListXattr(ctx, op)
	incrementCounterFsErrors("ListXattr", err)
	return err
}

func (fs *monitoring) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	incrementCounterFsRequests("SetXattr")
	err := fs.wrapped.SetXattr(ctx, op)
	incrementCounterFsErrors("SetXattr", err)
	return err
}

func (fs *monitoring) Fallocate(
	ctx context.Context,
	op *fuseops.FallocateOp) error {
	incrementCounterFsRequests("Fallocate")
	err := fs.wrapped.Fallocate(ctx, op)
	incrementCounterFsErrors("Fallocate", err)
	return err
}
