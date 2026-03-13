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
	// --- Cache and Prefetching Operations ---

	// FileCacheRead tracks read operations specifically from the local file cache.
	FileCacheRead = "file.cache.read"
	// FileCacheWrite tracks write or population operations into the local file cache.
	FileCacheWrite = "file.cache.write"
	// ReadPrefetchBlockPoolGen monitors the generation/lifecycle of the prefetch block pool.
	ReadPrefetchBlockPoolGen = "prefetch.block_pool_gen.read"
	// DownloadPrefetchBlock triggers a network request to pre-fill a specific data block.
	DownloadPrefetchBlock = "prefetch.block.download"
	// WaitForPrefetchBlock measures the time a process waits for a prefetch operation to complete.
	WaitForPrefetchBlock = "prefetch.block.wait"
	// ReadFromPrefetchBlock executes a read directly from a successfully prefetched data block.
	ReadFromPrefetchBlock = "prefetch.block.read"
	// ScheduleBlockForDownload adds a specific file block to the asynchronous download queue.
	ScheduleBlockForDownload = "download.block.schedule"

	// --- Metadata and Inode Operations ---

	// StatFS retrieves aggregate filesystem statistics (e.g., total capacity, free space).
	StatFS = "fs.stat_fs"
	// LookUpInode resolves a specific filename within a directory to its unique inode.
	LookUpInode = "fs.inode.lookup"
	// GetInodeAttributes retrieves metadata such as size, mode, and timestamps for an inode.
	GetInodeAttributes = "fs.inode.get_attributes"
	// SetInodeAttributes updates metadata (e.g., permissions or ownership) for an inode.
	SetInodeAttributes = "fs.inode.set_attributes"
	// ForgetInode informs the kernel that it no longer needs to track a specific inode.
	ForgetInode = "fs.inode.forget"
	// BatchForget allows the kernel to release multiple inode references in a single call.
	BatchForget = "fs.batch_forget"

	// --- Directory Lifecycle ---

	// MkDir creates a new directory entry.
	MkDir = "fs.dir.mk"
	// RmDir removes an existing, empty directory.
	RmDir = "fs.dir.rm"
	// OpenDir opens a directory for content enumeration.
	OpenDir = "fs.dir.open"
	// ReadDir reads entries from an open directory handle.
	ReadDir = "fs.dir.read"
	// ReadDirPlus reads directory entries along with their associated metadata/attributes.
	ReadDirPlus = "fs.dir.read_plus"
	// ReleaseDirHandle closes the handle for a directory and releases associated resources.
	ReleaseDirHandle = "fs.dir.release_handle"

	// --- File Lifecycle and I/O ---

	// CreateFile creates and opens a new file within the filesystem.
	CreateFile = "fs.file.create"
	// OpenFile opens an existing file for reading or writing.
	OpenFile = "fs.file.open"
	// ReadFile executes a standard read operation from a file handle.
	ReadFile = "fs.file.read"
	// WriteFile executes a standard write operation to a file handle.
	WriteFile = "fs.file.write"
	// SyncFile flushes buffered data for a specific file to stable storage.
	SyncFile = "fs.file.sync"
	// FlushFile is called on every close of a file descriptor to flush changes.
	FlushFile = "fs.file.flush"
	// ReleaseFileHandle closes the file handle and releases system resources.
	ReleaseFileHandle = "fs.file.release_handle"

	// --- Links and Extended Attributes ---

	// CreateLink creates a hard link to an existing inode.
	CreateLink = "fs.link.create"
	// CreateSymlink creates a symbolic (soft) link.
	CreateSymlink = "fs.symlink.create"
	// ReadSymlink reads the target path stored within a symbolic link.
	ReadSymlink = "fs.symlink.read"
	// Rename changes the name or location of a file or directory.
	Rename = "fs.rename"
	// Unlink removes a name from the filesystem; if it was the last link, the file is deleted.
	Unlink = "fs.unlink"
	// GetXattr retrieves the value of an extended attribute.
	GetXattr = "fs.xattr.get"
	// SetXattr sets or updates an extended attribute.
	SetXattr = "fs.xattr.set"
	// ListXattr lists the names of extended attributes assigned to a file.
	ListXattr = "fs.xattr.list"
	// RemoveXattr deletes an extended attribute from a file.
	RemoveXattr = "fs.xattr.remove"

	// --- Advanced Filesystem Operations ---

	// MkNode creates a filesystem node (file, device special file, or named pipe).
	MkNode = "fs.mknode"
	// Fallocate ensures that disk space is pre-allocated for a file.
	Fallocate = "fs.fallocate"
	// SyncFS flushes all buffered data for the entire filesystem to disk.
	SyncFS = "fs.sync_fs"
)
