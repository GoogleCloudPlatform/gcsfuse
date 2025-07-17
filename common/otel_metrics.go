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

package common

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
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
	gcsReaderCountIoMethodClosedAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", "closed")))
	gcsReaderCountIoMethodOpenedAttrSet                                                 = metric.WithAttributeSet(attribute.NewSet(attribute.String("io_method", "opened")))
	gcsRequestCountGcsMethodComposeObjectsAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")))
	gcsRequestCountGcsMethodCreateFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateFolder")))
	gcsRequestCountGcsMethodCreateObjectChunkWriterAttrSet                              = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")))
	gcsRequestCountGcsMethodDeleteFolderAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")))
	gcsRequestCountGcsMethodDeleteObjectAttrSet                                         = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteObject")))
	gcsRequestCountGcsMethodFinalizeUploadAttrSet                                       = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")))
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
	gcsRequestLatenciesGcsMethodCreateFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateFolder")))
	gcsRequestLatenciesGcsMethodCreateObjectChunkWriterAttrSet                          = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")))
	gcsRequestLatenciesGcsMethodDeleteFolderAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")))
	gcsRequestLatenciesGcsMethodDeleteObjectAttrSet                                     = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "DeleteObject")))
	gcsRequestLatenciesGcsMethodFinalizeUploadAttrSet                                   = metric.WithAttributeSet(attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")))
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

type MetricHandle interface {
	FileCacheReadBytesCount(
		inc int64, readType string,
	)
	FileCacheReadCount(
		inc int64, cacheHit bool, readType string,
	)
	FileCacheReadLatencies(
		ctx context.Context, duration time.Duration, cacheHit bool,
	)
	FsOpsCount(
		inc int64, fsOp string,
	)
	FsOpsErrorCount(
		inc int64, fsErrorCategory string, fsOp string,
	)
	FsOpsLatency(
		ctx context.Context, duration time.Duration, fsOp string,
	)
	GcsDownloadBytesCount(
		inc int64, readType string,
	)
	GcsReadBytesCount(
		inc int64,
	)
	GcsReadCount(
		inc int64, readType string,
	)
	GcsReaderCount(
		inc int64, ioMethod string,
	)
	GcsRequestCount(
		inc int64, gcsMethod string,
	)
	GcsRequestLatencies(
		ctx context.Context, duration time.Duration, gcsMethod string,
	)
	GcsRetryCount(
		inc int64, retryErrorCategory string,
	)
}

type histogramRecord struct {
	instrument metric.Int64Histogram
	value      int64
	attributes metric.RecordOption
}

type otelMetrics struct {
	ctx                                                                                context.Context
	ch                                                                                 chan histogramRecord
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
	gcsReaderCountIoMethodClosedAtomic                                                 *atomic.Int64
	gcsReaderCountIoMethodOpenedAtomic                                                 *atomic.Int64
	gcsRequestCountGcsMethodComposeObjectsAtomic                                       *atomic.Int64
	gcsRequestCountGcsMethodCreateFolderAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic                              *atomic.Int64
	gcsRequestCountGcsMethodDeleteFolderAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodDeleteObjectAtomic                                         *atomic.Int64
	gcsRequestCountGcsMethodFinalizeUploadAtomic                                       *atomic.Int64
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
	fileCacheReadLatencies                                                             metric.Int64Histogram
	fsOpsLatency                                                                       metric.Int64Histogram
	gcsRequestLatencies                                                                metric.Int64Histogram
}

func (o *otelMetrics) FileCacheReadBytesCount(
	inc int64, readType string,
) {
	switch readType {
	case "Parallel":
		o.fileCacheReadBytesCountReadTypeParallelAtomic.Add(inc)
	case "Random":
		o.fileCacheReadBytesCountReadTypeRandomAtomic.Add(inc)
	case "Sequential":
		o.fileCacheReadBytesCountReadTypeSequentialAtomic.Add(inc)
	}

}

func (o *otelMetrics) FileCacheReadCount(
	inc int64, cacheHit bool, readType string,
) {
	switch cacheHit {
	case true:
		switch readType {
		case "Parallel":
			o.fileCacheReadCountCacheHitTrueReadTypeParallelAtomic.Add(inc)
		case "Random":
			o.fileCacheReadCountCacheHitTrueReadTypeRandomAtomic.Add(inc)
		case "Sequential":
			o.fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic.Add(inc)
		}
	case false:
		switch readType {
		case "Parallel":
			o.fileCacheReadCountCacheHitFalseReadTypeParallelAtomic.Add(inc)
		case "Random":
			o.fileCacheReadCountCacheHitFalseReadTypeRandomAtomic.Add(inc)
		case "Sequential":
			o.fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic.Add(inc)
		}
	}

}

func (o *otelMetrics) FileCacheReadLatencies(
	ctx context.Context, latency time.Duration, cacheHit bool,
) {
	var record histogramRecord
	switch cacheHit {
	case true:
		record = histogramRecord{instrument: o.fileCacheReadLatencies, value: latency.Microseconds(), attributes: fileCacheReadLatenciesCacheHitTrueAttrSet}
	case false:
		record = histogramRecord{instrument: o.fileCacheReadLatencies, value: latency.Microseconds(), attributes: fileCacheReadLatenciesCacheHitFalseAttrSet}
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) FsOpsCount(
	inc int64, fsOp string,
) {
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
	}

}

func (o *otelMetrics) FsOpsErrorCount(
	inc int64, fsErrorCategory string, fsOp string,
) {
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
		}
	}

}

