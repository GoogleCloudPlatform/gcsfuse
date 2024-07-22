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
	"errors"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
)

type HNSDirTest struct {
	suite.Suite
	ctx        context.Context
	bucket     gcsx.SyncerBucket
	in         DirInode
	mockBucket *storage.TestifyMockBucket
}

func TestHNSDirSuite(testSuite *testing.T) { suite.Run(testSuite, new(HNSDirTest)) }

func (t *HNSDirTest) SetupTest() {
	t.ctx = context.Background()
	t.mockBucket = new(storage.TestifyMockBucket)
	t.bucket = gcsx.NewSyncerBucket(
		1,
		".gcsfuse_tmp/",
		t.mockBucket)
	t.resetDirInode(false, false, true)
}

func (t *HNSDirTest) resetDirInode(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool) {
	t.resetDirInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing, config.DefaultTypeCacheMaxSizeMB, typeCacheTTL)
}

func (t *HNSDirTest) resetDirInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool, typeCacheMaxSizeMB int, typeCacheTTL time.Duration) {
	var anyPastTime timeutil.SimulatedClock
	anyPastTime.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

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
		&anyPastTime,
		&anyPastTime,
		typeCacheMaxSizeMB,
		false,
	)

	d := t.in.(*dirInode)
	assert.NotEqual(t.T(), nil, d)
	t.in.Lock()
}

func (t *HNSDirTest) TearDownTest() {
	t.in.Unlock()
}

func (t *HNSDirTest) TestShouldFindExplicitHNSFolder() {
	const name = "qux"
	dirName := path.Join(dirInodeName, name) + "/"
	folder := &gcs.Folder{
		Name:           dirName,
		MetaGeneration: int64(1),
	}
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(folder, nil)

	// Look up with the name.
	result, err := findExplicitFolder(t.ctx, &t.bucket, NewDirName(t.in.Name(), name))

	assert.Nil(t.T(), err)
	assert.NotEqual(t.T(), nil, result.MinObject)
	assert.Equal(t.T(), dirName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirName, result.MinObject.Name)
	assert.Equal(t.T(), int64(1), result.MinObject.MetaGeneration)

}

func (t *HNSDirTest) TestShouldReturnNilWhenGCSFolderNotFoundForInHNS() {
	notFoundErr := &gcs.NotFoundError{Err: errors.New("storage: object doesn't exist")}
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	// Look up with the name.
	result, err := findExplicitFolder(t.ctx, &t.bucket, NewDirName(t.in.Name(), "not-present"))

	assert.Nil(t.T(), err)
	assert.Nil(t.T(), result)
}
