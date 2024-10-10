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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
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
	typeCache  metadata.TypeCache
	fixedTime  timeutil.SimulatedClock
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
	t.resetDirInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing, 4, typeCacheTTL)
}

func (t *HNSDirTest) resetDirInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool, typeCacheMaxSizeMB int64, typeCacheTTL time.Duration) {
	t.fixedTime.SetTime(time.Date(2024, 7, 22, 2, 15, 0, 0, time.Local))

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
		&t.fixedTime,
		&t.fixedTime,
		typeCacheMaxSizeMB,
		true,
	)

	d := t.in.(*dirInode)
	assert.NotNil(t.T(), d)
	t.typeCache = d.cache
	assert.NotNil(t.T(), t.typeCache)

	//Lock dir Inode
	t.in.Lock()
}

func (t *HNSDirTest) createDirInode(dirInodeName string) DirInode {
	return NewDirInode(
		5,
		NewDirName(NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		false,
		false,
		true,
		typeCacheTTL,
		&t.bucket,
		&t.fixedTime,
		&t.fixedTime,
		4,
		false,
	)
}

func (t *HNSDirTest) TearDownTest() {
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
	result, err := findExplicitFolder(t.ctx, &t.bucket, NewDirName(t.in.Name(), name))

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
	result, err := findExplicitFolder(t.ctx, &t.bucket, NewDirName(t.in.Name(), "not-present"))

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
	object := gcs.MinObject{Name: dirName}
	t.mockBucket.On("GetFolder", mock.Anything, dirName).Return(folder, nil)
	t.mockBucket.On("StatObject", mock.Anything, &statObjectRequest).Return(&object, &gcs.ExtendedObjectAttributes{}, nil)
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)

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
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
	t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.ExplicitDirType)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), dirName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirName, result.Folder.Name)
	assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
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
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
	assert.Equal(t.T(), metadata.UnknownType, t.typeCache.Get(t.fixedTime.Now(), name))
	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), dirName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirName, result.Folder.Name)
	assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
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
	t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.RegularFileType)
	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), fileName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), fileName, result.MinObject.Name)
	assert.Equal(t.T(), int64(2), result.MinObject.Generation)
	assert.Equal(t.T(), int64(1), result.MinObject.MetaGeneration)
	assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.fixedTime.Now(), name))
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
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
	t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.SymlinkType)
	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), fileName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), fileName, result.MinObject.Name)
	assert.Equal(t.T(), int64(2), result.MinObject.Generation)
	assert.Equal(t.T(), int64(1), result.MinObject.MetaGeneration)
	assert.Equal(t.T(), metadata.SymlinkType, t.typeCache.Get(t.fixedTime.Now(), name))
}

func (t *HNSDirTest) TestLookUpChildShouldCheckForHNSDirectoryWhenTypeIsNonExistentType() {
	const name = "file_type"
	t.typeCache.Insert(t.fixedTime.Now().Add(time.Minute), name, metadata.NonexistentType)
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

func (t *HNSDirTest) TestDeleteChildDir_WhenImplicitDirFlagTrueOnNonHNSBucket() {
	const folderName = "folder"
	dirName := path.Join(dirInodeName, folderName) + "/"
	dirIn := t.createDirInode(dirName)

	// Delete dir
	err := t.in.DeleteChildDir(t.ctx, folderName, true, dirIn)

	t.mockBucket.AssertExpectations(t.T()) // Verify mock interactions
	assert.NoError(t.T(), err)             // Ensure no error occurred
}

func (t *HNSDirTest) TestDeleteChildDir_WhenImplicitDirFlagFalseAndNonHNSBucket_DeleteObjectGiveSuccess() {
	const name = "dir"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("BucketType").Return(gcs.NonHierarchical)
	t.mockBucket.On("DeleteObject", t.ctx, &deleteObjectReq).Return(nil)
	dirIn := t.createDirInode(dirName)

	err := t.in.DeleteChildDir(t.ctx, name, false, dirIn)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
	assert.False(t.T(), dirIn.IsUnlinked())
}
func (t *HNSDirTest) TestDeleteChildDir_WithImplicitDirFlagFalseAndNonHNSBucket_DeleteObjectThrowAnError() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("BucketType").Return(gcs.NonHierarchical)
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
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
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
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
	t.mockBucket.On("DeleteObject", t.ctx, &deleteObjectReq).Return(fmt.Errorf("mock error"))
	t.mockBucket.On("DeleteFolder", t.ctx, dirName).Return(nil)
	dirIn := t.createDirInode(dirName)

	// Delete dir .
	err := t.in.DeleteChildDir(t.ctx, name, false, dirIn)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
	assert.True(t.T(), dirIn.IsUnlinked())
}

