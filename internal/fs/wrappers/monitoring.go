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
	"syscall"
	"time"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/vipnydav/gcsfuse/v3/metrics"
)

const name = "cloud.google.com/gcsfuse"

// Error categories
const (
	errDevice         = metrics.FsErrorCategoryDEVICEERRORAttr
	errDirNotEmpty    = metrics.FsErrorCategoryDIRNOTEMPTYAttr
	errFileExists     = metrics.FsErrorCategoryFILEEXISTSAttr
	errFileDir        = metrics.FsErrorCategoryFILEDIRERRORAttr
	errNotImplemented = metrics.FsErrorCategoryNOTIMPLEMENTEDAttr
	errIO             = metrics.FsErrorCategoryIOERRORAttr
	errInterrupt      = metrics.FsErrorCategoryINTERRUPTERRORAttr
	errInvalidArg     = metrics.FsErrorCategoryINVALIDARGUMENTAttr
	errInvalidOp      = metrics.FsErrorCategoryINVALIDOPERATIONAttr
	errMisc           = metrics.FsErrorCategoryMISCERRORAttr
	errNetwork        = metrics.FsErrorCategoryNETWORKERRORAttr
	errNoFileOrDir    = metrics.FsErrorCategoryNOFILEORDIRAttr
	errNotADir        = metrics.FsErrorCategoryNOTADIRAttr
	errPerm           = metrics.FsErrorCategoryPERMERRORAttr
	errProcessMgmt    = metrics.FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr
	errTooManyFiles   = metrics.FsErrorCategoryTOOMANYOPENFILESAttr
)

// categorize maps an error to an error-category.
// This helps reduce the cardinality of the labels to less than 30.
// This lower number of errors allows the various errors to get piped to Cloud metrics without getting dropped.
func categorize(err error) metrics.FsErrorCategory {
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
		return errDevice

	case syscall.ENOTEMPTY:
		return errDirNotEmpty

	case syscall.EEXIST:
		return errFileExists

	case syscall.EBADF,
		syscall.EBADFD,
		syscall.EFBIG,
		syscall.EISDIR,
		syscall.EISNAM,
		syscall.ENOTBLK:
		return errFileDir

	case syscall.ENOSYS:
		return errNotImplemented

	case syscall.EIO:
		return errIO

	case syscall.ECANCELED,
		syscall.EINTR:
		return errInterrupt

	case syscall.EINVAL:
		return errInvalidArg

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
		return errInvalidOp

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
		return errMisc

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
		return errNetwork

	case syscall.ENOENT:
		return errNoFileOrDir

	case syscall.ENOTDIR:
		return errNotADir

	case syscall.EACCES,
		syscall.EKEYEXPIRED,
		syscall.EKEYREJECTED,
		syscall.EKEYREVOKED,
		syscall.ENOKEY,
		syscall.EPERM,
		syscall.EROFS,
		syscall.ETXTBSY:
		return errPerm

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
		return errProcessMgmt

	case syscall.EMFILE,
		syscall.ENFILE:
		return errTooManyFiles
	}
	return errMisc
}

// Records file system operation count, failed operation count and the operation latency.
func recordOp(ctx context.Context, metricHandle metrics.MetricHandle, method metrics.FsOp, start time.Time, fsErr error) {
	metricHandle.FsOpsCount(1, method)

	// Recording opErrorCount.
	if fsErr != nil {
		errCategory := categorize(fsErr)
		metricHandle.FsOpsErrorCount(1, errCategory, method)
	}
	metricHandle.FsOpsLatency(ctx, time.Since(start), method)
}

// WithMonitoring takes a FileSystem, returns a FileSystem with monitoring
// on the counts of requests per API.
func WithMonitoring(fs fuseutil.FileSystem, metricHandle metrics.MetricHandle) fuseutil.FileSystem {
	return &monitoring{
		wrapped:      fs,
		metricHandle: metricHandle,
	}
}

type monitoring struct {
	wrapped      fuseutil.FileSystem
	metricHandle metrics.MetricHandle
}

func (fs *monitoring) Destroy() {
	fs.wrapped.Destroy()
}

type wrappedCall func(ctx context.Context) error

func (fs *monitoring) invokeWrapped(ctx context.Context, opName metrics.FsOp, w wrappedCall) error {
	startTime := time.Now()
	err := w(ctx)
	recordOp(ctx, fs.metricHandle, opName, startTime, err)
	return err
}

func (fs *monitoring) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOthersAttr, func(ctx context.Context) error { return fs.wrapped.StatFS(ctx, op) })
}

func (fs *monitoring) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpLookUpInodeAttr, func(ctx context.Context) error { return fs.wrapped.LookUpInode(ctx, op) })
}

