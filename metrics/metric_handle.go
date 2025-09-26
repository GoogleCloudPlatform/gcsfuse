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

// FsErrorCategory is a custom type for the fsErrorCategory attribute.
type FsErrorCategory string

const (
	// FsErrorCategoryDEVICEERRORAttr is the "DEVICE_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryDEVICEERRORAttr FsErrorCategory = "DEVICE_ERROR"
	// FsErrorCategoryDIRNOTEMPTYAttr is the "DIR_NOT_EMPTY" value for the "fsErrorCategory" attribute.
	FsErrorCategoryDIRNOTEMPTYAttr FsErrorCategory = "DIR_NOT_EMPTY"
	// FsErrorCategoryFILEDIRERRORAttr is the "FILE_DIR_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryFILEDIRERRORAttr FsErrorCategory = "FILE_DIR_ERROR"
	// FsErrorCategoryFILEEXISTSAttr is the "FILE_EXISTS" value for the "fsErrorCategory" attribute.
	FsErrorCategoryFILEEXISTSAttr FsErrorCategory = "FILE_EXISTS"
	// FsErrorCategoryINTERRUPTERRORAttr is the "INTERRUPT_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryINTERRUPTERRORAttr FsErrorCategory = "INTERRUPT_ERROR"
	// FsErrorCategoryINVALIDARGUMENTAttr is the "INVALID_ARGUMENT" value for the "fsErrorCategory" attribute.
	FsErrorCategoryINVALIDARGUMENTAttr FsErrorCategory = "INVALID_ARGUMENT"
	// FsErrorCategoryINVALIDOPERATIONAttr is the "INVALID_OPERATION" value for the "fsErrorCategory" attribute.
	FsErrorCategoryINVALIDOPERATIONAttr FsErrorCategory = "INVALID_OPERATION"
	// FsErrorCategoryIOERRORAttr is the "IO_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryIOERRORAttr FsErrorCategory = "IO_ERROR"
	// FsErrorCategoryMISCERRORAttr is the "MISC_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryMISCERRORAttr FsErrorCategory = "MISC_ERROR"
	// FsErrorCategoryNETWORKERRORAttr is the "NETWORK_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryNETWORKERRORAttr FsErrorCategory = "NETWORK_ERROR"
	// FsErrorCategoryNOFILEORDIRAttr is the "NO_FILE_OR_DIR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryNOFILEORDIRAttr FsErrorCategory = "NO_FILE_OR_DIR"
	// FsErrorCategoryNOTADIRAttr is the "NOT_A_DIR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryNOTADIRAttr FsErrorCategory = "NOT_A_DIR"
	// FsErrorCategoryNOTIMPLEMENTEDAttr is the "NOT_IMPLEMENTED" value for the "fsErrorCategory" attribute.
	FsErrorCategoryNOTIMPLEMENTEDAttr FsErrorCategory = "NOT_IMPLEMENTED"
	// FsErrorCategoryPERMERRORAttr is the "PERM_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryPERMERRORAttr FsErrorCategory = "PERM_ERROR"
	// FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr is the "PROCESS_RESOURCE_MGMT_ERROR" value for the "fsErrorCategory" attribute.
	FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr FsErrorCategory = "PROCESS_RESOURCE_MGMT_ERROR"
	// FsErrorCategoryTOOMANYOPENFILESAttr is the "TOO_MANY_OPEN_FILES" value for the "fsErrorCategory" attribute.
	FsErrorCategoryTOOMANYOPENFILESAttr FsErrorCategory = "TOO_MANY_OPEN_FILES"
)

// FsOp is a custom type for the fsOp attribute.
type FsOp string

