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

package fs

import (
	"context"
	"time"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	counterFsRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcsfuse_fs_requests",
			Help: "Number of requests per file system API.",
		},
		[]string{ // labels
			"method",
		},
	)
	latencyOpenFile = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "gcsfuse_fs_open_file_latency",
			Help: "The latency of OpenFile file system requests in ms.",
		},
	)
	latencyReadFile = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "gcsfuse_fs_read_file_latency",
			Help: "The latency of ReadFile file system requests in ms.",
		},
	)
)

// Initialize the prometheus metrics.
func init() {
	prometheus.MustRegister(counterFsRequests)
	prometheus.MustRegister(latencyOpenFile)
	prometheus.MustRegister(latencyReadFile)
}

func incrementCounterFsRequests(method string) {
	counterFsRequests.With(
		prometheus.Labels{
			"method": method,
		},
	).Inc()
}

func recordLatency(metric prometheus.Histogram, start time.Time) {
	latency := float64(time.Since(start).Milliseconds())
	metric.Observe(latency)
}

// WithMonitoring takes a FileSystem, returns a FileSystem with monitoring
// on the counts of requests per API.
func WithMonitoring(fs fuseutil.FileSystem) fuseutil.FileSystem {
	return &monitoringFileSystem{
		wrapped: fs,
	}
}

type monitoringFileSystem struct {
	wrapped fuseutil.FileSystem
}

func (fs *monitoringFileSystem) Destroy() {
	incrementCounterFsRequests("Destroy")
	fs.wrapped.Destroy()
}

func (fs *monitoringFileSystem) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	incrementCounterFsRequests("StatFS")
	return fs.wrapped.StatFS(ctx, op)
}

func (fs *monitoringFileSystem) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	incrementCounterFsRequests("LookUpInode")
	return fs.wrapped.LookUpInode(ctx, op)
}

func (fs *monitoringFileSystem) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	incrementCounterFsRequests("GetInodeAttributes")
	return fs.wrapped.GetInodeAttributes(ctx, op)
}

func (fs *monitoringFileSystem) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	incrementCounterFsRequests("SetInodeAttributes")
	return fs.wrapped.SetInodeAttributes(ctx, op)
}

func (fs *monitoringFileSystem) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	incrementCounterFsRequests("ForgetInode")
	return fs.wrapped.ForgetInode(ctx, op)
}

func (fs *monitoringFileSystem) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	incrementCounterFsRequests("MkDir")
	return fs.wrapped.MkDir(ctx, op)
}

func (fs *monitoringFileSystem) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	incrementCounterFsRequests("MkNode")
	return fs.wrapped.MkNode(ctx, op)
}

func (fs *monitoringFileSystem) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	incrementCounterFsRequests("CreateFile")
	return fs.wrapped.CreateFile(ctx, op)
}

func (fs *monitoringFileSystem) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	incrementCounterFsRequests("CreateSymlink")
	return fs.wrapped.CreateSymlink(ctx, op)
}

func (fs *monitoringFileSystem) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	incrementCounterFsRequests("Rename")
	return fs.wrapped.Rename(ctx, op)
}

func (fs *monitoringFileSystem) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	incrementCounterFsRequests("RmDir")
	return fs.wrapped.RmDir(ctx, op)
}

func (fs *monitoringFileSystem) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	incrementCounterFsRequests("Unlink")
	return fs.wrapped.Unlink(ctx, op)
}

func (fs *monitoringFileSystem) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	incrementCounterFsRequests("OpenDir")
	return fs.wrapped.OpenDir(ctx, op)
}

func (fs *monitoringFileSystem) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	incrementCounterFsRequests("ReadDir")
	return fs.wrapped.ReadDir(ctx, op)
}

func (fs *monitoringFileSystem) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	incrementCounterFsRequests("ReleaseDirHandle")
	return fs.wrapped.ReleaseDirHandle(ctx, op)
}

func (fs *monitoringFileSystem) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	incrementCounterFsRequests("OpenFile")
	defer recordLatency(latencyOpenFile, time.Now())
	return fs.wrapped.OpenFile(ctx, op)
}

func (fs *monitoringFileSystem) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	incrementCounterFsRequests("ReadFile")
	defer recordLatency(latencyReadFile, time.Now())
	return fs.wrapped.ReadFile(ctx, op)
}

func (fs *monitoringFileSystem) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	incrementCounterFsRequests("WriteFile")
	return fs.wrapped.WriteFile(ctx, op)
}

func (fs *monitoringFileSystem) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	incrementCounterFsRequests("SyncFile")
	return fs.wrapped.SyncFile(ctx, op)
}

func (fs *monitoringFileSystem) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	incrementCounterFsRequests("FlushFile")
	return fs.wrapped.FlushFile(ctx, op)
}

func (fs *monitoringFileSystem) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	incrementCounterFsRequests("ReleaseFileHandle")
	return fs.wrapped.ReleaseFileHandle(ctx, op)
}

func (fs *monitoringFileSystem) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	incrementCounterFsRequests("ReadSymlink")
	return fs.wrapped.ReadSymlink(ctx, op)
}

func (fs *monitoringFileSystem) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	incrementCounterFsRequests("RemoveXattr")
	return fs.wrapped.RemoveXattr(ctx, op)
}

func (fs *monitoringFileSystem) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	incrementCounterFsRequests("GetXattr")
	return fs.wrapped.GetXattr(ctx, op)
}

func (fs *monitoringFileSystem) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	incrementCounterFsRequests("ListXattr")
	return fs.wrapped.ListXattr(ctx, op)
}

func (fs *monitoringFileSystem) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	incrementCounterFsRequests("SetXattr")
	return fs.wrapped.SetXattr(ctx, op)
}
