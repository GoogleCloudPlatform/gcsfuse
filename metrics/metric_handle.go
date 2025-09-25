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

// MetricAttr is a string type for metric attributes.
type MetricAttr string

const (
	Attr1Attr                        MetricAttr = "attr1"
	Attr2Attr                        MetricAttr = "attr2"
	BatchForgetAttr                  MetricAttr = "BatchForget"
	CancelledAttr                    MetricAttr = "cancelled"
	ClosedAttr                       MetricAttr = "closed"
	ComposeObjectsAttr               MetricAttr = "ComposeObjects"
	CopyObjectAttr                   MetricAttr = "CopyObject"
	CreateAppendableObjectWriterAttr MetricAttr = "CreateAppendableObjectWriter"
	CreateFileAttr                   MetricAttr = "CreateFile"
	CreateFolderAttr                 MetricAttr = "CreateFolder"
	CreateLinkAttr                   MetricAttr = "CreateLink"
	CreateObjectAttr                 MetricAttr = "CreateObject"
	CreateObjectChunkWriterAttr      MetricAttr = "CreateObjectChunkWriter"
	CreateSymlinkAttr                MetricAttr = "CreateSymlink"
	DEVICEERRORAttr                  MetricAttr = "DEVICE_ERROR"
	DIRNOTEMPTYAttr                  MetricAttr = "DIR_NOT_EMPTY"
	DeleteFolderAttr                 MetricAttr = "DeleteFolder"
	DeleteObjectAttr                 MetricAttr = "DeleteObject"
	FILEDIRERRORAttr                 MetricAttr = "FILE_DIR_ERROR"
	FILEEXISTSAttr                   MetricAttr = "FILE_EXISTS"
	FailedAttr                       MetricAttr = "failed"
	FallocateAttr                    MetricAttr = "Fallocate"
	FinalizeUploadAttr               MetricAttr = "FinalizeUpload"
	FlushFileAttr                    MetricAttr = "FlushFile"
	FlushPendingWritesAttr           MetricAttr = "FlushPendingWrites"
	ForgetInodeAttr                  MetricAttr = "ForgetInode"
	GetFolderAttr                    MetricAttr = "GetFolder"
	GetInodeAttributesAttr           MetricAttr = "GetInodeAttributes"
	GetXattrAttr                     MetricAttr = "GetXattr"
	INTERRUPTERRORAttr               MetricAttr = "INTERRUPT_ERROR"
	INVALIDARGUMENTAttr              MetricAttr = "INVALID_ARGUMENT"
	INVALIDOPERATIONAttr             MetricAttr = "INVALID_OPERATION"
	IOERRORAttr                      MetricAttr = "IO_ERROR"
	InsufficientMemoryAttr           MetricAttr = "insufficient_memory"
	ListObjectsAttr                  MetricAttr = "ListObjects"
	ListXattrAttr                    MetricAttr = "ListXattr"
	LookUpInodeAttr                  MetricAttr = "LookUpInode"
	MISCERRORAttr                    MetricAttr = "MISC_ERROR"
	MkDirAttr                        MetricAttr = "MkDir"
	MkNodeAttr                       MetricAttr = "MkNode"
	MoveObjectAttr                   MetricAttr = "MoveObject"
	MultiRangeDownloaderAddAttr      MetricAttr = "MultiRangeDownloader::Add"
	NETWORKERRORAttr                 MetricAttr = "NETWORK_ERROR"
	NOFILEORDIRAttr                  MetricAttr = "NO_FILE_OR_DIR"
	NOTADIRAttr                      MetricAttr = "NOT_A_DIR"
	NOTIMPLEMENTEDAttr               MetricAttr = "NOT_IMPLEMENTED"
	NewMultiRangeDownloaderAttr      MetricAttr = "NewMultiRangeDownloader"
	NewReaderAttr                    MetricAttr = "NewReader"
	OTHERERRORSAttr                  MetricAttr = "OTHER_ERRORS"
	OpenDirAttr                      MetricAttr = "OpenDir"
	OpenFileAttr                     MetricAttr = "OpenFile"
	OpenedAttr                       MetricAttr = "opened"
	PERMERRORAttr                    MetricAttr = "PERM_ERROR"
	PROCESSRESOURCEMGMTERRORAttr     MetricAttr = "PROCESS_RESOURCE_MGMT_ERROR"
	ParallelAttr                     MetricAttr = "Parallel"
	RandomAttr                       MetricAttr = "Random"
	RandomReadDetectedAttr           MetricAttr = "random_read_detected"
	ReadDirAttr                      MetricAttr = "ReadDir"
	ReadDirPlusAttr                  MetricAttr = "ReadDirPlus"
	ReadFileAttr                     MetricAttr = "ReadFile"
	ReadHandleAttr                   MetricAttr = "ReadHandle"
	ReadSymlinkAttr                  MetricAttr = "ReadSymlink"
	ReleaseDirHandleAttr             MetricAttr = "ReleaseDirHandle"
	ReleaseFileHandleAttr            MetricAttr = "ReleaseFileHandle"
	RemoveXattrAttr                  MetricAttr = "RemoveXattr"
	RenameAttr                       MetricAttr = "Rename"
	RenameFolderAttr                 MetricAttr = "RenameFolder"
	RmDirAttr                        MetricAttr = "RmDir"
	STALLEDREADREQUESTAttr           MetricAttr = "STALLED_READ_REQUEST"
	SequentialAttr                   MetricAttr = "Sequential"
	SetInodeAttributesAttr           MetricAttr = "SetInodeAttributes"
	SetXattrAttr                     MetricAttr = "SetXattr"
	StatFSAttr                       MetricAttr = "StatFS"
	StatObjectAttr                   MetricAttr = "StatObject"
	SuccessfulAttr                   MetricAttr = "successful"
	SyncFSAttr                       MetricAttr = "SyncFS"
	SyncFileAttr                     MetricAttr = "SyncFile"
	TOOMANYOPENFILESAttr             MetricAttr = "TOO_MANY_OPEN_FILES"
	UnlinkAttr                       MetricAttr = "Unlink"
	UpdateObjectAttr                 MetricAttr = "UpdateObject"
	WriteFileAttr                    MetricAttr = "WriteFile"
)

