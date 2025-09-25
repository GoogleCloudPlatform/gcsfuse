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

// Constants for attribute FsErrorCategory
const (
	FsErrorCategoryDEVICEERROR              = "DEVICE_ERROR"
	FsErrorCategoryDIRNOTEMPTY              = "DIR_NOT_EMPTY"
	FsErrorCategoryFILEDIRERROR             = "FILE_DIR_ERROR"
	FsErrorCategoryFILEEXISTS               = "FILE_EXISTS"
	FsErrorCategoryINTERRUPTERROR           = "INTERRUPT_ERROR"
	FsErrorCategoryINVALIDARGUMENT          = "INVALID_ARGUMENT"
	FsErrorCategoryINVALIDOPERATION         = "INVALID_OPERATION"
	FsErrorCategoryIOERROR                  = "IO_ERROR"
	FsErrorCategoryMISCERROR                = "MISC_ERROR"
	FsErrorCategoryNETWORKERROR             = "NETWORK_ERROR"
	FsErrorCategoryNOFILEORDIR              = "NO_FILE_OR_DIR"
	FsErrorCategoryNOTADIR                  = "NOT_A_DIR"
	FsErrorCategoryNOTIMPLEMENTED           = "NOT_IMPLEMENTED"
	FsErrorCategoryPERMERROR                = "PERM_ERROR"
	FsErrorCategoryPROCESSRESOURCEMGMTERROR = "PROCESS_RESOURCE_MGMT_ERROR"
	FsErrorCategoryTOOMANYOPENFILES         = "TOO_MANY_OPEN_FILES"
)

// Constants for attribute FsOp
const (
	FsOpBatchForget        = "BatchForget"
	FsOpCreateFile         = "CreateFile"
	FsOpCreateLink         = "CreateLink"
	FsOpCreateSymlink      = "CreateSymlink"
	FsOpFallocate          = "Fallocate"
	FsOpFlushFile          = "FlushFile"
	FsOpForgetInode        = "ForgetInode"
	FsOpGetInodeAttributes = "GetInodeAttributes"
	FsOpGetXattr           = "GetXattr"
	FsOpListXattr          = "ListXattr"
	FsOpLookUpInode        = "LookUpInode"
	FsOpMkDir              = "MkDir"
	FsOpMkNode             = "MkNode"
	FsOpOpenDir            = "OpenDir"
	FsOpOpenFile           = "OpenFile"
	FsOpReadDir            = "ReadDir"
	FsOpReadDirPlus        = "ReadDirPlus"
	FsOpReadFile           = "ReadFile"
	FsOpReadSymlink        = "ReadSymlink"
	FsOpReleaseDirHandle   = "ReleaseDirHandle"
	FsOpReleaseFileHandle  = "ReleaseFileHandle"
	FsOpRemoveXattr        = "RemoveXattr"
	FsOpRename             = "Rename"
	FsOpRmDir              = "RmDir"
	FsOpSetInodeAttributes = "SetInodeAttributes"
	FsOpSetXattr           = "SetXattr"
	FsOpStatFS             = "StatFS"
	FsOpSyncFS             = "SyncFS"
	FsOpSyncFile           = "SyncFile"
	FsOpUnlink             = "Unlink"
	FsOpWriteFile          = "WriteFile"
)

// Constants for attribute GcsMethod
const (
	GcsMethodComposeObjects               = "ComposeObjects"
	GcsMethodCopyObject                   = "CopyObject"
	GcsMethodCreateAppendableObjectWriter = "CreateAppendableObjectWriter"
	GcsMethodCreateFolder                 = "CreateFolder"
	GcsMethodCreateObject                 = "CreateObject"
	GcsMethodCreateObjectChunkWriter      = "CreateObjectChunkWriter"
	GcsMethodDeleteFolder                 = "DeleteFolder"
	GcsMethodDeleteObject                 = "DeleteObject"
	GcsMethodFinalizeUpload               = "FinalizeUpload"
	GcsMethodFlushPendingWrites           = "FlushPendingWrites"
	GcsMethodGetFolder                    = "GetFolder"
	GcsMethodListObjects                  = "ListObjects"
	GcsMethodMoveObject                   = "MoveObject"
	GcsMethodMultiRangeDownloaderAdd      = "MultiRangeDownloader::Add"
	GcsMethodNewMultiRangeDownloader      = "NewMultiRangeDownloader"
	GcsMethodNewReader                    = "NewReader"
	GcsMethodRenameFolder                 = "RenameFolder"
	GcsMethodStatObject                   = "StatObject"
	GcsMethodUpdateObject                 = "UpdateObject"
)

