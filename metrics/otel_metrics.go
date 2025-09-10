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
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const logInterval = 5 * time.Minute

var (
	unrecognizedAttr                                                                    atomic.Value
	bufferedReadDownloadBlockLatencyStatusCancelledAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", "cancelled")))
	bufferedReadDownloadBlockLatencyStatusFailedAttrSet                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", "failed")))
	bufferedReadDownloadBlockLatencyStatusSuccessfulAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", "successful")))
	bufferedReadFallbackTriggerCountReasonInsufficientMemoryAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("reason", "insufficient_memory")))
	bufferedReadFallbackTriggerCountReasonRandomReadDetectedAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("reason", "random_read_detected")))
	bufferedReadScheduledBlockCountStatusCancelledAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", "cancelled")))
	bufferedReadScheduledBlockCountStatusFailedAttrSet                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", "failed")))
	bufferedReadScheduledBlockCountStatusSuccessfulAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", "successful")))
	fileCacheReadBytesCountReadTypeParallelAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Parallel")))
	fileCacheReadBytesCountReadTypeRandomAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Random")))
	fileCacheReadBytesCountReadTypeSequentialAttrSet                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Sequential")))
	fileCacheReadCountCacheHitTrueReadTypeParallelAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")))
	fileCacheReadCountCacheHitTrueReadTypeRandomAttrSet                                 = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")))
	fileCacheReadCountCacheHitTrueReadTypeSequentialAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Sequential")))
	fileCacheReadCountCacheHitFalseReadTypeParallelAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Parallel")))
	fileCacheReadCountCacheHitFalseReadTypeRandomAttrSet                                = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Random")))
	fileCacheReadCountCacheHitFalseReadTypeSequentialAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Sequential")))
	fileCacheReadLatenciesCacheHitTrueAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true)))
	fileCacheReadLatenciesCacheHitFalseAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false)))
	fsOpsCountFsOpBatchForgetAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "BatchForget")))
	fsOpsCountFsOpCreateFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateFile")))
	fsOpsCountFsOpCreateLinkAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateLink")))
	fsOpsCountFsOpCreateSymlinkAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateSymlink")))
	fsOpsCountFsOpFallocateAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Fallocate")))
	fsOpsCountFsOpFlushFileAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "FlushFile")))
	fsOpsCountFsOpForgetInodeAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ForgetInode")))
	fsOpsCountFsOpGetInodeAttributesAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsCountFsOpGetXattrAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "GetXattr")))
	fsOpsCountFsOpListXattrAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ListXattr")))
	fsOpsCountFsOpLookUpInodeAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "LookUpInode")))
	fsOpsCountFsOpMkDirAttrSet                                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkDir")))
	fsOpsCountFsOpMkNodeAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkNode")))
	fsOpsCountFsOpOpenDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenDir")))
	fsOpsCountFsOpOpenFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenFile")))
	fsOpsCountFsOpReadDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDir")))
	fsOpsCountFsOpReadDirPlusAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")))
	fsOpsCountFsOpReadFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadFile")))
	fsOpsCountFsOpReadSymlinkAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadSymlink")))
	fsOpsCountFsOpReleaseDirHandleAttrSet                                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsCountFsOpReleaseFileHandleAttrSet                                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsCountFsOpRemoveXattrAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "RemoveXattr")))
	fsOpsCountFsOpRenameAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Rename")))
	fsOpsCountFsOpRmDirAttrSet                                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "RmDir")))
	fsOpsCountFsOpSetInodeAttributesAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsCountFsOpSetXattrAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SetXattr")))
	fsOpsCountFsOpStatFSAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "StatFS")))
	fsOpsCountFsOpSyncFSAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SyncFS")))
	fsOpsCountFsOpSyncFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SyncFile")))
	fsOpsCountFsOpUnlinkAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Unlink")))
	fsOpsCountFsOpWriteFileAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAttrSet      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAttrSet   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAttrSet  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Fallocate")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetXattr")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ListXattr")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RemoveXattr")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetXattr")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "StatFS")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFS")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "WriteFile")))
	fsOpsLatencyFsOpBatchForgetAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "BatchForget")))
	fsOpsLatencyFsOpCreateFileAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateFile")))
	fsOpsLatencyFsOpCreateLinkAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateLink")))
	fsOpsLatencyFsOpCreateSymlinkAttrSet                                                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateSymlink")))
	fsOpsLatencyFsOpFallocateAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Fallocate")))
	fsOpsLatencyFsOpFlushFileAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "FlushFile")))
	fsOpsLatencyFsOpForgetInodeAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ForgetInode")))
	fsOpsLatencyFsOpGetInodeAttributesAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsLatencyFsOpGetXattrAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "GetXattr")))
	fsOpsLatencyFsOpListXattrAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ListXattr")))
	fsOpsLatencyFsOpLookUpInodeAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "LookUpInode")))
	fsOpsLatencyFsOpMkDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkDir")))
	fsOpsLatencyFsOpMkNodeAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkNode")))
	fsOpsLatencyFsOpOpenDirAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenDir")))
	fsOpsLatencyFsOpOpenFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenFile")))
	fsOpsLatencyFsOpReadDirAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDir")))
	fsOpsLatencyFsOpReadDirPlusAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")))
	fsOpsLatencyFsOpReadFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadFile")))
	fsOpsLatencyFsOpReadSymlinkAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadSymlink")))
	fsOpsLatencyFsOpReleaseDirHandleAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsLatencyFsOpReleaseFileHandleAttrSet                                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsLatencyFsOpRemoveXattrAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "RemoveXattr")))
	fsOpsLatencyFsOpRenameAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Rename")))
	fsOpsLatencyFsOpRmDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "RmDir")))
	fsOpsLatencyFsOpSetInodeAttributesAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsLatencyFsOpSetXattrAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SetXattr")))
	fsOpsLatencyFsOpStatFSAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "StatFS")))
	fsOpsLatencyFsOpSyncFSAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SyncFS")))
	fsOpsLatencyFsOpSyncFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SyncFile")))
	fsOpsLatencyFsOpUnlinkAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Unlink")))
	fsOpsLatencyFsOpWriteFileAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "WriteFile")))
	gcsDownloadBytesCountReadTypeParallelAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Parallel")))
	gcsDownloadBytesCountReadTypeRandomAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Random")))
	gcsDownloadBytesCountReadTypeSequentialAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Sequential")))
	gcsReadCountReadTypeParallelAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Parallel")))
	gcsReadCountReadTypeRandomAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Random")))
	gcsReadCountReadTypeSequentialAttrSet                                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Sequential")))
	gcsReaderCountIoMethodReadHandleAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", "ReadHandle")))
	gcsReaderCountIoMethodClosedAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", "closed")))
	gcsReaderCountIoMethodOpenedAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", "opened")))
	gcsRequestCountGcsMethodComposeObjectsAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")))
	gcsRequestCountGcsMethodCopyObjectAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CopyObject")))
	gcsRequestCountGcsMethodCreateAppendableObjectWriterAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateAppendableObjectWriter")))
	gcsRequestCountGcsMethodCreateFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateFolder")))
	gcsRequestCountGcsMethodCreateObjectAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateObject")))
	gcsRequestCountGcsMethodCreateObjectChunkWriterAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")))
	gcsRequestCountGcsMethodDeleteFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")))
	gcsRequestCountGcsMethodDeleteObjectAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteObject")))
	gcsRequestCountGcsMethodFinalizeUploadAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")))
	gcsRequestCountGcsMethodFlushPendingWritesAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "FlushPendingWrites")))
	gcsRequestCountGcsMethodGetFolderAttrSet                                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "GetFolder")))
	gcsRequestCountGcsMethodListObjectsAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "ListObjects")))
	gcsRequestCountGcsMethodMoveObjectAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "MoveObject")))
	gcsRequestCountGcsMethodMultiRangeDownloaderAddAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "MultiRangeDownloader::Add")))
	gcsRequestCountGcsMethodNewMultiRangeDownloaderAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "NewMultiRangeDownloader")))
	gcsRequestCountGcsMethodNewReaderAttrSet                                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "NewReader")))
	gcsRequestCountGcsMethodRenameFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "RenameFolder")))
	gcsRequestCountGcsMethodStatObjectAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "StatObject")))
	gcsRequestCountGcsMethodUpdateObjectAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "UpdateObject")))
	gcsRequestLatenciesGcsMethodComposeObjectsAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")))
	gcsRequestLatenciesGcsMethodCopyObjectAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CopyObject")))
	gcsRequestLatenciesGcsMethodCreateAppendableObjectWriterAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateAppendableObjectWriter")))
	gcsRequestLatenciesGcsMethodCreateFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateFolder")))
	gcsRequestLatenciesGcsMethodCreateObjectAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateObject")))
	gcsRequestLatenciesGcsMethodCreateObjectChunkWriterAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")))
	gcsRequestLatenciesGcsMethodDeleteFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")))
	gcsRequestLatenciesGcsMethodDeleteObjectAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteObject")))
	gcsRequestLatenciesGcsMethodFinalizeUploadAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")))
	gcsRequestLatenciesGcsMethodFlushPendingWritesAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "FlushPendingWrites")))
	gcsRequestLatenciesGcsMethodGetFolderAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "GetFolder")))
	gcsRequestLatenciesGcsMethodListObjectsAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "ListObjects")))
	gcsRequestLatenciesGcsMethodMoveObjectAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "MoveObject")))
	gcsRequestLatenciesGcsMethodMultiRangeDownloaderAddAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "MultiRangeDownloader::Add")))
	gcsRequestLatenciesGcsMethodNewMultiRangeDownloaderAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "NewMultiRangeDownloader")))
	gcsRequestLatenciesGcsMethodNewReaderAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "NewReader")))
	gcsRequestLatenciesGcsMethodRenameFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "RenameFolder")))
	gcsRequestLatenciesGcsMethodStatObjectAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "StatObject")))
	gcsRequestLatenciesGcsMethodUpdateObjectAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "UpdateObject")))
	gcsRetryCountRetryErrorCategoryOTHERERRORSAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")))
	gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")))
)

