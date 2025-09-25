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
	bufferedReadDownloadBlockLatencyStatusCancelledAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", string(CancelledAttr))))
	bufferedReadDownloadBlockLatencyStatusFailedAttrSet                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", string(FailedAttr))))
	bufferedReadDownloadBlockLatencyStatusSuccessfulAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", string(SuccessfulAttr))))
	bufferedReadFallbackTriggerCountReasonInsufficientMemoryAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("reason", string(InsufficientMemoryAttr))))
	bufferedReadFallbackTriggerCountReasonRandomReadDetectedAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("reason", string(RandomReadDetectedAttr))))
	bufferedReadScheduledBlockCountStatusCancelledAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", string(CancelledAttr))))
	bufferedReadScheduledBlockCountStatusFailedAttrSet                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", string(FailedAttr))))
	bufferedReadScheduledBlockCountStatusSuccessfulAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("status", string(SuccessfulAttr))))
	fileCacheReadBytesCountReadTypeParallelAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(ParallelAttr))))
	fileCacheReadBytesCountReadTypeRandomAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(RandomAttr))))
	fileCacheReadBytesCountReadTypeSequentialAttrSet                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(SequentialAttr))))
	fileCacheReadCountCacheHitTrueReadTypeParallelAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(ParallelAttr))))
	fileCacheReadCountCacheHitTrueReadTypeRandomAttrSet                                 = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(RandomAttr))))
	fileCacheReadCountCacheHitTrueReadTypeSequentialAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(SequentialAttr))))
	fileCacheReadCountCacheHitFalseReadTypeParallelAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", string(ParallelAttr))))
	fileCacheReadCountCacheHitFalseReadTypeRandomAttrSet                                = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", string(RandomAttr))))
	fileCacheReadCountCacheHitFalseReadTypeSequentialAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", string(SequentialAttr))))
	fileCacheReadLatenciesCacheHitTrueAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true)))
	fileCacheReadLatenciesCacheHitFalseAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false)))
	fsOpsCountFsOpBatchForgetAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsCountFsOpCreateFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsCountFsOpCreateLinkAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsCountFsOpCreateSymlinkAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsCountFsOpFallocateAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(FallocateAttr))))
	fsOpsCountFsOpFlushFileAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsCountFsOpForgetInodeAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsCountFsOpGetInodeAttributesAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsCountFsOpGetXattrAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsCountFsOpListXattrAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsCountFsOpLookUpInodeAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsCountFsOpMkDirAttrSet                                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(MkDirAttr))))
	fsOpsCountFsOpMkNodeAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsCountFsOpOpenDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsCountFsOpOpenFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsCountFsOpReadDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsCountFsOpReadDirPlusAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsCountFsOpReadFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsCountFsOpReadSymlinkAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsCountFsOpReleaseDirHandleAttrSet                                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsCountFsOpReleaseFileHandleAttrSet                                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsCountFsOpRemoveXattrAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsCountFsOpRenameAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(RenameAttr))))
	fsOpsCountFsOpRmDirAttrSet                                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(RmDirAttr))))
	fsOpsCountFsOpSetInodeAttributesAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsCountFsOpSetXattrAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsCountFsOpStatFSAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(StatFSAttr))))
	fsOpsCountFsOpSyncFSAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsCountFsOpSyncFileAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsCountFsOpUnlinkAttrSet                                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsCountFsOpWriteFileAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAttrSet               = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAttrSet                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAttrSet                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAttrSet                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAttrSet                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAttrSet      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAttrSet   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAttrSet  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAttrSet             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAttrSet                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAttrSet              = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(FallocateAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(MkDirAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAttrSet                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAttrSet           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAttrSet          = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAttrSet                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(RenameAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAttrSet                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(RmDirAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAttrSet         = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(StatFSAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAttrSet                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAttrSet                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(WriteFileAttr))))
	fsOpsLatencyFsOpBatchForgetAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(BatchForgetAttr))))
	fsOpsLatencyFsOpCreateFileAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(CreateFileAttr))))
	fsOpsLatencyFsOpCreateLinkAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(CreateLinkAttr))))
	fsOpsLatencyFsOpCreateSymlinkAttrSet                                                = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(CreateSymlinkAttr))))
	fsOpsLatencyFsOpFallocateAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(FallocateAttr))))
	fsOpsLatencyFsOpFlushFileAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(FlushFileAttr))))
	fsOpsLatencyFsOpForgetInodeAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ForgetInodeAttr))))
	fsOpsLatencyFsOpGetInodeAttributesAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(GetInodeAttributesAttr))))
	fsOpsLatencyFsOpGetXattrAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(GetXattrAttr))))
	fsOpsLatencyFsOpListXattrAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ListXattrAttr))))
	fsOpsLatencyFsOpLookUpInodeAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(LookUpInodeAttr))))
	fsOpsLatencyFsOpMkDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(MkDirAttr))))
	fsOpsLatencyFsOpMkNodeAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(MkNodeAttr))))
	fsOpsLatencyFsOpOpenDirAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(OpenDirAttr))))
	fsOpsLatencyFsOpOpenFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(OpenFileAttr))))
	fsOpsLatencyFsOpReadDirAttrSet                                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadDirAttr))))
	fsOpsLatencyFsOpReadDirPlusAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadDirPlusAttr))))
	fsOpsLatencyFsOpReadFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadFileAttr))))
	fsOpsLatencyFsOpReadSymlinkAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReadSymlinkAttr))))
	fsOpsLatencyFsOpReleaseDirHandleAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReleaseDirHandleAttr))))
	fsOpsLatencyFsOpReleaseFileHandleAttrSet                                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(ReleaseFileHandleAttr))))
	fsOpsLatencyFsOpRemoveXattrAttrSet                                                  = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(RemoveXattrAttr))))
	fsOpsLatencyFsOpRenameAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(RenameAttr))))
	fsOpsLatencyFsOpRmDirAttrSet                                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(RmDirAttr))))
	fsOpsLatencyFsOpSetInodeAttributesAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SetInodeAttributesAttr))))
	fsOpsLatencyFsOpSetXattrAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SetXattrAttr))))
	fsOpsLatencyFsOpStatFSAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(StatFSAttr))))
	fsOpsLatencyFsOpSyncFSAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SyncFSAttr))))
	fsOpsLatencyFsOpSyncFileAttrSet                                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(SyncFileAttr))))
	fsOpsLatencyFsOpUnlinkAttrSet                                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(UnlinkAttr))))
	fsOpsLatencyFsOpWriteFileAttrSet                                                    = metric.WithAttributeSet(attribute.NewSet(attribute.String("fs_op", string(WriteFileAttr))))
	gcsDownloadBytesCountReadTypeParallelAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(ParallelAttr))))
	gcsDownloadBytesCountReadTypeRandomAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(RandomAttr))))
	gcsDownloadBytesCountReadTypeSequentialAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(SequentialAttr))))
	gcsReadCountReadTypeParallelAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(ParallelAttr))))
	gcsReadCountReadTypeRandomAttrSet                                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(RandomAttr))))
	gcsReadCountReadTypeSequentialAttrSet                                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", string(SequentialAttr))))
	gcsReaderCountIoMethodReadHandleAttrSet                                             = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", string(ReadHandleAttr))))
	gcsReaderCountIoMethodClosedAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", string(ClosedAttr))))
	gcsReaderCountIoMethodOpenedAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", string(OpenedAttr))))
	gcsRequestCountGcsMethodComposeObjectsAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(ComposeObjectsAttr))))
	gcsRequestCountGcsMethodCopyObjectAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CopyObjectAttr))))
	gcsRequestCountGcsMethodCreateAppendableObjectWriterAttrSet                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateAppendableObjectWriterAttr))))
	gcsRequestCountGcsMethodCreateFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateFolderAttr))))
	gcsRequestCountGcsMethodCreateObjectAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateObjectAttr))))
	gcsRequestCountGcsMethodCreateObjectChunkWriterAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateObjectChunkWriterAttr))))
	gcsRequestCountGcsMethodDeleteFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(DeleteFolderAttr))))
	gcsRequestCountGcsMethodDeleteObjectAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(DeleteObjectAttr))))
	gcsRequestCountGcsMethodFinalizeUploadAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(FinalizeUploadAttr))))
	gcsRequestCountGcsMethodFlushPendingWritesAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(FlushPendingWritesAttr))))
	gcsRequestCountGcsMethodGetFolderAttrSet                                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(GetFolderAttr))))
	gcsRequestCountGcsMethodListObjectsAttrSet                                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(ListObjectsAttr))))
	gcsRequestCountGcsMethodMoveObjectAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(MoveObjectAttr))))
	gcsRequestCountGcsMethodMultiRangeDownloaderAddAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(MultiRangeDownloaderAddAttr))))
	gcsRequestCountGcsMethodNewMultiRangeDownloaderAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(NewMultiRangeDownloaderAttr))))
	gcsRequestCountGcsMethodNewReaderAttrSet                                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(NewReaderAttr))))
	gcsRequestCountGcsMethodRenameFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(RenameFolderAttr))))
	gcsRequestCountGcsMethodStatObjectAttrSet                                           = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(StatObjectAttr))))
	gcsRequestCountGcsMethodUpdateObjectAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(UpdateObjectAttr))))
	gcsRequestLatenciesGcsMethodComposeObjectsAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(ComposeObjectsAttr))))
	gcsRequestLatenciesGcsMethodCopyObjectAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CopyObjectAttr))))
	gcsRequestLatenciesGcsMethodCreateAppendableObjectWriterAttrSet                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateAppendableObjectWriterAttr))))
	gcsRequestLatenciesGcsMethodCreateFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateFolderAttr))))
	gcsRequestLatenciesGcsMethodCreateObjectAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateObjectAttr))))
	gcsRequestLatenciesGcsMethodCreateObjectChunkWriterAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(CreateObjectChunkWriterAttr))))
	gcsRequestLatenciesGcsMethodDeleteFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(DeleteFolderAttr))))
	gcsRequestLatenciesGcsMethodDeleteObjectAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(DeleteObjectAttr))))
	gcsRequestLatenciesGcsMethodFinalizeUploadAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(FinalizeUploadAttr))))
	gcsRequestLatenciesGcsMethodFlushPendingWritesAttrSet                               = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(FlushPendingWritesAttr))))
	gcsRequestLatenciesGcsMethodGetFolderAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(GetFolderAttr))))
	gcsRequestLatenciesGcsMethodListObjectsAttrSet                                      = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(ListObjectsAttr))))
	gcsRequestLatenciesGcsMethodMoveObjectAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(MoveObjectAttr))))
	gcsRequestLatenciesGcsMethodMultiRangeDownloaderAddAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(MultiRangeDownloaderAddAttr))))
	gcsRequestLatenciesGcsMethodNewMultiRangeDownloaderAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(NewMultiRangeDownloaderAttr))))
	gcsRequestLatenciesGcsMethodNewReaderAttrSet                                        = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(NewReaderAttr))))
	gcsRequestLatenciesGcsMethodRenameFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(RenameFolderAttr))))
	gcsRequestLatenciesGcsMethodStatObjectAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(StatObjectAttr))))
	gcsRequestLatenciesGcsMethodUpdateObjectAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", string(UpdateObjectAttr))))
	gcsRetryCountRetryErrorCategoryOTHERERRORSAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("retry_error_category", string(OTHERERRORSAttr))))
	gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAttrSet                            = metric.WithAttributeSet(attribute.NewSet(attribute.String("retry_error_category", string(STALLEDREADREQUESTAttr))))
	testUpdownCounterWithAttrsRequestTypeAttr1AttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("request_type", string(Attr1Attr))))
	testUpdownCounterWithAttrsRequestTypeAttr2AttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("request_type", string(Attr2Attr))))
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
	testUpdownCounterAtomic                                                            *atomic.Int64
	testUpdownCounterWithAttrsRequestTypeAttr1Atomic                                   *atomic.Int64
	testUpdownCounterWithAttrsRequestTypeAttr2Atomic                                   *atomic.Int64
	bufferedReadDownloadBlockLatency                                                   metric.Int64Histogram
	bufferedReadReadLatency                                                            metric.Int64Histogram
	fileCacheReadLatencies                                                             metric.Int64Histogram
	fsOpsLatency                                                                       metric.Int64Histogram
	gcsRequestLatencies                                                                metric.Int64Histogram
}