func (fs *monitoring) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpGetInodeAttributesAttr, func(ctx context.Context) error { return fs.wrapped.GetInodeAttributes(ctx, op) })
}

func (fs *monitoring) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpSetInodeAttributesAttr, func(ctx context.Context) error { return fs.wrapped.SetInodeAttributes(ctx, op) })
}

func (fs *monitoring) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpForgetInodeAttr, func(ctx context.Context) error { return fs.wrapped.ForgetInode(ctx, op) })
}

func (fs *monitoring) BatchForget(ctx context.Context, op *fuseops.BatchForgetOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpBatchForgetAttr, func(ctx context.Context) error { return fs.wrapped.BatchForget(ctx, op) })
}

func (fs *monitoring) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpMkDirAttr, func(ctx context.Context) error { return fs.wrapped.MkDir(ctx, op) })
}

func (fs *monitoring) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpMkNodeAttr, func(ctx context.Context) error { return fs.wrapped.MkNode(ctx, op) })
}

func (fs *monitoring) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpCreateFileAttr, func(ctx context.Context) error { return fs.wrapped.CreateFile(ctx, op) })
}

func (fs *monitoring) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpCreateLinkAttr, func(ctx context.Context) error { return fs.wrapped.CreateLink(ctx, op) })
}

func (fs *monitoring) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpCreateSymlinkAttr, func(ctx context.Context) error { return fs.wrapped.CreateSymlink(ctx, op) })
}

func (fs *monitoring) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpRenameAttr, func(ctx context.Context) error { return fs.wrapped.Rename(ctx, op) })
}

func (fs *monitoring) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpRmDirAttr, func(ctx context.Context) error { return fs.wrapped.RmDir(ctx, op) })
}

func (fs *monitoring) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpUnlinkAttr, func(ctx context.Context) error { return fs.wrapped.Unlink(ctx, op) })
}

func (fs *monitoring) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOpenDirAttr, func(ctx context.Context) error { return fs.wrapped.OpenDir(ctx, op) })
}

func (fs *monitoring) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpReadDirAttr, func(ctx context.Context) error { return fs.wrapped.ReadDir(ctx, op) })
}

func (fs *monitoring) ReadDirPlus(ctx context.Context, op *fuseops.ReadDirPlusOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpReadDirPlusAttr, func(ctx context.Context) error { return fs.wrapped.ReadDirPlus(ctx, op) })
}

func (fs *monitoring) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpReleaseDirHandleAttr, func(ctx context.Context) error { return fs.wrapped.ReleaseDirHandle(ctx, op) })
}

func (fs *monitoring) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOpenFileAttr, func(ctx context.Context) error { return fs.wrapped.OpenFile(ctx, op) })
}

func (fs *monitoring) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpReadFileAttr, func(ctx context.Context) error { return fs.wrapped.ReadFile(ctx, op) })
}

func (fs *monitoring) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpWriteFileAttr, func(ctx context.Context) error { return fs.wrapped.WriteFile(ctx, op) })
}

func (fs *monitoring) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpSyncFileAttr, func(ctx context.Context) error { return fs.wrapped.SyncFile(ctx, op) })
}

func (fs *monitoring) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpFlushFileAttr, func(ctx context.Context) error { return fs.wrapped.FlushFile(ctx, op) })
}

func (fs *monitoring) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpReleaseFileHandleAttr, func(ctx context.Context) error { return fs.wrapped.ReleaseFileHandle(ctx, op) })
}

func (fs *monitoring) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpReadSymlinkAttr, func(ctx context.Context) error { return fs.wrapped.ReadSymlink(ctx, op) })
}

func (fs *monitoring) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOthersAttr, func(ctx context.Context) error { return fs.wrapped.RemoveXattr(ctx, op) })
}

func (fs *monitoring) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOthersAttr, func(ctx context.Context) error { return fs.wrapped.GetXattr(ctx, op) })
}

func (fs *monitoring) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOthersAttr, func(ctx context.Context) error { return fs.wrapped.ListXattr(ctx, op) })
}

func (fs *monitoring) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOthersAttr, func(ctx context.Context) error { return fs.wrapped.SetXattr(ctx, op) })
}

func (fs *monitoring) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOthersAttr, func(ctx context.Context) error { return fs.wrapped.Fallocate(ctx, op) })
}

func (fs *monitoring) SyncFS(ctx context.Context, op *fuseops.SyncFSOp) error {
	return fs.invokeWrapped(ctx, metrics.FsOpOthersAttr, func(ctx context.Context) error { return fs.wrapped.SyncFS(ctx, op) })
}