// Constants for attribute IoMethod
const (
	IoMethodClosed     = "closed"
	IoMethodOpened     = "opened"
	IoMethodReadHandle = "ReadHandle"
)

// Constants for attribute ReadType
const (
	ReadTypeParallel   = "Parallel"
	ReadTypeRandom     = "Random"
	ReadTypeSequential = "Sequential"
	ReadTypeUnknown    = "Unknown"
)

// Constants for attribute Reason
const (
	ReasonInsufficientMemory = "insufficient_memory"
	ReasonRandomReadDetected = "random_read_detected"
)

// Constants for attribute RetryErrorCategory
const (
	RetryErrorCategoryOTHERERRORS        = "OTHER_ERRORS"
	RetryErrorCategorySTALLEDREADREQUEST = "STALLED_READ_REQUEST"
)

// Constants for attribute Status
const (
	StatusCancelled  = "cancelled"
	StatusSuccessful = "successful"
)

// MetricHandle provides an interface for recording metrics.
// The methods of this interface are auto-generated from metrics.yaml.
// Each method corresponds to a metric defined in metrics.yaml.
type MetricHandle interface {
	// BufferedReadDownloadBlockLatency - The cumulative distribution of block download latencies, along with status: successful, cancelled, or failed.
	BufferedReadDownloadBlockLatency(ctx context.Context, duration time.Duration, status string)

	// BufferedReadFallbackTriggerCount - The cumulative number of times the BufferedReader falls back to a different reader, along with the reason: random_read_detected or insufficient_memory.
	BufferedReadFallbackTriggerCount(inc int64, reason string)

	// BufferedReadReadLatency - The cumulative distribution of latencies for ReadAt calls served by the buffered reader.
	BufferedReadReadLatency(ctx context.Context, duration time.Duration)

	// BufferedReadScheduledBlockCount - The cumulative number of scheduled download blocks, along with their final status: successful, cancelled, or failed.
	BufferedReadScheduledBlockCount(inc int64, status string)

	// FileCacheReadBytesCount - The cumulative number of bytes read from file cache along with read type - Sequential/Random
	FileCacheReadBytesCount(inc int64, readType string)

	// FileCacheReadCount - Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false
	FileCacheReadCount(inc int64, cacheHit bool, readType string)

	// FileCacheReadLatencies - The cumulative distribution of the file cache read latencies along with cache hit - true/false.
	FileCacheReadLatencies(ctx context.Context, duration time.Duration, cacheHit bool)

	// FsOpsCount - The cumulative number of ops processed by the file system.
	FsOpsCount(inc int64, fsOp string)

	// FsOpsErrorCount - The cumulative number of errors generated by file system operations.
	FsOpsErrorCount(inc int64, fsErrorCategory string, fsOp string)

	// FsOpsLatency - The cumulative distribution of file system operation latencies
	FsOpsLatency(ctx context.Context, duration time.Duration, fsOp string)

	// GcsDownloadBytesCount - The cumulative number of bytes downloaded from GCS along with type - Sequential/Random
	GcsDownloadBytesCount(inc int64, readType string)

	// GcsReadBytesCount - The cumulative number of bytes read from GCS objects.
	GcsReadBytesCount(inc int64)

	// GcsReadCount - Specifies the number of gcs reads made along with type - Sequential/Random
	GcsReadCount(inc int64, readType string)

	// GcsReaderCount - The cumulative number of GCS object readers opened or closed.
	GcsReaderCount(inc int64, ioMethod string)

	// GcsRequestCount - The cumulative number of GCS requests processed along with the GCS method.
	GcsRequestCount(inc int64, gcsMethod string)

	// GcsRequestLatencies - The cumulative distribution of the GCS request latencies.
	GcsRequestLatencies(ctx context.Context, duration time.Duration, gcsMethod string)

	// GcsRetryCount - The cumulative number of retry requests made to GCS.
	GcsRetryCount(inc int64, retryErrorCategory string)

	// TestUpdownCounter - Test metric for updown counters.
	TestUpdownCounter(inc int64)

	// TestUpdownCounterWithAttrs - Test metric for updown counters with attributes.
	TestUpdownCounterWithAttrs(inc int64, requestType string)
}
