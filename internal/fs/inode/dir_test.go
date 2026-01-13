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
	"errors"
	"maps"
	"math"
	"os"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
	"golang.org/x/sync/semaphore"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const dirInodeID = 17
const dirInodeName = "foo/bar/"
const dirMode os.FileMode = 0712 | os.ModeDir
const typeCacheTTL = time.Second
const testSymlinkTarget = "blah"
const isTypeCacheDeprecationEnabled = false

type DirTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock

	in DirInode
	tc metadata.TypeCache
	suite.Suite
}

func TestDirTest(t *testing.T) {
	suite.Run(t, &DirTest{})
}

func (t *DirTest) SetupTest() {
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{})
	t.bucket = gcsx.NewSyncerBucket(
		1, // Append threshold
		ChunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		bucket)
	// Create the inode. No implicit dirs by default.
	t.resetInode(false, false)
}

func (t *DirTest) TearDownTestSuite() {
	t.in.Unlock()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type DirentSlice []fuseutil.Dirent

func (p DirentSlice) Len() int           { return len(p) }
func (p DirentSlice) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p DirentSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// NOTE: A limitation in the fake bucket's API prevents the direct creation of managed folders.
// This poses a challenge for writing unit tests for includeFoldersAsPrefixes.

func (t *DirTest) resetInode(implicitDirs, enableNonexistentTypeCache bool) {
	t.resetInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, true, 4, typeCacheTTL)
}

func (t *DirTest) resetInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool, typeCacheMaxSizeMB int64, typeCacheTTL time.Duration) {
	if t.in != nil {
		t.in.Unlock()
	}

	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: enableManagedFoldersListing},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: typeCacheMaxSizeMB},
		EnableHns:                    false,
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
		&t.clock,
		&t.clock,
		config,
	)

	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	t.tc = d.cache
	if !d.IsTypeCacheDeprecated() {
		require.NotNil(t.T(), t.tc)
	} else {
		require.Nil(t.T(), t.tc)
	}

	t.in.Lock()
}

func (t *DirTest) createDirInode(dirInodeName string) DirInode {
	return t.createDirInodeWithTypeCacheDeprecationFlag(dirInodeName, false)
}

func (t *DirTest) createDirInodeWithTypeCacheDeprecationFlag(dirInodeName string, isTypeCacheDeprecated bool) DirInode {
	config := &cfg.Config{
		List:                         cfg.ListConfig{EnableEmptyManagedFolders: false},
		MetadataCache:                cfg.MetadataCacheConfig{TypeCacheMaxSizeMb: 4},
		EnableHns:                    false,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   isTypeCacheDeprecated,
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
		&t.clock,
		&t.clock,
		config,
	)
}

func (t *DirTest) getTypeFromCache(name string) metadata.Type {
	return t.tc.Get(t.in.(*dirInode).cacheClock.Now(), name)
}

// Read all of the entries and sort them by name.
func (t *DirTest) readAllEntries() (entries []fuseutil.Dirent, err error) {
	tok := ""
	for {
		var tmp []fuseutil.Dirent
		tmp, _, tok, err = t.in.ReadEntries(t.ctx, tok)
		entries = append(entries, tmp...)
		if err != nil {
			return
		}

		if tok == "" {
			break
		}
	}

	sort.Sort(DirentSlice(entries))
	return
}

// Read all of the entry cores
func (t *DirTest) readAllEntryCores() (cores map[Name]*Core, unsupportedPaths []string, err error) {
	cores = make(map[Name]*Core)
	tok := ""
	for {
		var fetchedCores map[Name]*Core
		var fetchedUnsupportedPaths []string
		fetchedCores, fetchedUnsupportedPaths, tok, err = t.in.ReadEntryCores(t.ctx, tok)
		if err != nil {
			return nil, nil, err
		}
		maps.Copy(cores, fetchedCores)
		unsupportedPaths = append(unsupportedPaths, fetchedUnsupportedPaths...)

		if tok == "" {
			break
		}
	}

	return
}

func (t *DirTest) setSymlinkTarget(
	objName string) (err error) {
	target := testSymlinkTarget
	_, err = t.bucket.UpdateObject(
		t.ctx,
		&gcs.UpdateObjectRequest{
			Name: objName,
			Metadata: map[string]*string{
				SymlinkMetadataKey: &target,
			},
		})

	return
}

func (t *DirTest) createLocalFileInode(parent Name, name string, id fuseops.InodeID) (in Inode) {
	in = NewFileInode(
		id,
		NewFileName(parent, name),
		nil,
		fuseops.InodeAttributes{
			Uid:  123,
			Gid:  456,
			Mode: 0712,
		},
		&t.bucket,
		false, // localFileCache
		contentcache.New("", &t.clock),
		&t.clock,
		true, //localFile
		&cfg.Config{},
		semaphore.NewWeighted(math.MaxInt64),
		nil) // mrdCache
	return
}

func (t *DirTest) getLocalDirentKey(in Inode) string {
	return path.Base(in.Name().LocalName())
}

func (t *DirTest) validateCore(cores map[Name]*Core, entryName string, isDir bool, expectedType metadata.Type, expectedFullName string) {
	var name Name
	if isDir {
		name = NewDirName(t.in.Name(), entryName)
	} else {
		name = NewFileName(t.in.Name(), entryName)
	}

	core, ok := cores[name]
	require.True(t.T(), ok, "entry for "+entryName+" not found")
	assert.Equal(t.T(), expectedFullName, core.FullName.GcsObjectName())
	assert.Equal(t.T(), expectedType, core.Type())
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), expectedType, t.getTypeFromCache(entryName))
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirTest) TestID() {
	assert.EqualValues(t.T(), dirInodeID, t.in.ID())
}

