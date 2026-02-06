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
	bufferedReadFallbackTriggerCountReasonInsufficientMemoryAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("reason", "insufficient_memory")))
	bufferedReadFallbackTriggerCountReasonRandomReadDetectedAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("reason", "random_read_detected")))
	fileCacheReadBytesCountReadTypeParallelAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Parallel")))
	fileCacheReadBytesCountReadTypeRandomAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Random")))
	fileCacheReadBytesCountReadTypeSequentialAttrSet                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Sequential")))
	fileCacheReadBytesCountReadTypeUnknownAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Unknown")))
	fileCacheReadCountCacheHitTrueReadTypeParallelAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")))
	fileCacheReadCountCacheHitTrueReadTypeRandomAttrSet                                 = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")))
	fileCacheReadCountCacheHitTrueReadTypeSequentialAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Sequential")))
	fileCacheReadCountCacheHitTrueReadTypeUnknownAttrSet                                = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Unknown")))
	fileCacheReadCountCacheHitFalseReadTypeParallelAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Parallel")))
	fileCacheReadCountCacheHitFalseReadTypeRandomAttrSet                                = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Random")))
	fileCacheReadCountCacheHitFalseReadTypeSequentialAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Sequential")))
	fileCacheReadCountCacheHitFalseReadTypeUnknownAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Unknown")))
	fileCacheReadLatenciesCacheHitTrueAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true)))
	fileCacheReadLatenciesCacheHitFalseAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false)))
	fsOpsCountFsOpBatchForgetAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "BatchForget")))
	fsOpsCountFsOpCreateFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateFile")))
	fsOpsCountFsOpCreateLinkAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateLink")))
	fsOpsCountFsOpCreateSymlinkAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateSymlink")))
	fsOpsCountFsOpFlushFileAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "FlushFile")))
	fsOpsCountFsOpForgetInodeAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ForgetInode")))
	fsOpsCountFsOpGetInodeAttributesAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsCountFsOpLookUpInodeAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "LookUpInode")))
	fsOpsCountFsOpMkDirAttrSet                                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkDir")))
	fsOpsCountFsOpMkNodeAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkNode")))
	fsOpsCountFsOpOpenDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenDir")))
	fsOpsCountFsOpOpenFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenFile")))
	fsOpsCountFsOpOthersAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Others")))
	fsOpsCountFsOpReadDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDir")))
	fsOpsCountFsOpReadDirPlusAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")))
	fsOpsCountFsOpReadFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadFile")))
	fsOpsCountFsOpReadSymlinkAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadSymlink")))
	fsOpsCountFsOpReleaseDirHandleAttrSet                                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsCountFsOpReleaseFileHandleAttrSet                                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsCountFsOpRenameAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Rename")))
	fsOpsCountFsOpRmDirAttrSet                                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "RmDir")))
	fsOpsCountFsOpSetInodeAttributesAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsCountFsOpSyncFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SyncFile")))
	fsOpsCountFsOpUnlinkAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Unlink")))
	fsOpsCountFsOpWriteFileAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAttrSet      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAttrSet   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAttrSet  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "WriteFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "BatchForget")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateLink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateSymlink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "FlushFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ForgetInode")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "LookUpInode")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkNode")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Others")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDirPlus")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadSymlink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Rename")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RmDir")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFile")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Unlink")))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "WriteFile")))
	fsOpsLatencyFsOpBatchForgetAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "BatchForget")))
	fsOpsLatencyFsOpCreateFileAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateFile")))
	fsOpsLatencyFsOpCreateLinkAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateLink")))
	fsOpsLatencyFsOpCreateSymlinkAttrSet                                                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "CreateSymlink")))
	fsOpsLatencyFsOpFlushFileAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "FlushFile")))
	fsOpsLatencyFsOpForgetInodeAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ForgetInode")))
	fsOpsLatencyFsOpGetInodeAttributesAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")))
	fsOpsLatencyFsOpLookUpInodeAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "LookUpInode")))
	fsOpsLatencyFsOpMkDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkDir")))
	fsOpsLatencyFsOpMkNodeAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "MkNode")))
	fsOpsLatencyFsOpOpenDirAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenDir")))
	fsOpsLatencyFsOpOpenFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "OpenFile")))
	fsOpsLatencyFsOpOthersAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Others")))
	fsOpsLatencyFsOpReadDirAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDir")))
	fsOpsLatencyFsOpReadDirPlusAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")))
	fsOpsLatencyFsOpReadFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadFile")))
	fsOpsLatencyFsOpReadSymlinkAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReadSymlink")))
	fsOpsLatencyFsOpReleaseDirHandleAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")))
	fsOpsLatencyFsOpReleaseFileHandleAttrSet                                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")))
	fsOpsLatencyFsOpRenameAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Rename")))
	fsOpsLatencyFsOpRmDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "RmDir")))
	fsOpsLatencyFsOpSetInodeAttributesAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")))
	fsOpsLatencyFsOpSyncFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "SyncFile")))
	fsOpsLatencyFsOpUnlinkAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "Unlink")))
	fsOpsLatencyFsOpWriteFileAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", "WriteFile")))
	gcsDownloadBytesCountReadTypeBufferedAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Buffered")))
	gcsDownloadBytesCountReadTypeParallelAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Parallel")))
	gcsDownloadBytesCountReadTypeRandomAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Random")))
	gcsDownloadBytesCountReadTypeSequentialAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Sequential")))
	gcsReadCountReadTypeParallelAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Parallel")))
	gcsReadCountReadTypeRandomAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Random")))
	gcsReadCountReadTypeSequentialAttrSet                                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Sequential")))
	gcsReadCountReadTypeUnknownAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Unknown")))
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
	testUpdownCounterWithAttrsRequestTypeAttr1AttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("request_type", "attr1")))
	testUpdownCounterWithAttrsRequestTypeAttr2AttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("request_type", "attr2")))
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
	fileCacheReadBytesCountReadTypeParallelAtomic                                      *atomic.Int64
	fileCacheReadBytesCountReadTypeRandomAtomic                                        *atomic.Int64
	fileCacheReadBytesCountReadTypeSequentialAtomic                                    *atomic.Int64
	fileCacheReadBytesCountReadTypeUnknownAtomic                                       *atomic.Int64
	fileCacheReadCountCacheHitTrueReadTypeParallelAtomic                               *atomic.Int64
	fileCacheReadCountCacheHitTrueReadTypeRandomAtomic                                 *atomic.Int64
	fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic                             *atomic.Int64
	fileCacheReadCountCacheHitTrueReadTypeUnknownAtomic                                *atomic.Int64
	fileCacheReadCountCacheHitFalseReadTypeParallelAtomic                              *atomic.Int64
	fileCacheReadCountCacheHitFalseReadTypeRandomAtomic                                *atomic.Int64
	fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic                            *atomic.Int64
	fileCacheReadCountCacheHitFalseReadTypeUnknownAtomic                               *atomic.Int64
	fsOpsCountFsOpBatchForgetAtomic                                                    *atomic.Int64
	fsOpsCountFsOpCreateFileAtomic                                                     *atomic.Int64
	fsOpsCountFsOpCreateLinkAtomic                                                     *atomic.Int64
	fsOpsCountFsOpCreateSymlinkAtomic                                                  *atomic.Int64
	fsOpsCountFsOpFlushFileAtomic                                                      *atomic.Int64
	fsOpsCountFsOpForgetInodeAtomic                                                    *atomic.Int64
	fsOpsCountFsOpGetInodeAttributesAtomic                                             *atomic.Int64
	fsOpsCountFsOpLookUpInodeAtomic                                                    *atomic.Int64
	fsOpsCountFsOpMkDirAtomic                                                          *atomic.Int64
	fsOpsCountFsOpMkNodeAtomic                                                         *atomic.Int64
	fsOpsCountFsOpOpenDirAtomic                                                        *atomic.Int64
	fsOpsCountFsOpOpenFileAtomic                                                       *atomic.Int64
	fsOpsCountFsOpOthersAtomic                                                         *atomic.Int64
	fsOpsCountFsOpReadDirAtomic                                                        *atomic.Int64
	fsOpsCountFsOpReadDirPlusAtomic                                                    *atomic.Int64
	fsOpsCountFsOpReadFileAtomic                                                       *atomic.Int64
	fsOpsCountFsOpReadSymlinkAtomic                                                    *atomic.Int64
	fsOpsCountFsOpReleaseDirHandleAtomic                                               *atomic.Int64
	fsOpsCountFsOpReleaseFileHandleAtomic                                              *atomic.Int64
	fsOpsCountFsOpRenameAtomic                                                         *atomic.Int64
	fsOpsCountFsOpRmDirAtomic                                                          *atomic.Int64
	fsOpsCountFsOpSetInodeAttributesAtomic                                             *atomic.Int64
	fsOpsCountFsOpSyncFileAtomic                                                       *atomic.Int64
	fsOpsCountFsOpUnlinkAtomic                                                         *atomic.Int64
	fsOpsCountFsOpWriteFileAtomic                                                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic                               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic                              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic               *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic                        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic                           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic                       *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic                             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic                          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic                            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic                         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic            *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic        *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic             *atomic.Int64
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic                 *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic              *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic                  *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic                    *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic                *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic           *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic          *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic                      *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic         *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic                   *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic                     *atomic.Int64
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic                  *atomic.Int64
	gcsDownloadBytesCountReadTypeBufferedAtomic                                        *atomic.Int64
	gcsDownloadBytesCountReadTypeParallelAtomic                                        *atomic.Int64
	gcsDownloadBytesCountReadTypeRandomAtomic                                          *atomic.Int64
	gcsDownloadBytesCountReadTypeSequentialAtomic                                      *atomic.Int64
	gcsReadBytesCountAtomic                                                            *atomic.Int64
	gcsReadCountReadTypeParallelAtomic                                                 *atomic.Int64
	gcsReadCountReadTypeRandomAtomic                                                   *atomic.Int64
	gcsReadCountReadTypeSequentialAtomic                                               *atomic.Int64
	gcsReadCountReadTypeUnknownAtomic                                                  *atomic.Int64
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
	testUpdownCounterAtomic                                                            *atomic.Int64
	testUpdownCounterWithAttrsRequestTypeAttr1Atomic                                   *atomic.Int64
	testUpdownCounterWithAttrsRequestTypeAttr2Atomic                                   *atomic.Int64
	bufferedReadReadLatency                                                            metric.Int64Histogram
	fileCacheReadLatencies                                                             metric.Int64Histogram
	fsOpsLatency                                                                       metric.Int64Histogram
	gcsRequestLatencies                                                                metric.Int64Histogram
}