func (o *otelMetrics) BufferedReadDownloadBlockLatency(
	ctx context.Context, latency time.Duration, status MetricAttr) {
	var record histogramRecord
	switch status {
	case CancelledAttr:
		record = histogramRecord{ctx: ctx, instrument: o.bufferedReadDownloadBlockLatency, value: latency.Microseconds(), attributes: bufferedReadDownloadBlockLatencyStatusCancelledAttrSet}
	case FailedAttr:
		record = histogramRecord{ctx: ctx, instrument: o.bufferedReadDownloadBlockLatency, value: latency.Microseconds(), attributes: bufferedReadDownloadBlockLatencyStatusFailedAttrSet}
	case SuccessfulAttr:
		record = histogramRecord{ctx: ctx, instrument: o.bufferedReadDownloadBlockLatency, value: latency.Microseconds(), attributes: bufferedReadDownloadBlockLatencyStatusSuccessfulAttrSet}
	default:
		updateUnrecognizedAttribute(string(status))
		return
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) BufferedReadFallbackTriggerCount(
	inc int64, reason MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric buffered_read/fallback_trigger_count received a negative increment: %d", inc)
		return
	}
	switch reason {
	case InsufficientMemoryAttr:
		o.bufferedReadFallbackTriggerCountReasonInsufficientMemoryAtomic.Add(inc)
	case RandomReadDetectedAttr:
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

func (o *otelMetrics) BufferedReadScheduledBlockCount(
	inc int64, status MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric buffered_read/scheduled_block_count received a negative increment: %d", inc)
		return
	}
	switch status {
	case CancelledAttr:
		o.bufferedReadScheduledBlockCountStatusCancelledAtomic.Add(inc)
	case FailedAttr:
		o.bufferedReadScheduledBlockCountStatusFailedAtomic.Add(inc)
	case SuccessfulAttr:
		o.bufferedReadScheduledBlockCountStatusSuccessfulAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(status))
		return
	}
}

