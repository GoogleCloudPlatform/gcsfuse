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

package tracing

const (
	IS_CACHE_HIT   = "cache.hit"         // Indicates if the response was served from cache or not.
	BYTES_READ     = "read.size"         // Indicates the number of bytes read from the given reader
	BYTES_UPLOADED = "write.chunk.size"  // Indicates the number of bytes uploaded
	OBJECT_NAME    = "write.object_name" // Indicates the object name uploaded
	INODE_NAME        = "inode.name"        // Indicates the name of the inode looked up
	INODE_MODE        = "inode.mode"        // Indicates the mode/type of the inode looked up
	RENAME_SOURCE_DIR = "rename.source_dir" // Indicates the source directory GCS path for rename
	RENAME_TARGET_DIR = "rename.target_dir" // Indicates the target directory GCS path for rename
)