func (o *otelMetrics) FsOpsLatency(
	ctx context.Context, latency time.Duration, fsOp string,
) {
	var record histogramRecord
	switch fsOp {
	case "BatchForget":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpBatchForgetAttrSet}
	case "CreateFile":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateFileAttrSet}
	case "CreateLink":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateLinkAttrSet}
	case "CreateSymlink":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpCreateSymlinkAttrSet}
	case "Fallocate":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpFallocateAttrSet}
	case "FlushFile":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpFlushFileAttrSet}
	case "ForgetInode":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpForgetInodeAttrSet}
	case "GetInodeAttributes":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpGetInodeAttributesAttrSet}
	case "GetXattr":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpGetXattrAttrSet}
	case "ListXattr":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpListXattrAttrSet}
	case "LookUpInode":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpLookUpInodeAttrSet}
	case "MkDir":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkDirAttrSet}
	case "MkNode":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpMkNodeAttrSet}
	case "OpenDir":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenDirAttrSet}
	case "OpenFile":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpOpenFileAttrSet}
	case "ReadDir":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadDirAttrSet}
	case "ReadFile":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadFileAttrSet}
	case "ReadSymlink":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReadSymlinkAttrSet}
	case "ReleaseDirHandle":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseDirHandleAttrSet}
	case "ReleaseFileHandle":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpReleaseFileHandleAttrSet}
	case "RemoveXattr":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRemoveXattrAttrSet}
	case "Rename":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRenameAttrSet}
	case "RmDir":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpRmDirAttrSet}
	case "SetInodeAttributes":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSetInodeAttributesAttrSet}
	case "SetXattr":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSetXattrAttrSet}
	case "StatFS":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpStatFSAttrSet}
	case "SyncFS":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSyncFSAttrSet}
	case "SyncFile":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpSyncFileAttrSet}
	case "Unlink":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpUnlinkAttrSet}
	case "WriteFile":
		record = histogramRecord{instrument: o.fsOpsLatency, value: latency.Microseconds(), attributes: fsOpsLatencyFsOpWriteFileAttrSet}
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) GcsDownloadBytesCount(
	inc int64, readType string,
) {
	switch readType {
	case "Parallel":
		o.gcsDownloadBytesCountReadTypeParallelAtomic.Add(inc)
	case "Random":
		o.gcsDownloadBytesCountReadTypeRandomAtomic.Add(inc)
	case "Sequential":
		o.gcsDownloadBytesCountReadTypeSequentialAtomic.Add(inc)
	}

}

func (o *otelMetrics) GcsReadBytesCount(
	inc int64,
) {
	o.gcsReadBytesCountAtomic.Add(inc)

}

func (o *otelMetrics) GcsReadCount(
	inc int64, readType string,
) {
	switch readType {
	case "Parallel":
		o.gcsReadCountReadTypeParallelAtomic.Add(inc)
	case "Random":
		o.gcsReadCountReadTypeRandomAtomic.Add(inc)
	case "Sequential":
		o.gcsReadCountReadTypeSequentialAtomic.Add(inc)
	}

}

