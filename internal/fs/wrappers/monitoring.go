// Copyright 2020 Google LLC
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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor/tags"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TODO: name is subject to change.
const name = "cloud.google.com/gcsfuse"

var (
	opsCount      = stats.Int64("fs/ops_count", "The number of ops processed by the file system.", monitor.UnitDimensionless)
	opsLatency    = stats.Int64("fs/ops_latency", "The latency of a file system operation.", monitor.UnitMicroseconds)
	opsErrorCount = stats.Int64("fs/ops_error_count", "The number of errors generated by file system operation.", monitor.UnitDimensionless)

	tracer = otel.Tracer(name)
)

// Initialize the metrics.
func init() {

	// Register the view.
	if err := view.Register(
		&view.View{
			Name:        "fs/ops_count",
			Measure:     opsCount,
			Description: "The cumulative number of ops processed by the file system.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.FSOp},
		},
		&view.View{
			Name:        "fs/ops_error_count",
			Measure:     opsErrorCount,
			Description: "The cumulative number of errors generated by file system operations",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.FSOp, tags.FSErrCategory},
		},
		&view.View{
			Name:        "fs/ops_latency",
			Measure:     opsLatency,
			Description: "The cumulative distribution of file system operation latencies",
			Aggregation: ochttp.DefaultLatencyDistribution,
			TagKeys:     []tag.Key{tags.FSOp},
		}); err != nil {
		fmt.Printf("Failed to register metrics for the file system: %v\n", err)
	}
}

// categorize maps an error to an error-category.
// This helps reduce the cardinality of the labels to less than 30.
// This lower number of errors allows the various errors to get piped to Cloud metrics without getting dropped.
func categorize(err error) string {
	if err == nil {
		return ""
	}
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		errno = DefaultFSError
	}
	switch errno {
	case syscall.ELNRNG,
		syscall.ENODEV,
		syscall.ENONET,
		syscall.ENOSTR,
		syscall.ENOTSOCK,
		syscall.ENXIO,
		syscall.EPROTO,
		syscall.ERFKILL,
		syscall.EXDEV:
		return "device errors"

	case syscall.ENOTEMPTY:
		return "directory not empty"

	case syscall.EEXIST:
		return "file exists"

	case syscall.EBADF,
		syscall.EBADFD,
		syscall.EFBIG,
		syscall.EISDIR,
		syscall.EISNAM,
		syscall.ENOTBLK:
		return "file/directory errors"

	case syscall.ENOSYS:
		return "function not implemented"

	case syscall.EIO:
		return "input/output error"

	case syscall.ECANCELED,
		syscall.EINTR:
		return "interrupt errors"

	case syscall.EINVAL:
		return "invalid argument"

	case syscall.E2BIG,
		syscall.EALREADY,
		syscall.EBADE,
		syscall.EBADR,
		syscall.EDOM,
		syscall.EINPROGRESS,
		syscall.ENOEXEC,
		syscall.ENOTSUP,
		syscall.ENOTTY,
		syscall.ERANGE,
		syscall.ESPIPE:
		return "invalid operation"

	case syscall.EADV,
		syscall.EBADSLT,
		syscall.EBFONT,
		syscall.ECHRNG,
		syscall.EDOTDOT,
		syscall.EIDRM,
		syscall.EILSEQ,
		syscall.ELIBACC,
		syscall.ELIBBAD,
		syscall.ELIBEXEC,
		syscall.ELIBMAX,
		syscall.ELIBSCN,
		syscall.EMEDIUMTYPE,
		syscall.ENAVAIL,
		syscall.ENOANO,
		syscall.ENOCSI,
		syscall.ENODATA,
		syscall.ENOMEDIUM,
		syscall.ENOMSG,
		syscall.ENOPKG,
		syscall.ENOSR,
		syscall.ENOTNAM,
		syscall.ENOTRECOVERABLE,
		syscall.EOVERFLOW,
		syscall.ERESTART,
		syscall.ESRMNT,
		syscall.ESTALE,
		syscall.ETIME,
		syscall.ETOOMANYREFS,
		syscall.EUCLEAN,
		syscall.EUNATCH,
		syscall.EXFULL:
		return "miscellaneous errors"

	case syscall.EADDRINUSE,
		syscall.EADDRNOTAVAIL,
		syscall.EAFNOSUPPORT,
		syscall.EBADMSG,
		syscall.EBADRQC,
		syscall.ECOMM,
		syscall.ECONNABORTED,
		syscall.ECONNREFUSED,
		syscall.ECONNRESET,
		syscall.EDESTADDRREQ,
		syscall.EFAULT,
		syscall.EHOSTDOWN,
		syscall.EHOSTUNREACH,
		syscall.EISCONN,
		syscall.EL2HLT,
		syscall.EL2NSYNC,
		syscall.EL3HLT,
		syscall.EL3RST,
		syscall.EMSGSIZE,
		syscall.EMULTIHOP,
		syscall.ENETDOWN,
		syscall.ENETRESET,
		syscall.ENETUNREACH,
		syscall.ENOLINK,
		syscall.ENOPROTOOPT,
		syscall.ENOTCONN,
		syscall.ENOTUNIQ,
		syscall.EPFNOSUPPORT,
		syscall.EPIPE,
		syscall.EPROTONOSUPPORT,
		syscall.EPROTOTYPE,
		syscall.EREMCHG,
		syscall.EREMOTE,
		syscall.EREMOTEIO,
		syscall.ESHUTDOWN,
		syscall.ESOCKTNOSUPPORT,
		syscall.ESTRPIPE,
		syscall.ETIMEDOUT:
		return "network errors"

	case syscall.ENOENT:
		return "no such file or directory"

	case syscall.ENOTDIR:
		return "not a directory"

	case syscall.EACCES,
		syscall.EKEYEXPIRED,
		syscall.EKEYREJECTED,
		syscall.EKEYREVOKED,
		syscall.ENOKEY,
		syscall.EPERM,
		syscall.EROFS,
		syscall.ETXTBSY:
		return "permission errors"

	case syscall.EAGAIN,
		syscall.EBUSY,
		syscall.ECHILD,
		syscall.EDEADLK,
		syscall.EDQUOT,
		syscall.ELOOP,
		syscall.EMLINK,
		syscall.ENAMETOOLONG,
		syscall.ENOBUFS,
		syscall.ENOLCK,
		syscall.ENOMEM,
		syscall.ENOSPC,
		syscall.EOWNERDEAD,
		syscall.ESRCH,
		syscall.EUSERS:
		return "process/resource management errors"

	case syscall.EMFILE,
		syscall.ENFILE:
		return "too many open files"
	}
	return "miscellaneous errors"
}