func (o *otelMetrics) BufferedReadFallbackTriggerCount(
	inc int64, reason Reason) {
	if inc < 0 {
		logger.Errorf("Counter metric buffered_read/fallback_trigger_count received a negative increment: %d", inc)
		return
	}
	switch reason {
	case ReasonInsufficientMemoryAttr:
		o.bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic.Add(inc)
	case ReasonRandomReadDetectedAttr:
		o.bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(reason))
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

func (o *otelMetrics) FileCacheReadBytesCount(
	inc int64, readType ReadType) {
	if inc < 0 {
		logger.Errorf("Counter metric file_cache/read_bytes_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case ReadTypeParallelAttr:
		o.fileCacheReadBytesCountReadTypeParallelAtomic.Add(inc)
	case ReadTypeRandomAttr:
		o.fileCacheReadBytesCountReadTypeRandomAtomic.Add(inc)
	case ReadTypeSequentialAttr:
		o.fileCacheReadBytesCountReadTypeSequentialAtomic.Add(inc)
	case ReadTypeUnknownAttr:
		o.fileCacheReadBytesCountReadTypeUnknownAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(readType))
		return
	}
}

func (o *otelMetrics) FileCacheReadCount(
	inc int64, cacheHit bool, readType ReadType) {
	if inc < 0 {
		logger.Errorf("Counter metric file_cache/read_count received a negative increment: %d", inc)
		return
	}
	switch cacheHit {
	case true:
		switch readType {
		case ReadTypeParallelAttr:
			o.fileCacheReadCountCacheHitTrueReadTypeParallelAtomic.Add(inc)
		case ReadTypeRandomAttr:
			o.fileCacheReadCountCacheHitTrueReadTypeRandomAtomic.Add(inc)
		case ReadTypeSequentialAttr:
			o.fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic.Add(inc)
		case ReadTypeUnknownAttr:
			o.fileCacheReadCountCacheHitTrueReadTypeUnknownAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(readType))
			return
		}
	case false:
		switch readType {
		case ReadTypeParallelAttr:
			o.fileCacheReadCountCacheHitFalseReadTypeParallelAtomic.Add(inc)
		case ReadTypeRandomAttr:
			o.fileCacheReadCountCacheHitFalseReadTypeRandomAtomic.Add(inc)
		case ReadTypeSequentialAttr:
			o.fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic.Add(inc)
		case ReadTypeUnknownAttr:
			o.fileCacheReadCountCacheHitFalseReadTypeUnknownAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(readType))
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
	inc int64, fsOp FsOp) {
	if inc < 0 {
		logger.Errorf("Counter metric fs/ops_count received a negative increment: %d", inc)
		return
	}
	switch fsOp {
	case FsOpBatchForgetAttr:
		o.fsOpsCountFsOpBatchForgetAtomic.Add(inc)
	case FsOpCreateFileAttr:
		o.fsOpsCountFsOpCreateFileAtomic.Add(inc)
	case FsOpCreateLinkAttr:
		o.fsOpsCountFsOpCreateLinkAtomic.Add(inc)
	case FsOpCreateSymlinkAttr:
		o.fsOpsCountFsOpCreateSymlinkAtomic.Add(inc)
	case FsOpFlushFileAttr:
		o.fsOpsCountFsOpFlushFileAtomic.Add(inc)
	case FsOpForgetInodeAttr:
		o.fsOpsCountFsOpForgetInodeAtomic.Add(inc)
	case FsOpGetInodeAttributesAttr:
		o.fsOpsCountFsOpGetInodeAttributesAtomic.Add(inc)
	case FsOpLookUpInodeAttr:
		o.fsOpsCountFsOpLookUpInodeAtomic.Add(inc)
	case FsOpMkDirAttr:
		o.fsOpsCountFsOpMkDirAtomic.Add(inc)
	case FsOpMkNodeAttr:
		o.fsOpsCountFsOpMkNodeAtomic.Add(inc)
	case FsOpOpenDirAttr:
		o.fsOpsCountFsOpOpenDirAtomic.Add(inc)
	case FsOpOpenFileAttr:
		o.fsOpsCountFsOpOpenFileAtomic.Add(inc)
	case FsOpOthersAttr:
		o.fsOpsCountFsOpOthersAtomic.Add(inc)
	case FsOpReadDirAttr:
		o.fsOpsCountFsOpReadDirAtomic.Add(inc)
	case FsOpReadDirPlusAttr:
		o.fsOpsCountFsOpReadDirPlusAtomic.Add(inc)
	case FsOpReadFileAttr:
		o.fsOpsCountFsOpReadFileAtomic.Add(inc)
	case FsOpReadSymlinkAttr:
		o.fsOpsCountFsOpReadSymlinkAtomic.Add(inc)
	case FsOpReleaseDirHandleAttr:
		o.fsOpsCountFsOpReleaseDirHandleAtomic.Add(inc)
	case FsOpReleaseFileHandleAttr:
		o.fsOpsCountFsOpReleaseFileHandleAtomic.Add(inc)
	case FsOpRenameAttr:
		o.fsOpsCountFsOpRenameAtomic.Add(inc)
	case FsOpRmDirAttr:
		o.fsOpsCountFsOpRmDirAtomic.Add(inc)
	case FsOpSetInodeAttributesAttr:
		o.fsOpsCountFsOpSetInodeAttributesAtomic.Add(inc)
	case FsOpSyncFileAttr:
		o.fsOpsCountFsOpSyncFileAtomic.Add(inc)
	case FsOpUnlinkAttr:
		o.fsOpsCountFsOpUnlinkAtomic.Add(inc)
	case FsOpWriteFileAttr:
		o.fsOpsCountFsOpWriteFileAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(fsOp))
		return
	}
}