func (t *DirTest) TestName() {
	assert.Equal(t.T(), dirInodeName, t.in.Name().GcsObjectName())
}

func (t *DirTest) TestLookupCount() {
	// Increment thrice. The count should now be three.
	t.in.IncrementLookupCount()
	t.in.IncrementLookupCount()
	t.in.IncrementLookupCount()

	// Decrementing twice shouldn't cause destruction. But one more should.
	require.False(t.T(), t.in.DecrementLookupCount(2))
	assert.True(t.T(), t.in.DecrementLookupCount(1))
}

func (t *DirTest) TestAttributes_WithClobberedCheckTrue() {
	attrs, err := t.in.Attributes(t.ctx, true)

	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), uid, attrs.Uid)
	assert.EqualValues(t.T(), gid, attrs.Gid)
	assert.Equal(t.T(), dirMode|os.ModeDir, attrs.Mode)
}

func (t *DirTest) TestAttributes_WithClobberedCheckFalse() {
	attrs, err := t.in.Attributes(t.ctx, false)

	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), uid, attrs.Uid)
	assert.EqualValues(t.T(), gid, attrs.Gid)
	assert.Equal(t.T(), dirMode|os.ModeDir, attrs.Mode)
}

func (t *DirTest) TestLookUpChild_NonExistent() {
	const name = "qux"

	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
	}
}

func (t *DirTest) TestLookUpChild_FileOnly() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	createObj, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), objName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), objName, result.MinObject.Name)
	assert.Equal(t.T(), createObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), createObj.Size, result.MinObject.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	require.NoError(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}
}

func (t *DirTest) TestLookUpChild_DirOnly() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object.
	createObj, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	require.NoError(t.T(), err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), objName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), objName, result.MinObject.Name)
	assert.Equal(t.T(), createObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), createObj.Size, result.MinObject.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	require.NoError(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}
}

func (t *DirTest) TestLookUpChild_ImplicitDirOnly_Disabled() {
	const name = "qux"
	var err error

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	require.NoError(t.T(), err)

	// Looking up the name shouldn't work.
	result, err := t.in.LookUpChild(t.ctx, name)
	require.NoError(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
	}

	// Ditto with a conflict marker.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	require.NoError(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}
}

func (t *DirTest) TestLookUpChild_ImplicitDirOnly_Enabled() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Enable implicit dirs.
	t.resetInode(true, false)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	require.NoError(t.T(), err)

	// Looking up the name should work.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	assert.Nil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ImplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), objName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), metadata.ImplicitDirType, result.Type())

	// A conflict marker should not work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	require.NoError(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}
}

func (t *DirTest) TestLookUpChild_FileAndDir() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create backing objects.
	fileObj, err := storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	require.NoError(t.T(), err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	require.NoError(t.T(), err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), dirObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
	assert.Equal(t.T(), dirObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		require.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}

	assert.Equal(t.T(), fileObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), fileObjName, result.MinObject.Name)
	assert.Equal(t.T(), fileObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), fileObj.Size, result.MinObject.Size)
}

func (t *DirTest) TestLookUpChild_SymlinkAndDir() {
	const name = "qux"
	linkObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create backing objects.
	linkObj, err := storageutil.CreateObject(t.ctx, t.bucket, linkObjName, []byte("taco"))
	require.NoError(t.T(), err)

	err = t.setSymlinkTarget(linkObjName)
	require.NoError(t.T(), err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	require.NoError(t.T(), err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)

	// The following check should have been for metadata.SymlinkType,
	// because of the t.setSymlinkTarget call above, but it is not.
	// This is so because the above symlink is
	// created as a regular directory object on GCS,
	// and is read back the same to gcsfuse and is this stored in type-cache
	// also as a directory.
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), dirObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
	assert.Equal(t.T(), dirObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}

	assert.Equal(t.T(), linkObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), linkObjName, result.MinObject.Name)
	assert.Equal(t.T(), linkObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), linkObj.Size, result.MinObject.Size)
}

func (t *DirTest) TestLookUpChild_FileAndDirAndImplicitDir_Disabled() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create backing objects.
	fileObj, err := storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	require.NoError(t.T(), err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	require.NoError(t.T(), err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	require.NoError(t.T(), err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(path.Join(dirInodeName, name)))
	}

	assert.Equal(t.T(), dirObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
	assert.Equal(t.T(), dirObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)

	assert.Equal(t.T(), fileObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), fileObjName, result.MinObject.Name)
	assert.Equal(t.T(), fileObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), fileObj.Size, result.MinObject.Size)
}

func (t *DirTest) TestLookUpChild_FileAndDirAndImplicitDir_Enabled() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Enable implicit dirs.
	t.resetInode(true, false)

	// Create backing objects.
	fileObj, err := storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	require.NoError(t.T(), err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	require.NoError(t.T(), err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	require.NoError(t.T(), err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(path.Join(dirInodeName, name)))
	}

	assert.Equal(t.T(), dirObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
	assert.Equal(t.T(), dirObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}

	assert.Equal(t.T(), fileObjName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), fileObjName, result.MinObject.Name)
	assert.Equal(t.T(), fileObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), fileObj.Size, result.MinObject.Size)
}

func (t *DirTest) TestLookUpChild_TypeCaching() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object for a file.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	require.NoError(t.T(), err)

	// Look up; we should get the file.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), fileObjName, result.MinObject.Name)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	require.NoError(t.T(), err)

	// Look up again. Even though the directory should shadow the file, because
	// we've cached only seeing the file that's what we should get back.
	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), fileObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
}

