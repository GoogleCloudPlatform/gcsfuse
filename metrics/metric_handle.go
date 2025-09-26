// Copyright 2025 Google LLC
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

// **** DO NOT EDIT - FILE IS AUTO-GENERATED ****
package metrics

import (
	"context"
	"time"
)

// FsErrorCategory is a user-defined type for the FsErrorCategory attribute.
type FsErrorCategory string

const (
	FsErrorCategoryDEVICEERROR              FsErrorCategory = "DEVICE_ERROR"
	FsErrorCategoryDIRNOTEMPTY              FsErrorCategory = "DIR_NOT_EMPTY"
	FsErrorCategoryFILEDIRERROR             FsErrorCategory = "FILE_DIR_ERROR"
	FsErrorCategoryFILEEXISTS               FsErrorCategory = "FILE_EXISTS"
	FsErrorCategoryINTERRUPTERROR           FsErrorCategory = "INTERRUPT_ERROR"
	FsErrorCategoryINVALIDARGUMENT          FsErrorCategory = "INVALID_ARGUMENT"
	FsErrorCategoryINVALIDOPERATION         FsErrorCategory = "INVALID_OPERATION"
	FsErrorCategoryIOERROR                  FsErrorCategory = "IO_ERROR"
	FsErrorCategoryMISCERROR                FsErrorCategory = "MISC_ERROR"
	FsErrorCategoryNETWORKERROR             FsErrorCategory = "NETWORK_ERROR"
	FsErrorCategoryNOFILEORDIR              FsErrorCategory = "NO_FILE_OR_DIR"
	FsErrorCategoryNOTADIR                  FsErrorCategory = "NOT_A_DIR"
	FsErrorCategoryNOTIMPLEMENTED           FsErrorCategory = "NOT_IMPLEMENTED"
	FsErrorCategoryPERMERROR                FsErrorCategory = "PERM_ERROR"
	FsErrorCategoryPROCESSRESOURCEMGMTERROR FsErrorCategory = "PROCESS_RESOURCE_MGMT_ERROR"
	FsErrorCategoryTOOMANYOPENFILES         FsErrorCategory = "TOO_MANY_OPEN_FILES"
)

// FsOp is a user-defined type for the FsOp attribute.
type FsOp string

const (
	FsOpBatchForget        FsOp = "BatchForget"
	FsOpCreateFile         FsOp = "CreateFile"
	FsOpCreateLink         FsOp = "CreateLink"
	FsOpCreateSymlink      FsOp = "CreateSymlink"
	FsOpFallocate          FsOp = "Fallocate"
	FsOpFlushFile          FsOp = "FlushFile"
	FsOpForgetInode        FsOp = "ForgetInode"
	FsOpGetInodeAttributes FsOp = "GetInodeAttributes"
	FsOpGetXattr           FsOp = "GetXattr"
	FsOpListXattr          FsOp = "ListXattr"
	FsOpLookUpInode        FsOp = "LookUpInode"
	FsOpMkDir              FsOp = "MkDir"
	FsOpMkNode             FsOp = "MkNode"
	FsOpOpenDir            FsOp = "OpenDir"
	FsOpOpenFile           FsOp = "OpenFile"
	FsOpReadDir            FsOp = "ReadDir"
	FsOpReadDirPlus        FsOp = "ReadDirPlus"
	FsOpReadFile           FsOp = "ReadFile"
	FsOpReadSymlink        FsOp = "ReadSymlink"
	FsOpReleaseDirHandle   FsOp = "ReleaseDirHandle"
	FsOpReleaseFileHandle  FsOp = "ReleaseFileHandle"
	FsOpRemoveXattr        FsOp = "RemoveXattr"
	FsOpRename             FsOp = "Rename"
	FsOpRmDir              FsOp = "RmDir"
	FsOpSetInodeAttributes FsOp = "SetInodeAttributes"
	FsOpSetXattr           FsOp = "SetXattr"
	FsOpStatFS             FsOp = "StatFS"
	FsOpSyncFS             FsOp = "SyncFS"
	FsOpSyncFile           FsOp = "SyncFile"
	FsOpUnlink             FsOp = "Unlink"
	FsOpWriteFile          FsOp = "WriteFile"
)

// GcsMethod is a user-defined type for the GcsMethod attribute.
type GcsMethod string

const (
	GcsMethodComposeObjects               GcsMethod = "ComposeObjects"
	GcsMethodCopyObject                   GcsMethod = "CopyObject"
	GcsMethodCreateAppendableObjectWriter GcsMethod = "CreateAppendableObjectWriter"
	GcsMethodCreateFolder                 GcsMethod = "CreateFolder"
	GcsMethodCreateObject                 GcsMethod = "CreateObject"
	GcsMethodCreateObjectChunkWriter      GcsMethod = "CreateObjectChunkWriter"
	GcsMethodDeleteFolder                 GcsMethod = "DeleteFolder"
	GcsMethodDeleteObject                 GcsMethod = "DeleteObject"
	GcsMethodFinalizeUpload               GcsMethod = "FinalizeUpload"
	GcsMethodFlushPendingWrites           GcsMethod = "FlushPendingWrites"
	GcsMethodGetFolder                    GcsMethod = "GetFolder"
	GcsMethodListObjects                  GcsMethod = "ListObjects"
	GcsMethodMoveObject                   GcsMethod = "MoveObject"
	GcsMethodMultiRangeDownloaderAdd      GcsMethod = "MultiRangeDownloader::Add"
	GcsMethodNewMultiRangeDownloader      GcsMethod = "NewMultiRangeDownloader"
	GcsMethodNewReader                    GcsMethod = "NewReader"
	GcsMethodRenameFolder                 GcsMethod = "RenameFolder"
	GcsMethodStatObject                   GcsMethod = "StatObject"
	GcsMethodUpdateObject                 GcsMethod = "UpdateObject"
)

