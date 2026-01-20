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

package inode

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/mock"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
)

type hnsDirTest struct {
	suite.Suite
	ctx        context.Context
	bucket     gcsx.SyncerBucket
	in         DirInode
	mockBucket *storagemock.TestifyMockBucket
	typeCache  metadata.TypeCache
	fixedTime  timeutil.SimulatedClock
}

type HNSDirTest struct {
	hnsDirTest
}

type NonHNSDirTest struct {
	hnsDirTest
}

func TestHNSDirSuiteWithHierarchicalBucket(testSuite *testing.T) {
	suite.Run(testSuite, &HNSDirTest{})
}

func TestHNSDirSuiteWithNonHierarchicalBucket(testSuite *testing.T) {
	suite.Run(testSuite, &NonHNSDirTest{})
}

func (t *hnsDirTest) setupTestSuite(hierarchical bool) {
	t.ctx = context.Background()
	t.mockBucket = new(storagemock.TestifyMockBucket)
	t.mockBucket.On("BucketType").Return(gcs.BucketType{Hierarchical: hierarchical})
	t.bucket = gcsx.NewSyncerBucket(
		1,
		ChunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		t.mockBucket)
	t.resetDirInode(false, false, true)
}

func (t *HNSDirTest) SetupTest() {
	t.setupTestSuite(true)
}

func (t *NonHNSDirTest) SetupTest() {
	t.setupTestSuite(false)
}

func (t *hnsDirTest) resetDirInode(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool) {
	t.resetDirInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing, 4, typeCacheTTL)
}

func (t *hnsDirTest) resetDirInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool, typeCacheMaxSizeMB int64, typeCacheTTL time.Duration) {
	t.fixedTime.SetTime(time.Date(2024, 7, 22, 2, 15, 0, 0, time.Local))

	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: enableManagedFoldersListing},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: typeCacheMaxSizeMB},
		EnableHns:                    true,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   isTypeCacheDeprecationEnabled,
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
		enableNonexistentTypeCache,
		typeCacheTTL,
		&t.bucket,
		&t.fixedTime,
		&t.fixedTime,
		semaphore.NewWeighted(10),
		config,
	)

	d := t.in.(*dirInode)
	assert.NotNil(t.T(), d)
	t.typeCache = d.cache
	if !d.IsTypeCacheDeprecated() {
		assert.NotNil(t.T(), t.typeCache)
	} else {
		assert.Nil(t.T(), t.typeCache)
	}

	//Lock dir Inode
	t.in.Lock()
}

func (t *hnsDirTest) createDirInode(dirInodeName string) DirInode {
	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: false},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: 4},
		EnableHns:                    false,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   isTypeCacheDeprecationEnabled,
	}

	return NewDirInode(
		5,
		NewDirName(NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		false,
		true,
		typeCacheTTL,
		&t.bucket,
		&t.fixedTime,
		&t.fixedTime,
		semaphore.NewWeighted(10),
		config,
	)
}

func (t *HNSDirTest) TearDownTest() {
	t.in.Unlock()
}

func (t *NonHNSDirTest) TearDownTest() {
	t.in.Unlock()
}

func (t *HNSDirTest) TestShouldFindExplicitHNSFolder() {
	const name = "qux"
	dirName := path.Join(dirInodeName, name) + "/"
	folder := &gcs.Folder{
		Name: dirName,
	}
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(folder, nil)

	// Look up with the name.
	result, err := findExplicitFolder(t.ctx, &t.bucket, NewDirName(t.in.Name(), name), false)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.NotEqual(t.T(), nil, result.MinObject)
	assert.Equal(t.T(), dirName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirName, result.Folder.Name)
}

func (t *HNSDirTest) TestShouldReturnNilWhenGCSFolderNotFoundForInHNS() {
	notFoundErr := &gcs.NotFoundError{Err: errors.New("storage: object doesn't exist")}
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	// Look up with the name.
	result, err := findExplicitFolder(t.ctx, &t.bucket, NewDirName(t.in.Name(), "not-present"), false)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.Nil(t.T(), result)
}