// Records file system operation count, failed operation count and the operation latency.
func recordOp(ctx context.Context, method string, start time.Time, fsErr error) {
	// Recording opCount.
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.FSOp, method),
		},
		opsCount.M(1),
	); err != nil {
		// Error in recording opCount.
		logger.Errorf("Cannot record file system op: %v", err)
	}

	// Recording opErrorCount.
	if fsErr != nil {
		errCategory := categorize(fsErr)
		if err := stats.RecordWithTags(
			ctx,
			[]tag.Mutator{
				tag.Upsert(tags.FSOp, method),
				tag.Upsert(tags.FSErrCategory, errCategory),
			},
			opsErrorCount.M(1),
		); err != nil {
			// Error in recording opErrorCount.
			logger.Errorf("Cannot record error count of the file system failed operations: %v", err)
		}
	}

	// Recording opLatency.
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.FSOp, method),
		},
		opsLatency.M(time.Since(start).Microseconds()),
	); err != nil {
		// Error in opLatency.
		logger.Errorf("Cannot record file system operation latency: %v", err)
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

type wrappedCall func() error

func invokeWrapped(ctx context.Context, opName string, w wrappedCall) error {
	ctx, span := tracer.Start(ctx, opName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	startTime := time.Now()
	err := w()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	recordOp(ctx, opName, startTime, err)
	return err
}

func (fs *monitoring) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return invokeWrapped(ctx, "StatFS", func() error { return fs.wrapped.StatFS(ctx, op) })
}

func (fs *monitoring) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	return invokeWrapped(ctx, "LookUpInode", func() error { return fs.wrapped.LookUpInode(ctx, op) })
}

func (fs *monitoring) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	return invokeWrapped(ctx, "GetInodeAttributes", func() error { return fs.wrapped.GetInodeAttributes(ctx, op) })
}