// IoMethod is a user-defined type for the IoMethod attribute.
type IoMethod string

const (
	IoMethodClosed     IoMethod = "closed"
	IoMethodOpened     IoMethod = "opened"
	IoMethodReadHandle IoMethod = "ReadHandle"
)

// ReadType is a user-defined type for the ReadType attribute.
type ReadType string

const (
	ReadTypeParallel   ReadType = "Parallel"
	ReadTypeRandom     ReadType = "Random"
	ReadTypeSequential ReadType = "Sequential"
)

// Reason is a user-defined type for the Reason attribute.
type Reason string

const (
	ReasonInsufficientMemory Reason = "insufficient_memory"
	ReasonRandomReadDetected Reason = "random_read_detected"
)

// RequestType is a user-defined type for the RequestType attribute.
type RequestType string

const (
	RequestTypeAttr1 RequestType = "attr1"
	RequestTypeAttr2 RequestType = "attr2"
)

// RetryErrorCategory is a user-defined type for the RetryErrorCategory attribute.
type RetryErrorCategory string

const (
	RetryErrorCategoryOTHERERRORS        RetryErrorCategory = "OTHER_ERRORS"
	RetryErrorCategorySTALLEDREADREQUEST RetryErrorCategory = "STALLED_READ_REQUEST"
)

// Status is a user-defined type for the Status attribute.
type Status string

const (
	StatusCancelled  Status = "cancelled"
	StatusFailed     Status = "failed"
	StatusSuccessful Status = "successful"
)

// MetricHandle provides an interface for recording metrics.
// The methods of this interface are auto-generated from metrics.yaml.
// Each method corresponds to a metric defined in metrics.yaml.
type MetricHandle interface {
	// BufferedReadDownloadBlockLatency - The cumulative distribution of block download latencies, along with status: successful, cancelled, or failed.
	BufferedReadDownloadBlockLatency(ctx context.Context, duration time.Duration, status Status)

	// BufferedReadFallbackTriggerCount - The cumulative number of times the BufferedReader falls back to a different reader, along with the reason: random_read_detected or insufficient_memory.
	BufferedReadFallbackTriggerCount(inc int64, reason Reason)

	// BufferedReadReadLatency - The cumulative distribution of latencies for ReadAt calls served by the buffered reader.
	BufferedReadReadLatency(ctx context.Context, duration time.Duration)

	// BufferedReadScheduledBlockCount - The cumulative number of scheduled download blocks, along with their final status: successful, cancelled, or failed.
	BufferedReadScheduledBlockCount(inc int64, status Status)

	// FileCacheReadBytesCount - The cumulative number of bytes read from file cache along with read type - Sequential/Random
	FileCacheReadBytesCount(inc int64, readType ReadType)

	// FileCacheReadCount - Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false
	FileCacheReadCount(inc int64, cacheHit bool, readType ReadType)

	// FileCacheReadLatencies - The cumulative distribution of the file cache read latencies along with cache hit - true/false.
	FileCacheReadLatencies(ctx context.Context, duration time.Duration, cacheHit bool)

	// FsOpsCount - The cumulative number of ops processed by the file system.
	FsOpsCount(inc int64, fsOp FsOp)

	// FsOpsErrorCount - The cumulative number of errors generated by file system operations.
	FsOpsErrorCount(inc int64, fsErrorCategory FsErrorCategory, fsOp FsOp)

	// FsOpsLatency - The cumulative distribution of file system operation latencies
	FsOpsLatency(ctx context.Context, duration time.Duration, fsOp FsOp)

	// GcsDownloadBytesCount - The cumulative number of bytes downloaded from GCS along with type - Sequential/Random
	GcsDownloadBytesCount(inc int64, readType ReadType)

	// GcsReadBytesCount - The cumulative number of bytes read from GCS objects.
	GcsReadBytesCount(inc int64)

	// GcsReadCount - Specifies the number of gcs reads made along with type - Sequential/Random
	GcsReadCount(inc int64, readType ReadType)

	// GcsReaderCount - The cumulative number of GCS object readers opened or closed.
	GcsReaderCount(inc int64, ioMethod IoMethod)

	// GcsRequestCount - The cumulative number of GCS requests processed along with the GCS method.
	GcsRequestCount(inc int64, gcsMethod GcsMethod)

	// GcsRequestLatencies - The cumulative distribution of the GCS request latencies.
	GcsRequestLatencies(ctx context.Context, duration time.Duration, gcsMethod GcsMethod)

	// GcsRetryCount - The cumulative number of retry requests made to GCS.
	GcsRetryCount(inc int64, retryErrorCategory RetryErrorCategory)

	// TestUpdownCounter - Test metric for updown counters.
	TestUpdownCounter(inc int64)

	// TestUpdownCounterWithAttrs - Test metric for updown counters with attributes.
	TestUpdownCounterWithAttrs(inc int64, requestType RequestType)
}
