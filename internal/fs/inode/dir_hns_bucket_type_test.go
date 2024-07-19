// Copyright 2024 Google Inc. All Rights Reserved.
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
	"context"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DirTestHNSBucketType struct {
	suite.Suite
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock

	in DirInode
	tc metadata.TypeCache
}

func TestDirHandleHNSBucketTypet(t *testing.T) {
	suite.Run(t, new(DirTestHNSBucketType))
}

func (t *DirTestHNSBucketType) SetupTest() {
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := fake.NewFakeBucket(&t.clock, "some_bucket", gcs.Hierarchical)
	t.bucket = gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		bucket)
	// Create the inode. No implicit dirs by default.
	t.resetInode(false, false, true)
}

func (t *DirTestHNSBucketType) TearDownTest() {
	t.in.Unlock()
}

func (t *DirTestHNSBucketType) resetInode(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool) {
	t.resetInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing, config.DefaultTypeCacheMaxSizeMB, typeCacheTTL)
}

func (t *DirTestHNSBucketType) getTypeFromCache(name string) metadata.Type {
	return t.tc.Get(t.in.(*dirInode).cacheClock.Now(), name)
}

func (t *DirTestHNSBucketType) resetInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool, typeCacheMaxSizeMB int, typeCacheTTL time.Duration) {
	if t.in != nil {
		t.in.Unlock()
	}

	t.in = NewDirInode(
		dirInodeID,
		NewDirName(NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		implicitDirs,
		enableManagedFoldersListing,
		enableNonexistentTypeCache,
		typeCacheTTL,
		&t.bucket,
		&t.clock,
		&t.clock,
		typeCacheMaxSizeMB,
		false,
	)

	d := t.in.(*dirInode)
	assert.NotNil(t.T(), d)
	t.tc = d.cache
	assert.NotNil(t.T(), t.tc)

	t.in.Lock()
}

func (t *DirTestHNSBucketType) Test_CreateChildDir_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Call the inode.
	result, err := t.in.CreateChildDir(t.ctx, name)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), result)
	assert.NotNil(t.T(), result.MinObject)
	assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	assert.Equal(t.T(), t.bucket.Name(), result.Bucket.Name())
	assert.Equal(t.T(), result.FullName.GcsObjectName(), result.MinObject.Name)
	assert.Equal(t.T(), objName, result.MinObject.Name)
	assert.False(t.T(), IsSymlink(result.MinObject))
}

func (t *DirTestHNSBucketType) Test_CreateChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Create an existing backing object.
	_, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	assert.NoError(t.T(), err)

	// Call the inode.
	_, err = t.in.CreateChildDir(t.ctx, name)
	assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
}