func (o *otelMetrics) FsOpsErrorCount(
	inc int64, fsErrorCategory FsErrorCategory, fsOp FsOp) {
	if inc < 0 {
		logger.Errorf("Counter metric fs/ops_error_count received a negative increment: %d", inc)
		return
	}
	switch fsErrorCategory {
	case FsErrorCategoryDEVICEERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryDIRNOTEMPTYAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryFILEDIRERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryFILEEXISTSAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryINTERRUPTERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryINVALIDARGUMENTAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryINVALIDOPERATIONAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryIOERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryMISCERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryNETWORKERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryNOTADIRAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryNOTIMPLEMENTEDAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryNOFILEORDIRAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryPERMERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FsErrorCategoryTOOMANYOPENFILESAttr:
		switch fsOp {
		case FsOpBatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic.Add(inc)
		case FsOpCreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic.Add(inc)
		case FsOpCreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic.Add(inc)
		case FsOpCreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic.Add(inc)
		case FsOpFlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic.Add(inc)
		case FsOpForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic.Add(inc)
		case FsOpGetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic.Add(inc)
		case FsOpLookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic.Add(inc)
		case FsOpMkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic.Add(inc)
		case FsOpMkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic.Add(inc)
		case FsOpOpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic.Add(inc)
		case FsOpOpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic.Add(inc)
		case FsOpOthersAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAtomic.Add(inc)
		case FsOpReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic.Add(inc)
		case FsOpReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic.Add(inc)
		case FsOpReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic.Add(inc)
		case FsOpReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic.Add(inc)
		case FsOpReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic.Add(inc)
		case FsOpReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic.Add(inc)
		case FsOpRenameAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic.Add(inc)
		case FsOpRmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic.Add(inc)
		case FsOpSetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic.Add(inc)
		case FsOpSyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic.Add(inc)
		case FsOpUnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic.Add(inc)
		case FsOpWriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	default:
		updateUnrecognizedAttribute(string(fsErrorCategory))
		return
	}
}

func (o *otelMetrics) FsOpsLatency(
	ctx context.Context, latency time.Duration, fsOp FsOp) {
	var record histogramRecord
	switch fsOp {
	case FsOpBatchForgetAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpBatchForgetAttrSet}
	case FsOpCreateFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateFileAttrSet}
	case FsOpCreateLinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateLinkAttrSet}
	case FsOpCreateSymlinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateSymlinkAttrSet}
	case FsOpFlushFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpFlushFileAttrSet}
	case FsOpForgetInodeAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpForgetInodeAttrSet}
	case FsOpGetInodeAttributesAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpGetInodeAttributesAttrSet}
	case FsOpLookUpInodeAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpLookUpInodeAttrSet}
	case FsOpMkDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkDirAttrSet}
	case FsOpMkNodeAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkNodeAttrSet}
	case FsOpOpenDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenDirAttrSet}
	case FsOpOpenFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenFileAttrSet}
	case FsOpOthersAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOthersAttrSet}
	case FsOpReadDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadDirAttrSet}
	case FsOpReadDirPlusAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadDirPlusAttrSet}
	case FsOpReadFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadFileAttrSet}
	case FsOpReadSymlinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadSymlinkAttrSet}
	case FsOpReleaseDirHandleAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseDirHandleAttrSet}
	case FsOpReleaseFileHandleAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseFileHandleAttrSet}
	case FsOpRenameAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRenameAttrSet}
	case FsOpRmDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRmDirAttrSet}
	case FsOpSetInodeAttributesAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSetInodeAttributesAttrSet}
	case FsOpSyncFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSyncFileAttrSet}
	case FsOpUnlinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpUnlinkAttrSet}
	case FsOpWriteFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpWriteFileAttrSet}
	default:
		updateUnrecognizedAttribute(string(fsOp))
		return
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) GcsDownloadBytesCount(
	inc int64, readType ReadType) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/download_bytes_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case ReadTypeBufferedAttr:
		o.gcsDownloadBytesCountReadTypeBufferedAtomic.Add(inc)
	case ReadTypeParallelAttr:
		o.gcsDownloadBytesCountReadTypeParallelAtomic.Add(inc)
	case ReadTypeRandomAttr:
		o.gcsDownloadBytesCountReadTypeRandomAtomic.Add(inc)
	case ReadTypeSequentialAttr:
		o.gcsDownloadBytesCountReadTypeSequentialAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(readType))
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
	inc int64, readType ReadType) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/read_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case ReadTypeParallelAttr:
		o.gcsReadCountReadTypeParallelAtomic.Add(inc)
	case ReadTypeRandomAttr:
		o.gcsReadCountReadTypeRandomAtomic.Add(inc)
	case ReadTypeSequentialAttr:
		o.gcsReadCountReadTypeSequentialAtomic.Add(inc)
	case ReadTypeUnknownAttr:
		o.gcsReadCountReadTypeUnknownAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(readType))
		return
	}
}

