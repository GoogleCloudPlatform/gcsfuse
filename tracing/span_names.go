// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tracing contains the constants and utilities for OpenTelemetry
// instrumentation within gcsfuse.
package tracing

// Span name constants for GCSFuse operations.
// These constants define the canonical names for spans created during FUSE
// operations. Using these constants ensures consistency and better readability and a single source of truth where all the span names are listed.

const (
	StatFS             = "StatFS"
	LookUpInode        = "LookUpInode"
	GetInodeAttributes = "GetInodeAttributes"
	SetInodeAttributes = "SetInodeAttributes"
	ForgetInode        = "ForgetInode"
	BatchForget        = "BatchForget"
	MkDir              = "MkDir"
	MkNode             = "MkNode"
	CreateFile         = "CreateFile"
	CreateLink         = "CreateLink"
	CreateSymlink      = "CreateSymlink"
	Rename             = "Rename"
	RmDir              = "RmDir"
	Unlink             = "Unlink"
	OpenDir            = "OpenDir"
	ReadDir            = "ReadDir"
	ReadDirPlus        = "ReadDirPlus"
	ReleaseDirHandle   = "ReleaseDirHandle"
	OpenFile           = "OpenFile"
	ReadFile           = "ReadFile"
	WriteFile          = "WriteFile"
	SyncFile           = "SyncFile"
	FlushFile          = "FlushFile"
	ReleaseFileHandle  = "ReleaseFileHandle"
	ReadSymlink        = "ReadSymlink"
	RemoveXattr        = "RemoveXattr"
	GetXattr           = "GetXattr"
	ListXattr          = "ListXattr"
	SetXattr           = "SetXattr"
	Fallocate          = "Fallocate"
	SyncFS             = "SyncFS"
)