func (t *DirTest) TestLookUpChild_NonExistentTypeCache_ImplicitDirsDisabled() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	// Enable enableNonexistentTypeCache for type cache
	t.resetInode(false, true)

	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Look up nonexistent object, return nil
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.Nil(t.T(), result)

	// Create a backing object.
	createObj, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	require.NoError(t.T(), err)

	// Look up again, should still return nil due to cache
	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.NonexistentType, t.getTypeFromCache(name))
	}

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	// Look up again, should return correct object
	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), objName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), objName, result.MinObject.Name)
	assert.Equal(t.T(), createObj.Generation, result.MinObject.Generation)
	assert.Equal(t.T(), createObj.Size, result.MinObject.Size)
}

func (t *DirTest) TestLookUpChild_NonExistentTypeCache_ImplicitDirsEnabled() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	// Enable implicitDirs and enableNonexistentTypeCache for type cache
	t.resetInode(true, true)

	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Look up nonexistent object, return nil
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.NonexistentType, t.getTypeFromCache(name))
	}

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	require.NoError(t.T(), err)

	// Look up again, should still return nil due to cache
	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.NonexistentType, t.getTypeFromCache(name))
	}

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	// Look up again, should return correct object
	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	assert.Nil(t.T(), result.MinObject)

	assert.Equal(t.T(), objName, result.FullName.GcsObjectName())
	assert.Equal(t.T(), metadata.ImplicitDirType, result.Type())

	// A conflict marker should not work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	require.NoError(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
	}
}

func (t *DirTest) TestLookUpChild_TypeCacheEnabled() {
	inputs := []struct {
		typeCacheMaxSizeMB int64
		typeCacheTTL       time.Duration
	}{{
		typeCacheMaxSizeMB: 4,
		typeCacheTTL:       time.Second,
	}, {
		typeCacheMaxSizeMB: -1,
		typeCacheTTL:       time.Second,
	}}

	for _, input := range inputs {
		t.resetInodeWithTypeCacheConfigs(true, true, true, input.typeCacheMaxSizeMB, input.typeCacheTTL)

		const name = "qux"
		objName := path.Join(dirInodeName, name)

		// Create a backing object.
		o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))

		require.NoError(t.T(), err)
		require.NotNil(t.T(), o)

		// Look up nonexistent object, return nil
		result, err := t.in.LookUpChild(t.ctx, name)

		require.NoError(t.T(), err)
		require.NotNil(t.T(), result)
		if !t.in.IsTypeCacheDeprecated() {
			assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
		}
	}
}

func (t *DirTest) TestLookUpChild_TypeCacheDisabled() {
	inputs := []struct {
		typeCacheMaxSizeMB int64
		typeCacheTTL       time.Duration
	}{
		{
			typeCacheMaxSizeMB: 0,
			typeCacheTTL:       time.Second,
		}, {
			typeCacheMaxSizeMB: 4,
			typeCacheTTL:       0,
		},
	}

	for _, input := range inputs {
		t.resetInodeWithTypeCacheConfigs(true, true, true, input.typeCacheMaxSizeMB, input.typeCacheTTL)

		const name = "qux"
		objName := path.Join(dirInodeName, name)

		// Create a backing object.
		o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))

		require.NoError(t.T(), err)
		require.NotNil(t.T(), o)

		// Look up nonexistent object, return nil
		result, err := t.in.LookUpChild(t.ctx, name)

		require.NoError(t.T(), err)
		require.NotNil(t.T(), result)
		if !t.in.IsTypeCacheDeprecated() {
			assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
		}
	}
}

func (t *DirTest) TestReadDescendants_Empty() {
	descendants, err := t.in.ReadDescendants(t.ctx, 10)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(descendants))

}

func (t *DirTest) TestReadDescendants_NonEmpty() {
	var err error

	// Set up contents.
	objs := []string{
		dirInodeName + "backed_dir_empty/",
		dirInodeName + "backed_dir_nonempty/",
		dirInodeName + "backed_dir_nonempty/blah",
		dirInodeName + "file",
		dirInodeName + "implicit_dir/blah",
		dirInodeName + "symlink",
	}

	err = storageutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
	require.NoError(t.T(), err)

	descendants, err := t.in.ReadDescendants(t.ctx, 10)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), 6, len(descendants))

	descendants, err = t.in.ReadDescendants(t.ctx, 2)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), 2, len(descendants))
}