type histogramRecord struct {
	ctx        context.Context
	instrument metric.Int64Histogram
	value      int64
	attributes metric.RecordOption
}

type otelMetrics struct {
	ch                                                                                 chan histogramRecord
	wg                                                                                 *sync.WaitGroup
	bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic                     *atomic.Int64
	bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic                     *atomic.Int64
	bufferedReadScheduledBlockCountStatusCancelledAtomic                               *atomic.Int64
	bufferedReadScheduledBlockCountStatusFailedAtomic                                  *atomic.Int64
	bufferedReadScheduledBlockCountStatusSuccessfulAtomic                              *atomic.Int64
	fileCacheReadBytesCountReadTypeParallelAtomic                                      *atomic.Int64
	fileCacheReadBytesCountReadTypeRandomAtomic                                        *atomic.Int64
	fileCacheReadBytesCountReadTypeSequentialAtomic                                    *atomic.Int64
	fileCacheReadCountCacheHitTrueReadTypeParallelAtomic                               *atomic.Int64
	fileCacheReadCountCacheHitTrueReadTypeRandomAtomic                                 *atomic.Int64
	fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic                             *atomic.Int64
	fileCacheReadCountCacheHitFalseReadTypeParallelAtomic                              *atomic.Int64
	fileCacheReadCountCacheHitFalseReadTypeRandomAtomic                                *atomic.Int64
	fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic                            *atomic.Int64
	fsOpsCountFsOpBatchForgetAtomic                                                    *atomic.Int64
	fsOpsCountFsOpCreateFileAtomic                                                     *atomic.Int64
	fsOpsCountFsOpCreateLinkAtomic                                                     *atomic.Int64
	fsOpsCountFsOpCreateSymlinkAtomic                                                  *atomic.Int64
	fsOpsCountFsOpFallocateAtomic                                                      *atomic.Int64
	fsOpsCountFsOpFlushFileAtomic                                                      *atomic.Int64
	fsOpsCountFsOpForgetInodeAtomic                                                    *atomic.Int64
	fsOpsCountFsOpGetInodeAttributesAtomic                                             *atomic.Int64
	fsOpsCountFsOpGetXattrAtomic                                                       *atomic.Int64
	fsOpsCountFsOpListXattrAtomic                                                      *atomic.Int64
	fsOpsCountFsOpLookUpInodeAtomic                                                    *atomic.Int64
	fsOpsCountFsOpMkDirAtomic                                                          *atomic.Int64
	fsOpsCountFsOpMkNodeAtomic                                                         *atomic.Int64
	fsOpsCountFsOpOpenDirAtomic                                                        *atomic.Int64
	fsOpsCountFsOpOpenFileAtomic                                                       *atomic.Int64
	fsOpsCountFsOpReadDirAtomic                                                        *atomic.Int64
	fsOpsCountFsOpReadDirPlusAtomic                                                    *atomic.Int64
	fsOpsCountFsOpReadFileAtomic                                                       *atomic.Int64
	fsOpsCountFsOpReadSymlinkAtomic                                                    *atomic.Int64
	fsOpsCountFsOpReleaseDirHandleAtomic                                               *atomic.Int64
	fsOpsCountFsOpReleaseFileHandleAtomic                                              *atomic.Int64
	fsOpsCountFsOpRemoveXattrAtomic                                                    *atomic.Int64
	fsOpsCountFsOpRenameAtomic                                                         *atomic.Int64
	fsOpsCountFsOpRmDirAtomic                                                          *atomic.Int64
	fsOpsCountFsOpSetInodeAttributesAtomic                                             *atomic.Int64
	fsOpsCountFsOpSetXattrAtomic                                                       *atomic.Int64
	fsOpsCountFsOpStatFSAtomic                                                         *atomic.Int64
	fsOpsCountFsOpSyncFSAtomic                                                         *atomic.Int64
	fsOpsCountFsOpSyncFileAtomic                                                       *atomic.Int64
	fsOpsCountFsOpUnlinkAtomic                                                         *atomic.Int64
	fsOpsCountFsOpWriteFileAtomic                                                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic                  *atomic.Int64
	gcsDownloadBytesCountReadTypeParallelAtomic                                        *atomic.Int64
	gcsDownloadBytesCountReadTypeRandomAtomic                                          *atomic.Int64
	gcsDownloadBytesCountReadTypeSequentialAtomic                                      *atomic.Int64
	gcsReadBytesCountAtomic                                                            *atomic.Int64
	gcsReadCountReadTypeParallelAtomic                                                 *atomic.Int64
	gcsReadCountReadTypeRandomAtomic                                                   *atomic.Int64
	gcsReadCountReadTypeSequentialAtomic                                               *atomic.Int64
	gcsReaderCountIoMethodReadHandleAtomic                                             *atomic.Int64
	gcsReaderCountIoMethodClosedAtomic                                                 *atomic.Int64
	gcsReaderCountIoMethodOpenedAtomic                                                 *atomic.Int64
	gcsRequestCountGcsMethodComposeObjectsAtomic                                       *atomic.Int64
	gcsRequestCountGcsMethodCopyObjectAtomic                                           *atomic.Int64
	gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic                         *atomic.Int64
	gcsRequestCountGcsMethodCreateFolderAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodCreateObjectAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic                              *atomic.Int64
	gcsRequestCountGcsMethodDeleteFolderAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodDeleteObjectAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodFinalizeUploadAtomic                                       *atomic.Int64
	gcsRequestCountGcsMethodFlushPendingWritesAtomic                                   *atomic.Int64
	gcsRequestCountGcsMethodGetFolderAtomic                                            *atomic.Int64
	gcsRequestCountGcsMethodListObjectsAtomic                                          *atomic.Int64
	gcsRequestCountGcsMethodMoveObjectAtomic                                           *atomic.Int64
	gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic                              *atomic.Int64
	gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic                              *atomic.Int64
	gcsRequestCountGcsMethodNewReaderAtomic                                            *atomic.Int64
	gcsRequestCountGcsMethodRenameFolderAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodStatObjectAtomic                                           *atomic.Int64
	gcsRequestCountGcsMethodUpdateObjectAtomic                                         *atomic.Int64
	gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic                                   *atomic.Int64
	gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic                            *atomic.Int64
	bufferedReadDownloadBlockLatency                                                   metric.Int64Histogram
	bufferedReadReadLatency                                                            metric.Int64Histogram
	fileCacheReadLatencies                                                             metric.Int64Histogram
	fsOpsLatency                                                                       metric.Int64Histogram
	gcsRequestLatencies                                                                metric.Int64Histogram
}