func (o *otelMetrics) GcsReaderCount(
	inc int64, ioMethod string,
) {
	switch ioMethod {
	case "closed":
		o.gcsReaderCountIoMethodClosedAtomic.Add(inc)
	case "opened":
		o.gcsReaderCountIoMethodOpenedAtomic.Add(inc)
	}

}

func (o *otelMetrics) GcsRequestCount(
	inc int64, gcsMethod string,
) {
	switch gcsMethod {
	case "ComposeObjects":
		o.gcsRequestCountGcsMethodComposeObjectsAtomic.Add(inc)
	case "CreateFolder":
		o.gcsRequestCountGcsMethodCreateFolderAtomic.Add(inc)
	case "CreateObjectChunkWriter":
		o.gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic.Add(inc)
	case "DeleteFolder":
		o.gcsRequestCountGcsMethodDeleteFolderAtomic.Add(inc)
	case "DeleteObject":
		o.gcsRequestCountGcsMethodDeleteObjectAtomic.Add(inc)
	case "FinalizeUpload":
		o.gcsRequestCountGcsMethodFinalizeUploadAtomic.Add(inc)
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
	}

}

func (o *otelMetrics) GcsRequestLatencies(
	ctx context.Context, latency time.Duration, gcsMethod string,
) {
	var record histogramRecord
	switch gcsMethod {
	case "ComposeObjects":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodComposeObjectsAttrSet}
	case "CreateFolder":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateFolderAttrSet}
	case "CreateObjectChunkWriter":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodCreateObjectChunkWriterAttrSet}
	case "DeleteFolder":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteFolderAttrSet}
	case "DeleteObject":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodDeleteObjectAttrSet}
	case "FinalizeUpload":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodFinalizeUploadAttrSet}
	case "GetFolder":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodGetFolderAttrSet}
	case "ListObjects":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodListObjectsAttrSet}
	case "MoveObject":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMoveObjectAttrSet}
	case "MultiRangeDownloader::Add":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodMultiRangeDownloaderAddAttrSet}
	case "NewMultiRangeDownloader":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewMultiRangeDownloaderAttrSet}
	case "NewReader":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodNewReaderAttrSet}
	case "RenameFolder":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodRenameFolderAttrSet}
	case "StatObject":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodStatObjectAttrSet}
	case "UpdateObject":
		record = histogramRecord{instrument: o.gcsRequestLatencies, value: latency.Milliseconds(), attributes: gcsRequestLatenciesGcsMethodUpdateObjectAttrSet}
	}

	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func (o *otelMetrics) GcsRetryCount(
	inc int64, retryErrorCategory string,
) {
	switch retryErrorCategory {
	case "OTHER_ERRORS":
		o.gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic.Add(inc)
	case "STALLED_READ_REQUEST":
		o.gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic.Add(inc)
	}

}