const (
	// FsOpBatchForgetAttr is the "BatchForget" value for the "fsOp" attribute.
	FsOpBatchForgetAttr FsOp = "BatchForget"
	// FsOpCreateFileAttr is the "CreateFile" value for the "fsOp" attribute.
	FsOpCreateFileAttr FsOp = "CreateFile"
	// FsOpCreateLinkAttr is the "CreateLink" value for the "fsOp" attribute.
	FsOpCreateLinkAttr FsOp = "CreateLink"
	// FsOpCreateSymlinkAttr is the "CreateSymlink" value for the "fsOp" attribute.
	FsOpCreateSymlinkAttr FsOp = "CreateSymlink"
	// FsOpFallocateAttr is the "Fallocate" value for the "fsOp" attribute.
	FsOpFallocateAttr FsOp = "Fallocate"
	// FsOpFlushFileAttr is the "FlushFile" value for the "fsOp" attribute.
	FsOpFlushFileAttr FsOp = "FlushFile"
	// FsOpForgetInodeAttr is the "ForgetInode" value for the "fsOp" attribute.
	FsOpForgetInodeAttr FsOp = "ForgetInode"
	// FsOpGetInodeAttributesAttr is the "GetInodeAttributes" value for the "fsOp" attribute.
	FsOpGetInodeAttributesAttr FsOp = "GetInodeAttributes"
	// FsOpGetXattrAttr is the "GetXattr" value for the "fsOp" attribute.
	FsOpGetXattrAttr FsOp = "GetXattr"
	// FsOpListXattrAttr is the "ListXattr" value for the "fsOp" attribute.
	FsOpListXattrAttr FsOp = "ListXattr"
	// FsOpLookUpInodeAttr is the "LookUpInode" value for the "fsOp" attribute.
	FsOpLookUpInodeAttr FsOp = "LookUpInode"
	// FsOpMkDirAttr is the "MkDir" value for the "fsOp" attribute.
	FsOpMkDirAttr FsOp = "MkDir"
	// FsOpMkNodeAttr is the "MkNode" value for the "fsOp" attribute.
	FsOpMkNodeAttr FsOp = "MkNode"
	// FsOpOpenDirAttr is the "OpenDir" value for the "fsOp" attribute.
	FsOpOpenDirAttr FsOp = "OpenDir"
	// FsOpOpenFileAttr is the "OpenFile" value for the "fsOp" attribute.
	FsOpOpenFileAttr FsOp = "OpenFile"
	// FsOpReadDirAttr is the "ReadDir" value for the "fsOp" attribute.
	FsOpReadDirAttr FsOp = "ReadDir"
	// FsOpReadDirPlusAttr is the "ReadDirPlus" value for the "fsOp" attribute.
	FsOpReadDirPlusAttr FsOp = "ReadDirPlus"
	// FsOpReadFileAttr is the "ReadFile" value for the "fsOp" attribute.
	FsOpReadFileAttr FsOp = "ReadFile"
	// FsOpReadSymlinkAttr is the "ReadSymlink" value for the "fsOp" attribute.
	FsOpReadSymlinkAttr FsOp = "ReadSymlink"
	// FsOpReleaseDirHandleAttr is the "ReleaseDirHandle" value for the "fsOp" attribute.
	FsOpReleaseDirHandleAttr FsOp = "ReleaseDirHandle"
	// FsOpReleaseFileHandleAttr is the "ReleaseFileHandle" value for the "fsOp" attribute.
	FsOpReleaseFileHandleAttr FsOp = "ReleaseFileHandle"
	// FsOpRemoveXattrAttr is the "RemoveXattr" value for the "fsOp" attribute.
	FsOpRemoveXattrAttr FsOp = "RemoveXattr"
	// FsOpRenameAttr is the "Rename" value for the "fsOp" attribute.
	FsOpRenameAttr FsOp = "Rename"
	// FsOpRmDirAttr is the "RmDir" value for the "fsOp" attribute.
	FsOpRmDirAttr FsOp = "RmDir"
	// FsOpSetInodeAttributesAttr is the "SetInodeAttributes" value for the "fsOp" attribute.
	FsOpSetInodeAttributesAttr FsOp = "SetInodeAttributes"
	// FsOpSetXattrAttr is the "SetXattr" value for the "fsOp" attribute.
	FsOpSetXattrAttr FsOp = "SetXattr"
	// FsOpStatFSAttr is the "StatFS" value for the "fsOp" attribute.
	FsOpStatFSAttr FsOp = "StatFS"
	// FsOpSyncFSAttr is the "SyncFS" value for the "fsOp" attribute.
	FsOpSyncFSAttr FsOp = "SyncFS"
	// FsOpSyncFileAttr is the "SyncFile" value for the "fsOp" attribute.
	FsOpSyncFileAttr FsOp = "SyncFile"
	// FsOpUnlinkAttr is the "Unlink" value for the "fsOp" attribute.
	FsOpUnlinkAttr FsOp = "Unlink"
	// FsOpWriteFileAttr is the "WriteFile" value for the "fsOp" attribute.
	FsOpWriteFileAttr FsOp = "WriteFile"
)