func (o *otelMetrics) BufferedReadDownloadBlockLatency(
	ctx context.Context, latency time.Duration, status string) {
	var record histogramRecord
	switch status {
	case "cancelled":
		record = histogramRecord{ctx: ctx, instrument: o.bufferedReadDownloadBlockLatency, value: latency.Microseconds(), attributes: bufferedReadDownloadBlockLatencyStatusCancelledAttrSet}
	case "failed":
		record = histogramRecord{ctx: ctx, instrument: o.bufferedReadDownloadBlockLatency, value: latency.Microseconds(), attributes: bufferedReadDownloadBlockLatencyStatusFailedAttrSet}
	case "successful":
		record = histogramRecord{ctx: ctx, instrument: o.bufferedReadDownloadBlockLatency, value: latency.Microseconds(), attributes: bufferedReadDownloadBlockLatencyStatusSuccessfulAttrSet}
	default:
		updateUnrecognizedAttribute(status)
		return
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) BufferedReadFallbackTriggerCount(
	inc int64, reason string) {
	if inc < 0 {
		logger.Errorf("Counter metric buffered_read/fallback_trigger_count received a negative increment: %d", inc)
		return
	}
	switch reason {
	case "insufficient_memory":
		o.bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic.Add(inc)
	case "random_read_detected":
		o.bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(reason)
		return
	}

}

func (o *otelMetrics) BufferedReadReadLatency(
	ctx context.Context, latency time.Duration) {
	var record histogramRecord
	record = histogramRecord{ctx: ctx, instrument: o.bufferedReadReadLatency, value: latency.Microseconds()}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) BufferedReadScheduledBlockCount(
	inc int64, status string) {
	if inc < 0 {
		logger.Errorf("Counter metric buffered_read/scheduled_block_count received a negative increment: %d", inc)
		return
	}
	switch status {
	case "cancelled":
		o.bufferedReadScheduledBlockCountStatusCancelledAtomic.Add(inc)
	case "failed":
		o.bufferedReadScheduledBlockCountStatusFailedAtomic.Add(inc)
	case "successful":
		o.bufferedReadScheduledBlockCountStatusSuccessfulAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(status)
		return
	}

}

func (o *otelMetrics) FileCacheReadBytesCount(
	inc int64, readType string) {
	if inc < 0 {
		logger.Errorf("Counter metric file_cache/read_bytes_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case "Parallel":
		o.fileCacheReadBytesCountReadTypeParallelAtomic.Add(inc)
	case "Random":
		o.fileCacheReadBytesCountReadTypeRandomAtomic.Add(inc)
	case "Sequential":
		o.fileCacheReadBytesCountReadTypeSequentialAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(readType)
		return
	}

}

func (o *otelMetrics) FileCacheReadCount(
	inc int64, cacheHit bool, readType string) {
	if inc < 0 {
		logger.Errorf("Counter metric file_cache/read_count received a negative increment: %d", inc)
		return
	}
	switch cacheHit {
	case true:
		switch readType {
		case "Parallel":
			o.fileCacheReadCountCacheHitTrueReadTypeParallelAtomic.Add(inc)
		case "Random":
			o.fileCacheReadCountCacheHitTrueReadTypeRandomAtomic.Add(inc)
		case "Sequential":
			o.fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(readType)
			return
		}
	case false:
		switch readType {
		case "Parallel":
			o.fileCacheReadCountCacheHitFalseReadTypeParallelAtomic.Add(inc)
		case "Random":
			o.fileCacheReadCountCacheHitFalseReadTypeRandomAtomic.Add(inc)
		case "Sequential":
			o.fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(readType)
			return
		}
	}

}

func (o *otelMetrics) FileCacheReadLatencies(
	ctx context.Context, latency time.Duration, cacheHit bool) {
	var record histogramRecord
	switch cacheHit {
	case true:
		record = histogramRecord{ctx: ctx, instrument: o.fileCacheReadLatencies, value: latency.Microseconds(), attributes: fileCacheReadLatenciesCacheHitTrueAttrSet}
	case false:
		record = histogramRecord{ctx: ctx, instrument: o.fileCacheReadLatencies, value: latency.Microseconds(), attributes: fileCacheReadLatenciesCacheHitFalseAttrSet}
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) FsOpsCount(
	inc int64, fsOp string) {
	if inc < 0 {
		logger.Errorf("Counter metric fs/ops_count received a negative increment: %d", inc)
		return
	}
	switch fsOp {
	case "BatchForget":
		o.fsOpsCountFsOpBatchForgetAtomic.Add(inc)
	case "CreateFile":
		o.fsOpsCountFsOpCreateFileAtomic.Add(inc)
	case "CreateLink":
		o.fsOpsCountFsOpCreateLinkAtomic.Add(inc)
	case "CreateSymlink":
		o.fsOpsCountFsOpCreateSymlinkAtomic.Add(inc)
	case "Fallocate":
		o.fsOpsCountFsOpFallocateAtomic.Add(inc)
	case "FlushFile":
		o.fsOpsCountFsOpFlushFileAtomic.Add(inc)
	case "ForgetInode":
		o.fsOpsCountFsOpForgetInodeAtomic.Add(inc)
	case "GetInodeAttributes":
		o.fsOpsCountFsOpGetInodeAttributesAtomic.Add(inc)
	case "GetXattr":
		o.fsOpsCountFsOpGetXattrAtomic.Add(inc)
	case "ListXattr":
		o.fsOpsCountFsOpListXattrAtomic.Add(inc)
	case "LookUpInode":
		o.fsOpsCountFsOpLookUpInodeAtomic.Add(inc)
	case "MkDir":
		o.fsOpsCountFsOpMkDirAtomic.Add(inc)
	case "MkNode":
		o.fsOpsCountFsOpMkNodeAtomic.Add(inc)
	case "OpenDir":
		o.fsOpsCountFsOpOpenDirAtomic.Add(inc)
	case "OpenFile":
		o.fsOpsCountFsOpOpenFileAtomic.Add(inc)
	case "ReadDir":
		o.fsOpsCountFsOpReadDirAtomic.Add(inc)
	case "ReadDirPlus":
		o.fsOpsCountFsOpReadDirPlusAtomic.Add(inc)
	case "ReadFile":
		o.fsOpsCountFsOpReadFileAtomic.Add(inc)
	case "ReadSymlink":
		o.fsOpsCountFsOpReadSymlinkAtomic.Add(inc)
	case "ReleaseDirHandle":
		o.fsOpsCountFsOpReleaseDirHandleAtomic.Add(inc)
	case "ReleaseFileHandle":
		o.fsOpsCountFsOpReleaseFileHandleAtomic.Add(inc)
	case "RemoveXattr":
		o.fsOpsCountFsOpRemoveXattrAtomic.Add(inc)
	case "Rename":
		o.fsOpsCountFsOpRenameAtomic.Add(inc)
	case "RmDir":
		o.fsOpsCountFsOpRmDirAtomic.Add(inc)
	case "SetInodeAttributes":
		o.fsOpsCountFsOpSetInodeAttributesAtomic.Add(inc)
	case "SetXattr":
		o.fsOpsCountFsOpSetXattrAtomic.Add(inc)
	case "StatFS":
		o.fsOpsCountFsOpStatFSAtomic.Add(inc)
	case "SyncFS":
		o.fsOpsCountFsOpSyncFSAtomic.Add(inc)
	case "SyncFile":
		o.fsOpsCountFsOpSyncFileAtomic.Add(inc)
	case "Unlink":
		o.fsOpsCountFsOpUnlinkAtomic.Add(inc)
	case "WriteFile":
		o.fsOpsCountFsOpWriteFileAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(fsOp)
		return
	}

}

func (o *otelMetrics) FsOpsErrorCount(
	inc int64, fsErrorCategory string, fsOp string) {
	if inc < 0 {
		logger.Errorf("Counter metric fs/ops_error_count received a negative increment: %d", inc)
		return
	}
	switch fsErrorCategory {
	case "DEVICE_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "DIR_NOT_EMPTY":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "FILE_DIR_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "FILE_EXISTS":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "INTERRUPT_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "INVALID_ARGUMENT":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "INVALID_OPERATION":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "IO_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "MISC_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "NETWORK_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "NOT_A_DIR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "NOT_IMPLEMENTED":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "NO_FILE_OR_DIR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "PERM_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "PROCESS_RESOURCE_MGMT_ERROR":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	case "TOO_MANY_OPEN_FILES":
		switch fsOp {
		case "BatchForget":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic.Add(inc)
		case "CreateFile":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic.Add(inc)
		case "CreateLink":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic.Add(inc)
		case "CreateSymlink":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic.Add(inc)
		case "Fallocate":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic.Add(inc)
		case "FlushFile":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic.Add(inc)
		case "ForgetInode":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic.Add(inc)
		case "GetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic.Add(inc)
		case "GetXattr":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic.Add(inc)
		case "ListXattr":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic.Add(inc)
		case "LookUpInode":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic.Add(inc)
		case "MkDir":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic.Add(inc)
		case "MkNode":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic.Add(inc)
		case "OpenDir":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic.Add(inc)
		case "OpenFile":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic.Add(inc)
		case "ReadDir":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic.Add(inc)
		case "ReadDirPlus":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic.Add(inc)
		case "ReadFile":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic.Add(inc)
		case "ReadSymlink":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic.Add(inc)
		case "ReleaseDirHandle":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic.Add(inc)
		case "ReleaseFileHandle":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic.Add(inc)
		case "RemoveXattr":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic.Add(inc)
		case "Rename":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic.Add(inc)
		case "RmDir":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic.Add(inc)
		case "SetInodeAttributes":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic.Add(inc)
		case "SetXattr":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic.Add(inc)
		case "StatFS":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic.Add(inc)
		case "SyncFS":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic.Add(inc)
		case "SyncFile":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic.Add(inc)
		case "Unlink":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic.Add(inc)
		case "WriteFile":
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(fsOp)
			return
		}
	default:
		updateUnrecognizedAttribute(fsErrorCategory)
		return
	}

}

func (o *otelMetrics) FsOpsLatency(
	ctx context.Context, latency time.Duration, fsOp string) {
	var record histogramRecord
	switch fsOp {
	case "BatchForget":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpBatchForgetAttrSet}
	case "CreateFile":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateFileAttrSet}
	case "CreateLink":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateLinkAttrSet}
	case "CreateSymlink":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateSymlinkAttrSet}
	case "Fallocate":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpFallocateAttrSet}
	case "FlushFile":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpFlushFileAttrSet}
	case "ForgetInode":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpForgetInodeAttrSet}
	case "GetInodeAttributes":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpGetInodeAttributesAttrSet}
	case "GetXattr":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpGetXattrAttrSet}
	case "ListXattr":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpListXattrAttrSet}
	case "LookUpInode":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpLookUpInodeAttrSet}
	case "MkDir":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkDirAttrSet}
	case "MkNode":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkNodeAttrSet}
	case "OpenDir":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenDirAttrSet}
	case "OpenFile":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenFileAttrSet}
	case "ReadDir":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadDirAttrSet}
	case "ReadDirPlus":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadDirPlusAttrSet}
	case "ReadFile":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadFileAttrSet}
	case "ReadSymlink":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadSymlinkAttrSet}
	case "ReleaseDirHandle":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseDirHandleAttrSet}
	case "ReleaseFileHandle":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseFileHandleAttrSet}
	case "RemoveXattr":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRemoveXattrAttrSet}
	case "Rename":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRenameAttrSet}
	case "RmDir":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRmDirAttrSet}
	case "SetInodeAttributes":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSetInodeAttributesAttrSet}
	case "SetXattr":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSetXattrAttrSet}
	case "StatFS":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpStatFSAttrSet}
	case "SyncFS":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSyncFSAttrSet}
	case "SyncFile":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSyncFileAttrSet}
	case "Unlink":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpUnlinkAttrSet}
	case "WriteFile":
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpWriteFileAttrSet}
	default:
		updateUnrecognizedAttribute(fsOp)
		return
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) GcsDownloadBytesCount(
	inc int64, readType string) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/download_bytes_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case "Parallel":
		o.gcsDownloadBytesCountReadTypeParallelAtomic.Add(inc)
	case "Random":
		o.gcsDownloadBytesCountReadTypeRandomAtomic.Add(inc)
	case "Sequential":
		o.gcsDownloadBytesCountReadTypeSequentialAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(readType)
		return
	}

}