func NewOTelMetrics(ctx context.Context, workers int, bufferSize int) (*otelMetrics, error) {
	ch := make(chan histogramRecord, bufferSize)
	for range workers {
		go func() {
			for record := range ch {
				record.instrument.Record(ctx, record.value, record.attributes)
			}
		}()
	}
	meter := otel.Meter("gcsfuse")
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

	var gcsReaderCountIoMethodClosedAtomic,
		gcsReaderCountIoMethodOpenedAtomic atomic.Int64

	var gcsRequestCountGcsMethodComposeObjectsAtomic,
		gcsRequestCountGcsMethodCreateFolderAtomic,
		gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic,
		gcsRequestCountGcsMethodDeleteFolderAtomic,
		gcsRequestCountGcsMethodDeleteObjectAtomic,
		gcsRequestCountGcsMethodFinalizeUploadAtomic,
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

	_, err0 := meter.Int64ObservableCounter("file_cache/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from file cache along with read type - Sequential/Random"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(fileCacheReadBytesCountReadTypeParallelAtomic.Load(), fileCacheReadBytesCountReadTypeParallelAttrSet)
			obsrv.Observe(fileCacheReadBytesCountReadTypeRandomAtomic.Load(), fileCacheReadBytesCountReadTypeRandomAttrSet)
			obsrv.Observe(fileCacheReadBytesCountReadTypeSequentialAtomic.Load(), fileCacheReadBytesCountReadTypeSequentialAttrSet)
			return nil
		}))

	_, err1 := meter.Int64ObservableCounter("file_cache/read_count",
		metric.WithDescription("Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false"),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(fileCacheReadCountCacheHitTrueReadTypeParallelAtomic.Load(), fileCacheReadCountCacheHitTrueReadTypeParallelAttrSet)
			obsrv.Observe(fileCacheReadCountCacheHitTrueReadTypeRandomAtomic.Load(), fileCacheReadCountCacheHitTrueReadTypeRandomAttrSet)
			obsrv.Observe(fileCacheReadCountCacheHitTrueReadTypeSequentialAtomic.Load(), fileCacheReadCountCacheHitTrueReadTypeSequentialAttrSet)
			obsrv.Observe(fileCacheReadCountCacheHitFalseReadTypeParallelAtomic.Load(), fileCacheReadCountCacheHitFalseReadTypeParallelAttrSet)
			obsrv.Observe(fileCacheReadCountCacheHitFalseReadTypeRandomAtomic.Load(), fileCacheReadCountCacheHitFalseReadTypeRandomAttrSet)
			obsrv.Observe(fileCacheReadCountCacheHitFalseReadTypeSequentialAtomic.Load(), fileCacheReadCountCacheHitFalseReadTypeSequentialAttrSet)
			return nil
		}))

	fileCacheReadLatencies, err2 := meter.Int64Histogram("file_cache/read_latencies",
		metric.WithDescription("The cumulative distribution of the file cache read latencies along with cache hit - true/false."),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err3 := meter.Int64ObservableCounter("fs/ops_count",
		metric.WithDescription("The cumulative number of ops processed by the file system."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(fsOpsCountFsOpBatchForgetAtomic.Load(), fsOpsCountFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsCountFsOpCreateFileAtomic.Load(), fsOpsCountFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsCountFsOpCreateLinkAtomic.Load(), fsOpsCountFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsCountFsOpCreateSymlinkAtomic.Load(), fsOpsCountFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsCountFsOpFallocateAtomic.Load(), fsOpsCountFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsCountFsOpFlushFileAtomic.Load(), fsOpsCountFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsCountFsOpForgetInodeAtomic.Load(), fsOpsCountFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsCountFsOpGetInodeAttributesAtomic.Load(), fsOpsCountFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsCountFsOpGetXattrAtomic.Load(), fsOpsCountFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsCountFsOpListXattrAtomic.Load(), fsOpsCountFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsCountFsOpLookUpInodeAtomic.Load(), fsOpsCountFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsCountFsOpMkDirAtomic.Load(), fsOpsCountFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsCountFsOpMkNodeAtomic.Load(), fsOpsCountFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsCountFsOpOpenDirAtomic.Load(), fsOpsCountFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsCountFsOpOpenFileAtomic.Load(), fsOpsCountFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsCountFsOpReadDirAtomic.Load(), fsOpsCountFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsCountFsOpReadFileAtomic.Load(), fsOpsCountFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsCountFsOpReadSymlinkAtomic.Load(), fsOpsCountFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsCountFsOpReleaseDirHandleAtomic.Load(), fsOpsCountFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsCountFsOpReleaseFileHandleAtomic.Load(), fsOpsCountFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsCountFsOpRemoveXattrAtomic.Load(), fsOpsCountFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsCountFsOpRenameAtomic.Load(), fsOpsCountFsOpRenameAttrSet)
			obsrv.Observe(fsOpsCountFsOpRmDirAtomic.Load(), fsOpsCountFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsCountFsOpSetInodeAttributesAtomic.Load(), fsOpsCountFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsCountFsOpSetXattrAtomic.Load(), fsOpsCountFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsCountFsOpStatFSAtomic.Load(), fsOpsCountFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsCountFsOpSyncFSAtomic.Load(), fsOpsCountFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsCountFsOpSyncFileAtomic.Load(), fsOpsCountFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsCountFsOpUnlinkAtomic.Load(), fsOpsCountFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsCountFsOpWriteFileAtomic.Load(), fsOpsCountFsOpWriteFileAttrSet)
			return nil
		}))

	_, err4 := meter.Int64ObservableCounter("fs/ops_error_count",
		metric.WithDescription("The cumulative number of errors generated by file system operations."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDEVICEERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryDIRNOTEMPTYFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEDIRERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryFILEEXISTSFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINTERRUPTERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDARGUMENTFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryINVALIDOPERATIONFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryIOERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryMISCERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNETWORKERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTADIRFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOTIMPLEMENTEDFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryNOFILEORDIRFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPERMERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryPROCESSRESOURCEMGMTERRORFsOpWriteFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpBatchForgetAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateLinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpCreateSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFallocateAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpFlushFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpForgetInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpGetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpListXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpLookUpInodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpMkNodeAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpOpenFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReadSymlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseDirHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpReleaseFileHandleAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRemoveXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRenameAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpRmDirAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetInodeAttributesAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSetXattrAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpStatFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFSAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpSyncFileAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpUnlinkAttrSet)
			obsrv.Observe(fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAtomic.Load(), fsOpsErrorCountFsErrorCategoryTOOMANYOPENFILESFsOpWriteFileAttrSet)
			return nil
		}))

	fsOpsLatency, err5 := meter.Int64Histogram("fs/ops_latency",
		metric.WithDescription("The cumulative distribution of file system operation latencies"),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err6 := meter.Int64ObservableCounter("gcs/download_bytes_count",
		metric.WithDescription("The cumulative number of bytes downloaded from GCS along with type - Sequential/Random"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(gcsDownloadBytesCountReadTypeParallelAtomic.Load(), gcsDownloadBytesCountReadTypeParallelAttrSet)
			obsrv.Observe(gcsDownloadBytesCountReadTypeRandomAtomic.Load(), gcsDownloadBytesCountReadTypeRandomAttrSet)
			obsrv.Observe(gcsDownloadBytesCountReadTypeSequentialAtomic.Load(), gcsDownloadBytesCountReadTypeSequentialAttrSet)
			return nil
		}))

	_, err7 := meter.Int64ObservableCounter("gcs/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from GCS objects."),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(gcsReadBytesCountAtomic.Load())
			return nil
		}))

	_, err8 := meter.Int64ObservableCounter("gcs/read_count",
		metric.WithDescription("Specifies the number of gcs reads made along with type - Sequential/Random"),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(gcsReadCountReadTypeParallelAtomic.Load(), gcsReadCountReadTypeParallelAttrSet)
			obsrv.Observe(gcsReadCountReadTypeRandomAtomic.Load(), gcsReadCountReadTypeRandomAttrSet)
			obsrv.Observe(gcsReadCountReadTypeSequentialAtomic.Load(), gcsReadCountReadTypeSequentialAttrSet)
			return nil
		}))

	_, err9 := meter.Int64ObservableCounter("gcs/reader_count",
		metric.WithDescription("The cumulative number of GCS object readers opened or closed."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(gcsReaderCountIoMethodClosedAtomic.Load(), gcsReaderCountIoMethodClosedAttrSet)
			obsrv.Observe(gcsReaderCountIoMethodOpenedAtomic.Load(), gcsReaderCountIoMethodOpenedAttrSet)
			return nil
		}))

	_, err10 := meter.Int64ObservableCounter("gcs/request_count",
		metric.WithDescription("The cumulative number of GCS requests processed along with the GCS method."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(gcsRequestCountGcsMethodComposeObjectsAtomic.Load(), gcsRequestCountGcsMethodComposeObjectsAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodCreateFolderAtomic.Load(), gcsRequestCountGcsMethodCreateFolderAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic.Load(), gcsRequestCountGcsMethodCreateObjectChunkWriterAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodDeleteFolderAtomic.Load(), gcsRequestCountGcsMethodDeleteFolderAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodDeleteObjectAtomic.Load(), gcsRequestCountGcsMethodDeleteObjectAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodFinalizeUploadAtomic.Load(), gcsRequestCountGcsMethodFinalizeUploadAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodGetFolderAtomic.Load(), gcsRequestCountGcsMethodGetFolderAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodListObjectsAtomic.Load(), gcsRequestCountGcsMethodListObjectsAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodMoveObjectAtomic.Load(), gcsRequestCountGcsMethodMoveObjectAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic.Load(), gcsRequestCountGcsMethodMultiRangeDownloaderAddAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic.Load(), gcsRequestCountGcsMethodNewMultiRangeDownloaderAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodNewReaderAtomic.Load(), gcsRequestCountGcsMethodNewReaderAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodRenameFolderAtomic.Load(), gcsRequestCountGcsMethodRenameFolderAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodStatObjectAtomic.Load(), gcsRequestCountGcsMethodStatObjectAttrSet)
			obsrv.Observe(gcsRequestCountGcsMethodUpdateObjectAtomic.Load(), gcsRequestCountGcsMethodUpdateObjectAttrSet)
			return nil
		}))

	gcsRequestLatencies, err11 := meter.Int64Histogram("gcs/request_latencies",
		metric.WithDescription("The cumulative distribution of the GCS request latencies."),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	_, err12 := meter.Int64ObservableCounter("gcs/retry_count",
		metric.WithDescription("The cumulative number of retry requests made to GCS."),
		metric.WithUnit(""),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic.Load(), gcsRetryCountRetryErrorCategoryOTHERERRORSAttrSet)
			obsrv.Observe(gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic.Load(), gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAttrSet)
			return nil
		}))

	errs := []error{err0, err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return &otelMetrics{
		ch: ch,
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
		gcsDownloadBytesCountReadTypeParallelAtomic:             &gcsDownloadBytesCountReadTypeParallelAtomic,
		gcsDownloadBytesCountReadTypeRandomAtomic:               &gcsDownloadBytesCountReadTypeRandomAtomic,
		gcsDownloadBytesCountReadTypeSequentialAtomic:           &gcsDownloadBytesCountReadTypeSequentialAtomic,
		gcsReadBytesCountAtomic:                                 &gcsReadBytesCountAtomic,
		gcsReadCountReadTypeParallelAtomic:                      &gcsReadCountReadTypeParallelAtomic,
		gcsReadCountReadTypeRandomAtomic:                        &gcsReadCountReadTypeRandomAtomic,
		gcsReadCountReadTypeSequentialAtomic:                    &gcsReadCountReadTypeSequentialAtomic,
		gcsReaderCountIoMethodClosedAtomic:                      &gcsReaderCountIoMethodClosedAtomic,
		gcsReaderCountIoMethodOpenedAtomic:                      &gcsReaderCountIoMethodOpenedAtomic,
		gcsRequestCountGcsMethodComposeObjectsAtomic:            &gcsRequestCountGcsMethodComposeObjectsAtomic,
		gcsRequestCountGcsMethodCreateFolderAtomic:              &gcsRequestCountGcsMethodCreateFolderAtomic,
		gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic:   &gcsRequestCountGcsMethodCreateObjectChunkWriterAtomic,
		gcsRequestCountGcsMethodDeleteFolderAtomic:              &gcsRequestCountGcsMethodDeleteFolderAtomic,
		gcsRequestCountGcsMethodDeleteObjectAtomic:              &gcsRequestCountGcsMethodDeleteObjectAtomic,
		gcsRequestCountGcsMethodFinalizeUploadAtomic:            &gcsRequestCountGcsMethodFinalizeUploadAtomic,
		gcsRequestCountGcsMethodGetFolderAtomic:                 &gcsRequestCountGcsMethodGetFolderAtomic,
		gcsRequestCountGcsMethodListObjectsAtomic:               &gcsRequestCountGcsMethodListObjectsAtomic,
		gcsRequestCountGcsMethodMoveObjectAtomic:                &gcsRequestCountGcsMethodMoveObjectAtomic,
		gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic:   &gcsRequestCountGcsMethodMultiRangeDownloaderAddAtomic,
		gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic:   &gcsRequestCountGcsMethodNewMultiRangeDownloaderAtomic,
		gcsRequestCountGcsMethodNewReaderAtomic:                 &gcsRequestCountGcsMethodNewReaderAtomic,
		gcsRequestCountGcsMethodRenameFolderAtomic:              &gcsRequestCountGcsMethodRenameFolderAtomic,
		gcsRequestCountGcsMethodStatObjectAtomic:                &gcsRequestCountGcsMethodStatObjectAtomic,
		gcsRequestCountGcsMethodUpdateObjectAtomic:              &gcsRequestCountGcsMethodUpdateObjectAtomic,
		gcsRequestLatencies:                                     gcsRequestLatencies,
		gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic:        &gcsRetryCountRetryErrorCategoryOTHERERRORSAtomic,
		gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic: &gcsRetryCountRetryErrorCategorySTALLEDREADREQUESTAtomic,
	}, nil
}
