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
	"sync"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/vipnydav/gcsfuse/v3/internal/gcsx"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
	"golang.org/x/net/context"
)

// When this custom metadata key is present in an object record, it is to be
// treated as a symlink. For use in testing only; other users should detect
// this with IsSymlink.
const SymlinkMetadataKey = "gcsfuse_symlink_target"

// IsSymlink Does the supplied object represent a symlink inode?
func IsSymlink(m *gcs.MinObject) bool {
	if m == nil {
		return false
	}

	_, ok := m.Metadata[SymlinkMetadataKey]
	return ok
}

type SymlinkInode struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	id               fuseops.InodeID
	name             Name
	bucket           *gcsx.SyncerBucket
	sourceGeneration Generation
	attrs            fuseops.InodeAttributes
	target           string

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu sync.Mutex

	// GUARDED_BY(mu)
	lc lookupCount
}

var _ Inode = &SymlinkInode{}

// Create a symlink inode for the supplied object record.
//
// REQUIRES: IsSymlink(o)
func NewSymlinkInode(
	id fuseops.InodeID,
	name Name,
	bucket *gcsx.SyncerBucket,
	m *gcs.MinObject,
	attrs fuseops.InodeAttributes) (s *SymlinkInode) {
	// Create the inode.
	s = &SymlinkInode{
		id:     id,
		name:   name,
		bucket: bucket,
		sourceGeneration: Generation{
			Object:   m.Generation,
			Metadata: m.MetaGeneration,
			Size:     m.Size,
		},
		attrs: fuseops.InodeAttributes{
			Nlink: 1,
			Uid:   attrs.Uid,
			Gid:   attrs.Gid,
			Mode:  attrs.Mode,
			Atime: m.Updated,
			Ctime: m.Updated,
			Mtime: m.Updated,
		},
		target: m.Metadata[SymlinkMetadataKey],
	}

	// Set up lookup counting.
	s.lc.Init(id)

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (s *SymlinkInode) Lock() {
	s.mu.Lock()
}

func (s *SymlinkInode) Unlock() {
	s.mu.Unlock()
}

func (s *SymlinkInode) ID() fuseops.InodeID {
	return s.id
}

func (s *SymlinkInode) Name() Name {
	return s.name
}

// SourceGeneration returns the object generation from which this inode was branched.
//
// LOCKS_REQUIRED(s)
func (s *SymlinkInode) SourceGeneration() Generation {
	return s.sourceGeneration
}

// LOCKS_REQUIRED(s.mu)
func (s *SymlinkInode) IncrementLookupCount() {
	s.lc.Inc()
}

// LOCKS_REQUIRED(s.mu)
func (s *SymlinkInode) DecrementLookupCount(n uint64) (destroy bool) {
	destroy = s.lc.Dec(n)
	return
}

// LOCKS_REQUIRED(s.mu)
func (s *SymlinkInode) Destroy() (err error) {
	// Nothing to do.
	return
}

func (s *SymlinkInode) Attributes(
	ctx context.Context, clobberedCheck bool) (attrs fuseops.InodeAttributes, err error) {
	attrs = s.attrs
	return
}

// Target returns the target of the symlink.
func (s *SymlinkInode) Target() (target string) {
	target = s.target
	return
}

func (s *SymlinkInode) Unlink() {
}

// Bucket returns the bucket that owns this inode.
func (s *SymlinkInode) Bucket() *gcsx.SyncerBucket {
	return s.bucket
}

// Source returns the MinObject from which this inode was created.
func (s *SymlinkInode) Source() *gcs.MinObject {
	return &gcs.MinObject{
		Name:           s.name.GcsObjectName(),
		Generation:     s.sourceGeneration.Object,
		MetaGeneration: s.sourceGeneration.Metadata,
		Size:           s.sourceGeneration.Size,
		Metadata: map[string]string{
			SymlinkMetadataKey: s.target,
		},
		Updated: s.attrs.Mtime,
	}
}