func (o *otelMetrics) GcsReaderCount(
	inc int64, ioMethod IoMethod) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/reader_count received a negative increment: %d", inc)
		return
	}
	switch ioMethod {
	case IoMethodClosedAttr:
		o.gcsReaderCountIoMethodClosedAtomic.Add(inc)
	case IoMethodOpenedAttr:
		o.gcsReaderCountIoMethodOpenedAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(ioMethod))
		return
	}
}

func (o *otelMetrics) GcsRequestCount(
	inc int64, gcsMethod GcsMethod) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/request_count received a negative increment: %d", inc)
		return
	}
	switch gcsMethod {
	case GcsMethodComposeObjectsAttr:
		o.gcsRequestCountGcsMethodComposeObjectsAtomic.Add(inc)
	case GcsMethodCopyObjectAttr:
		o.gcsRequestCountGcsMethodCopyObjectAtomic.Add(inc)
	case GcsMethodCreateAppendableObjectWriterAttr:
		o.gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic.Add(inc)
	case GcsMethodCreateFolderAttr:
		o.gcsRequestCountGcsMethodCreateFolderAtomic.Add(inc)
	case GcsMethodCreateObjectAttr:
		o.gcsRequestCountGcsMethodCreateObjectAtomic.Add(inc)
	case GcsMethodCreateObjectChunkWriterAttr:
		o.gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic.Add(inc)
	case GcsMethodDeleteFolderAttr:
		o.gcsRequestCountGcsMethodDeleteFolderAtomic.Add(inc)
	case GcsMethodDeleteObjectAttr:
		o.gcsRequestCountGcsMethodDeleteObjectAtomic.Add(inc)
	case GcsMethodFinalizeUploadAttr:
		o.gcsRequestCountGcsMethodFinalizeUploadAtomic.Add(inc)
	case GcsMethodFlushPendingWritesAttr:
		o.gcsRequestCountGcsMethodFlushPendingWritesAtomic.Add(inc)
	case GcsMethodGetFolderAttr:
		o.gcsRequestCountGcsMethodGetFolderAtomic.Add(inc)
	case GcsMethodListObjectsAttr:
		o.gcsRequestCountGcsMethodListObjectsAtomic.Add(inc)
	case GcsMethodMoveObjectAttr:
		o.gcsRequestCountGcsMethodMoveObjectAtomic.Add(inc)
	case GcsMethodMultiRangeDownloaderAddAttr:
		o.gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic.Add(inc)
	case GcsMethodNewMultiRangeDownloaderAttr:
		o.gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic.Add(inc)
	case GcsMethodNewReaderAttr:
		o.gcsRequestCountGcsMethodNewReaderAtomic.Add(inc)
	case GcsMethodRenameFolderAttr:
		o.gcsRequestCountGcsMethodRenameFolderAtomic.Add(inc)
	case GcsMethodStatObjectAttr:
		o.gcsRequestCountGcsMethodStatObjectAtomic.Add(inc)
	case GcsMethodUpdateObjectAttr:
		o.gcsRequestCountGcsMethodUpdateObjectAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(gcsMethod))
		return
	}
}