// GcsMethod is a custom type for the gcsMethod attribute.
type GcsMethod string

const (
	// GcsMethodComposeObjectsAttr is the "ComposeObjects" value for the "gcsMethod" attribute.
	GcsMethodComposeObjectsAttr GcsMethod = "ComposeObjects"
	// GcsMethodCopyObjectAttr is the "CopyObject" value for the "gcsMethod" attribute.
	GcsMethodCopyObjectAttr GcsMethod = "CopyObject"
	// GcsMethodCreateAppendableObjectWriterAttr is the "CreateAppendableObjectWriter" value for the "gcsMethod" attribute.
	GcsMethodCreateAppendableObjectWriterAttr GcsMethod = "CreateAppendableObjectWriter"
	// GcsMethodCreateFolderAttr is the "CreateFolder" value for the "gcsMethod" attribute.
	GcsMethodCreateFolderAttr GcsMethod = "CreateFolder"
	// GcsMethodCreateObjectAttr is the "CreateObject" value for the "gcsMethod" attribute.
	GcsMethodCreateObjectAttr GcsMethod = "CreateObject"
	// GcsMethodCreateObjectChunkWriterAttr is the "CreateObjectChunkWriter" value for the "gcsMethod" attribute.
	GcsMethodCreateObjectChunkWriterAttr GcsMethod = "CreateObjectChunkWriter"
	// GcsMethodDeleteFolderAttr is the "DeleteFolder" value for the "gcsMethod" attribute.
	GcsMethodDeleteFolderAttr GcsMethod = "DeleteFolder"
	// GcsMethodDeleteObjectAttr is the "DeleteObject" value for the "gcsMethod" attribute.
	GcsMethodDeleteObjectAttr GcsMethod = "DeleteObject"
	// GcsMethodFinalizeUploadAttr is the "FinalizeUpload" value for the "gcsMethod" attribute.
	GcsMethodFinalizeUploadAttr GcsMethod = "FinalizeUpload"
	// GcsMethodFlushPendingWritesAttr is the "FlushPendingWrites" value for the "gcsMethod" attribute.
	GcsMethodFlushPendingWritesAttr GcsMethod = "FlushPendingWrites"
	// GcsMethodGetFolderAttr is the "GetFolder" value for the "gcsMethod" attribute.
	GcsMethodGetFolderAttr GcsMethod = "GetFolder"
	// GcsMethodListObjectsAttr is the "ListObjects" value for the "gcsMethod" attribute.
	GcsMethodListObjectsAttr GcsMethod = "ListObjects"
	// GcsMethodMoveObjectAttr is the "MoveObject" value for the "gcsMethod" attribute.
	GcsMethodMoveObjectAttr GcsMethod = "MoveObject"
	// GcsMethodMultiRangeDownloaderAddAttr is the "MultiRangeDownloader::Add" value for the "gcsMethod" attribute.
	GcsMethodMultiRangeDownloaderAddAttr GcsMethod = "MultiRangeDownloader::Add"
	// GcsMethodNewMultiRangeDownloaderAttr is the "NewMultiRangeDownloader" value for the "gcsMethod" attribute.
	GcsMethodNewMultiRangeDownloaderAttr GcsMethod = "NewMultiRangeDownloader"
	// GcsMethodNewReaderAttr is the "NewReader" value for the "gcsMethod" attribute.
	GcsMethodNewReaderAttr GcsMethod = "NewReader"
	// GcsMethodRenameFolderAttr is the "RenameFolder" value for the "gcsMethod" attribute.
	GcsMethodRenameFolderAttr GcsMethod = "RenameFolder"
	// GcsMethodStatObjectAttr is the "StatObject" value for the "gcsMethod" attribute.
	GcsMethodStatObjectAttr GcsMethod = "StatObject"
	// GcsMethodUpdateObjectAttr is the "UpdateObject" value for the "gcsMethod" attribute.
	GcsMethodUpdateObjectAttr GcsMethod = "UpdateObject"
)

