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

// FsErrorCategory is a custom type for the fs_error_category attribute.
type FsErrorCategory string

const (
	FsErrorCategoryDEVICEERRORAttr              FsErrorCategory = "DEVICE_ERROR"
	FsErrorCategoryDIRNOTEMPTYAttr              FsErrorCategory = "DIR_NOT_EMPTY"
	FsErrorCategoryFILEDIRERRORAttr             FsErrorCategory = "FILE_DIR_ERROR"
	FsErrorCategoryFILEEXISTSAttr               FsErrorCategory = "FILE_EXISTS"
	FsErrorCategoryINTERRUPTERRORAttr           FsErrorCategory = "INTERRUPT_ERROR"
	FsErrorCategoryINVALIDARGUMENTAttr          FsErrorCategory = "INVALID_ARGUMENT"
	FsErrorCategoryINVALIDOPERATIONAttr         FsErrorCategory = "INVALID_OPERATION"
	FsErrorCategoryIOERRORAttr                  FsErrorCategory = "IO_ERROR"
	FsErrorCategoryMISCERRORAttr                FsErrorCategory = "MISC_ERROR"
	FsErrorCategoryNETWORKERRORAttr             FsErrorCategory = "NETWORK_ERROR"
	FsErrorCategoryNOTADIRAttr                  FsErrorCategory = "NOT_A_DIR"
	FsErrorCategoryNOTIMPLEMENTEDAttr           FsErrorCategory = "NOT_IMPLEMENTED"
	FsErrorCategoryNOFILEORDIRAttr              FsErrorCategory = "NO_FILE_OR_DIR"
	FsErrorCategoryPERMERRORAttr                FsErrorCategory = "PERM_ERROR"
	FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr FsErrorCategory = "PROCESS_RESOURCE_MGMT_ERROR"
	FsErrorCategoryTOOMANYOPENFILESAttr         FsErrorCategory = "TOO_MANY_OPEN_FILES"
)

// FsOp is a custom type for the fs_op attribute.
type FsOp string

const (
	FsOpBatchForgetAttr        FsOp = "BatchForget"
	FsOpCreateFileAttr         FsOp = "CreateFile"
	FsOpCreateLinkAttr         FsOp = "CreateLink"
	FsOpCreateSymlinkAttr      FsOp = "CreateSymlink"
	FsOpFlushFileAttr          FsOp = "FlushFile"
	FsOpForgetInodeAttr        FsOp = "ForgetInode"
	FsOpGetInodeAttributesAttr FsOp = "GetInodeAttributes"
	FsOpLookUpInodeAttr        FsOp = "LookUpInode"
	FsOpMkDirAttr              FsOp = "MkDir"
	FsOpMkNodeAttr             FsOp = "MkNode"
	FsOpOpenDirAttr            FsOp = "OpenDir"
	FsOpOpenFileAttr           FsOp = "OpenFile"
	FsOpOthersAttr             FsOp = "Others"
	FsOpReadDirAttr            FsOp = "ReadDir"
	FsOpReadDirPlusAttr        FsOp = "ReadDirPlus"
	FsOpReadFileAttr           FsOp = "ReadFile"
	FsOpReadSymlinkAttr        FsOp = "ReadSymlink"
	FsOpReleaseDirHandleAttr   FsOp = "ReleaseDirHandle"
	FsOpReleaseFileHandleAttr  FsOp = "ReleaseFileHandle"
	FsOpRenameAttr             FsOp = "Rename"
	FsOpRmDirAttr              FsOp = "RmDir"
	FsOpSetInodeAttributesAttr FsOp = "SetInodeAttributes"
	FsOpSyncFileAttr           FsOp = "SyncFile"
	FsOpUnlinkAttr             FsOp = "Unlink"
	FsOpWriteFileAttr          FsOp = "WriteFile"
)

// GcsMethod is a custom type for the gcs_method attribute.
type GcsMethod string