func (t *HNSDirTest) TestLookUpChildWithConflictMarkerName() {
	const name = "qux"
	dirName := path.Join(dirInodeName, name) + "/"
	folder := &gcs.Folder{
		Name: dirName,
	}
	statObjectRequest := gcs.StatObjectRequest{
		Name: path.Join(dirInodeName, name),
	}
	getFolderRequest := gcs.GetFolderRequest{
		Name: dirName,
	}
	object := gcs.MinObject{Name: dirName}
	t.mockBucket.On("GetFolder", mock.Anything, &getFolderRequest).Return(folder, nil)
	t.mockBucket.On("StatObject", mock.Anything, &statObjectRequest).Return(&object, &gcs.ExtendedObjectAttributes{}, nil)

	c, err := t.in.LookUpChild(t.ctx, name+"\n")

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), dirName, c.MinObject.Name)
}

func (t *HNSDirTest) TestLookUpChildShouldCheckOnlyForExplicitHNSDirectory() {
	const name = "qux"
	dirName := path.Join(dirInodeName, name) + "/"
	// mock get folder call
	folder := &gcs.Folder{
		Name: dirName,
	}
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(folder, nil)
	if !t.in.IsTypeCacheDeprecated() {
		t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.ExplicitDirType)
	}

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), dirName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirName, result.Folder.Name)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
	}
}

func (t *HNSDirTest) TestLookUpChildShouldCheckForHNSDirectoryWhenTypeNotPresent() {
	const name = "unknown_type"
	dirName := path.Join(dirInodeName, name) + "/"
	// mock get folder call
	folder := &gcs.Folder{
		Name: dirName,
	}
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(folder, nil)
	notFoundErr := &gcs.NotFoundError{Err: errors.New("storage: object doesn't exist")}
	t.mockBucket.On("StatObject", mock.Anything, mock.Anything).Return(nil, nil, notFoundErr)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.typeCache.Get(t.fixedTime.Now(), name))
	}
	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), dirName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirName, result.Folder.Name)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
	}
}

func (t *HNSDirTest) TestLookUpChildShouldCheckForHNSDirectoryWhenTypeIsRegularFileType() {
	const name = "file_type"
	fileName := path.Join(dirInodeName, name)
	// mock stat object call
	minObject := &gcs.MinObject{
		Name:           fileName,
		MetaGeneration: int64(1),
		Generation:     int64(2),
	}
	attrs := &gcs.ExtendedObjectAttributes{
		ContentType:  "plain/text",
		StorageClass: "DEFAULT",
		CacheControl: "some-value",
	}
	t.mockBucket.On("StatObject", mock.Anything, mock.Anything).Return(minObject, attrs, nil)
	if !t.in.IsTypeCacheDeprecated() {
		t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.RegularFileType)
	}
	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), fileName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), fileName, result.MinObject.Name)
	assert.Equal(t.T(), int64(2), result.MinObject.Generation)
	assert.Equal(t.T(), int64(1), result.MinObject.MetaGeneration)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.fixedTime.Now(), name))
	}
}

func (t *HNSDirTest) TestLookUpChildShouldCheckForHNSDirectoryWhenTypeIsSymlinkType() {
	const name = "file_type"
	fileName := path.Join(dirInodeName, name)
	// mock stat object call
	minObject := &gcs.MinObject{
		Name:           fileName,
		MetaGeneration: int64(1),
		Generation:     int64(2),
		Metadata:       map[string]string{"gcsfuse_symlink_target": "link"},
	}
	attrs := &gcs.ExtendedObjectAttributes{
		ContentType:  "plain/text",
		StorageClass: "DEFAULT",
		CacheControl: "some-value",
	}
	t.mockBucket.On("StatObject", mock.Anything, mock.Anything).Return(minObject, attrs, nil)
	if !t.in.IsTypeCacheDeprecated() {
		t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.SymlinkType)
	}
	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), fileName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), fileName, result.MinObject.Name)
	assert.Equal(t.T(), int64(2), result.MinObject.Generation)
	assert.Equal(t.T(), int64(1), result.MinObject.MetaGeneration)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.SymlinkType, t.typeCache.Get(t.fixedTime.Now(), name))
	}
}