// MetricHandle provides an interface for recording metrics.
// The methods of this interface are auto-generated from metrics.yaml.
// Each method corresponds to a metric defined in metrics.yaml.
type MetricHandle interface {
	// BufferedReadDownloadBlockLatency - The cumulative distribution of block download latencies, along with status: successful, cancelled, or failed.
	BufferedReadDownloadBlockLatency(ctx context.Context, duration time.Duration, status MetricAttr)

	// BufferedReadFallbackTriggerCount - The cumulative number of times the BufferedReader falls back to a different reader, along with the reason: random_read_detected or insufficient_memory.
	BufferedReadFallbackTriggerCount(inc int64, reason MetricAttr)

	// BufferedReadReadLatency - The cumulative distribution of latencies for ReadAt calls served by the buffered reader.
	BufferedReadReadLatency(ctx context.Context, duration time.Duration)

	// BufferedReadScheduledBlockCount - The cumulative number of scheduled download blocks, along with their final status: successful, cancelled, or failed.
	BufferedReadScheduledBlockCount(inc int64, status MetricAttr)

	// FileCacheReadBytesCount - The cumulative number of bytes read from file cache along with read type - Sequential/Random
	FileCacheReadBytesCount(inc int64, readType MetricAttr)

	// FileCacheReadCount - Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false
	FileCacheReadCount(inc int64, cacheHit bool, readType MetricAttr)

	// FileCacheReadLatencies - The cumulative distribution of the file cache read latencies along with cache hit - true/false.
	FileCacheReadLatencies(ctx context.Context, duration time.Duration, cacheHit bool)

	// FsOpsCount - The cumulative number of ops processed by the file system.
	FsOpsCount(inc int64, fsOp MetricAttr)

	// FsOpsErrorCount - The cumulative number of errors generated by file system operations.
	FsOpsErrorCount(inc int64, fsErrorCategory MetricAttr, fsOp MetricAttr)

	// FsOpsLatency - The cumulative distribution of file system operation latencies
	FsOpsLatency(ctx context.Context, duration time.Duration, fsOp MetricAttr)

	// GcsDownloadBytesCount - The cumulative number of bytes downloaded from GCS along with type - Sequential/Random
	GcsDownloadBytesCount(inc int64, readType MetricAttr)

	// GcsReadBytesCount - The cumulative number of bytes read from GCS objects.
	GcsReadBytesCount(inc int64)

	// GcsReadCount - Specifies the number of gcs reads made along with type - Sequential/Random
	GcsReadCount(inc int64, readType MetricAttr)

	// GcsReaderCount - The cumulative number of GCS object readers opened or closed.
	GcsReaderCount(inc int64, ioMethod MetricAttr)

	// GcsRequestCount - The cumulative number of GCS requests processed along with the GCS method.
	GcsRequestCount(inc int64, gcsMethod MetricAttr)

	// GcsRequestLatencies - The cumulative distribution of the GCS request latencies.
	GcsRequestLatencies(ctx context.Context, duration time.Duration, gcsMethod MetricAttr)

	// GcsRetryCount - The cumulative number of retry requests made to GCS.
	GcsRetryCount(inc int64, retryErrorCategory MetricAttr)

	// TestUpdownCounter - Test metric for updown counters.
	TestUpdownCounter(inc int64)

	// TestUpdownCounterWithAttrs - Test metric for updown counters with attributes.
	TestUpdownCounterWithAttrs(inc int64, requestType MetricAttr)
}