func (o *otelMetrics) FileCacheReadBytesCount(
	inc int64, readType MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric file_cache/read_bytes_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case ParallelAttr:
		o.fileCacheReadBytesCountReadTypeParallelAtomic.Add(inc)
	case RandomAttr:
		o.fileCacheReadBytesCountReadTypeRandomAtomic.Add(inc)
	case SequentialAttr:
		o.fileCacheReadBytesCountReadTypeSequentialAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(readType))
		return
	}
}

func (o *otelMetrics) FileCacheReadCount(
	inc int64, cacheHit bool, readType MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric file_cache/read_count received a negative increment: %d", inc)
		return
	}
	switch cacheHit {
	case true:
		switch readType {
		case ParallelAttr:
			o.fileCacheReadCountCacheHitTrueReadTypeParallelAtomic.Add(inc)
		case RandomAttr:
			o.fileCacheReadCountCacheHitTrueReadTypeRandomAtomic.Add(inc)
		case SequentialAttr:
			o.fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(readType))
			return
		}
	case false:
		switch readType {
		case ParallelAttr:
			o.fileCacheReadCountCacheHitFalseReadTypeParallelAtomic.Add(inc)
		case RandomAttr:
			o.fileCacheReadCountCacheHitFalseReadTypeRandomAtomic.Add(inc)
		case SequentialAttr:
			o.fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic.Add(inc)
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
	inc int64, fsOp MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric fs/ops_count received a negative increment: %d", inc)
		return
	}
	switch fsOp {
	case BatchForgetAttr:
		o.fsOpsCountFsOpBatchForgetAtomic.Add(inc)
	case CreateFileAttr:
		o.fsOpsCountFsOpCreateFileAtomic.Add(inc)
	case CreateLinkAttr:
		o.fsOpsCountFsOpCreateLinkAtomic.Add(inc)
	case CreateSymlinkAttr:
		o.fsOpsCountFsOpCreateSymlinkAtomic.Add(inc)
	case FallocateAttr:
		o.fsOpsCountFsOpFallocateAtomic.Add(inc)
	case FlushFileAttr:
		o.fsOpsCountFsOpFlushFileAtomic.Add(inc)
	case ForgetInodeAttr:
		o.fsOpsCountFsOpForgetInodeAtomic.Add(inc)
	case GetInodeAttributesAttr:
		o.fsOpsCountFsOpGetInodeAttributesAtomic.Add(inc)
	case GetXattrAttr:
		o.fsOpsCountFsOpGetXattrAtomic.Add(inc)
	case ListXattrAttr:
		o.fsOpsCountFsOpListXattrAtomic.Add(inc)
	case LookUpInodeAttr:
		o.fsOpsCountFsOpLookUpInodeAtomic.Add(inc)
	case MkDirAttr:
		o.fsOpsCountFsOpMkDirAtomic.Add(inc)
	case MkNodeAttr:
		o.fsOpsCountFsOpMkNodeAtomic.Add(inc)
	case OpenDirAttr:
		o.fsOpsCountFsOpOpenDirAtomic.Add(inc)
	case OpenFileAttr:
		o.fsOpsCountFsOpOpenFileAtomic.Add(inc)
	case ReadDirAttr:
		o.fsOpsCountFsOpReadDirAtomic.Add(inc)
	case ReadDirPlusAttr:
		o.fsOpsCountFsOpReadDirPlusAtomic.Add(inc)
	case ReadFileAttr:
		o.fsOpsCountFsOpReadFileAtomic.Add(inc)
	case ReadSymlinkAttr:
		o.fsOpsCountFsOpReadSymlinkAtomic.Add(inc)
	case ReleaseDirHandleAttr:
		o.fsOpsCountFsOpReleaseDirHandleAtomic.Add(inc)
	case ReleaseFileHandleAttr:
		o.fsOpsCountFsOpReleaseFileHandleAtomic.Add(inc)
	case RemoveXattrAttr:
		o.fsOpsCountFsOpRemoveXattrAtomic.Add(inc)
	case RenameAttr:
		o.fsOpsCountFsOpRenameAtomic.Add(inc)
	case RmDirAttr:
		o.fsOpsCountFsOpRmDirAtomic.Add(inc)
	case SetInodeAttributesAttr:
		o.fsOpsCountFsOpSetInodeAttributesAtomic.Add(inc)
	case SetXattrAttr:
		o.fsOpsCountFsOpSetXattrAtomic.Add(inc)
	case StatFSAttr:
		o.fsOpsCountFsOpStatFSAtomic.Add(inc)
	case SyncFSAttr:
		o.fsOpsCountFsOpSyncFSAtomic.Add(inc)
	case SyncFileAttr:
		o.fsOpsCountFsOpSyncFileAtomic.Add(inc)
	case UnlinkAttr:
		o.fsOpsCountFsOpUnlinkAtomic.Add(inc)
	case WriteFileAttr:
		o.fsOpsCountFsOpWriteFileAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(fsOp))
		return
	}
}