func (t *HNSDirTest) TestLookUpChildShouldCheckForHNSDirectoryWhenTypeIsNonExistentType() {
	const name = "file_type"
	if !t.in.IsTypeCacheDeprecated() {
		t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.NonexistentType)
	}
	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	assert.Nil(t.T(), err)
	assert.Nil(t.T(), result)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *HNSDirTest) TestRenameFolderWithGivenName() {
	const (
		dirName       = "qux"
		renameDirName = "rename"
	)
	folderName := path.Join(dirInodeName, dirName) + "/"
	renameFolderName := path.Join(dirInodeName, renameDirName) + "/"
	renameFolder := gcs.Folder{Name: renameFolderName}
	t.mockBucket.On("RenameFolder", t.ctx, folderName, renameFolderName).Return(&renameFolder, nil)

	// Attempt to rename the folder.
	f, err := t.in.RenameFolder(t.ctx, folderName, renameFolderName)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	// Verify the renamed folder exists.
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), renameFolderName, f.Name)
}

func (t *HNSDirTest) TestRenameFolderWithNonExistentSourceFolder() {
	var notFoundErr *gcs.NotFoundError
	const (
		dirName       = "qux"
		renameDirName = "rename"
	)
	folderName := path.Join(dirInodeName, dirName) + "/"
	renameFolderName := path.Join(dirInodeName, renameDirName) + "/"
	t.mockBucket.On("RenameFolder", t.ctx, folderName, renameFolderName).Return(nil, &gcs.NotFoundError{})

	// Attempt to rename the folder.
	f, err := t.in.RenameFolder(t.ctx, folderName, renameFolderName)

	t.mockBucket.AssertExpectations(t.T())
	assert.True(t.T(), errors.As(err, &notFoundErr))
	assert.Nil(t.T(), f)
}

func (t *HNSDirTest) TestRenameFileWithGivenName() {
	const (
		fileName       = "qux"
		renameFileName = "rename"
	)
	oldObjName := path.Join(dirInodeName, fileName)
	newObjName := path.Join(dirInodeName, renameFileName)
	var metaGeneration int64 = 0
	moveObjectReq := gcs.MoveObjectRequest{
		SrcName:                       oldObjName,
		DstName:                       newObjName,
		SrcGeneration:                 0,
		SrcMetaGenerationPrecondition: &metaGeneration,
	}
	oldObj := gcs.MinObject{Name: oldObjName}
	newObj := gcs.Object{Name: newObjName}
	t.mockBucket.On("MoveObject", t.ctx, &moveObjectReq).Return(&newObj, nil)

	// Attempt to rename the file.
	f, err := t.in.RenameFile(t.ctx, &oldObj, path.Join(dirInodeName, renameFileName))

	t.mockBucket.AssertExpectations(t.T())
	// Verify the renamed file exists.
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), newObjName, f.Name)
}

func (t *HNSDirTest) TestRenameFileWithNonExistentSourceFile() {
	const (
		fileName       = "qux"
		renameFileName = "rename"
	)
	oldObjName := path.Join(dirInodeName, fileName)
	newObjName := path.Join(dirInodeName, renameFileName)
	var metaGeneration int64 = 0
	moveObjectReq := gcs.MoveObjectRequest{
		SrcName:                       oldObjName,
		DstName:                       newObjName,
		SrcGeneration:                 0,
		SrcMetaGenerationPrecondition: &metaGeneration,
	}
	oldObj := gcs.MinObject{Name: oldObjName}
	var notFoundErr *gcs.NotFoundError
	t.mockBucket.On("MoveObject", t.ctx, &moveObjectReq).Return(nil, &gcs.NotFoundError{})

	// Attempt to rename the file.
	f, err := t.in.RenameFile(t.ctx, &oldObj, newObjName)

	t.mockBucket.AssertExpectations(t.T())
	assert.True(t.T(), errors.As(err, &notFoundErr))
	assert.Nil(t.T(), f)
}