const (
	GcsMethodComposeObjectsAttr               GcsMethod = "ComposeObjects"
	GcsMethodCopyObjectAttr                   GcsMethod = "CopyObject"
	GcsMethodCreateAppendableObjectWriterAttr GcsMethod = "CreateAppendableObjectWriter"
	GcsMethodCreateFolderAttr                 GcsMethod = "CreateFolder"
	GcsMethodCreateObjectAttr                 GcsMethod = "CreateObject"
	GcsMethodCreateObjectChunkWriterAttr      GcsMethod = "CreateObjectChunkWriter"
	GcsMethodDeleteFolderAttr                 GcsMethod = "DeleteFolder"
	GcsMethodDeleteObjectAttr                 GcsMethod = "DeleteObject"
	GcsMethodFinalizeUploadAttr               GcsMethod = "FinalizeUpload"
	GcsMethodFlushPendingWritesAttr           GcsMethod = "FlushPendingWrites"
	GcsMethodGetFolderAttr                    GcsMethod = "GetFolder"
	GcsMethodListObjectsAttr                  GcsMethod = "ListObjects"
	GcsMethodMoveObjectAttr                   GcsMethod = "MoveObject"
	GcsMethodMultiRangeDownloaderAddAttr      GcsMethod = "MultiRangeDownloader::Add"
	GcsMethodNewMultiRangeDownloaderAttr      GcsMethod = "NewMultiRangeDownloader"
	GcsMethodNewReaderAttr                    GcsMethod = "NewReader"
	GcsMethodRenameFolderAttr                 GcsMethod = "RenameFolder"
	GcsMethodStatObjectAttr                   GcsMethod = "StatObject"
	GcsMethodUpdateObjectAttr                 GcsMethod = "UpdateObject"
)

// IoMethod is a custom type for the io_method attribute.
type IoMethod string

const (
	IoMethodReadHandleAttr IoMethod = "ReadHandle"
	IoMethodClosedAttr     IoMethod = "closed"
	IoMethodOpenedAttr     IoMethod = "opened"
)

// ReadType is a custom type for the read_type attribute.
type ReadType string

const (
	ReadTypeBufferedAttr   ReadType = "Buffered"
	ReadTypeParallelAttr   ReadType = "Parallel"
	ReadTypeRandomAttr     ReadType = "Random"
	ReadTypeSequentialAttr ReadType = "Sequential"
	ReadTypeUnknownAttr    ReadType = "Unknown"
)

// Reason is a custom type for the reason attribute.
type Reason string

const (
	ReasonInsufficientMemoryAttr Reason = "insufficient_memory"
	ReasonRandomReadDetectedAttr Reason = "random_read_detected"
)

// RequestType is a custom type for the request_type attribute.
type RequestType string

const (
	RequestTypeAttr1Attr RequestType = "attr1"
	RequestTypeAttr2Attr RequestType = "attr2"
)

// RetryErrorCategory is a custom type for the retry_error_category attribute.
type RetryErrorCategory string

const (
	RetryErrorCategoryOTHERERRORSAttr        RetryErrorCategory = "OTHER_ERRORS"
	RetryErrorCategorySTALLEDREADREQUESTAttr RetryErrorCategory = "STALLED_READ_REQUEST"
)

// MetricHandle provides an interface for recording metrics.
// The methods of this interface are auto-generated from metrics.yaml.
// Each method corresponds to a metric defined in metrics.yaml.
type MetricHandle interface {
	// BufferedReadFallbackTriggerCount - The cumulative number of times the BufferedReader falls back to a different reader, along with the reason: random_read_detected or insufficient_memory.
	BufferedReadFallbackTriggerCount(inc int64, reason Reason)

	// BufferedReadReadLatency - The cumulative distribution of latencies for ReadAt calls served by the buffered reader.
	BufferedReadReadLatency(ctx context.Context, latency time.Duration)

	// FileCacheReadBytesCount - The cumulative number of bytes read from file cache along with read type - Sequential/Random
	FileCacheReadBytesCount(inc int64, readType ReadType)

	// FileCacheReadCount - Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false
	FileCacheReadCount(inc int64, cacheHit bool, readType ReadType)

	// FileCacheReadLatencies - The cumulative distribution of the file cache read latencies along with cache hit - true/false.
	FileCacheReadLatencies(ctx context.Context, latency time.Duration, cacheHit bool)

	// FsOpsCount - The cumulative number of ops processed by the file system.
	FsOpsCount(inc int64, fsOp FsOp)

	// FsOpsErrorCount - The cumulative number of errors generated by file system operations.
	FsOpsErrorCount(inc int64, fsErrorCategory FsErrorCategory, fsOp FsOp)

	// FsOpsLatency - The cumulative distribution of file system operation latencies
	FsOpsLatency(ctx context.Context, latency time.Duration, fsOp FsOp)

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
	GcsRequestLatencies(ctx context.Context, latency time.Duration, gcsMethod GcsMethod)

	// GcsRetryCount - The cumulative number of retry requests made to GCS.
	GcsRetryCount(inc int64, retryErrorCategory RetryErrorCategory)

	// TestUpdownCounter - Test metric for updown counters.
	TestUpdownCounter(inc int64)

	// TestUpdownCounterWithAttrs - Test metric for updown counters with attributes.
	TestUpdownCounterWithAttrs(inc int64, requestType RequestType)
}