func (t *DirTest) TestReadEntries_Empty() {
	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	require.True(t.T(), d.prevDirListingTimeStamp.IsZero())
	entries, err := t.readAllEntries()

	require.NoError(t.T(), err)
	assert.ElementsMatch(t.T(), []fuseutil.Dirent{}, entries)
	// Make sure prevDirListingTimeStamp is initialized.
	require.False(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) TestReadEntries_NonEmpty_ImplicitDirsDisabled() {
	var err error
	var entry fuseutil.Dirent

	// Set up contents.
	objs := []string{
		dirInodeName + "backed_dir_empty/",
		dirInodeName + "backed_dir_nonempty/",
		dirInodeName + "backed_dir_nonempty/blah",
		dirInodeName + "file",
		dirInodeName + "implicit_dir/blah",
		dirInodeName + "symlink",
	}

	err = storageutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
	require.NoError(t.T(), err)

	// Set up the symlink target.
	err = t.setSymlinkTarget(dirInodeName + "symlink")
	require.NoError(t.T(), err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	require.True(t.T(), d.prevDirListingTimeStamp.IsZero())

	// Read entries.
	entries, err := t.readAllEntries()

	require.NoError(t.T(), err)
	require.Equal(t.T(), 4, len(entries))

	entry = entries[0]
	assert.Equal(t.T(), "backed_dir_empty", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_Directory, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_empty"))
	}

	entry = entries[1]
	assert.Equal(t.T(), "backed_dir_nonempty", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_Directory, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_nonempty"))
	}

	entry = entries[2]
	assert.Equal(t.T(), "file", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_File, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache("file"))
	}

	entry = entries[3]
	assert.Equal(t.T(), "symlink", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_Link, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.SymlinkType, t.getTypeFromCache("symlink"))
	}

	// Make sure prevDirListingTimeStamp is initialized.
	require.False(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) TestReadEntries_NonEmpty_ImplicitDirsEnabled() {
	var err error
	var entry fuseutil.Dirent

	// Enable implicit dirs.
	t.resetInode(true, false)

	// Set up contents.
	objs := []string{
		dirInodeName + "backed_dir_empty/",
		dirInodeName + "backed_dir_nonempty/",
		dirInodeName + "backed_dir_nonempty/blah",
		dirInodeName + "file",
		dirInodeName + "implicit_dir/blah",
		dirInodeName + "symlink",
	}

	err = storageutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
	require.NoError(t.T(), err)

	// Set up the symlink target.
	err = t.setSymlinkTarget(dirInodeName + "symlink")
	require.NoError(t.T(), err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	require.True(t.T(), d.prevDirListingTimeStamp.IsZero())

	// Read entries.
	entries, err := t.readAllEntries()

	require.NoError(t.T(), err)
	require.Equal(t.T(), 5, len(entries))

	entry = entries[0]
	assert.Equal(t.T(), "backed_dir_empty", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_Directory, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_empty"))
	}

	entry = entries[1]
	assert.Equal(t.T(), "backed_dir_nonempty", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_Directory, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_nonempty"))
	}

	entry = entries[2]
	assert.Equal(t.T(), "file", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_File, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache("file"))
	}

	entry = entries[3]
	assert.Equal(t.T(), "implicit_dir", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_Directory, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ImplicitDirType, t.getTypeFromCache("implicit_dir"))
	}

	entry = entries[4]
	assert.Equal(t.T(), "symlink", entry.Name)
	assert.Equal(t.T(), fuseutil.DT_Link, entry.Type)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.SymlinkType, t.getTypeFromCache("symlink"))
	}

	// Make sure prevDirListingTimeStamp is initialized.
	require.False(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) TestReadEntries_TypeCaching() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object for a file.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	require.NoError(t.T(), err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	require.True(t.T(), d.prevDirListingTimeStamp.IsZero())

	// Read the directory, priming the type cache.
	_, err = t.readAllEntries()
	require.NoError(t.T(), err)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
	}

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	require.NoError(t.T(), err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), fileObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), dirObjName, result.MinObject.Name)

	// Make sure prevDirListingTimeStamp is initialized.
	require.False(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) TestReadEntryCores_Empty() {
	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	require.True(t.T(), d.prevDirListingTimeStamp.IsZero())

	cores, unsupportedPaths, err := t.readAllEntryCores()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(cores))
	assert.Equal(t.T(), 0, len(unsupportedPaths))
	// Make sure prevDirListingTimeStamp is initialized.
	require.False(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) TestReadEntryCores_NonEmpty_ImplicitDirsDisabled() {
	var err error
	var cores map[Name]*Core

	// Set up contents.
	backedDirEmptyName := path.Join(dirInodeName, "backed_dir_empty") + "/"
	backedDirNonEmptyName := path.Join(dirInodeName, "backed_dir_nonempty") + "/"
	backedDirNonEmptyFileName := path.Join(backedDirNonEmptyName, "blah")
	testFileName := path.Join(dirInodeName, "file")
	implicitDirObjName := path.Join(dirInodeName, "implicit_dir") + "/blah"
	symlinkName := path.Join(dirInodeName, "symlink")

	objs := []string{
		backedDirEmptyName,
		backedDirNonEmptyName,
		backedDirNonEmptyFileName,
		testFileName,
		implicitDirObjName,
		symlinkName,
	}

	err = storageutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
	require.NoError(t.T(), err)

	// Set up the symlink target.
	err = t.setSymlinkTarget(dirInodeName + "symlink")
	require.NoError(t.T(), err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	require.True(t.T(), d.prevDirListingTimeStamp.IsZero())

	// Read cores.
	cores, _, _, err = t.in.ReadEntryCores(t.ctx, "")

	require.NoError(t.T(), err)
	require.Equal(t.T(), 4, len(cores))
	t.validateCore(cores, "backed_dir_empty", true, metadata.ExplicitDirType, backedDirEmptyName)
	t.validateCore(cores, "backed_dir_nonempty", true, metadata.ExplicitDirType, backedDirNonEmptyName)
	t.validateCore(cores, "file", false, metadata.RegularFileType, testFileName)
	t.validateCore(cores, "symlink", false, metadata.SymlinkType, symlinkName)
	// Make sure prevDirListingTimeStamp is initialized.
	require.False(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) TestReadEntryCores_NonEmpty_ImplicitDirsEnabled() {
	var err error
	var cores map[Name]*Core
	var unsupportedPaths []string

	// Enable implicit dirs.
	t.resetInode(true, false)

	// Set up contents.
	backedDirEmptyName := path.Join(dirInodeName, "backed_dir_empty") + "/"
	backedDirNonEmptyName := path.Join(dirInodeName, "backed_dir_nonempty") + "/"
	backedDirNonEmptyFileName := path.Join(backedDirNonEmptyName, "blah")
	testFileName := path.Join(dirInodeName, "file")
	implicitDirObjName := path.Join(dirInodeName, "implicit_dir") + "/blah"
	symlinkName := path.Join(dirInodeName, "symlink")
	unsupportedPathName1 := dirInodeName + "//" + "a.txt"
	unsupportedPathName2 := dirInodeName + "../" + "b.txt"

	objs := []string{
		backedDirEmptyName,
		backedDirNonEmptyName,
		backedDirNonEmptyFileName,
		testFileName,
		implicitDirObjName,
		symlinkName,
		unsupportedPathName1,
		unsupportedPathName2,
	}

	err = storageutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
	require.NoError(t.T(), err)

	// Set up the symlink target.
	err = t.setSymlinkTarget(dirInodeName + "symlink")
	require.NoError(t.T(), err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	require.NotNil(t.T(), d)
	require.True(t.T(), d.prevDirListingTimeStamp.IsZero())

	// Read cores.
	cores, unsupportedPaths, err = t.readAllEntryCores()

	require.NoError(t.T(), err)
	require.Equal(t.T(), 5, len(cores))
	require.Equal(t.T(), 2, len(unsupportedPaths))
	t.validateCore(cores, "backed_dir_empty", true, metadata.ExplicitDirType, backedDirEmptyName)
	t.validateCore(cores, "backed_dir_nonempty", true, metadata.ExplicitDirType, backedDirNonEmptyName)
	t.validateCore(cores, "file", false, metadata.RegularFileType, testFileName)
	t.validateCore(cores, "implicit_dir", true, metadata.ImplicitDirType, path.Join(dirInodeName, "implicit_dir")+"/")
	t.validateCore(cores, "symlink", false, metadata.SymlinkType, symlinkName)
	assert.ElementsMatch(t.T(), []string{dirInodeName + "../", dirInodeName + "/"}, unsupportedPaths)
	// Make sure prevDirListingTimeStamp is initialized.
	require.False(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) TestCreateChildFile_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	result, err := t.in.CreateChildFile(t.ctx, name)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), result)
	require.NotNil(t.T(), result.MinObject)

	assert.Equal(t.T(), t.bucket.Name(), result.Bucket.Name())
	assert.Equal(t.T(), result.FullName.GcsObjectName(), result.MinObject.Name)
	assert.Equal(t.T(), objName, result.MinObject.Name)
	assert.False(t.T(), IsSymlink(result.MinObject))
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), 1, len(result.MinObject.Metadata))
	assert.Equal(t.T(),
		t.clock.Now().UTC().Format(time.RFC3339Nano),
		result.MinObject.Metadata["gcsfuse_mtime"])
}