func (t *NonHNSDirTest) TestDeleteChildDir_WhenImplicitDirFlagTrueOnNonHNSBucket() {
	const folderName = "folder"
	dirName := path.Join(dirInodeName, folderName) + "/"
	dirIn := t.createDirInode(dirName)

	// Delete dir
	err := t.in.DeleteChildDir(t.ctx, folderName, true, dirIn)

	t.mockBucket.AssertExpectations(t.T()) // Verify mock interactions
	assert.NoError(t.T(), err)             // Ensure no error occurred
}

func (t *NonHNSDirTest) TestDeleteChildDir_WhenImplicitDirFlagFalseAndNonHNSBucket_DeleteObjectGiveSuccess() {
	const name = "dir"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("DeleteObject", t.ctx, &deleteObjectReq).Return(nil)
	dirIn := t.createDirInode(dirName)

	err := t.in.DeleteChildDir(t.ctx, name, false, dirIn)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
	}
	assert.False(t.T(), dirIn.IsUnlinked())
}
func (t *NonHNSDirTest) TestDeleteChildDir_WithImplicitDirFlagFalseAndNonHNSBucket_DeleteObjectThrowAnError() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("DeleteObject", t.ctx, &deleteObjectReq).Return(fmt.Errorf("mock error"))
	dirIn := t.createDirInode(dirName)

	// Delete dir .
	err := t.in.DeleteChildDir(t.ctx, name, false, dirIn)

	t.mockBucket.AssertExpectations(t.T())
	assert.NotNil(t.T(), err)
	assert.False(t.T(), dirIn.IsUnlinked())
}

func (t *HNSDirTest) TestDeleteChildDir_WithImplicitDirFlagFalseAndBucketTypeIsHNS_DeleteObjectGiveSuccessDeleteFolderThrowAnError() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("DeleteObject", t.ctx, &deleteObjectReq).Return(nil)
	t.mockBucket.On("DeleteFolder", t.ctx, dirName).Return(fmt.Errorf("mock error"))
	dirIn := t.createDirInode(dirName)

	// Delete dir .
	err := t.in.DeleteChildDir(t.ctx, name, false, dirIn)

	t.mockBucket.AssertExpectations(t.T())
	assert.NotNil(t.T(), err)
	assert.False(t.T(), dirIn.IsUnlinked())
}

func (t *HNSDirTest) TestDeleteChildDir_WithImplicitDirFlagFalseAndBucketTypeIsHNS_DeleteObjectThrowAnErrorDeleteFolderGiveSuccess() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("DeleteObject", t.ctx, &deleteObjectReq).Return(fmt.Errorf("mock error"))
	t.mockBucket.On("DeleteFolder", t.ctx, dirName).Return(nil)
	dirIn := t.createDirInode(dirName)

	// Delete dir .
	err := t.in.DeleteChildDir(t.ctx, name, false, dirIn)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
	}
	assert.True(t.T(), dirIn.IsUnlinked())
}

func (t *HNSDirTest) TestDeleteChildDir_WithImplicitDirFlagFalseAndBucketTypeIsHNS_DeleteObjectAndDeleteFolderThrowAnError() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("DeleteObject", t.ctx, &deleteObjectReq).Return(fmt.Errorf("mock error"))
	t.mockBucket.On("DeleteFolder", t.ctx, dirName).Return(fmt.Errorf("mock delete folder error"))
	dirIn := t.createDirInode(dirName)

	// Delete dir .
	err := t.in.DeleteChildDir(t.ctx, name, false, dirIn)

	t.mockBucket.AssertExpectations(t.T())
	assert.NotNil(t.T(), err)
	// It will ignore the error that came from deleteObject.
	assert.Equal(t.T(), err.Error(), "DeleteFolder: mock delete folder error")
	assert.False(t.T(), dirIn.IsUnlinked())
}