func (fs *monitoring) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return invokeWrapped(ctx, "SetInodeAttributes", func() error { return fs.wrapped.SetInodeAttributes(ctx, op) })
}

func (fs *monitoring) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	return invokeWrapped(ctx, "ForgetInode", func() error { return fs.wrapped.ForgetInode(ctx, op) })
}

func (fs *monitoring) BatchForget(ctx context.Context, op *fuseops.BatchForgetOp) error {
	return invokeWrapped(ctx, "BatchForget", func() error { return fs.wrapped.BatchForget(ctx, op) })
}

func (fs *monitoring) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return invokeWrapped(ctx, "MkDir", func() error { return fs.wrapped.MkDir(ctx, op) })
}

func (fs *monitoring) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	return invokeWrapped(ctx, "MkNode", func() error { return fs.wrapped.MkNode(ctx, op) })
}

func (fs *monitoring) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) error {
	return invokeWrapped(ctx, "CreateFile", func() error { return fs.wrapped.CreateFile(ctx, op) })
}

func (fs *monitoring) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return invokeWrapped(ctx, "CreateLink", func() error { return fs.wrapped.CreateLink(ctx, op) })
}

func (fs *monitoring) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return invokeWrapped(ctx, "CreateSymlink", func() error { return fs.wrapped.CreateSymlink(ctx, op) })
}

func (fs *monitoring) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	return invokeWrapped(ctx, "Rename", func() error { return fs.wrapped.Rename(ctx, op) })
}

func (fs *monitoring) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return invokeWrapped(ctx, "RmDir", func() error { return fs.wrapped.RmDir(ctx, op) })
}

func (fs *monitoring) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	return invokeWrapped(ctx, "Unlink", func() error { return fs.wrapped.Unlink(ctx, op) })
}

func (fs *monitoring) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return invokeWrapped(ctx, "OpenDir", func() error { return fs.wrapped.OpenDir(ctx, op) })
}

func (fs *monitoring) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	return invokeWrapped(ctx, "ReadDir", func() error { return fs.wrapped.ReadDir(ctx, op) })
}

func (fs *monitoring) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	return invokeWrapped(ctx, "ReleaseDirHandle", func() error { return fs.wrapped.ReleaseDirHandle(ctx, op) })
}

func (fs *monitoring) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return invokeWrapped(ctx, "OpenFile", func() error { return fs.wrapped.OpenFile(ctx, op) })
}

func (fs *monitoring) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	return invokeWrapped(ctx, "ReadFile", func() error { return fs.wrapped.ReadFile(ctx, op) })
}

func (fs *monitoring) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return invokeWrapped(ctx, "WriteFile", func() error { return fs.wrapped.WriteFile(ctx, op) })
}

func (fs *monitoring) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) error {
	return invokeWrapped(ctx, "SyncFile", func() error { return fs.wrapped.SyncFile(ctx, op) })
}

func (fs *monitoring) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) error {
	return invokeWrapped(ctx, "FlushFile", func() error { return fs.wrapped.FlushFile(ctx, op) })
}

func (fs *monitoring) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	return invokeWrapped(ctx, "ReleaseFileHandle", func() error { return fs.wrapped.ReleaseFileHandle(ctx, op) })
}

func (fs *monitoring) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return invokeWrapped(ctx, "ReadSymlink", func() error { return fs.wrapped.ReadSymlink(ctx, op) })
}

func (fs *monitoring) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return invokeWrapped(ctx, "RemoveXattr", func() error { return fs.wrapped.RemoveXattr(ctx, op) })
}

func (fs *monitoring) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	return invokeWrapped(ctx, "GetXattr", func() error { return fs.wrapped.GetXattr(ctx, op) })
}

func (fs *monitoring) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	return invokeWrapped(ctx, "ListXattr", func() error { return fs.wrapped.ListXattr(ctx, op) })
}

func (fs *monitoring) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return invokeWrapped(ctx, "SetXattr", func() error { return fs.wrapped.SetXattr(ctx, op) })
}

func (fs *monitoring) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return invokeWrapped(ctx, "Fallocate", func() error { return fs.wrapped.Fallocate(ctx, op) })
}
