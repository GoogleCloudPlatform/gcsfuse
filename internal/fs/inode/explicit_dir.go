// Copyright 2015 Google LLC
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

package inode

import (
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
)

// An inode representing a directory backed by an object in GCS with a specific
// generation.
type ExplicitDirInode interface {
	DirInode
	SourceGeneration() Generation
}

// Create an explicit dir inode backed by the supplied object. See notes on
// NewDirInode for more.
func NewExplicitDirInode(
	id fuseops.InodeID,
	name Name,
	m *gcs.MinObject,
	attrs fuseops.InodeAttributes,
	implicitDirs bool,
	includeFoldersAsPrefixes bool,
	enableNonexistentTypeCache bool,
	typeCacheTTL time.Duration,
	bucket *gcsx.SyncerBucket,
	mtimeClock timeutil.Clock,
	cacheClock timeutil.Clock,
	typeCacheMaxSizeMB int64,
	enableHNS bool) (d ExplicitDirInode) {
	wrapped := NewDirInode(
		id,
		name,
		attrs,
		implicitDirs,
		includeFoldersAsPrefixes,
		enableNonexistentTypeCache,
		typeCacheTTL,
		bucket,
		mtimeClock,
		cacheClock,
		typeCacheMaxSizeMB,
		enableHNS)

	dirInode := &explicitDirInode{
		dirInode: wrapped.(*dirInode),
	}

	if m != nil {
		dirInode.generation = Generation{
			Object:   m.Generation,
			Metadata: m.MetaGeneration,
			Size:     m.Size,
		}
	}

	d = dirInode
	return
}

type explicitDirInode struct {
	*dirInode
	generation Generation
}

func (d *explicitDirInode) SourceGeneration() (gen Generation) {
	gen = d.generation
	return
}