func (t *HNSDirTest) TestCreateChildDirWhenBucketTypeIsHNSWithFailure() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	t.mockBucket.On("CreateFolder", t.ctx, dirName).Return(nil, fmt.Errorf("mock error"))

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NotNil(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
	}
}

func (t *HNSDirTest) TestCreateChildDirWhenBucketTypeIsHNSWithSuccess() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	folder := gcs.Folder{Name: dirName}
	t.mockBucket.On("CreateFolder", t.ctx, dirName).Return(&folder, nil)

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), result)
	assert.Equal(t.T(), dirName, result.Folder.Name)
	assert.Equal(t.T(), dirName, result.FullName.objectName)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
	}
}

func (t *NonHNSDirTest) TestCreateChildDirWhenBucketTypeIsNonHNSWithFailure() {
	const name = "folder"
	var preCond int64
	dirName := path.Join(dirInodeName, name) + "/"
	createObjectReq := gcs.CreateObjectRequest{Name: dirName, Contents: strings.NewReader(""), GenerationPrecondition: &preCond}
	t.mockBucket.On("CreateObject", t.ctx, &createObjectReq).Return(nil, fmt.Errorf("mock error"))

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NotNil(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
	}
}

func (t *NonHNSDirTest) TestCreateChildDirWhenBucketTypeIsNonHNSWithSuccess() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	var preCond int64
	createObjectReq := gcs.CreateObjectRequest{Name: dirName, Contents: strings.NewReader(""), GenerationPrecondition: &preCond}
	object := gcs.Object{Name: dirName}
	t.mockBucket.On("CreateObject", t.ctx, &createObjectReq).Return(&object, nil)

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), result)
	assert.Equal(t.T(), dirName, result.MinObject.Name)
	assert.Equal(t.T(), dirName, result.FullName.objectName)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
	}
}