func (o *otelMetrics) FsOpsErrorCount(
	inc int64, fsErrorCategory MetricAttr, fsOp MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric fs/ops_error_count received a negative increment: %d", inc)
		return
	}
	switch fsErrorCategory {
	case DEVICEERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case DIRNOTEMPTYAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FILEDIRERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case FILEEXISTSAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case INTERRUPTERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case INVALIDARGUMENTAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case INVALIDOPERATIONAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case IOERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case MISCERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case NETWORKERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case NOTADIRAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case NOTIMPLEMENTEDAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case NOFILEORDIRAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case PERMERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case PROCESSRESOURCEMGMTERRORAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
			o.fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic.Add(inc)
		default:
			updateUnrecognizedAttribute(string(fsOp))
			return
		}
	case TOOMANYOPENFILESAttr:
		switch fsOp {
		case BatchForgetAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic.Add(inc)
		case CreateFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic.Add(inc)
		case CreateLinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic.Add(inc)
		case CreateSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic.Add(inc)
		case FallocateAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic.Add(inc)
		case FlushFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic.Add(inc)
		case ForgetInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic.Add(inc)
		case GetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic.Add(inc)
		case GetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic.Add(inc)
		case ListXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic.Add(inc)
		case LookUpInodeAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic.Add(inc)
		case MkDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic.Add(inc)
		case MkNodeAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic.Add(inc)
		case OpenDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic.Add(inc)
		case OpenFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic.Add(inc)
		case ReadDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic.Add(inc)
		case ReadDirPlusAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirPlusAtomic.Add(inc)
		case ReadFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic.Add(inc)
		case ReadSymlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic.Add(inc)
		case ReleaseDirHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic.Add(inc)
		case ReleaseFileHandleAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic.Add(inc)
		case RemoveXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic.Add(inc)
		case RenameAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic.Add(inc)
		case RmDirAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic.Add(inc)
		case SetInodeAttributesAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic.Add(inc)
		case SetXattrAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic.Add(inc)
		case StatFSAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic.Add(inc)
		case SyncFSAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic.Add(inc)
		case SyncFileAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic.Add(inc)
		case UnlinkAttr:
			o.fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic.Add(inc)
		case WriteFileAttr:
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
	ctx context.Context, latency time.Duration, fsOp MetricAttr) {
	var record histogramRecord
	switch fsOp {
	case BatchForgetAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpBatchForgetAttrSet}
	case CreateFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateFileAttrSet}
	case CreateLinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateLinkAttrSet}
	case CreateSymlinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateSymlinkAttrSet}
	case FallocateAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpFallocateAttrSet}
	case FlushFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpFlushFileAttrSet}
	case ForgetInodeAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpForgetInodeAttrSet}
	case GetInodeAttributesAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpGetInodeAttributesAttrSet}
	case GetXattrAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpGetXattrAttrSet}
	case ListXattrAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpListXattrAttrSet}
	case LookUpInodeAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpLookUpInodeAttrSet}
	case MkDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkDirAttrSet}
	case MkNodeAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkNodeAttrSet}
	case OpenDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenDirAttrSet}
	case OpenFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenFileAttrSet}
	case ReadDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadDirAttrSet}
	case ReadDirPlusAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadDirPlusAttrSet}
	case ReadFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadFileAttrSet}
	case ReadSymlinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadSymlinkAttrSet}
	case ReleaseDirHandleAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseDirHandleAttrSet}
	case ReleaseFileHandleAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseFileHandleAttrSet}
	case RemoveXattrAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRemoveXattrAttrSet}
	case RenameAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRenameAttrSet}
	case RmDirAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRmDirAttrSet}
	case SetInodeAttributesAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSetInodeAttributesAttrSet}
	case SetXattrAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSetXattrAttrSet}
	case StatFSAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpStatFSAttrSet}
	case SyncFSAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSyncFSAttrSet}
	case SyncFileAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSyncFileAttrSet}
	case UnlinkAttr:
		record = histogramRecord{ctx: ctx, instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpUnlinkAttrSet}
	case WriteFileAttr:
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
	inc int64, readType MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/download_bytes_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case ParallelAttr:
		o.gcsDownloadBytesCountReadTypeParallelAtomic.Add(inc)
	case RandomAttr:
		o.gcsDownloadBytesCountReadTypeRandomAtomic.Add(inc)
	case SequentialAttr:
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
	inc int64, readType MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/read_count received a negative increment: %d", inc)
		return
	}
	switch readType {
	case ParallelAttr:
		o.gcsReadCountReadTypeParallelAtomic.Add(inc)
	case RandomAttr:
		o.gcsReadCountReadTypeRandomAtomic.Add(inc)
	case SequentialAttr:
		o.gcsReadCountReadTypeSequentialAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(readType))
		return
	}
}