func (o *otelMetrics) GcsReadBytesCount(
	inc int64) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/read_bytes_count received a negative increment: %d", inc)
		return
	}
	o.gcsReadBytesCountAtomic.Add(inc)

}

func (o *otelMetrics) GcsReadCount(
	inc int64, readType string) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/read_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case "Parallel":
		o.gcsReadCountReadTypeParallelAtomic.Add(inc)
	case "Random":
		o.gcsReadCountReadTypeRandomAtomic.Add(inc)
	case "Sequential":
		o.gcsReadCountReadTypeSequentialAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(readType)
		return
	}

}

func (o *otelMetrics) GcsReaderCount(
	inc int64, ioMethod string) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/reader_count received a negative increment: %d", inc)
		return
	}
	switch ioMethod {
	case "ReadHandle":
		o.gcsReaderCountIoMethodReadHandleAtomic.Add(inc)
	case "closed":
		o.gcsReaderCountIoMethodClosedAtomic.Add(inc)
	case "opened":
		o.gcsReaderCountIoMethodOpenedAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(ioMethod)
		return
	}

}

func (o *otelMetrics) GcsRequestCount(
	inc int64, gcsMethod string) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/request_count received a negative increment: %d", inc)
		return
	}
	switch gcsMethod {
	case "ComposeObjects":
		o.gcsRequestCountGcsMethodComposeObjectsAtomic.Add(inc)
	case "CopyObject":
		o.gcsRequestCountGcsMethodCopyObjectAtomic.Add(inc)
	case "CreateAppendableObjectWriter":
		o.gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic.Add(inc)
	case "CreateFolder":
		o.gcsRequestCountGcsMethodCreateFolderAtomic.Add(inc)
	case "CreateObject":
		o.gcsRequestCountGcsMethodCreateObjectAtomic.Add(inc)
	case "CreateObjectChunkWriter":
		o.gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic.Add(inc)
	case "DeleteFolder":
		o.gcsRequestCountGcsMethodDeleteFolderAtomic.Add(inc)
	case "DeleteObject":
		o.gcsRequestCountGcsMethodDeleteObjectAtomic.Add(inc)
	case "FinalizeUpload":
		o.gcsRequestCountGcsMethodFinalizeUploadAtomic.Add(inc)
	case "FlushPendingWrites":
		o.gcsRequestCountGcsMethodFlushPendingWritesAtomic.Add(inc)
	case "GetFolder":
		o.gcsRequestCountGcsMethodGetFolderAtomic.Add(inc)
	case "ListObjects":
		o.gcsRequestCountGcsMethodListObjectsAtomic.Add(inc)
	case "MoveObject":
		o.gcsRequestCountGcsMethodMoveObjectAtomic.Add(inc)
	case "MultiRangeDownloader::Add":
		o.gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic.Add(inc)
	case "NewMultiRangeDownloader":
		o.gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic.Add(inc)
	case "NewReader":
		o.gcsRequestCountGcsMethodNewReaderAtomic.Add(inc)
	case "RenameFolder":
		o.gcsRequestCountGcsMethodRenameFolderAtomic.Add(inc)
	case "StatObject":
		o.gcsRequestCountGcsMethodStatObjectAtomic.Add(inc)
	case "UpdateObject":
		o.gcsRequestCountGcsMethodUpdateObjectAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(gcsMethod)
		return
	}

}