func (t *HNSDirTest) TestDeleteObjects() {
	// Arrange
	objectNames := []string{"dir1/file1.txt", "dir2/"}
	t.mockBucket.On("DeleteObject", t.ctx, &gcs.DeleteObjectRequest{Name: "dir1/file1.txt"}).Return(nil)
	// Mock for recursive deletion of dir2/
	listReq := &gcs.ListObjectsRequest{
		Prefix:                   "dir2/",
		MaxResults:               MaxResultsForListObjectsCall,
		Delimiter:                "/",
		ContinuationToken:        "",
		IncludeFoldersAsPrefixes: true,
		IsTypeCacheDeprecated:    t.in.IsTypeCacheDeprecated(),
	}
	listResp := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir2/file2.txt"},
		},
		CollapsedRuns: []string{"dir2/subdir/"},
	}
	t.mockBucket.On("ListObjects", mock.Anything, listReq).Return(listResp, nil)
	t.mockBucket.On("DeleteObject", mock.Anything, &gcs.DeleteObjectRequest{Name: "dir2/file2.txt"}).Return(nil)
	// Mock for recursive call on subdir/
	listReqSubdir := &gcs.ListObjectsRequest{
		Prefix:                   "dir2/subdir/",
		MaxResults:               MaxResultsForListObjectsCall,
		Delimiter:                "/",
		ContinuationToken:        "",
		IncludeFoldersAsPrefixes: true,
		IsTypeCacheDeprecated:    t.in.IsTypeCacheDeprecated(),
	}
	listRespSubdir := &gcs.Listing{}
	t.mockBucket.On("ListObjects", mock.Anything, listReqSubdir).Return(listRespSubdir, nil)
	t.mockBucket.On("DeleteObject", mock.Anything, &gcs.DeleteObjectRequest{Name: "dir2/subdir/"}).Return(nil)
	t.mockBucket.On("DeleteFolder", mock.Anything, "dir2/subdir/").Return(nil)
	t.mockBucket.On("DeleteObject", mock.Anything, &gcs.DeleteObjectRequest{Name: "dir2/"}).Return(nil)
	t.mockBucket.On("DeleteFolder", mock.Anything, "dir2/").Return(nil)

	// Act
	err := t.in.DeleteObjects(t.ctx, objectNames)

	// Assert
	assert.NoError(t.T(), err)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *HNSDirTest) TestReadEntriesInHierarchicalBucket() {
	t.resetDirInode(false, false, true)
	const (
		folder1     = "folder1"
		folder2     = "folder2"
		file1       = "file1"
		file2       = "file2"
		file3       = "file3"
		implicitDir = "implicitDir" // In Hierarchical bucket implicitDir will also become folder.
	)
	tok := ""
	obj1 := gcs.MinObject{Name: path.Join(dirInodeName, folder1) + "/"}
	obj2 := gcs.MinObject{Name: path.Join(dirInodeName, folder2) + "/"}
	obj3 := gcs.MinObject{Name: path.Join(dirInodeName, folder2, file1)}
	obj4 := gcs.MinObject{Name: path.Join(dirInodeName, file2)}
	obj5 := gcs.MinObject{Name: path.Join(dirInodeName, implicitDir, file3)}
	minObjects := []*gcs.MinObject{&obj1, &obj2, &obj3, &obj4, &obj5}
	collapsedRuns := []string{path.Join(dirInodeName, folder1) + "/", path.Join(dirInodeName, folder2) + "/", path.Join(dirInodeName, implicitDir) + "/"}
	listing := gcs.Listing{
		MinObjects:    minObjects,
		CollapsedRuns: collapsedRuns,
	}
	listObjectReq := gcs.ListObjectsRequest{
		Prefix:                   dirInodeName,
		Delimiter:                "/",
		IncludeFoldersAsPrefixes: true,
		IncludeTrailingDelimiter: false,
		MaxResults:               5000,
		ProjectionVal:            gcs.NoAcl,
		IsTypeCacheDeprecated:    t.in.IsTypeCacheDeprecated(),
	}
	t.mockBucket.On("ListObjects", t.ctx, &listObjectReq).Return(&listing, nil)

	entries, _, _, err := t.in.ReadEntries(t.ctx, tok)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 6, len(entries))
	for i := range 6 {
		switch entries[i].Name {
		case folder1:
			assert.Equal(t.T(), folder1, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_Directory, entries[i].Type)
			if !t.in.IsTypeCacheDeprecated() {
				assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), folder1))
			}
		case folder2:
			assert.Equal(t.T(), folder2, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_Directory, entries[i].Type)
			if !t.in.IsTypeCacheDeprecated() {
				assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), folder2))
			}
		case implicitDir:
			assert.Equal(t.T(), implicitDir, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_Directory, entries[i].Type)
			if !t.in.IsTypeCacheDeprecated() {
				assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), implicitDir))
			}
		case file1:
			assert.Equal(t.T(), file1, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_File, entries[i].Type)
			if !t.in.IsTypeCacheDeprecated() {
				assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), file1))
			}
		case file2:
			assert.Equal(t.T(), file2, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_File, entries[i].Type)
			if !t.in.IsTypeCacheDeprecated() {
				assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), file2))
			}
		case file3:
			assert.Equal(t.T(), file3, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_File, entries[i].Type)
			if !t.in.IsTypeCacheDeprecated() {
				assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), file3))
			}
		}
	}
}