func (t *HNSDirTest) TestDeleteChildDir_WithImplicitDirFlagFalseAndBucketTypeIsHNS_DeleteObjectAndDeleteFolderThrowAnError() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	deleteObjectReq := gcs.DeleteObjectRequest{
		Name:       dirName,
		Generation: 0,
	}
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
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
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
	t.mockBucket.On("CreateFolder", t.ctx, dirName).Return(nil, fmt.Errorf("mock error"))

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NotNil(t.T(), err)
	assert.Nil(t.T(), result)
	assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
}

func (t *HNSDirTest) TestCreateChildDirWhenBucketTypeIsHNSWithSuccess() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	folder := gcs.Folder{Name: dirName}
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
	t.mockBucket.On("CreateFolder", t.ctx, dirName).Return(&folder, nil)

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), result)
	assert.Equal(t.T(), dirName, result.Folder.Name)
	assert.Equal(t.T(), dirName, result.FullName.objectName)
	assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
}

func (t *HNSDirTest) TestCreateChildDirWhenBucketTypeIsNonHNSWithFailure() {
	const name = "folder"
	var preCond int64
	dirName := path.Join(dirInodeName, name) + "/"
	createObjectReq := gcs.CreateObjectRequest{Name: dirName, Contents: strings.NewReader(""), GenerationPrecondition: &preCond}
	t.mockBucket.On("BucketType").Return(gcs.NonHierarchical)
	t.mockBucket.On("CreateObject", t.ctx, &createObjectReq).Return(nil, fmt.Errorf("mock error"))

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NotNil(t.T(), err)
	assert.Nil(t.T(), result)
	assert.Equal(t.T(), metadata.Type(0), t.typeCache.Get(t.fixedTime.Now(), dirName))
}

func (t *HNSDirTest) TestCreateChildDirWhenBucketTypeIsNonHNSWithSuccess() {
	const name = "folder"
	dirName := path.Join(dirInodeName, name) + "/"
	var preCond int64
	createObjectReq := gcs.CreateObjectRequest{Name: dirName, Contents: strings.NewReader(""), GenerationPrecondition: &preCond}
	object := gcs.Object{Name: dirName}
	t.mockBucket.On("BucketType").Return(gcs.NonHierarchical)
	t.mockBucket.On("CreateObject", t.ctx, &createObjectReq).Return(&object, nil)

	result, err := t.in.CreateChildDir(t.ctx, name)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), result)
	assert.Equal(t.T(), dirName, result.MinObject.Name)
	assert.Equal(t.T(), dirName, result.FullName.objectName)
	assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.fixedTime.Now(), name))
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
	obj1 := gcs.Object{Name: path.Join(dirInodeName, folder1) + "/"}
	obj2 := gcs.Object{Name: path.Join(dirInodeName, folder2) + "/"}
	obj3 := gcs.Object{Name: path.Join(dirInodeName, folder2, file1)}
	obj4 := gcs.Object{Name: path.Join(dirInodeName, file2)}
	obj5 := gcs.Object{Name: path.Join(dirInodeName, implicitDir, file3)}
	objects := []*gcs.Object{&obj1, &obj2, &obj3, &obj4, &obj5}
	collapsedRuns := []string{path.Join(dirInodeName, folder1) + "/", path.Join(dirInodeName, folder2) + "/", path.Join(dirInodeName, implicitDir) + "/"}
	listing := gcs.Listing{
		Objects:       objects,
		CollapsedRuns: collapsedRuns,
	}
	listObjectReq := gcs.ListObjectsRequest{
		Prefix:                   dirInodeName,
		Delimiter:                "/",
		IncludeFoldersAsPrefixes: true,
		IncludeTrailingDelimiter: true,
		MaxResults:               5000,
		ProjectionVal:            gcs.NoAcl,
	}
	t.mockBucket.On("BucketType").Return(gcs.Hierarchical)
	t.mockBucket.On("ListObjects", t.ctx, &listObjectReq).Return(&listing, nil)

	entries, _, err := t.in.ReadEntries(t.ctx, tok)

	t.mockBucket.AssertExpectations(t.T())
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 6, len(entries))
	for i := 0; i < 6; i++ {
		switch entries[i].Name {
		case folder1:
			assert.Equal(t.T(), folder1, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_Directory, entries[i].Type)
			assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), folder1))
		case folder2:
			assert.Equal(t.T(), folder2, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_Directory, entries[i].Type)
			assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), folder2))
		case implicitDir:
			assert.Equal(t.T(), implicitDir, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_Directory, entries[i].Type)
			assert.Equal(t.T(), metadata.ExplicitDirType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), implicitDir))
		case file1:
			assert.Equal(t.T(), file1, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_File, entries[i].Type)
			assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), file1))
		case file2:
			assert.Equal(t.T(), file2, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_File, entries[i].Type)
			assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), file2))
		case file3:
			assert.Equal(t.T(), file3, entries[i].Name)
			assert.Equal(t.T(), fuseutil.DT_File, entries[i].Type)
			assert.Equal(t.T(), metadata.RegularFileType, t.typeCache.Get(t.in.(*dirInode).cacheClock.Now(), file3))
		}
	}
}