func (o *otelMetrics) GcsRequestLatencies(
	ctx context.Context, latency time.Duration, gcsMethod string) {
	var record histogramRecord
	switch gcsMethod {
	case "ComposeObjects":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodComposeObjectsAttrSet}
	case "CopyObject":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCopyObjectAttrSet}
	case "CreateAppendableObjectWriter":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateAppendableObjectWriterAttrSet}
	case "CreateFolder":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateFolderAttrSet}
	case "CreateObject":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateObjectAttrSet}
	case "CreateObjectChunkWriter":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateObjectChunkWriterAttrSet}
	case "DeleteFolder":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteFolderAttrSet}
	case "DeleteObject":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteObjectAttrSet}
	case "FinalizeUpload":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodFinalizeUploadAttrSet}
	case "FlushPendingWrites":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodFlushPendingWritesAttrSet}
	case "GetFolder":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodGetFolderAttrSet}
	case "ListObjects":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodListObjectsAttrSet}
	case "MoveObject":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMoveObjectAttrSet}
	case "MultiRangeDownloader::Add":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMultiRangeDownloaderAddAttrSet}
	case "NewMultiRangeDownloader":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewMultiRangeDownloaderAttrSet}
	case "NewReader":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewReaderAttrSet}
	case "RenameFolder":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodRenameFolderAttrSet}
	case "StatObject":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodStatObjectAttrSet}
	case "UpdateObject":
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodUpdateObjectAttrSet}
	default:
		updateUnrecognizedAttribute(gcsMethod)
		return
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) GcsRetryCount(
	inc int64, retryErrorCategory string) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/retry_count received a negative increment: %d", inc)
		return
	}
	switch retryErrorCategory {
	case "OTHER_ERRORS":
		o.gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic.Add(inc)
	case "STALLED_READ_REQUEST":
		o.gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(retryErrorCategory)
		return
	}

}