func (t *DirTest) TestCreateChildFile_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create an existing backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	// Call the inode.
	_, err = t.in.CreateChildFile(t.ctx, name)
	assert.ErrorContains(t.T(), err, "Precondition")
	assert.ErrorContains(t.T(), err, "exists")
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
	}
}

func (t *DirTest) TestCreateChildFile_TypeCaching() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create the name.
	_, err = t.in.CreateChildFile(t.ctx, name)
	require.NoError(t.T(), err)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	require.NoError(t.T(), err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), fileObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
}

func (t *DirTest) TestCloneToChildFile_SourceDoesntExist() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	var err error

	// Create and then delete the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte(""))
	require.NoError(t.T(), err)

	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{Name: srcName})

	require.NoError(t.T(), err)

	// Call the inode.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	_, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	var notFoundErr *gcs.NotFoundError
	assert.True(t.T(), errors.As(err, &notFoundErr))
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(dstName))
	}
}

func (t *DirTest) TestCloneToChildFile_DestinationDoesntExist() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte("taco"))
	require.NoError(t.T(), err)

	// Call the inode.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	result, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), result)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache("qux"))
	}

	assert.Equal(t.T(), t.bucket.Name(), result.Bucket.Name())
	assert.Equal(t.T(), result.FullName.GcsObjectName(), result.MinObject.Name)
	assert.Equal(t.T(), dstName, result.MinObject.Name)
	assert.False(t.T(), IsSymlink(result.MinObject))

	// Check resulting contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, dstName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "taco", string(contents))
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache("qux"))
	}
}

func (t *DirTest) TestCloneToChildFile_DestinationExists() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte("taco"))
	require.NoError(t.T(), err)

	// And a destination object that will be overwritten.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dstName, []byte(""))
	require.NoError(t.T(), err)

	// Call the inode.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	result, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), result)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache("qux"))
	}

	assert.Equal(t.T(), t.bucket.Name(), result.Bucket.Name())
	assert.Equal(t.T(), result.FullName.GcsObjectName(), result.MinObject.Name)
	assert.Equal(t.T(), dstName, result.MinObject.Name)
	assert.False(t.T(), IsSymlink(result.MinObject))
	assert.EqualValues(t.T(), len("taco"), result.MinObject.Size)

	// Check resulting contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, dstName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "taco", string(contents))
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache("qux"))
	}
}

func (t *DirTest) TestCloneToChildFile_TypeCaching() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	var err error

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte(""))
	require.NoError(t.T(), err)

	// Clone to the destination.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	_, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	require.NoError(t.T(), err)

	// Create a backing object for a directory.
	dirObjName := dstName + "/"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	require.NoError(t.T(), err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, path.Base(dstName))

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache("qux"))
	}

	assert.Equal(t.T(), dstName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, path.Base(dstName))

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache("qux"))
	}

	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
}