func (t *NonHNSDirTest) TestDeleteChildDir_TypeCacheDeprecated() {
	testCases := []struct {
		name                string
		isImplicitDir       bool
		onlyDeleteFromCache bool
	}{
		{
			name:                "ImplicitDir",
			isImplicitDir:       true,
			onlyDeleteFromCache: true,
		},
		{
			name:                "ExplicitDir",
			isImplicitDir:       false,
			onlyDeleteFromCache: false,
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(st *testing.T) {
			// Enable type cache deprecation
			config := &cfg.Config{
				EnableTypeCacheDeprecation: true,
			}
			dirInode := NewDirInode(
				dirInodeID,
				NewDirName(NewRootName(""), dirInodeName),
				fuseops.InodeAttributes{
					Uid:  uid,
					Gid:  gid,
					Mode: dirMode,
				},
				true,  // implicitDirs
				false, // enableNonexistentTypeCache
				typeCacheTTL,
				&t.bucket,
				&t.fixedTime,
				&t.fixedTime,
				semaphore.NewWeighted(10),
				config,
			)
			dirName := path.Join(dirInodeName, tc.name) + "/"
			// Expectation: DeleteObject called with OnlyDeleteFromCache
			expectedReq := &gcs.DeleteObjectRequest{
				Name:                dirName,
				Generation:          0,
				OnlyDeleteFromCache: tc.onlyDeleteFromCache,
			}
			t.mockBucket.On("DeleteObject", t.ctx, expectedReq).Return(nil)

			err := dirInode.DeleteChildDir(t.ctx, tc.name, tc.isImplicitDir, nil)

			assert.NoError(st, err)
			t.mockBucket.AssertExpectations(st)
		})
	}
}

func (t *HNSDirTest) TestLookUpChild_TypeCacheDeprecated_File() {
	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: true},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: 4},
		EnableHns:                    true,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   true,
	}
	t.in.Unlock()
	t.in = NewDirInode(
		dirInodeID,
		NewDirName(NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		false, // implicitDirs
		false, // enableNonexistentTypeCache
		typeCacheTTL,
		&t.bucket,
		&t.fixedTime,
		&t.fixedTime,
		semaphore.NewWeighted(10),
		config,
	)
	t.in.Lock()
	const name = "file"
	objName := path.Join(dirInodeName, name)
	minObject := &gcs.MinObject{
		Name:           objName,
		MetaGeneration: int64(1),
		Generation:     int64(2),
	}
	attrs := &gcs.ExtendedObjectAttributes{}
	// Mock StatObject for file lookup
	t.mockBucket.On("StatObject", mock.Anything, mock.Anything).Return(minObject, attrs, nil)
	// Mock GetFolder for dir lookup (should return not found or nil)
	notFoundErr := &gcs.NotFoundError{Err: errors.New("not found")}
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	entry, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), entry)
	assert.Equal(t.T(), objName, entry.FullName.GcsObjectName())
	assert.Equal(t.T(), metadata.RegularFileType, entry.Type())
}

func (t *HNSDirTest) TestLookUpChild_TypeCacheDeprecated_Folder() {
	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: true},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: 4},
		EnableHns:                    true,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   true,
	}
	t.in.Unlock()
	t.in = NewDirInode(
		dirInodeID,
		NewDirName(NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		false, // implicitDirs
		false, // enableNonexistentTypeCache
		typeCacheTTL,
		&t.bucket,
		&t.fixedTime,
		&t.fixedTime,
		semaphore.NewWeighted(10),
		config,
	)
	t.in.Lock()
	const name = "folder"
	folderName := path.Join(dirInodeName, name) + "/"
	folder := &gcs.Folder{Name: folderName}
	// Mock GetFolder for dir lookup
	t.mockBucket.On("GetFolder", mock.Anything, mock.Anything).Return(folder, nil)
	// Mock StatObject for file lookup (should return not found)
	notFoundErr := &gcs.NotFoundError{Err: errors.New("not found")}
	t.mockBucket.On("StatObject", mock.Anything, mock.Anything).Return(nil, nil, notFoundErr)

	entry, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), entry)
	assert.Equal(t.T(), folderName, entry.FullName.GcsObjectName())
	assert.Equal(t.T(), metadata.ExplicitDirType, entry.Type())
}