func (o *otelMetrics) GcsReaderCount(
	inc int64, ioMethod MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/reader_count received a negative increment: %d", inc)
		return
	}
	switch ioMethod {
	case ReadHandleAttr:
		o.gcsReaderCountIoMethodReadHandleAtomic.Add(inc)
	case ClosedAttr:
		o.gcsReaderCountIoMethodClosedAtomic.Add(inc)
	case OpenedAttr:
		o.gcsReaderCountIoMethodOpenedAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(ioMethod))
		return
	}
}

func (o *otelMetrics) GcsRequestCount(
	inc int64, gcsMethod MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/request_count received a negative increment: %d", inc)
		return
	}
	switch gcsMethod {
	case ComposeObjectsAttr:
		o.gcsRequestCountGcsMethodComposeObjectsAtomic.Add(inc)
	case CopyObjectAttr:
		o.gcsRequestCountGcsMethodCopyObjectAtomic.Add(inc)
	case CreateAppendableObjectWriterAttr:
		o.gcsRequestCountGcsMethodCreateAppendableObjectWriterAtomic.Add(inc)
	case CreateFolderAttr:
		o.gcsRequestCountGcsMethodCreateFolderAtomic.Add(inc)
	case CreateObjectAttr:
		o.gcsRequestCountGcsMethodCreateObjectAtomic.Add(inc)
	case CreateObjectChunkWriterAttr:
		o.gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic.Add(inc)
	case DeleteFolderAttr:
		o.gcsRequestCountGcsMethodDeleteFolderAtomic.Add(inc)
	case DeleteObjectAttr:
		o.gcsRequestCountGcsMethodDeleteObjectAtomic.Add(inc)
	case FinalizeUploadAttr:
		o.gcsRequestCountGcsMethodFinalizeUploadAtomic.Add(inc)
	case FlushPendingWritesAttr:
		o.gcsRequestCountGcsMethodFlushPendingWritesAtomic.Add(inc)
	case GetFolderAttr:
		o.gcsRequestCountGcsMethodGetFolderAtomic.Add(inc)
	case ListObjectsAttr:
		o.gcsRequestCountGcsMethodListObjectsAtomic.Add(inc)
	case MoveObjectAttr:
		o.gcsRequestCountGcsMethodMoveObjectAtomic.Add(inc)
	case MultiRangeDownloaderAddAttr:
		o.gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic.Add(inc)
	case NewMultiRangeDownloaderAttr:
		o.gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic.Add(inc)
	case NewReaderAttr:
		o.gcsRequestCountGcsMethodNewReaderAtomic.Add(inc)
	case RenameFolderAttr:
		o.gcsRequestCountGcsMethodRenameFolderAtomic.Add(inc)
	case StatObjectAttr:
		o.gcsRequestCountGcsMethodStatObjectAtomic.Add(inc)
	case UpdateObjectAttr:
		o.gcsRequestCountGcsMethodUpdateObjectAtomic.Add(inc)
	default:
		updateUnrecognizedAttribute(string(gcsMethod))
		return
	}
}