func (t *DirTest) TestCreateChildSymlink_DoesntExist() {
	const name = "qux"
	const target = "taco"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	result, err := t.in.CreateChildSymlink(t.ctx, name, target)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), result)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.SymlinkType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), t.bucket.Name(), result.Bucket.Name())
	assert.Equal(t.T(), result.FullName.GcsObjectName(), result.MinObject.Name)
	assert.Equal(t.T(), objName, result.MinObject.Name)
	assert.Equal(t.T(), target, result.MinObject.Metadata[SymlinkMetadataKey])
}

func (t *DirTest) TestCreateChildSymlink_Exists() {
	const name = "qux"
	const target = "taco"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create an existing backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	require.NoError(t.T(), err)

	// Call the inode.
	_, err = t.in.CreateChildSymlink(t.ctx, name, target)
	assert.ErrorContains(t.T(), err, "Precondition")
	assert.ErrorContains(t.T(), err, "exists")
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
	}
}

func (t *DirTest) TestCreateChildSymlink_TypeCaching() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	const name = "qux"
	linkObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create the name.
	_, err = t.in.CreateChildSymlink(t.ctx, name, "")
	require.NoError(t.T(), err)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	require.NoError(t.T(), err)

	// Look up the name. Even though the directory should shadow the symlink,
	// because we've cached only seeing the symlink that's what we should get
	// back.
	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.SymlinkType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), linkObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
}

func (t *DirTest) TestDeleteChildFile_Succeeds_TypeCacheEvicted() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)
	var err error
	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)
	// Prime the type cache.
	t.in.InsertFileIntoTypeCache(name)
	assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))

	// Call the inode.
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &o.MetaGeneration)

	require.NoError(t.T(), err)
	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	assert.ErrorAs(t.T(), err, &notFoundErr)
	// Check that the type cache has been updated.
	assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
}

func (t *DirTest) TestDeleteChildFile_ReturnsError_TypeCacheRetained() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)
	var err error
	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)
	// Prime the type cache.
	t.in.InsertFileIntoTypeCache(name)
	assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))

	// Call the inode with a meta-generation that will cause a precondition error.
	wrongMetaGeneration := o.MetaGeneration + 1
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &wrongMetaGeneration)

	assert.ErrorContains(t.T(), err, "DeleteObject: gcs.PreconditionError")
	assert.Equal(t.T(), metadata.RegularFileType, t.getTypeFromCache(name))
}

func (t *DirTest) TestCreateChildDir_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Call the inode.
	result, err := t.in.CreateChildDir(t.ctx, name)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), result)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), t.bucket.Name(), result.Bucket.Name())
	assert.Equal(t.T(), result.FullName.GcsObjectName(), result.MinObject.Name)
	assert.Equal(t.T(), objName, result.MinObject.Name)
	assert.False(t.T(), IsSymlink(result.MinObject))
}

func (t *DirTest) TestCreateChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create an existing backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	// Call the inode.
	_, err = t.in.CreateChildDir(t.ctx, name)
	assert.ErrorContains(t.T(), err, "Precondition")
	assert.ErrorContains(t.T(), err, "exists")
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
	}
}

func (t *DirTest) TestDeleteChildFile_DoesntExist() {
	const name = "qux"

	err := t.in.DeleteChildFile(t.ctx, name, 0, nil)
	require.NoError(t.T(), err)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache(name))
	}
}

func (t *DirTest) TestDeleteChildFile_WrongGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	// Call the inode with the wrong generation. No error should be returned.
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation+1, &o.MetaGeneration)
	require.NoError(t.T(), err)

	// The original generation should still be there.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, objName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "taco", string(contents))
}

func (t *DirTest) TestDeleteChildFile_WrongMetaGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	// Call the inode with the wrong meta-generation. No error should be
	// returned.
	precond := o.MetaGeneration + 1
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &precond)

	assert.ErrorContains(t.T(), err, "Precondition")
	assert.ErrorContains(t.T(), err, "meta-generation")

	// The original generation should still be there.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, objName)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "taco", string(contents))
}

func (t *DirTest) TestDeleteChildFile_LatestGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	// Call the inode.
	err = t.in.DeleteChildFile(t.ctx, name, 0, nil)
	require.NoError(t.T(), err)

	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	assert.True(t.T(), errors.As(err, &notFoundErr))
}

func (t *DirTest) TestDeleteChildFile_ParticularGenerationAndMetaGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	// Call the inode.
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &o.MetaGeneration)
	require.NoError(t.T(), err)

	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	assert.True(t.T(), errors.As(err, &notFoundErr))
}

func (t *DirTest) TestDeleteChildFile_TypeCaching() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create the name, priming the type cache.
	_, err = t.in.CreateChildFile(t.ctx, name)
	require.NoError(t.T(), err)

	// Create a backing object for a directory. It should be shadowed by the
	// file.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	require.NoError(t.T(), err)

	result, err := t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	require.Equal(t.T(), fileObjName, result.MinObject.Name)

	// But after deleting the file via the inode, the directory should be
	// revealed.
	err = t.in.DeleteChildFile(t.ctx, name, 0, nil)
	require.NoError(t.T(), err)

	result, err = t.in.LookUpChild(t.ctx, name)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), result.MinObject)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.ExplicitDirType, t.getTypeFromCache(name))
	}

	assert.Equal(t.T(), dirObjName, result.MinObject.Name)
}

func (t *DirTest) TestDeleteChildDir_DoesntExist() {
	const name = "qux"

	err := t.in.DeleteChildDir(t.ctx, name, false, nil)
	require.NoError(t.T(), err)
}

