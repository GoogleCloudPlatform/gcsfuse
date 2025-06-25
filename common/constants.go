// Copyright 2024 Google LLC
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

package common

// FUSE operation names
const (
	OpStatFS             = "StatFS"
	OpLookUpInode        = "LookUpInode"
	OpGetInodeAttributes = "GetInodeAttributes"
	OpSetInodeAttributes = "SetInodeAttributes"
	OpForgetInode        = "ForgetInode"
	OpBatchForget        = "BatchForget"
	OpMkDir              = "MkDir"
	OpMkNode             = "MkNode"
	OpCreateFile         = "CreateFile"
	OpCreateLink         = "CreateLink"
	OpCreateSymlink      = "CreateSymlink"
	OpRename             = "Rename"
	OpRmDir              = "RmDir"
	OpUnlink             = "Unlink"
	OpOpenDir            = "OpenDir"
	OpReadDir            = "ReadDir"
	OpReleaseDirHandle   = "ReleaseDirHandle"
	OpOpenFile           = "OpenFile"
	OpReadFile           = "ReadFile"
	OpWriteFile          = "WriteFile"
	OpSyncFile           = "SyncFile"
	OpFlushFile          = "FlushFile"
	OpReleaseFileHandle  = "ReleaseFileHandle"
	OpReadSymlink        = "ReadSymlink"
	OpRemoveXattr        = "RemoveXattr"
	OpGetXattr           = "GetXattr"
	OpListXattr          = "ListXattr"
	OpSetXattr           = "SetXattr"
	OpFallocate          = "Fallocate"
	OpSyncFS             = "SyncFS"
)
