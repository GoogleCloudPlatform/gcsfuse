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
	counterFsErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcsfuse_fs_errors",
			Help: "Number of errors per file system API.",
		},
		[]string{ // labels
			"method",
		},
	)
	latencyOpenFile = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "gcsfuse_fs_open_file_latency",
			Help: "The latency of executing an OpenFile request in ms.",

			// 32 buckets: [0.1ms, 0.15ms, ..., 28.8s, +Inf]
			Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 32),
		},
	)
	latencyReadFile = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "gcsfuse_fs_read_file_latency",
			Help: "The latency of executing a ReadFile request in ms.",

			// 32 buckets: [0.1ms, 0.15ms, ..., 28.8s, +Inf]
			Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 32),
		},
	)
)

// Initialize the prometheus metrics.
func init() {
	prometheus.MustRegister(counterFsRequests)
	prometheus.MustRegister(counterFsErrors)
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
func incrementCounterFsErrors(method string, err error) {
	if err != nil {
		counterFsErrors.With(
			prometheus.Labels{
				"method": method,
			},
		).Inc()
	}
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
	err := fs.wrapped.StatFS(ctx, op)
	incrementCounterFsErrors("StatFS", err)
	return err
}

func (fs *monitoringFileSystem) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	incrementCounterFsRequests("LookUpInode")
	err := fs.wrapped.LookUpInode(ctx, op)
	incrementCounterFsErrors("LookUpInode", err)
	return err
}

func (fs *monitoringFileSystem) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	incrementCounterFsRequests("GetInodeAttributes")
	err := fs.wrapped.GetInodeAttributes(ctx, op)
	incrementCounterFsErrors("GetInodeAttributes", err)
	return err
}

func (fs *monitoringFileSystem) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	incrementCounterFsRequests("SetInodeAttributes")
	err := fs.wrapped.SetInodeAttributes(ctx, op)
	incrementCounterFsErrors("SetInodeAttributes", err)
	return err
}

func (fs *monitoringFileSystem) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) error {
	incrementCounterFsRequests("ForgetInode")
	err := fs.wrapped.ForgetInode(ctx, op)
	incrementCounterFsErrors("ForgetInode", err)
	return err
}

func (fs *monitoringFileSystem) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) error {
	incrementCounterFsRequests("MkDir")
	err := fs.wrapped.MkDir(ctx, op)
	incrementCounterFsErrors("MkDir", err)
	return err
}

func (fs *monitoringFileSystem) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) error {
	incrementCounterFsRequests("MkNode")
	err := fs.wrapped.MkNode(ctx, op)
	incrementCounterFsErrors("MkNode", err)
	return err
}

func (fs *monitoringFileSystem) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) error {
	incrementCounterFsRequests("CreateFile")
	err := fs.wrapped.CreateFile(ctx, op)
	incrementCounterFsErrors("CreateFile", err)
	return err
}

func (fs *monitoringFileSystem) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) error {
	incrementCounterFsRequests("CreateSymlink")
	err := fs.wrapped.CreateSymlink(ctx, op)
	incrementCounterFsErrors("CreateSymlink", err)
	return err
}

func (fs *monitoringFileSystem) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) error {
	incrementCounterFsRequests("Rename")
	err := fs.wrapped.Rename(ctx, op)
	incrementCounterFsErrors("Rename", err)
	return err
}

func (fs *monitoringFileSystem) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) error {
	incrementCounterFsRequests("RmDir")
	err := fs.wrapped.RmDir(ctx, op)
	incrementCounterFsErrors("RmDir", err)
	return err
}

func (fs *monitoringFileSystem) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) error {
	incrementCounterFsRequests("Unlink")
	err := fs.wrapped.Unlink(ctx, op)
	incrementCounterFsErrors("Unlink", err)
	return err
}

func (fs *monitoringFileSystem) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	incrementCounterFsRequests("OpenDir")
	err := fs.wrapped.OpenDir(ctx, op)
	incrementCounterFsErrors("OpenDir", err)
	return err
}

func (fs *monitoringFileSystem) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	incrementCounterFsRequests("ReadDir")
	err := fs.wrapped.ReadDir(ctx, op)
	incrementCounterFsErrors("ReadDir", err)
	return err
}

func (fs *monitoringFileSystem) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) error {
	incrementCounterFsRequests("ReleaseDirHandle")
	err := fs.wrapped.ReleaseDirHandle(ctx, op)
	incrementCounterFsErrors("ReleaseDirHandle", err)
	return err
}

func (fs *monitoringFileSystem) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	incrementCounterFsRequests("OpenFile")
	defer recordLatency(latencyOpenFile, time.Now())
	err := fs.wrapped.OpenFile(ctx, op)
	incrementCounterFsErrors("OpenFile", err)
	return err
}

func (fs *monitoringFileSystem) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	incrementCounterFsRequests("ReadFile")
	defer recordLatency(latencyReadFile, time.Now())
	err := fs.wrapped.ReadFile(ctx, op)
	incrementCounterFsErrors("ReadFile", err)
	return err
}

func (fs *monitoringFileSystem) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) error {
	incrementCounterFsRequests("WriteFile")
	err := fs.wrapped.WriteFile(ctx, op)
	incrementCounterFsErrors("WriteFile", err)
	return err
}

func (fs *monitoringFileSystem) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) error {
	incrementCounterFsRequests("SyncFile")
	err := fs.wrapped.SyncFile(ctx, op)
	incrementCounterFsErrors("SyncFile", err)
	return err
}

func (fs *monitoringFileSystem) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) error {
	incrementCounterFsRequests("FlushFile")
	err := fs.wrapped.FlushFile(ctx, op)
	incrementCounterFsErrors("FlushFile", err)
	return err
}

func (fs *monitoringFileSystem) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) error {
	incrementCounterFsRequests("ReleaseFileHandle")
	err := fs.wrapped.ReleaseFileHandle(ctx, op)
	incrementCounterFsErrors("ReleaseFileHandle", err)
	return err
}

func (fs *monitoringFileSystem) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) error {
	incrementCounterFsRequests("ReadSymlink")
	err := fs.wrapped.ReadSymlink(ctx, op)
	incrementCounterFsErrors("ReadSymlink", err)
	return err
}

func (fs *monitoringFileSystem) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) error {
	incrementCounterFsRequests("RemoveXattr")
	err := fs.wrapped.RemoveXattr(ctx, op)
	incrementCounterFsErrors("RemoveXattr", err)
	return err
}

func (fs *monitoringFileSystem) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	incrementCounterFsRequests("GetXattr")
	err := fs.wrapped.GetXattr(ctx, op)
	incrementCounterFsErrors("GetXattr", err)
	return err
}

func (fs *monitoringFileSystem) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	incrementCounterFsRequests("ListXattr")
	err := fs.wrapped.ListXattr(ctx, op)
	incrementCounterFsErrors("ListXattr", err)
	return err
}

func (fs *monitoringFileSystem) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) error {
	incrementCounterFsRequests("SetXattr")
	err := fs.wrapped.SetXattr(ctx, op)
	incrementCounterFsErrors("SetXattr", err)
	return err
}