func (t *DirTest) TestDeleteChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	require.NoError(t.T(), err)

	dirIn := t.createDirInode(objName)
	// Call the inode.
	err = t.in.DeleteChildDir(t.ctx, name, false, dirIn)
	require.NoError(t.T(), err)

	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	assert.True(t.T(), errors.As(err, &notFoundErr))
	assert.False(t.T(), dirIn.IsUnlinked())
}

func (t *DirTest) TestDeleteChildDir_ImplicitDirTrue() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	dirIn := t.createDirInode(objName)
	err := t.in.DeleteChildDir(t.ctx, name, true, dirIn)

	require.NoError(t.T(), err)
	assert.False(t.T(), dirIn.IsUnlinked())
}

func (t *DirTest) TestLocalChildFileCore() {
	core, err := t.in.CreateLocalChildFileCore("qux")

	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.bucket.Name(), core.Bucket.Name())
	assert.Equal(t.T(), "foo/bar/qux", core.FullName.objectName)
	assert.True(t.T(), core.Local)
	assert.Nil(t.T(), core.MinObject)
	result, err := t.in.LookUpChild(t.ctx, "qux")
	require.NoError(t.T(), err)
	assert.Nil(t.T(), result)
	if !t.in.IsTypeCacheDeprecated() {
		assert.Equal(t.T(), metadata.UnknownType, t.getTypeFromCache("qux"))
	}
}

func (t *DirTest) TestInsertIntoTypeCache() {
	t.in.InsertFileIntoTypeCache("abc")

	if !t.in.IsTypeCacheDeprecated() {
		d := t.in.(*dirInode)
		tp := t.tc.Get(d.cacheClock.Now(), "abc")
		assert.EqualValues(t.T(), 2, tp)
	}
}

func (t *DirTest) TestEraseFromTypeCache() {
	if t.in.IsTypeCacheDeprecated() {
		return
	}
	t.in.InsertFileIntoTypeCache("abc")

	t.in.EraseFromTypeCache("abc")

	d := t.in.(*dirInode)
	tp := d.cache.Get(d.cacheClock.Now(), "abc")
	require.EqualValues(t.T(), 0, tp)
}

func (t *DirTest) TestDeleteObjects() {
	// Arrange
	parentDirGcsName := t.in.Name().GcsObjectName() // e.g., "foo/bar/"
	d := t.in.(*dirInode)
	// Define supported objects to create.
	objectsToCreate := map[string]string{
		parentDirGcsName + "dir_to_delete/":                           "", // Explicit dir
		parentDirGcsName + "dir_to_delete/file1.txt":                  "content1",
		parentDirGcsName + "dir_to_delete/nested_dir/":                "",
		parentDirGcsName + "dir_to_delete/nested_dir/nested_file.txt": "content_nested",
		parentDirGcsName + "file_to_delete.txt":                       "content_file",
	}
	for objName, content := range objectsToCreate {
		_, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(content))
		require.NoError(t.T(), err)
	}
	// Verify initial state: all created objects exist.
	for objName := range objectsToCreate {
		_, err := storageutil.ReadObject(t.ctx, t.bucket, objName)
		require.NoError(t.T(), err)
	}
	// Act: Call DeleteObjects with the list of supported objects.
	objectsToDelete := []string{
		parentDirGcsName + "dir_to_delete/",
		parentDirGcsName + "file_to_delete.txt",
	}

	err := d.DeleteObjects(t.ctx, objectsToDelete)

	require.NoError(t.T(), err)
	// Assert: All specified objects and their contents should be deleted.
	for _, objName := range objectsToDelete {
		_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
		var notFoundErr *gcs.NotFoundError
		assert.True(t.T(), errors.As(err, &notFoundErr), "Object %s should be deleted. Error: %v", objName, err)
	}
}

func (t *DirTest) TestLocalFileEntriesEmpty() {
	localFileInodes := map[Name]Inode{}

	entries := t.in.LocalFileEntries(localFileInodes)

	require.Equal(t.T(), 0, len(entries))
}

func (t *DirTest) TestLocalFileEntriesWith2LocalChildFiles() {
	in1 := t.createLocalFileInode(t.in.Name(), "1_localChildInode", 1)
	in2 := t.createLocalFileInode(t.in.Name(), "2_localChildInode", 2)
	in3 := t.createLocalFileInode(Name{bucketName: "abc", objectName: "def/"}, "3_localNonChildInode", 3)
	localFileInodes := map[Name]Inode{
		in1.Name(): in1,
		in2.Name(): in2,
		in3.Name(): in3,
	}

	entries := t.in.LocalFileEntries(localFileInodes)

	require.Equal(t.T(), 2, len(entries))
	require.Equal(t.T(), entries[t.getLocalDirentKey(in1)].Name, "1_localChildInode")
	require.Equal(t.T(), entries[t.getLocalDirentKey(in2)].Name, "2_localChildInode")
}

func (t *DirTest) TestLocalFileEntriesWithNoLocalChildFiles() {
	in1 := t.createLocalFileInode(Name{bucketName: "abc", objectName: "def/"}, "1_localNonChildInode", 4)
	in2 := t.createLocalFileInode(Name{bucketName: "abc", objectName: "def/"}, "2_localNonChildInode", 5)
	localFileInodes := map[Name]Inode{
		in1.Name(): in1,
		in2.Name(): in2,
	}

	entries := t.in.LocalFileEntries(localFileInodes)

	require.Equal(t.T(), 0, len(entries))
}