func (t *HNSDirTest) TestLookUpChild_TypeCacheDeprecated_CacheMiss() {
	config := &cfg.Config{
		List: cfg.ListConfig{EnableEmptyManagedFolders: true},
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs:            60,
			TypeCacheMaxSizeMb: 4,
		},
		EnableHns:                    true,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   true,
	}
	t.in.Unlock()
	t.in = NewDirInode(
		dirInodeID,
		NewDirName(NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		false, // implicitDirs
		false, // enableNonexistentTypeCache
		typeCacheTTL,
		&t.bucket,
		&t.fixedTime,
		&t.fixedTime,
		semaphore.NewWeighted(10),
		config,
	)
	t.in.Lock()

	const name = "file"
	objName := path.Join(dirInodeName, name)
	dirObjName := objName + "/"

	cacheMissErr := &caching.CacheMissError{}

	// Expect cache lookup for file -> CacheMiss
	t.mockBucket.On("StatObject", mock.Anything, mock.MatchedBy(func(req *gcs.StatObjectRequest) bool {
		return req.Name == objName && req.FetchOnlyFromCache == true
	})).Return(nil, nil, cacheMissErr).Once()

	// Expect cache lookup for dir -> CacheMiss
	t.mockBucket.On("GetFolder", mock.Anything, mock.MatchedBy(func(req *gcs.GetFolderRequest) bool {
		return req.Name == dirObjName && req.FetchOnlyFromCache == true
	})).Return(nil, cacheMissErr).Once()

	// Expect actual lookup for file -> Success
	minObject := &gcs.MinObject{
		Name:           objName,
		Generation:     1,
		MetaGeneration: 1,
		Size:           100,
	}
	t.mockBucket.On("StatObject", mock.Anything, mock.MatchedBy(func(req *gcs.StatObjectRequest) bool {
		return req.Name == objName && req.FetchOnlyFromCache == false
	})).Return(minObject, &gcs.ExtendedObjectAttributes{}, nil).Once()

	// Expect actual lookup for dir -> NotFound
	t.mockBucket.On("GetFolder", mock.Anything, mock.MatchedBy(func(req *gcs.GetFolderRequest) bool {
		return req.Name == dirObjName && req.FetchOnlyFromCache == false
	})).Return(nil, &gcs.NotFoundError{}).Once()

	entry, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), entry)
	assert.Equal(t.T(), objName, entry.FullName.GcsObjectName())
	t.mockBucket.AssertExpectations(t.T())
}

func (t *HNSDirTest) TestLookUpChild_TypeCacheDeprecated_CacheHit() {
	config := &cfg.Config{
		List: cfg.ListConfig{EnableEmptyManagedFolders: true},
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs:            60,
			TypeCacheMaxSizeMb: 4,
		},
		EnableHns:                    true,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   true,
	}
	t.in.Unlock()
	t.in = NewDirInode(
		dirInodeID,
		NewDirName(NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		false, // implicitDirs
		false, // enableNonexistentTypeCache
		typeCacheTTL,
		&t.bucket,
		&t.fixedTime,
		&t.fixedTime,
		semaphore.NewWeighted(10),
		config,
	)
	t.in.Lock()
	const name = "file"
	objName := path.Join(dirInodeName, name)
	dirObjName := objName + "/"

	// Expect cache lookup for file -> Success
	minObject := &gcs.MinObject{
		Name:           objName,
		Generation:     1,
		MetaGeneration: 1,
		Size:           100,
	}
	t.mockBucket.On("StatObject", mock.Anything, mock.MatchedBy(func(req *gcs.StatObjectRequest) bool {
		return req.Name == objName && req.FetchOnlyFromCache == true
	})).Return(minObject, &gcs.ExtendedObjectAttributes{}, nil).Once()

	// Expect cache lookup for dir -> NotFound (nil, nil)
	t.mockBucket.On("GetFolder", mock.Anything, mock.MatchedBy(func(req *gcs.GetFolderRequest) bool {
		return req.Name == dirObjName && req.FetchOnlyFromCache == true
	})).Return(nil, &gcs.NotFoundError{}).Once()

	entry, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), entry)
	assert.Equal(t.T(), objName, entry.FullName.GcsObjectName())
	t.mockBucket.AssertExpectations(t.T())
}