func (o *otelMetrics) GcsRequestLatencies(
	ctx context.Context, latency time.Duration, gcsMethod GcsMethod) {
	var record histogramRecord
	switch gcsMethod {
	case GcsMethodComposeObjectsAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodComposeObjectsAttrSet}
	case GcsMethodCopyObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCopyObjectAttrSet}
	case GcsMethodCreateAppendableObjectWriterAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateAppendableObjectWriterAttrSet}
	case GcsMethodCreateFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateFolderAttrSet}
	case GcsMethodCreateObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateObjectAttrSet}
	case GcsMethodCreateObjectChunkWriterAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateObjectChunkWriterAttrSet}
	case GcsMethodDeleteFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteFolderAttrSet}
	case GcsMethodDeleteObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteObjectAttrSet}
	case GcsMethodFinalizeUploadAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodFinalizeUploadAttrSet}
	case GcsMethodFlushPendingWritesAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodFlushPendingWritesAttrSet}
	case GcsMethodGetFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodGetFolderAttrSet}
	case GcsMethodListObjectsAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodListObjectsAttrSet}
	case GcsMethodMoveObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMoveObjectAttrSet}
	case GcsMethodMultiRangeDownloaderAddAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMultiRangeDownloaderAddAttrSet}
	case GcsMethodNewMultiRangeDownloaderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewMultiRangeDownloaderAttrSet}
	case GcsMethodNewReaderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewReaderAttrSet}
	case GcsMethodRenameFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodRenameFolderAttrSet}
	case GcsMethodStatObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodStatObjectAttrSet}
	case GcsMethodUpdateObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodUpdateObjectAttrSet}
	default:
		updateUnrecognizedAttribute(string(gcsMethod))
		return
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) GcsRetryCount(
	inc int64, retryErrorCategory RetryErrorCategory) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/retry_count received a negative increment: %d", inc)
		return
	}
	switch retryErrorCategory {
	case RetryErrorCategoryOTHERERRORSAttr:
		o.gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic.Add(inc)
	case RetryErrorCategorySTALLEDREADREQUESTAttr:
		o.gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(retryErrorCategory))
		return
	}
}

func (o *otelMetrics) TestUpdownCounter(
	inc int64) {
	o.testUpdownCounterAtomic.Add(inc)
}