func (t *DirTest) TestLocalFileEntriesWithUnlinkedLocalChildFiles() {
	// Create 2 local child inodes and 1 non child inode.
	in1 := t.createLocalFileInode(t.in.Name(), "1_localChildInode", 1)
	in2 := t.createLocalFileInode(t.in.Name(), "2_localChildInode", 2)
	in3 := t.createLocalFileInode(Name{bucketName: "abc", objectName: "def/"}, "3_localNonChildInode", 3)
	// Unlink local file inode 2.
	filein2, _ := in2.(*FileInode)
	filein2.Unlink()
	// Create local file inodes map.
	localFileInodes := map[Name]Inode{
		in1.Name(): in1,
		in2.Name(): in2,
		in3.Name(): in3,
	}

	entries := t.in.LocalFileEntries(localFileInodes)

	// Validate entries contains only linked child files.
	require.Equal(t.T(), 1, len(entries))
	require.Equal(t.T(), entries[t.getLocalDirentKey(in1)].Name, "1_localChildInode")
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_ListingNotHappenedYet() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = time.Time{}

	// Irrespective of the ttl value, this should always return true.
	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(util.MaxTimeDuration)

	require.Equal(t.T(), true, shouldInvalidate)
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_WithinTtl() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	ttl := time.Second * 10
	t.clock.AdvanceTime(ttl / 2)

	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(ttl)

	require.Equal(t.T(), false, shouldInvalidate)
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_ExpiredTtl() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	ttl := 10 * time.Second
	t.clock.AdvanceTime(ttl + time.Second)

	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(ttl)

	require.Equal(t.T(), true, shouldInvalidate)
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_ZeroTtl() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	ttl := time.Duration(0)

	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(ttl)

	require.Equal(t.T(), true, shouldInvalidate)
}

func (t *DirTest) Test_InvalidateKernelListCache() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	assert.False(t.T(), d.prevDirListingTimeStamp.IsZero())

	t.in.InvalidateKernelListCache()

	assert.True(t.T(), d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) Test_ReadObjectsUnlocked() {
	testCases := []struct {
		name               string
		enableImplicitDirs bool
		expectedCoresCount int
		expectedUnsupCount int
	}{
		{
			name:               "ImplicitDirsDisabled",
			enableImplicitDirs: false,
			expectedCoresCount: 4, // backed_dir_empty, backed_dir_nonempty, file2, symlink
			expectedUnsupCount: 0,
		},
		{
			name:               "ImplicitDirsEnabled",
			enableImplicitDirs: true,
			expectedCoresCount: 5, // Above + implicit_dir
			expectedUnsupCount: 1, // dirInodeName + "//"
		},
	}

	// 1. Setup - Create a superset of all object types once for all cases.
	objs := []string{
		path.Join(dirInodeName, "backed_dir_empty") + "/",
		path.Join(dirInodeName, "backed_dir_nonempty") + "/",
		path.Join(dirInodeName, "backed_dir_nonempty", "file1"),
		path.Join(dirInodeName, "file2"),
		path.Join(dirInodeName, "implicit_dir", "file3"),
		path.Join(dirInodeName, "symlink"),
		dirInodeName + "//" + "invalid",
	}
	err := storageutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
	require.NoError(t.T(), err)
	err = t.setSymlinkTarget(path.Join(dirInodeName, "symlink"))
	require.NoError(t.T(), err)

	for _, tc := range testCases {
		t.T().Run(tc.name, func(st *testing.T) {
			t.resetInode(tc.enableImplicitDirs, false)
			d := t.in.(*dirInode)

			// Execute with lock management
			t.in.Unlock()
			cores, unsupported, _, err := d.readObjectsUnlocked(t.ctx, "")
			t.in.Lock()

			require.NoError(st, err)
			assert.Equal(st, tc.expectedCoresCount, len(cores))
			assert.Equal(st, tc.expectedUnsupCount, len(unsupported))
			t.validateCore(cores, "backed_dir_empty", true, metadata.ExplicitDirType, path.Join(dirInodeName, "backed_dir_empty")+"/")
			t.validateCore(cores, "backed_dir_nonempty", true, metadata.ExplicitDirType, path.Join(dirInodeName, "backed_dir_nonempty")+"/")
			t.validateCore(cores, "file2", false, metadata.RegularFileType, path.Join(dirInodeName, "file2"))
			t.validateCore(cores, "symlink", false, metadata.SymlinkType, path.Join(dirInodeName, "symlink"))
			if tc.enableImplicitDirs {
				t.validateCore(cores, "implicit_dir", true, metadata.ImplicitDirType, path.Join(dirInodeName, "implicit_dir")+"/")
			}
		})
	}
}

func (t *DirTest) Test_readObjectsUnlocked_Empty() {
	// readObjectsUnlocked needs inode in unlocked state.
	t.in.Unlock()
	defer t.in.Lock()
	d := t.in.(*dirInode)
	assert.NotNil(t.T(), d)

	cores, unsupportedPaths, newTok, err := d.readObjectsUnlocked(t.ctx, "")

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(cores))
	assert.Equal(t.T(), 0, len(unsupportedPaths))
	assert.Equal(t.T(), "", newTok)
}

func (t *DirTest) Test_IsTypeCacheDeprecated_false() {
	dInode := t.createDirInodeWithTypeCacheDeprecationFlag(dirInodeName, false)

	assert.False(t.T(), dInode.IsTypeCacheDeprecated())
}

func (t *DirTest) Test_IsTypeCacheDeprecated_true() {
	dInode := t.createDirInodeWithTypeCacheDeprecationFlag(dirInodeName, true)

	assert.True(t.T(), dInode.IsTypeCacheDeprecated())
}