// IoMethod is a custom type for the ioMethod attribute.
type IoMethod string

const (
	// IoMethodClosedAttr is the "closed" value for the "ioMethod" attribute.
	IoMethodClosedAttr IoMethod = "closed"
	// IoMethodOpenedAttr is the "opened" value for the "ioMethod" attribute.
	IoMethodOpenedAttr IoMethod = "opened"
	// IoMethodReadHandleAttr is the "ReadHandle" value for the "ioMethod" attribute.
	IoMethodReadHandleAttr IoMethod = "ReadHandle"
)

// ReadType is a custom type for the readType attribute.
type ReadType string

const (
	// ReadTypeParallelAttr is the "Parallel" value for the "readType" attribute.
	ReadTypeParallelAttr ReadType = "Parallel"
	// ReadTypeRandomAttr is the "Random" value for the "readType" attribute.
	ReadTypeRandomAttr ReadType = "Random"
	// ReadTypeSequentialAttr is the "Sequential" value for the "readType" attribute.
	ReadTypeSequentialAttr ReadType = "Sequential"
)

// Reason is a custom type for the reason attribute.
type Reason string

const (
	// ReasonInsufficientMemoryAttr is the "insufficient_memory" value for the "reason" attribute.
	ReasonInsufficientMemoryAttr Reason = "insufficient_memory"
	// ReasonRandomReadDetectedAttr is the "random_read_detected" value for the "reason" attribute.
	ReasonRandomReadDetectedAttr Reason = "random_read_detected"
)

// RequestType is a custom type for the requestType attribute.
type RequestType string

const (
	// RequestTypeAttr1Attr is the "attr1" value for the "requestType" attribute.
	RequestTypeAttr1Attr RequestType = "attr1"
	// RequestTypeAttr2Attr is the "attr2" value for the "requestType" attribute.
	RequestTypeAttr2Attr RequestType = "attr2"
)

// RetryErrorCategory is a custom type for the retryErrorCategory attribute.
type RetryErrorCategory string

const (
	// RetryErrorCategoryOTHERERRORSAttr is the "OTHER_ERRORS" value for the "retryErrorCategory" attribute.
	RetryErrorCategoryOTHERERRORSAttr RetryErrorCategory = "OTHER_ERRORS"
	// RetryErrorCategorySTALLEDREADREQUESTAttr is the "STALLED_READ_REQUEST" value for the "retryErrorCategory" attribute.
	RetryErrorCategorySTALLEDREADREQUESTAttr RetryErrorCategory = "STALLED_READ_REQUEST"
)

// Status is a custom type for the status attribute.
type Status string

const (
	// StatusCancelledAttr is the "cancelled" value for the "status" attribute.
	StatusCancelledAttr Status = "cancelled"
	// StatusFailedAttr is the "failed" value for the "status" attribute.
	StatusFailedAttr Status = "failed"
	// StatusSuccessfulAttr is the "successful" value for the "status" attribute.
	StatusSuccessfulAttr Status = "successful"
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