func (o *otelMetrics) GcsRequestLatencies(
	ctx context.Context, latency time.Duration, gcsMethod MetricAttr) {
	var record histogramRecord
	switch gcsMethod {
	case ComposeObjectsAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodComposeObjectsAttrSet}
	case CopyObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCopyObjectAttrSet}
	case CreateAppendableObjectWriterAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateAppendableObjectWriterAttrSet}
	case CreateFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateFolderAttrSet}
	case CreateObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateObjectAttrSet}
	case CreateObjectChunkWriterAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateObjectChunkWriterAttrSet}
	case DeleteFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteFolderAttrSet}
	case DeleteObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteObjectAttrSet}
	case FinalizeUploadAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodFinalizeUploadAttrSet}
	case FlushPendingWritesAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodFlushPendingWritesAttrSet}
	case GetFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodGetFolderAttrSet}
	case ListObjectsAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodListObjectsAttrSet}
	case MoveObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMoveObjectAttrSet}
	case MultiRangeDownloaderAddAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMultiRangeDownloaderAddAttrSet}
	case NewMultiRangeDownloaderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewMultiRangeDownloaderAttrSet}
	case NewReaderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewReaderAttrSet}
	case RenameFolderAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodRenameFolderAttrSet}
	case StatObjectAttr:
		record = histogramRecord{ctx: ctx, instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodStatObjectAttrSet}
	case UpdateObjectAttr:
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
	inc int64, retryErrorCategory MetricAttr) {
	if inc < 0 {
		logger.Errorf("Counter metric gcs/retry_count received a negative increment: %d", inc)
		return
	}
	switch retryErrorCategory {
	case OTHERERRORSAttr:
		o.gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic.Add(inc)
	case STALLEDREADREQUESTAttr:
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
	inc int64, requestType MetricAttr) {
	switch requestType {
	case Attr1Attr:
		o.testUpdownCounterWithAttrsRequestTypeAttr1Atomic.Add(inc)
	case Attr2Attr:
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

	var testUpdownCounterAtomic atomic.Int64

	var testUpdownCounterWithAttrsRequestTypeAttr1Atomic,
		testUpdownCounterWithAttrsRequestTypeAttr2Atomic atomic.Int64

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

	_, err17 := meter.Int64ObservableUpDownCounter("test/updown_counter",
		metric.WithDescription("Test metric for updown counters."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			observeUpDownCounter(obsrv, &testUpdownCounterAtomic)
			return nil
		}))

	_, err18 := meter.Int64ObservableUpDownCounter("test/updown_counter_with_attrs",
		metric.WithDescription("Test metric for updown counters with attributes."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			observeUpDownCounter(obsrv, &testUpdownCounterWithAttrsRequestTypeAttr1Atomic, testUpdownCounterWithAttrsRequestTypeAttr1AttrSet)
			observeUpDownCounter(obsrv, &testUpdownCounterWithAttrsRequestTypeAttr2Atomic, testUpdownCounterWithAttrsRequestTypeAttr2AttrSet)
			return nil
		}))

	errs := []error{err0, err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12, err13, err14, err15, err16, err17, err18}
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