func (o *otelMetrics) TestUpdownCounterWithAttrs(
	inc int64, requestType RequestType) {
	switch requestType {
	case RequestTypeAttr1Attr:
		o.testUpdownCounterWithAttrsRequestTypeAttr1Atomic.Add(inc)
	case RequestTypeAttr2Attr:
		o.testUpdownCounterWithAttrsRequestTypeAttr2Atomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(requestType))
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

	var fileCacheReadBytesCountReadTypeParallelAtomic,
		fileCacheReadBytesCountReadTypeRandomAtomic,
		fileCacheReadBytesCountReadTypeSequentialAtomic,
		fileCacheReadBytesCountReadTypeUnknownAtomic atomic.Int64

	var fileCacheReadCountCacheHitTrueReadTypeParallelAtomic,
		fileCacheReadCountCacheHitTrueReadTypeRandomAtomic,
		fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic,
		fileCacheReadCountCacheHitTrueReadTypeUnknownAtomic,
		fileCacheReadCountCacheHitFalseReadTypeParallelAtomic,
		fileCacheReadCountCacheHitFalseReadTypeRandomAtomic,
		fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic,
		fileCacheReadCountCacheHitFalseReadTypeUnknownAtomic atomic.Int64

	var fsOpsCountFsOpBatchForgetAtomic,
		fsOpsCountFsOpCreateFileAtomic,
		fsOpsCountFsOpCreateLinkAtomic,
		fsOpsCountFsOpCreateSymlinkAtomic,
		fsOpsCountFsOpFlushFileAtomic,
		fsOpsCountFsOpForgetInodeAtomic,
		fsOpsCountFsOpGetInodeAttributesAtomic,
		fsOpsCountFsOpLookUpInodeAtomic,
		fsOpsCountFsOpMkDirAtomic,
		fsOpsCountFsOpMkNodeAtomic,
		fsOpsCountFsOpOpenDirAtomic,
		fsOpsCountFsOpOpenFileAtomic,
		fsOpsCountFsOpOthersAtomic,
		fsOpsCountFsOpReadDirAtomic,
		fsOpsCountFsOpReadDirPlusAtomic,
		fsOpsCountFsOpReadFileAtomic,
		fsOpsCountFsOpReadSymlinkAtomic,
		fsOpsCountFsOpReleaseDirHandleAtomic,
		fsOpsCountFsOpReleaseFileHandleAtomic,
		fsOpsCountFsOpRenameAtomic,
		fsOpsCountFsOpRmDirAtomic,
		fsOpsCountFsOpSetInodeAttributesAtomic,
		fsOpsCountFsOpSyncFileAtomic,
		fsOpsCountFsOpUnlinkAtomic,
		fsOpsCountFsOpWriteFileAtomic atomic.Int64

	var fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic atomic.Int64

	var gcsDownloadBytesCountReadTypeBufferedAtomic,
		gcsDownloadBytesCountReadTypeParallelAtomic,
		gcsDownloadBytesCountReadTypeRandomAtomic,
		gcsDownloadBytesCountReadTypeSequentialAtomic atomic.Int64

	var gcsReadBytesCountAtomic atomic.Int64

	var gcsReadCountReadTypeParallelAtomic,
		gcsReadCountReadTypeRandomAtomic,
		gcsReadCountReadTypeSequentialAtomic,
		gcsReadCountReadTypeUnknownAtomic atomic.Int64

	var gcsReaderCountIoMethodClosedAtomic,
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

	var testUpdownCounterAtomic atomic.Int64

	var testUpdownCounterWithAttrsRequestTypeAttr1Atomic,
		testUpdownCounterWithAttrsRequestTypeAttr2Atomic atomic.Int64

	_, err0 := meter.Int64ObservableCounter("buffered_read/fallback_trigger_count",
		metric.WithDescription("The cumulative number of times the BufferedReader falls back to a different reader, along with the reason: random_read_detected or insufficient_memory."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic, bufferedReadFallbackTriggerCountReasonInsufficientMemoryAttrSet)
			conditionallyObserve(obsrv, &bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic, bufferedReadFallbackTriggerCountReasonRandomReadDetectedAttrSet)
			return nil
		}))

	bufferedReadReadLatency, err1 := meter.Int64Histogram("buffered_read/read_latency",
		metric.WithDescription("The cumulative distribution of latencies for ReadAt calls served by the buffered reader."),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(50, 100, 200, 400, 800, 1200, 2000, 5000, 10000, 20000, 50000, 100000, 200000, 500000, 1000000, 2000000, 5000000, 10000000, 50000000, 100000000, 300000000, 500000000))

	_, err2 := meter.Int64ObservableCounter("file_cache/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from file cache along with read type - Sequential/Random"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fileCacheReadBytesCountReadTypeParallelAtomic, fileCacheReadBytesCountReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadBytesCountReadTypeRandomAtomic, fileCacheReadBytesCountReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadBytesCountReadTypeSequentialAtomic, fileCacheReadBytesCountReadTypeSequentialAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadBytesCountReadTypeUnknownAtomic, fileCacheReadBytesCountReadTypeUnknownAttrSet)
			return nil
		}))

	_, err3 := meter.Int64ObservableCounter("file_cache/read_count",
		metric.WithDescription("Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false"),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitTrueReadTypeParallelAtomic, fileCacheReadCountCacheHitTrueReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitTrueReadTypeRandomAtomic, fileCacheReadCountCacheHitTrueReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic, fileCacheReadCountCacheHitTrueReadTypeSequentialAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitTrueReadTypeUnknownAtomic, fileCacheReadCountCacheHitTrueReadTypeUnknownAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitFalseReadTypeParallelAtomic, fileCacheReadCountCacheHitFalseReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitFalseReadTypeRandomAtomic, fileCacheReadCountCacheHitFalseReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic, fileCacheReadCountCacheHitFalseReadTypeSequentialAttrSet)
			conditionallyObserve(obsrv, &fileCacheReadCountCacheHitFalseReadTypeUnknownAtomic, fileCacheReadCountCacheHitFalseReadTypeUnknownAttrSet)
			return nil
		}))

	fileCacheReadLatencies, err4 := meter.Int64Histogram("file_cache/read_latencies",
		metric.WithDescription("The cumulative distribution of the file cache read latencies along with cache hit - true/false."),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(50, 100, 200, 400, 800, 1200, 2000, 5000, 10000, 20000, 50000, 100000, 200000, 500000, 1000000, 2000000, 5000000, 10000000, 50000000, 100000000, 300000000, 500000000))

	_, err5 := meter.Int64ObservableCounter("fs/ops_count",
		metric.WithDescription("The cumulative number of ops processed by the file system."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fsOpsCountFsOpBatchForgetAtomic, fsOpsCountFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpCreateFileAtomic, fsOpsCountFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpCreateLinkAtomic, fsOpsCountFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpCreateSymlinkAtomic, fsOpsCountFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpFlushFileAtomic, fsOpsCountFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpForgetInodeAtomic, fsOpsCountFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpGetInodeAttributesAtomic, fsOpsCountFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpLookUpInodeAtomic, fsOpsCountFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpMkDirAtomic, fsOpsCountFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpMkNodeAtomic, fsOpsCountFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpOpenDirAtomic, fsOpsCountFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpOpenFileAtomic, fsOpsCountFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpOthersAtomic, fsOpsCountFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadDirAtomic, fsOpsCountFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadDirPlusAtomic, fsOpsCountFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadFileAtomic, fsOpsCountFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReadSymlinkAtomic, fsOpsCountFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReleaseDirHandleAtomic, fsOpsCountFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpReleaseFileHandleAtomic, fsOpsCountFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpRenameAtomic, fsOpsCountFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpRmDirAtomic, fsOpsCountFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpSetInodeAttributesAtomic, fsOpsCountFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpSyncFileAtomic, fsOpsCountFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpUnlinkAtomic, fsOpsCountFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsCountFsOpWriteFileAtomic, fsOpsCountFsOpWriteFileAttrSet)
			return nil
		}))

	_, err6 := meter.Int64ObservableCounter("fs/ops_error_count",
		metric.WithDescription("The cumulative number of errors generated by file system operations."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAttrSet)
			conditionallyObserve(obsrv, &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic, fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAttrSet)
			return nil
		}))

	fsOpsLatency, err7 := meter.Int64Histogram("fs/ops_latency",
		metric.WithDescription("The cumulative distribution of file system operation latencies"),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(50, 100, 200, 400, 800, 1200, 2000, 5000, 10000, 20000, 50000, 100000, 200000, 500000, 1000000, 2000000, 5000000, 10000000, 50000000, 100000000, 300000000, 500000000))

	_, err8 := meter.Int64ObservableCounter("gcs/download_bytes_count",
		metric.WithDescription("The cumulative number of bytes downloaded from GCS along with type - Sequential/Random"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsDownloadBytesCountReadTypeBufferedAtomic, gcsDownloadBytesCountReadTypeBufferedAttrSet)
			conditionallyObserve(obsrv, &gcsDownloadBytesCountReadTypeParallelAtomic, gcsDownloadBytesCountReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &gcsDownloadBytesCountReadTypeRandomAtomic, gcsDownloadBytesCountReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &gcsDownloadBytesCountReadTypeSequentialAtomic, gcsDownloadBytesCountReadTypeSequentialAttrSet)
			return nil
		}))

	_, err9 := meter.Int64ObservableCounter("gcs/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from GCS objects."),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsReadBytesCountAtomic)
			return nil
		}))

	_, err10 := meter.Int64ObservableCounter("gcs/read_count",
		metric.WithDescription("Specifies the number of gcs reads made along with type - Sequential/Random"),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsReadCountReadTypeParallelAtomic, gcsReadCountReadTypeParallelAttrSet)
			conditionallyObserve(obsrv, &gcsReadCountReadTypeRandomAtomic, gcsReadCountReadTypeRandomAttrSet)
			conditionallyObserve(obsrv, &gcsReadCountReadTypeSequentialAtomic, gcsReadCountReadTypeSequentialAttrSet)
			conditionallyObserve(obsrv, &gcsReadCountReadTypeUnknownAtomic, gcsReadCountReadTypeUnknownAttrSet)
			return nil
		}))

	_, err11 := meter.Int64ObservableCounter("gcs/reader_count",
		metric.WithDescription("The cumulative number of GCS object readers opened or closed."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsReaderCountIoMethodClosedAtomic, gcsReaderCountIoMethodClosedAttrSet)
			conditionallyObserve(obsrv, &gcsReaderCountIoMethodOpenedAtomic, gcsReaderCountIoMethodOpenedAttrSet)
			return nil
		}))

	_, err12 := meter.Int64ObservableCounter("gcs/request_count",
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

	gcsRequestLatencies, err13 := meter.Int64Histogram("gcs/request_latencies",
		metric.WithDescription("The cumulative distribution of the GCS request latencies."),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(50, 100, 150, 200, 300, 400, 500, 700, 1000, 2000, 5000, 7000, 10000, 20000, 50000, 100000, 200000, 500000))

	_, err14 := meter.Int64ObservableCounter("gcs/retry_count",
		metric.WithDescription("The cumulative number of retry requests made to GCS."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			conditionallyObserve(obsrv, &gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic, gcsRetryCountRetryErrorCategoryOTHERERRORSAttrSet)
			conditionallyObserve(obsrv, &gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic, gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAttrSet)
			return nil
		}))

	_, err15 := meter.Int64ObservableUpDownCounter("test/updown_counter",
		metric.WithDescription("Test metric for updown counters."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			observeUpDownCounter(obsrv, &testUpdownCounterAtomic)
			return nil
		}))

	_, err16 := meter.Int64ObservableUpDownCounter("test/updown_counter_with_attrs",
		metric.WithDescription("Test metric for updown counters with attributes."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			observeUpDownCounter(obsrv, &testUpdownCounterWithAttrsRequestTypeAttr1Atomic, testUpdownCounterWithAttrsRequestTypeAttr1AttrSet)
			observeUpDownCounter(obsrv, &testUpdownCounterWithAttrsRequestTypeAttr2Atomic, testUpdownCounterWithAttrsRequestTypeAttr2AttrSet)
			return nil
		}))

	errs := []error{err0, err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12, err13, err14, err15, err16}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return &otelMetrics{
		ch: ch,
		wg: &wg,
		bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic: &bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic,
		bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic: &bufferedReadFallbackTriggerCountReasonRandomReadDetectedAtomic,
		bufferedReadReadLatency:                                                            bufferedReadReadLatency,
		fileCacheReadBytesCountReadTypeParallelAtomic:                                      &fileCacheReadBytesCountReadTypeParallelAtomic,
		fileCacheReadBytesCountReadTypeRandomAtomic:                                        &fileCacheReadBytesCountReadTypeRandomAtomic,
		fileCacheReadBytesCountReadTypeSequentialAtomic:                                    &fileCacheReadBytesCountReadTypeSequentialAtomic,
		fileCacheReadBytesCountReadTypeUnknownAtomic:                                       &fileCacheReadBytesCountReadTypeUnknownAtomic,
		fileCacheReadCountCacheHitTrueReadTypeParallelAtomic:                               &fileCacheReadCountCacheHitTrueReadTypeParallelAtomic,
		fileCacheReadCountCacheHitTrueReadTypeRandomAtomic:                                 &fileCacheReadCountCacheHitTrueReadTypeRandomAtomic,
		fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic:                             &fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic,
		fileCacheReadCountCacheHitTrueReadTypeUnknownAtomic:                                &fileCacheReadCountCacheHitTrueReadTypeUnknownAtomic,
		fileCacheReadCountCacheHitFalseReadTypeParallelAtomic:                              &fileCacheReadCountCacheHitFalseReadTypeParallelAtomic,
		fileCacheReadCountCacheHitFalseReadTypeRandomAtomic:                                &fileCacheReadCountCacheHitFalseReadTypeRandomAtomic,
		fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic:                            &fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic,
		fileCacheReadCountCacheHitFalseReadTypeUnknownAtomic:                               &fileCacheReadCountCacheHitFalseReadTypeUnknownAtomic,
		fileCacheReadLatencies:                                                             fileCacheReadLatencies,
		fsOpsCountFsOpBatchForgetAtomic:                                                    &fsOpsCountFsOpBatchForgetAtomic,
		fsOpsCountFsOpCreateFileAtomic:                                                     &fsOpsCountFsOpCreateFileAtomic,
		fsOpsCountFsOpCreateLinkAtomic:                                                     &fsOpsCountFsOpCreateLinkAtomic,
		fsOpsCountFsOpCreateSymlinkAtomic:                                                  &fsOpsCountFsOpCreateSymlinkAtomic,
		fsOpsCountFsOpFlushFileAtomic:                                                      &fsOpsCountFsOpFlushFileAtomic,
		fsOpsCountFsOpForgetInodeAtomic:                                                    &fsOpsCountFsOpForgetInodeAtomic,
		fsOpsCountFsOpGetInodeAttributesAtomic:                                             &fsOpsCountFsOpGetInodeAttributesAtomic,
		fsOpsCountFsOpLookUpInodeAtomic:                                                    &fsOpsCountFsOpLookUpInodeAtomic,
		fsOpsCountFsOpMkDirAtomic:                                                          &fsOpsCountFsOpMkDirAtomic,
		fsOpsCountFsOpMkNodeAtomic:                                                         &fsOpsCountFsOpMkNodeAtomic,
		fsOpsCountFsOpOpenDirAtomic:                                                        &fsOpsCountFsOpOpenDirAtomic,
		fsOpsCountFsOpOpenFileAtomic:                                                       &fsOpsCountFsOpOpenFileAtomic,
		fsOpsCountFsOpOthersAtomic:                                                         &fsOpsCountFsOpOthersAtomic,
		fsOpsCountFsOpReadDirAtomic:                                                        &fsOpsCountFsOpReadDirAtomic,
		fsOpsCountFsOpReadDirPlusAtomic:                                                    &fsOpsCountFsOpReadDirPlusAtomic,
		fsOpsCountFsOpReadFileAtomic:                                                       &fsOpsCountFsOpReadFileAtomic,
		fsOpsCountFsOpReadSymlinkAtomic:                                                    &fsOpsCountFsOpReadSymlinkAtomic,
		fsOpsCountFsOpReleaseDirHandleAtomic:                                               &fsOpsCountFsOpReleaseDirHandleAtomic,
		fsOpsCountFsOpReleaseFileHandleAtomic:                                              &fsOpsCountFsOpReleaseFileHandleAtomic,
		fsOpsCountFsOpRenameAtomic:                                                         &fsOpsCountFsOpRenameAtomic,
		fsOpsCountFsOpRmDirAtomic:                                                          &fsOpsCountFsOpRmDirAtomic,
		fsOpsCountFsOpSetInodeAttributesAtomic:                                             &fsOpsCountFsOpSetInodeAttributesAtomic,
		fsOpsCountFsOpSyncFileAtomic:                                                       &fsOpsCountFsOpSyncFileAtomic,
		fsOpsCountFsOpUnlinkAtomic:                                                         &fsOpsCountFsOpUnlinkAtomic,
		fsOpsCountFsOpWriteFileAtomic:                                                      &fsOpsCountFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic:                      &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic:                      &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic:                   &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic:                &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic:               &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic:                          &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic:                      &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic:                      &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic:                   &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic:                         &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic:                &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic:               &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic:                           &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic:                        &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic:                          &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic:                       &fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic:                     &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic:                     &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic:               &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic:              &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic:                       &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic:                    &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic:               &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic:                            &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic:                          &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic:                      &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic:                 &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic:                &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic:                            &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic:               &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic:                         &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic:                           &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic:                        &fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic:                   &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic:                        &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic:             &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic:            &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic:                        &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic:                     &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic:               &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic:          &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic:                       &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic:            &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic:           &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic:                       &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic:          &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic:                 &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic:              &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic:                    &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic:           &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic:          &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic:                      &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic:                   &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic:                  &fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic:                          &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic:                          &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic:                           &fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic:                               &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic:                             &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic:                             &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic:                    &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic:                   &fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic:                               &fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic:                            &fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic:                              &fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic:                           &fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic:                        &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic:                        &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic:                         &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic:                             &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic:                           &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic:                           &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic:                  &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic:                 &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic:                             &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic:                          &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic:                            &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic:                         &fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic:                     &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic:                      &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic:                          &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic:                    &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic:               &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic:              &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic:                          &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic:             &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic:                      &fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic:                          &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic:                          &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic:                           &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic:                               &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic:                             &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic:                             &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic:                         &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic:                   &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic:                               &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic:                            &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic:                              &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic:                           &fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic:                   &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic:                   &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic:                      &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic:                      &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic:                  &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic:             &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic:            &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic:                        &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic:           &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic:                     &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic:                    &fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic:                      &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic:                      &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic:                   &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic:                           &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic:                         &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic:                         &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic:                &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic:               &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic:                           &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic:              &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic:                        &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic:                          &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic:                       &fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic:                        &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic:                        &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic:                         &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic:                             &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic:                           &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic:                           &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic:                       &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic:                  &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic:                 &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic:                             &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic:                &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic:                          &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic:                            &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic:                         &fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic:         &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic:         &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic:      &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic:          &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic: &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic:              &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic:            &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic:            &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic:        &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic:   &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic:  &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic:              &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic: &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic:           &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic:             &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic:          &fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic:                 &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic:                 &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic:              &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic:                  &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic:                      &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic:                    &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOthersAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic:                    &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic:                &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic:           &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic:          &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic:                      &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic:         &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic:                   &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic:                     &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic,
		fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic:                  &fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic,
		fsOpsLatency: fsOpsLatency,
		gcsDownloadBytesCountReadTypeBufferedAtomic:                &gcsDownloadBytesCountReadTypeBufferedAtomic,
		gcsDownloadBytesCountReadTypeParallelAtomic:                &gcsDownloadBytesCountReadTypeParallelAtomic,
		gcsDownloadBytesCountReadTypeRandomAtomic:                  &gcsDownloadBytesCountReadTypeRandomAtomic,
		gcsDownloadBytesCountReadTypeSequentialAtomic:              &gcsDownloadBytesCountReadTypeSequentialAtomic,
		gcsReadBytesCountAtomic:                                    &gcsReadBytesCountAtomic,
		gcsReadCountReadTypeParallelAtomic:                         &gcsReadCountReadTypeParallelAtomic,
		gcsReadCountReadTypeRandomAtomic:                           &gcsReadCountReadTypeRandomAtomic,
		gcsReadCountReadTypeSequentialAtomic:                       &gcsReadCountReadTypeSequentialAtomic,
		gcsReadCountReadTypeUnknownAtomic:                          &gcsReadCountReadTypeUnknownAtomic,
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
		testUpdownCounterAtomic:                                    &testUpdownCounterAtomic,
		testUpdownCounterWithAttrsRequestTypeAttr1Atomic:           &testUpdownCounterWithAttrsRequestTypeAttr1Atomic,
		testUpdownCounterWithAttrsRequestTypeAttr2Atomic:           &testUpdownCounterWithAttrsRequestTypeAttr2Atomic,
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

func observeUpDownCounter(obsrv metric.Int64Observer, counter *atomic.Int64, obsrvOptions ...metric.ObserveOption) {
	obsrv.Observe(counter.Load(), obsrvOptions...)
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