func NewOTelMetrics(ctx context.Context, workers int, bufferSize int) (*otelMetrics, error) {
	ch := make(chan histogramRecord, bufferSize)
	var wg sync.WaitGroup
	startSampledLogging(ctx)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range ch {
				if record.attributes != nil {
					record.instrument.Record(record.ctx, record.value, record.attributes)
				} else {
					record.instrument.Record(record.ctx, record.value)
				}
			}
		}()
	}
	meter := otel.Meter("gcsfuse")

	var bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic,
		bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic atomic.Int64

	var bufferedReadScheduledBlockCountStatusCancelledAtomic,
		bufferedReadScheduledBlockCountStatusFailedAtomic,
		bufferedReadScheduledBlockCountStatusSuccessfulAtomic atomic.Int64

	var fileCacheReadBytesCountReadTypeParallelAtomic,
		fileCacheReadBytesCountReadTypeRandomAtomic,
		fileCacheReadBytesCountReadTypeSequentialAtomic atomic.Int64

	var fileCacheReadCountCacheHitTrueReadTypeParallelAtomic,
		fileCacheReadCountCacheHitTrueReadTypeRandomAtomic,
		fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic,
		fileCacheReadCountCacheHitFalseReadTypeParallelAtomic,
		fileCacheReadCountCacheHitFalseReadTypeRandomAtomic,
		fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic atomic.Int64

	var fsOpsCountFsOpBatchForgetAtomic,
		fsOpsCountFsOpCreateFileAtomic,
		fsOpsCountFsOpCreateLinkAtomic,
		fsOpsCountFsOpCreateSymlinkAtomic,
		fsOpsCountFsOpFallocateAtomic,
		fsOpsCountFsOpFlushFileAtomic,
		fsOpsCountFsOpForgetInodeAtomic,
		fsOpsCountFsOpGetInodeAttributesAtomic,
		fsOpsCountFsOpGetXattrAtomic,
		fsOpsCountFsOpListXattrAtomic,
		fsOpsCountFsOpLookUpInodeAtomic,
		fsOpsCountFsOpMkDirAtomic,
		fsOpsCountFsOpMkNodeAtomic,
		fsOpsCountFsOpOpenDirAtomic,
		fsOpsCountFsOpOpenFileAtomic,
		fsOpsCountFsOpReadDirAtomic,
		fsOpsCountFsOpReadDirPlusAtomic,
		fsOpsCountFsOpReadFileAtomic,
		fsOpsCountFsOpReadSymlinkAtomic,
		fsOpsCountFsOpReleaseDirHandleAtomic,
		fsOpsCountFsOpReleaseFileHandleAtomic,
		fsOpsCountFsOpRemoveXattrAtomic,
		fsOpsCountFsOpRenameAtomic,
		fsOpsCountFsOpRmDirAtomic,
		fsOpsCountFsOpSetInodeAttributesAtomic,
		fsOpsCountFsOpSetXattrAtomic,
		fsOpsCountFsOpStatFSAtomic,
		fsOpsCountFsOpSyncFSAtomic,
		fsOpsCountFsOpSyncFileAtomic,
		fsOpsCountFsOpUnlinkAtomic,
		fsOpsCountFsOpWriteFileAtomic atomic.Int64

	var fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic atomic.Int64

	var gcsDownloadBytesCountReadTypeParallelAtomic,
		gcsDownloadBytesCountReadTypeRandomAtomic,
		gcsDownloadBytesCountReadTypeSequentialAtomic atomic.Int64

	var gcsReadBytesCountAtomic atomic.Int64

	var gcsReadCountReadTypeParallelAtomic,
		gcsReadCountReadTypeRandomAtomic,
		gcsReadCountReadTypeSequentialAtomic atomic.Int64

	var gcsReaderCountIoMethodReadHandleAtomic,
		gcsReaderCountIoMethodClosedAtomic,
		gcsReaderCountIoMethodOpenedAtomic atomic.Int64

	var gcsRequestCountGcsMethodComposeObjectsAtomic,
		gcsRequestCountGcsMethodCopyObjectAtomic,
		gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic,
		gcsRequestCountGcsMethodCreateFolderAtomic,
		gcsRequestCountGcsMethodCreateObjectAtomic,
		gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic,
		gcsRequestCountGcsMethodDeleteFolderAtomic,
		gcsRequestCountGcsMethodDeleteObjectAtomic,
		gcsRequestCountGcsMethodFinalizeUploadAtomic,
		gcsRequestCountGcsMethodFlushPendingWritesAtomic,
		gcsRequestCountGcsMethodGetFolderAtomic,
		gcsRequestCountGcsMethodListObjectsAtomic,
		gcsRequestCountGcsMethodMoveObjectAtomic,
		gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic,
		gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic,
		gcsRequestCountGcsMethodNewReaderAtomic,
		gcsRequestCountGcsMethodRenameFolderAtomic,
		gcsRequestCountGcsMethodStatObjectAtomic,
		gcsRequestCountGcsMethodUpdateObjectAtomic atomic.Int64

	var gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic,
		gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic atomic.Int64

	bufferedReadDownloadBlockLatency, err0 := meter.Int64Histogram("buffered_read/download_block_latency",
		metric.WithDescription("The cumulative distribution of block download latencies, along with status: successful, cancelled, or failed."),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err1 := meter.Int64ObservableCounter("buffered_read/fallback_trigger_count",
		metric.WithDescription("The cumulative number of times the BufferedReader falls back to a different reader, along with the reason: random_read_detected or insufficient_memory."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic, bufferedReadFallbackTriggerCountReasonInsufficientMemoryAttrSet)
			conditionallyObserve(obsrv, &bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic, bufferedReadFallbackTriggerCountReasonRandomReadDetectedAttrSet)
			return nil
		}))

	bufferedReadReadLatency, err2 := meter.Int64Histogram("buffered_read/read_latency",
		metric.WithDescription("The cumulative distribution of latencies for ReadAt calls served by the buffered reader."),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err3 := meter.Int64ObservableCounter("buffered_read/scheduled_block_count",
		metric.WithDescription("The cumulative number of scheduled download blocks, along with their final status: successful, cancelled, or failed."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &bufferedReadScheduledBlockCountStatusCancelledAtomic, bufferedReadScheduledBlockCountStatusCancelledAttrSet)
			conditionallyObserve(obsrv, &bufferedReadScheduledBlockCountStatusFailedAtomic, bufferedReadScheduledBlockCountStatusFailedAttrSet)
			conditionallyObserve(obsrv, &bufferedReadScheduledBlockCountStatusSuccessfulAtomic, bufferedReadScheduledBlockCountStatusSuccessfulAttrSet)
			return nil
		}))

	_, err4 := meter.Int64ObservableCounter("file_cache/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from file cache along with read type - Sequential/Random"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fileCacheReadBytesCountReadTypeParallelAtomic, fileCacheReadBytesCountReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadBytesCountReadTypeRandomAtomic, fileCacheReadBytesCountReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadBytesCountReadTypeSequentialAtomic, fileCacheReadBytesCountReadTypeSequentialAttrSet)
			return nil
		}))

	_, err5 := meter.Int64ObservableCounter("file_cache/read_count",
		metric.WithDescription("Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false"),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitTrueReadTypeParallelAtomic, fileCacheReadCountCacheHitTrueReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitTrueReadTypeRandomAtomic, fileCacheReadCountCacheHitTrueReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic, fileCacheReadCountCacheHitTrueReadTypeSequentialAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitFalseReadTypeParallelAtomic, fileCacheReadCountCacheHitFalseReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitFalseReadTypeRandomAtomic, fileCacheReadCountCacheHitFalseReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic, fileCacheReadCountCacheHitFalseReadTypeSequentialAttrSet)
			return nil
		}))

	fileCacheReadLatencies, err6 := meter.Int64Histogram("file_cache/read_latencies",
		metric.WithDescription("The cumulative distribution of the file cache read latencies along with cache hit - true/false."),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err7 := meter.Int64ObservableCounter("fs/ops_count",
		metric.WithDescription("The cumulative number of ops processed by the file system."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fsOpsCountFsOpBatchForgetAtomic, fsOpsCountFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpCreateFileAtomic, fsOpsCountFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpCreateLinkAtomic, fsOpsCountFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpCreateSymlinkAtomic, fsOpsCountFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpFallocateAtomic, fsOpsCountFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpFlushFileAtomic, fsOpsCountFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpForgetInodeAtomic, fsOpsCountFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpGetInodeAttributesAtomic, fsOpsCountFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpGetXattrAtomic, fsOpsCountFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpListXattrAtomic, fsOpsCountFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpLookUpInodeAtomic, fsOpsCountFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpMkDirAtomic, fsOpsCountFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpMkNodeAtomic, fsOpsCountFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpOpenDirAtomic, fsOpsCountFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpOpenFileAtomic, fsOpsCountFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadDirAtomic, fsOpsCountFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadDirPlusAtomic, fsOpsCountFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadFileAtomic, fsOpsCountFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadSymlinkAtomic, fsOpsCountFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReleaseDirHandleAtomic, fsOpsCountFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReleaseFileHandleAtomic, fsOpsCountFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpRemoveXattrAtomic, fsOpsCountFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpRenameAtomic, fsOpsCountFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpRmDirAtomic, fsOpsCountFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpSetInodeAttributesAtomic, fsOpsCountFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpSetXattrAtomic, fsOpsCountFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpStatFSAtomic, fsOpsCountFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpSyncFSAtomic, fsOpsCountFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpSyncFileAtomic, fsOpsCountFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpUnlinkAtomic, fsOpsCountFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpWriteFileAtomic, fsOpsCountFsOpWriteFileAttrSet)
			return nil
		}))

	_, err8 := meter.Int64ObservableCounter("fs/ops_error_count",
		metric.WithDescription("The cumulative number of errors generated by file system operations."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAttrSet)
			return nil
		}))

	fsOpsLatency, err9 := meter.Int64Histogram("fs/ops_latency",
		metric.WithDescription("The cumulative distribution of file system operation latencies"),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err10 := meter.Int64ObservableCounter("gcs/download_bytes_count",
		metric.WithDescription("The cumulative number of bytes downloaded from GCS along with type - Sequential/Random"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsDownloadBytesCountReadTypeParallelAtomic, gcsDownloadBytesCountReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &gcsDownloadBytesCountReadTypeRandomAtomic, gcsDownloadBytesCountReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &gcsDownloadBytesCountReadTypeSequentialAtomic, gcsDownloadBytesCountReadTypeSequentialAttrSet)
			return nil
		}))

	_, err11 := meter.Int64ObservableCounter("gcs/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from GCS objects."),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsReadBytesCountAtomic)
			return nil
		}))

	_, err12 := meter.Int64ObservableCounter("gcs/read_count",
		metric.WithDescription("Specifies the number of gcs reads made along with type - Sequential/Random"),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsReadCountReadTypeParallelAtomic, gcsReadCountReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &gcsReadCountReadTypeRandomAtomic, gcsReadCountReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &gcsReadCountReadTypeSequentialAtomic, gcsReadCountReadTypeSequentialAttrSet)
			return nil
		}))

	_, err13 := meter.Int64ObservableCounter("gcs/reader_count",
		metric.WithDescription("The cumulative number of GCS object readers opened or closed."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsReaderCountIoMethodReadHandleAtomic, gcsReaderCountIoMethodReadHandleAttrSet)
			conditionallyObserve(obsrv, &gcsReaderCountIoMethodClosedAtomic, gcsReaderCountIoMethodClosedAttrSet)
			conditionallyObserve(obsrv, &gcsReaderCountIoMethodOpenedAtomic, gcsReaderCountIoMethodOpenedAttrSet)
			return nil
		}))

	_, err14 := meter.Int64ObservableCounter("gcs/request_count",
		metric.WithDescription("The cumulative number of GCS requests processed along with the GCS method."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodComposeObjectsAtomic, gcsRequestCountGcsMethodComposeObjectsAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodCopyObjectAtomic, gcsRequestCountGcsMethodCopyObjectAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic, gcsRequestCountGcsMethodCreateAppendableObjectWriterAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodCreateFolderAtomic, gcsRequestCountGcsMethodCreateFolderAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodCreateObjectAtomic, gcsRequestCountGcsMethodCreateObjectAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic, gcsRequestCountGcsMethodCreateObjectChunkWriterAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodDeleteFolderAtomic, gcsRequestCountGcsMethodDeleteFolderAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodDeleteObjectAtomic, gcsRequestCountGcsMethodDeleteObjectAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodFinalizeUploadAtomic, gcsRequestCountGcsMethodFinalizeUploadAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodFlushPendingWritesAtomic, gcsRequestCountGcsMethodFlushPendingWritesAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodGetFolderAtomic, gcsRequestCountGcsMethodGetFolderAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodListObjectsAtomic, gcsRequestCountGcsMethodListObjectsAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodMoveObjectAtomic, gcsRequestCountGcsMethodMoveObjectAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic, gcsRequestCountGcsMethodMultiRangeDownloaderAddAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic, gcsRequestCountGcsMethodNewMultiRangeDownloaderAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodNewReaderAtomic, gcsRequestCountGcsMethodNewReaderAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodRenameFolderAtomic, gcsRequestCountGcsMethodRenameFolderAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodStatObjectAtomic, gcsRequestCountGcsMethodStatObjectAttrSet)
			conditionallyObserve(obsrv, &gcsRequestCountGcsMethodUpdateObjectAtomic, gcsRequestCountGcsMethodUpdateObjectAttrSet)
			return nil
		}))

	gcsRequestLatencies, err15 := meter.Int64Histogram("gcs/request_latencies",
		metric.WithDescription("The cumulative distribution of the GCS request latencies."),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err16 := meter.Int64ObservableCounter("gcs/retry_count",
		metric.WithDescription("The cumulative number of retry requests made to GCS."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic, gcsRetryCountRetryErrorCategoryOTHERERRORSAttrSet)
			conditionallyObserve(obsrv, &gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic, gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAttrSet)
			return nil
		}))

	errs := []error{err0, err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12, err13, err14, err15, err16}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return &otelMetrics{
		ch:                               ch,
		wg:                               &wg,
		bufferedReadDownloadBlockLatency: bufferedReadDownloadBlockLatency,
		bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic: &bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic,
		bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic: &bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic,
		bufferedReadReadLatency:                                                            bufferedReadReadLatency,
		bufferedReadScheduledBlockCountStatusCancelledAtomic:                               &bufferedReadScheduledBlockCountStatusCancelledAtomic,
		bufferedReadScheduledBlockCountStatusFailedAtomic:                                  &bufferedReadScheduledBlockCountStatusFailedAtomic,
		bufferedReadScheduledBlockCountStatusSuccessfulAtomic:                              &bufferedReadScheduledBlockCountStatusSuccessfulAtomic,
		fileCacheReadBytesCountReadTypeParallelAtomic:                                      &fileCacheReadBytesCountReadTypeParallelAtomic,
		fileCacheReadBytesCountReadTypeRandomAtomic:                                        &fileCacheReadBytesCountReadTypeRandomAtomic,
		fileCacheReadBytesCountReadTypeSequentialAtomic:                                    &fileCacheReadBytesCountReadTypeSequentialAtomic,
		fileCacheReadCountCacheHitTrueReadTypeParallelAtomic:                               &fileCacheReadCountCacheHitTrueReadTypeParallelAtomic,
		fileCacheReadCountCacheHitTrueReadTypeRandomAtomic:                                 &fileCacheReadCountCacheHitTrueReadTypeRandomAtomic,
		fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic:                             &fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic,
		fileCacheReadCountCacheHitFalseReadTypeParallelAtomic:                              &fileCacheReadCountCacheHitFalseReadTypeParallelAtomic,
		fileCacheReadCountCacheHitFalseReadTypeRandomAtomic:                                &fileCacheReadCountCacheHitFalseReadTypeRandomAtomic,
		fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic:                            &fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic,
		fileCacheReadLatencies:                                                             fileCacheReadLatencies,
		fsOpsCountFsOpBatchForgetAtomic:                                                    &fsOpsCountFsOpBatchForgetAtomic,
		fsOpsCountFsOpCreateFileAtomic:                                                     &fsOpsCountFsOpCreateFileAtomic,
		fsOpsCountFsOpCreateLinkAtomic:                                                     &fsOpsCountFsOpCreateLinkAtomic,
		fsOpsCountFsOpCreateSymlinkAtomic:                                                  &fsOpsCountFsOpCreateSymlinkAtomic,
		fsOpsCountFsOpFallocateAtomic:                                                      &fsOpsCountFsOpFallocateAtomic,
		fsOpsCountFsOpFlushFileAtomic:                                                      &fsOpsCountFsOpFlushFileAtomic,
		fsOpsCountFsOpForgetInodeAtomic:                                                    &fsOpsCountFsOpForgetInodeAtomic,
		fsOpsCountFsOpGetInodeAttributesAtomic:                                             &fsOpsCountFsOpGetInodeAttributesAtomic,
		fsOpsCountFsOpGetXattrAtomic:                                                       &fsOpsCountFsOpGetXattrAtomic,
		fsOpsCountFsOpListXattrAtomic:                                                      &fsOpsCountFsOpListXattrAtomic,
		fsOpsCountFsOpLookUpInodeAtomic:                                                    &fsOpsCountFsOpLookUpInodeAtomic,
		fsOpsCountFsOpMkDirAtomic:                                                          &fsOpsCountFsOpMkDirAtomic,
		fsOpsCountFsOpMkNodeAtomic:                                                         &fsOpsCountFsOpMkNodeAtomic,
		fsOpsCountFsOpOpenDirAtomic:                                                        &fsOpsCountFsOpOpenDirAtomic,
		fsOpsCountFsOpOpenFileAtomic:                                                       &fsOpsCountFsOpOpenFileAtomic,
		fsOpsCountFsOpReadDirAtomic:                                                        &fsOpsCountFsOpReadDirAtomic,
		fsOpsCountFsOpReadDirPlusAtomic:                                                    &fsOpsCountFsOpReadDirPlusAtomic,
		fsOpsCountFsOpReadFileAtomic:                                                       &fsOpsCountFsOpReadFileAtomic,
		fsOpsCountFsOpReadSymlinkAtomic:                                                    &fsOpsCountFsOpReadSymlinkAtomic,
		fsOpsCountFsOpReleaseDirHandleAtomic:                                               &fsOpsCountFsOpReleaseDirHandleAtomic,
		fsOpsCountFsOpReleaseFileHandleAtomic:                                              &fsOpsCountFsOpReleaseFileHandleAtomic,
		fsOpsCountFsOpRemoveXattrAtomic:                                                    &fsOpsCountFsOpRemoveXattrAtomic,
		fsOpsCountFsOpRenameAtomic:                                                         &fsOpsCountFsOpRenameAtomic,
		fsOpsCountFsOpRmDirAtomic:                                                          &fsOpsCountFsOpRmDirAtomic,
		fsOpsCountFsOpSetInodeAttributesAtomic:                                             &fsOpsCountFsOpSetInodeAttributesAtomic,
		fsOpsCountFsOpSetXattrAtomic:                                                       &fsOpsCountFsOpSetXattrAtomic,
		fsOpsCountFsOpStatFSAtomic:                                                         &fsOpsCountFsOpStatFSAtomic,
		fsOpsCountFsOpSyncFSAtomic:                                                         &fsOpsCountFsOpSyncFSAtomic,
		fsOpsCountFsOpSyncFileAtomic:                                                       &fsOpsCountFsOpSyncFileAtomic,
		fsOpsCountFsOpUnlinkAtomic:                                                         &fsOpsCountFsOpUnlinkAtomic,
		fsOpsCountFsOpWriteFileAtomic:                                                      &fsOpsCountFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic:                      &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic:                      &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic:                   &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic:                       &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic:                &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic:               &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic:                      &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic:                      &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic:                   &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic:                       &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic:                &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic:               &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic:                     &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic:                     &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic:               &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic:              &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic:               &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic:                            &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic:                 &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic:                &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic:                            &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic:               &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic:                   &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic:                    &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic:                    &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic:                        &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic:             &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic:            &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic:                        &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic:               &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic:          &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic:                       &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic:            &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic:           &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic:                       &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic:          &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic:              &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic:           &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic:          &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic:                          &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic:                          &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic:                           &fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic:                           &fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic:                           &fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic:                               &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic:                             &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic:                             &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic:                    &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic:                   &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic:                               &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic:                           &fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic:                        &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic:                        &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic:                         &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic:                         &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic:                         &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic:                             &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic:                           &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic:                           &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic:                  &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic:                 &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic:                             &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic:                         &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic:                     &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic:                      &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic:                      &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic:                      &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic:                          &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic:               &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic:              &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic:                          &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic:                      &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic:                          &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic:                          &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic:                           &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic:                           &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic:                           &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic:                               &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic:                             &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic:                             &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic:                   &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic:                               &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic:                           &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic:                   &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic:                   &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic:                      &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic:                      &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic:             &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic:            &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic:                      &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic:                      &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic:                   &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic:                       &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic:                           &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic:                         &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic:                         &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic:                &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic:               &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic:                           &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic:                        &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic:                        &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic:                         &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic:                         &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic:                         &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic:                             &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic:                           &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic:                           &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic:                  &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic:                 &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic:                             &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic:                         &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic:         &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic:         &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic:      &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic:          &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic:          &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic: &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic:          &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic:              &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic:            &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic:            &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic:   &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic:  &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic:              &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic: &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic:          &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic:                 &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic:                 &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic:              &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic:                  &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic:                  &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic:                  &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic:                      &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic:                    &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic:                    &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic:           &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic:          &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic:                      &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic:                  &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic,
		fsOpsLatency: fsOpsLatency,
		gcsDownloadBytesCountReadTypeParallelAtomic:                &gcsDownloadBytesCountReadTypeParallelAtomic,
		gcsDownloadBytesCountReadTypeRandomAtomic:                  &gcsDownloadBytesCountReadTypeRandomAtomic,
		gcsDownloadBytesCountReadTypeSequentialAtomic:              &gcsDownloadBytesCountReadTypeSequentialAtomic,
		gcsReadBytesCountAtomic:                                    &gcsReadBytesCountAtomic,
		gcsReadCountReadTypeParallelAtomic:                         &gcsReadCountReadTypeParallelAtomic,
		gcsReadCountReadTypeRandomAtomic:                           &gcsReadCountReadTypeRandomAtomic,
		gcsReadCountReadTypeSequentialAtomic:                       &gcsReadCountReadTypeSequentialAtomic,
		gcsReaderCountIoMethodReadHandleAtomic:                     &gcsReaderCountIoMethodReadHandleAtomic,
		gcsReaderCountIoMethodClosedAtomic:                         &gcsReaderCountIoMethodClosedAtomic,
		gcsReaderCountIoMethodOpenedAtomic:                         &gcsReaderCountIoMethodOpenedAtomic,
		gcsRequestCountGcsMethodComposeObjectsAtomic:               &gcsRequestCountGcsMethodComposeObjectsAtomic,
		gcsRequestCountGcsMethodCopyObjectAtomic:                   &gcsRequestCountGcsMethodCopyObjectAtomic,
		gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic: &gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic,
		gcsRequestCountGcsMethodCreateFolderAtomic:                 &gcsRequestCountGcsMethodCreateFolderAtomic,
		gcsRequestCountGcsMethodCreateObjectAtomic:                 &gcsRequestCountGcsMethodCreateObjectAtomic,
		gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic:      &gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic,
		gcsRequestCountGcsMethodDeleteFolderAtomic:                 &gcsRequestCountGcsMethodDeleteFolderAtomic,
		gcsRequestCountGcsMethodDeleteObjectAtomic:                 &gcsRequestCountGcsMethodDeleteObjectAtomic,
		gcsRequestCountGcsMethodFinalizeUploadAtomic:               &gcsRequestCountGcsMethodFinalizeUploadAtomic,
		gcsRequestCountGcsMethodFlushPendingWritesAtomic:           &gcsRequestCountGcsMethodFlushPendingWritesAtomic,
		gcsRequestCountGcsMethodGetFolderAtomic:                    &gcsRequestCountGcsMethodGetFolderAtomic,
		gcsRequestCountGcsMethodListObjectsAtomic:                  &gcsRequestCountGcsMethodListObjectsAtomic,
		gcsRequestCountGcsMethodMoveObjectAtomic:                   &gcsRequestCountGcsMethodMoveObjectAtomic,
		gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic:      &gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic,
		gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic:      &gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic,
		gcsRequestCountGcsMethodNewReaderAtomic:                    &gcsRequestCountGcsMethodNewReaderAtomic,
		gcsRequestCountGcsMethodRenameFolderAtomic:                 &gcsRequestCountGcsMethodRenameFolderAtomic,
		gcsRequestCountGcsMethodStatObjectAtomic:                   &gcsRequestCountGcsMethodStatObjectAtomic,
		gcsRequestCountGcsMethodUpdateObjectAtomic:                 &gcsRequestCountGcsMethodUpdateObjectAtomic,
		gcsRequestLatencies:                                        gcsRequestLatencies,
		gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic:           &gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic,
		gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic:    &gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic,
	}, nil
}

func (o *otelMetrics) Close() {
	close(o.ch)
	o.wg.Wait()
}

func conditionallyObserve(obsrv metric.Int64Observer, counter *atomic.Int64, obsrvOptions ...metric.ObserveOption) {
	if val := counter.Load(); val > 0 {
		obsrv.Observe(val, obsrvOptions...)
	}
}

func updateUnrecognizedAttribute(newValue string) {
	unrecognizedAttr.CompareAndSwap("", newValue)
}

// StartSampledLogging starts a goroutine that logs unrecognized attributes periodically.
func startSampledLogging(ctx context.Context) {
	// Init the atomic.Value
	unrecognizedAttr.Store("")

	go func() {
		ticker := time.NewTicker(logInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logUnrecognizedAttribute()
			}
		}
	}()
}

// logUnrecognizedAttribute retrieves and logs any unrecognized attributes.
func logUnrecognizedAttribute() {
	// Atomically load and reset the attribute name, then generate a log
	// if an unrecognized attribute was encountered.
	if currentAttr := unrecognizedAttr.Swap("").(string); currentAttr != "" {
		logger.Tracef("Attribute %s is not declared", currentAttr)
	}
}
