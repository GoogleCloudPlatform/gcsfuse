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
	"os"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestDir(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const dirInodeID = 17
const dirInodeName = "foo/bar/"
const dirMode os.FileMode = 0712 | os.ModeDir
const typeCacheTTL = time.Second

type DirTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock

	in DirInode
	tc metadata.TypeCache
}

var _ SetUpInterface = &DirTest{}
var _ TearDownInterface = &DirTest{}

func init() { RegisterTestSuite(&DirTest{}) }

func (t *DirTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := fake.NewFakeBucket(&t.clock, "some_bucket", gcs.NonHierarchical)
	t.bucket = gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		bucket)
	// Create the inode. No implicit dirs by default.
	t.resetInode(false, false, true)
}

func (t *DirTest) TearDown() {
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

func (t *DirTest) resetInode(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool) {
	t.resetInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing, 4, typeCacheTTL)
}

func (t *DirTest) resetInodeWithTypeCacheConfigs(implicitDirs, enableNonexistentTypeCache, enableManagedFoldersListing bool, typeCacheMaxSizeMB int64, typeCacheTTL time.Duration) {
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
	AssertNe(nil, d)
	t.tc = d.cache
	AssertNe(nil, t.tc)

	t.in.Lock()
}

func (t *DirTest) createDirInode(dirInodeName string) DirInode {
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
		&t.clock,
		&t.clock,
		4,
		false,
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
		tmp, tok, err = t.in.ReadEntries(t.ctx, tok)
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

func (t *DirTest) setSymlinkTarget(
	objName string,
	target string) (err error) {
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
		true) //localFile
	return
}

func (t *DirTest) getLocalDirentKey(in Inode) string {
	return path.Base(in.Name().LocalName())
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirTest) ID() {
	ExpectEq(dirInodeID, t.in.ID())
}

func (t *DirTest) Name() {
	ExpectEq(dirInodeName, t.in.Name().GcsObjectName())
}

func (t *DirTest) LookupCount() {
	// Increment thrice. The count should now be three.
	t.in.IncrementLookupCount()
	t.in.IncrementLookupCount()
	t.in.IncrementLookupCount()

	// Decrementing twice shouldn't cause destruction. But one more should.
	AssertFalse(t.in.DecrementLookupCount(2))
	ExpectTrue(t.in.DecrementLookupCount(1))
}

func (t *DirTest) Attributes() {
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	ExpectEq(uid, attrs.Uid)
	ExpectEq(gid, attrs.Gid)
	ExpectEq(dirMode|os.ModeDir, attrs.Mode)
}

func (t *DirTest) LookUpChild_NonExistent() {
	const name = "qux"

	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))
}

func (t *DirTest) LookUpChild_FileOnly() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	createObj, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, result.MinObject.Name)
	ExpectEq(createObj.Generation, result.MinObject.Generation)
	ExpectEq(createObj.Size, result.MinObject.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
}

func (t *DirTest) LookUpChild_DirOnly() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object.
	createObj, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, result.MinObject.Name)
	ExpectEq(createObj.Generation, result.MinObject.Generation)
	ExpectEq(createObj.Size, result.MinObject.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Disabled() {
	const name = "qux"
	var err error

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Looking up the name shouldn't work.
	result, err := t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	ExpectEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))

	// Ditto with a conflict marker.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Enabled() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Enable implicit dirs.
	t.resetInode(true, false, true)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Looking up the name should work.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	ExpectEq(nil, result.MinObject)
	ExpectEq(metadata.ImplicitDirType, t.getTypeFromCache(name))

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(metadata.ImplicitDirType, result.Type())

	// A conflict marker should not work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
}

func (t *DirTest) LookUpChild_FileAndDir() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create backing objects.
	fileObj, err := storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.MinObject.Name)
	ExpectEq(dirObj.Generation, result.MinObject.Generation)
	ExpectEq(dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	AssertEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, result.MinObject.Name)
	ExpectEq(fileObj.Generation, result.MinObject.Generation)
	ExpectEq(fileObj.Size, result.MinObject.Size)
}

func (t *DirTest) LookUpChild_SymlinkAndDir() {
	const name = "qux"
	linkObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create backing objects.
	linkObj, err := storageutil.CreateObject(t.ctx, t.bucket, linkObjName, []byte("taco"))
	AssertEq(nil, err)

	err = t.setSymlinkTarget(linkObjName, "blah")
	AssertEq(nil, err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)

	// The following check should have been for metadata.SymlinkType,
	// because of the t.setSymlinkTarget call above, but it is not.
	// This is so because the above symlink is
	// created as a regular directory object on GCS,
	// and is read back the same to gcsfuse and is this stored in type-cache
	// also as a directory.
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.MinObject.Name)
	ExpectEq(dirObj.Generation, result.MinObject.Generation)
	ExpectEq(dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))

	ExpectEq(linkObjName, result.FullName.GcsObjectName())
	ExpectEq(linkObjName, result.MinObject.Name)
	ExpectEq(linkObj.Generation, result.MinObject.Generation)
	ExpectEq(linkObj.Size, result.MinObject.Size)
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Disabled() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create backing objects.
	fileObj, err := storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(path.Join(dirInodeName, name)))

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.MinObject.Name)
	ExpectEq(dirObj.Generation, result.MinObject.Generation)
	ExpectEq(dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, result.MinObject.Name)
	ExpectEq(fileObj.Generation, result.MinObject.Generation)
	ExpectEq(fileObj.Size, result.MinObject.Size)
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Enabled() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Enable implicit dirs.
	t.resetInode(true, false, true)

	// Create backing objects.
	fileObj, err := storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	dirObj, err := storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(path.Join(dirInodeName, name)))

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.MinObject.Name)
	ExpectEq(dirObj.Generation, result.MinObject.Generation)
	ExpectEq(dirObj.Size, result.MinObject.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, result.MinObject.Name)
	ExpectEq(fileObj.Generation, result.MinObject.Generation)
	ExpectEq(fileObj.Size, result.MinObject.Size)
}

func (t *DirTest) LookUpChild_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object for a file.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up; we should get the file.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))

	ExpectEq(fileObjName, result.MinObject.Name)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up again. Even though the directory should shadow the file, because
	// we've cached only seeing the file that's what we should get back.
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))

	ExpectEq(fileObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(dirObjName, result.MinObject.Name)
}

func (t *DirTest) LookUpChild_NonExistentTypeCache_ImplicitDirsDisabled() {
	// Enable enableNonexistentTypeCache for type cache
	t.resetInode(false, true, true)

	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Look up nonexistent object, return nil
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertEq(nil, result)

	// Create a backing object.
	createObj, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	AssertEq(nil, err)

	// Look up again, should still return nil due to cache
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertEq(nil, result)
	ExpectEq(metadata.NonexistentType, t.getTypeFromCache(name))

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	// Look up again, should return correct object
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, result.MinObject.Name)
	ExpectEq(createObj.Generation, result.MinObject.Generation)
	ExpectEq(createObj.Size, result.MinObject.Size)
}

func (t *DirTest) LookUpChild_NonExistentTypeCache_ImplicitDirsEnabled() {
	// Enable implicitDirs and enableNonexistentTypeCache for type cache
	t.resetInode(true, true, true)

	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Look up nonexistent object, return nil
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertEq(nil, result)
	ExpectEq(metadata.NonexistentType, t.getTypeFromCache(name))

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Look up again, should still return nil due to cache
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertEq(nil, result)
	ExpectEq(metadata.NonexistentType, t.getTypeFromCache(name))

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	// Look up again, should return correct object
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	ExpectEq(nil, result.MinObject)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(metadata.ImplicitDirType, result.Type())

	// A conflict marker should not work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name+ConflictingFileNameSuffix))
}

func (t *DirTest) LookUpChild_TypeCacheEnabled() {
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

		AssertEq(nil, err)
		AssertNe(nil, o)

		// Look up nonexistent object, return nil
		result, err := t.in.LookUpChild(t.ctx, name)

		AssertEq(nil, err)
		AssertNe(nil, result)
		ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))
	}
}

func (t *DirTest) LookUpChild_TypeCacheDisabled() {
	inputs := []struct {
		typeCacheMaxSizeMB int64
		typeCacheTTL       time.Duration
	}{{
		typeCacheMaxSizeMB: 0,
		typeCacheTTL:       time.Second,
	}, {
		typeCacheMaxSizeMB: 4,
		typeCacheTTL:       0,
	}}

	for _, input := range inputs {
		t.resetInodeWithTypeCacheConfigs(true, true, true, input.typeCacheMaxSizeMB, input.typeCacheTTL)

		const name = "qux"
		objName := path.Join(dirInodeName, name)

		// Create a backing object.
		o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))

		AssertEq(nil, err)
		AssertNe(nil, o)

		// Look up nonexistent object, return nil
		result, err := t.in.LookUpChild(t.ctx, name)

		AssertEq(nil, err)
		AssertNe(nil, result)
		ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))
	}
}

func (t *DirTest) ReadDescendants_Empty() {
	descendants, err := t.in.ReadDescendants(t.ctx, 10)

	AssertEq(nil, err)
	ExpectEq(0, len(descendants))

}

func (t *DirTest) ReadDescendants_NonEmpty() {
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
	AssertEq(nil, err)

	descendants, err := t.in.ReadDescendants(t.ctx, 10)
	AssertEq(nil, err)
	ExpectEq(6, len(descendants))

	descendants, err = t.in.ReadDescendants(t.ctx, 2)
	AssertEq(nil, err)
	ExpectEq(2, len(descendants))
}

func (t *DirTest) ReadEntries_Empty() {
	d := t.in.(*dirInode)
	AssertNe(nil, d)
	AssertTrue(d.prevDirListingTimeStamp.IsZero())
	entries, err := t.readAllEntries()

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
	// Make sure prevDirListingTimeStamp is initialized.
	AssertFalse(d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) ReadEntries_NonEmpty_ImplicitDirsDisabled() {
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
	AssertEq(nil, err)

	// Set up the symlink target.
	err = t.setSymlinkTarget(dirInodeName+"symlink", "blah")
	AssertEq(nil, err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	AssertNe(nil, d)
	AssertTrue(d.prevDirListingTimeStamp.IsZero())

	// Read entries.
	entries, err := t.readAllEntries()

	AssertEq(nil, err)
	AssertEq(4, len(entries))

	entry = entries[0]
	ExpectEq("backed_dir_empty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_empty"))

	entry = entries[1]
	ExpectEq("backed_dir_nonempty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_nonempty"))

	entry = entries[2]
	ExpectEq("file", entry.Name)
	ExpectEq(fuseutil.DT_File, entry.Type)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache("file"))

	entry = entries[3]
	ExpectEq("symlink", entry.Name)
	ExpectEq(fuseutil.DT_Link, entry.Type)
	ExpectEq(metadata.SymlinkType, t.getTypeFromCache("symlink"))

	// Make sure prevDirListingTimeStamp is initialized.
	AssertFalse(d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) ReadEntries_NonEmpty_ImplicitDirsEnabled() {
	var err error
	var entry fuseutil.Dirent

	// Enable implicit dirs.
	t.resetInode(true, false, true)

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
	AssertEq(nil, err)

	// Set up the symlink target.
	err = t.setSymlinkTarget(dirInodeName+"symlink", "blah")
	AssertEq(nil, err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	AssertNe(nil, d)
	AssertTrue(d.prevDirListingTimeStamp.IsZero())

	// Read entries.
	entries, err := t.readAllEntries()

	AssertEq(nil, err)
	AssertEq(5, len(entries))

	entry = entries[0]
	ExpectEq("backed_dir_empty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_empty"))

	entry = entries[1]
	ExpectEq("backed_dir_nonempty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache("backed_dir_nonempty"))

	entry = entries[2]
	ExpectEq("file", entry.Name)
	ExpectEq(fuseutil.DT_File, entry.Type)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache("file"))

	entry = entries[3]
	ExpectEq("implicit_dir", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)
	ExpectEq(metadata.ImplicitDirType, t.getTypeFromCache("implicit_dir"))

	entry = entries[4]
	ExpectEq("symlink", entry.Name)
	ExpectEq(fuseutil.DT_Link, entry.Type)
	ExpectEq(metadata.SymlinkType, t.getTypeFromCache("symlink"))

	// Make sure prevDirListingTimeStamp is initialized.
	AssertFalse(d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) ReadEntries_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object for a file.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	// Nil prevDirListingTimeStamp
	d := t.in.(*dirInode)
	AssertNe(nil, d)
	AssertTrue(d.prevDirListingTimeStamp.IsZero())

	// Read the directory, priming the type cache.
	_, err = t.readAllEntries()
	AssertEq(nil, err)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))

	ExpectEq(fileObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(dirObjName, result.MinObject.Name)

	// Make sure prevDirListingTimeStamp is initialized.
	AssertFalse(d.prevDirListingTimeStamp.IsZero())
}

func (t *DirTest) CreateChildFile_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	result, err := t.in.CreateChildFile(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.MinObject)

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.MinObject.Name)
	ExpectEq(objName, result.MinObject.Name)
	ExpectFalse(IsSymlink(result.MinObject))
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))

	ExpectEq(1, len(result.MinObject.Metadata))
	ExpectEq(
		t.clock.Now().UTC().Format(time.RFC3339Nano),
		result.MinObject.Metadata["gcsfuse_mtime"])
}

func (t *DirTest) CreateChildFile_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create an existing backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	_, err = t.in.CreateChildFile(t.ctx, name)
	ExpectThat(err, Error(HasSubstr("Precondition")))
	ExpectThat(err, Error(HasSubstr("exists")))
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))
}

func (t *DirTest) CreateChildFile_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create the name.
	_, err = t.in.CreateChildFile(t.ctx, name)
	AssertEq(nil, err)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache(name))

	ExpectEq(fileObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(dirObjName, result.MinObject.Name)
}

func (t *DirTest) CloneToChildFile_SourceDoesntExist() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	var err error

	// Create and then delete the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte(""))
	AssertEq(nil, err)

	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{Name: srcName})

	AssertEq(nil, err)

	// Call the inode.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	_, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(dstName))
}

func (t *DirTest) CloneToChildFile_DestinationDoesntExist() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	result, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache("qux"))

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.MinObject.Name)
	ExpectEq(dstName, result.MinObject.Name)
	ExpectFalse(IsSymlink(result.MinObject))

	// Check resulting contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, dstName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache("qux"))
}

func (t *DirTest) CloneToChildFile_DestinationExists() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte("taco"))
	AssertEq(nil, err)

	// And a destination object that will be overwritten.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dstName, []byte(""))
	AssertEq(nil, err)

	// Call the inode.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	result, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache("qux"))

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.MinObject.Name)
	ExpectEq(dstName, result.MinObject.Name)
	ExpectFalse(IsSymlink(result.MinObject))
	ExpectEq(len("taco"), result.MinObject.Size)

	// Check resulting contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, dstName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache("qux"))
}

func (t *DirTest) CloneToChildFile_TypeCaching() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	var err error

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte(""))
	AssertEq(nil, err)

	// Clone to the destination.
	srcMinObject := storageutil.ConvertObjToMinObject(src)
	_, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), srcMinObject)
	AssertEq(nil, err)

	// Create a backing object for a directory.
	dirObjName := dstName + "/"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, path.Base(dstName))

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.RegularFileType, t.getTypeFromCache("qux"))

	ExpectEq(dstName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, path.Base(dstName))

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache("qux"))

	ExpectEq(dirObjName, result.MinObject.Name)
}

func (t *DirTest) CreateChildSymlink_DoesntExist() {
	const name = "qux"
	const target = "taco"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	result, err := t.in.CreateChildSymlink(t.ctx, name, target)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.SymlinkType, t.getTypeFromCache(name))

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.MinObject.Name)
	ExpectEq(objName, result.MinObject.Name)
	ExpectEq(target, result.MinObject.Metadata[SymlinkMetadataKey])
}

func (t *DirTest) CreateChildSymlink_Exists() {
	const name = "qux"
	const target = "taco"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create an existing backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	AssertEq(nil, err)

	// Call the inode.
	_, err = t.in.CreateChildSymlink(t.ctx, name, target)
	ExpectThat(err, Error(HasSubstr("Precondition")))
	ExpectThat(err, Error(HasSubstr("exists")))
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))
}

func (t *DirTest) CreateChildSymlink_TypeCaching() {
	const name = "qux"
	linkObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create the name.
	_, err = t.in.CreateChildSymlink(t.ctx, name, "")
	AssertEq(nil, err)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the symlink,
	// because we've cached only seeing the symlink that's what we should get
	// back.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.SymlinkType, t.getTypeFromCache(name))

	ExpectEq(linkObjName, result.MinObject.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(dirObjName, result.MinObject.Name)
}

func (t *DirTest) CreateChildDir_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Call the inode.
	result, err := t.in.CreateChildDir(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.MinObject.Name)
	ExpectEq(objName, result.MinObject.Name)
	ExpectFalse(IsSymlink(result.MinObject))
}

func (t *DirTest) CreateChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create an existing backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	_, err = t.in.CreateChildDir(t.ctx, name)
	ExpectThat(err, Error(HasSubstr("Precondition")))
	ExpectThat(err, Error(HasSubstr("exists")))
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))
}

func (t *DirTest) DeleteChildFile_DoesntExist() {
	const name = "qux"

	err := t.in.DeleteChildFile(t.ctx, name, 0, nil)
	ExpectEq(nil, err)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache(name))
}

func (t *DirTest) DeleteChildFile_WrongGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode with the wrong generation. No error should be returned.
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation+1, &o.MetaGeneration)
	AssertEq(nil, err)

	// The original generation should still be there.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, objName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *DirTest) DeleteChildFile_WrongMetaGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode with the wrong meta-generation. No error should be
	// returned.
	precond := o.MetaGeneration + 1
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &precond)

	ExpectThat(err, Error(HasSubstr("Precondition")))
	ExpectThat(err, Error(HasSubstr("meta-generation")))

	// The original generation should still be there.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, objName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *DirTest) DeleteChildFile_LatestGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	err = t.in.DeleteChildFile(t.ctx, name, 0, nil)
	AssertEq(nil, err)

	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *DirTest) DeleteChildFile_ParticularGenerationAndMetaGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &o.MetaGeneration)
	AssertEq(nil, err)

	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *DirTest) DeleteChildFile_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create the name, priming the type cache.
	_, err = t.in.CreateChildFile(t.ctx, name)
	AssertEq(nil, err)

	// Create a backing object for a directory. It should be shadowed by the
	// file.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	AssertEq(fileObjName, result.MinObject.Name)

	// But after deleting the file via the inode, the directory should be
	// revealed.
	err = t.in.DeleteChildFile(t.ctx, name, 0, nil)
	AssertEq(nil, err)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.MinObject)
	ExpectEq(metadata.ExplicitDirType, t.getTypeFromCache(name))

	ExpectEq(dirObjName, result.MinObject.Name)
}

func (t *DirTest) DeleteChildDir_DoesntExist() {
	const name = "qux"

	err := t.in.DeleteChildDir(t.ctx, name, false, nil)
	ExpectEq(nil, err)
}

func (t *DirTest) DeleteChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	dirIn := t.createDirInode(objName)
	// Call the inode.
	err = t.in.DeleteChildDir(t.ctx, name, false, dirIn)
	AssertEq(nil, err)

	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
	ExpectFalse(dirIn.IsUnlinked())
}

func (t *DirTest) DeleteChildDir_ImplicitDirTrue() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	dirIn := t.createDirInode(objName)
	err := t.in.DeleteChildDir(t.ctx, name, true, dirIn)

	ExpectEq(nil, err)
	ExpectFalse(dirIn.IsUnlinked())
}

func (t *DirTest) LocalChildFileCore() {
	core, err := t.in.CreateLocalChildFileCore("qux")

	AssertEq(nil, err)
	AssertEq(t.bucket.Name(), core.Bucket.Name())
	AssertEq("foo/bar/qux", core.FullName.objectName)
	AssertTrue(core.Local)
	AssertEq(nil, core.MinObject)
	result, err := t.in.LookUpChild(t.ctx, "qux")
	AssertEq(nil, err)
	AssertEq(nil, result)
	ExpectEq(metadata.UnknownType, t.getTypeFromCache("qux"))
}

func (t *DirTest) InsertIntoTypeCache() {
	t.in.InsertFileIntoTypeCache("abc")

	d := t.in.(*dirInode)
	tp := t.tc.Get(d.cacheClock.Now(), "abc")
	AssertEq(2, tp)
}

func (t *DirTest) EraseFromTypeCache() {
	t.in.InsertFileIntoTypeCache("abc")

	t.in.EraseFromTypeCache("abc")

	d := t.in.(*dirInode)
	tp := d.cache.Get(d.cacheClock.Now(), "abc")
	AssertEq(0, tp)
}

func (t *DirTest) LocalFileEntriesEmpty() {
	localFileInodes := map[Name]Inode{}

	entries := t.in.LocalFileEntries(localFileInodes)

	AssertEq(0, len(entries))
}

func (t *DirTest) LocalFileEntriesWith2LocalChildFiles() {
	in1 := t.createLocalFileInode(t.in.Name(), "1_localChildInode", 1)
	in2 := t.createLocalFileInode(t.in.Name(), "2_localChildInode", 2)
	in3 := t.createLocalFileInode(Name{bucketName: "abc", objectName: "def/"}, "3_localNonChildInode", 3)
	localFileInodes := map[Name]Inode{
		in1.Name(): in1,
		in2.Name(): in2,
		in3.Name(): in3,
	}

	entries := t.in.LocalFileEntries(localFileInodes)

	AssertEq(2, len(entries))
	AssertEq(entries[t.getLocalDirentKey(in1)].Name, "1_localChildInode")
	AssertEq(entries[t.getLocalDirentKey(in2)].Name, "2_localChildInode")
}

func (t *DirTest) LocalFileEntriesWithNoLocalChildFiles() {
	in1 := t.createLocalFileInode(Name{bucketName: "abc", objectName: "def/"}, "1_localNonChildInode", 4)
	in2 := t.createLocalFileInode(Name{bucketName: "abc", objectName: "def/"}, "2_localNonChildInode", 5)
	localFileInodes := map[Name]Inode{
		in1.Name(): in1,
		in2.Name(): in2,
	}

	entries := t.in.LocalFileEntries(localFileInodes)

	AssertEq(0, len(entries))
}

func (t *DirTest) LocalFileEntriesWithUnlinkedLocalChildFiles() {
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
	AssertEq(1, len(entries))
	AssertEq(entries[t.getLocalDirentKey(in1)].Name, "1_localChildInode")
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_ListingNotHappenedYet() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = time.Time{}

	// Irrespective of the ttl value, this should always return true.
	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(util.MaxTimeDuration)

	AssertEq(true, shouldInvalidate)
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_WithinTtl() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	ttl := time.Second * 10
	t.clock.AdvanceTime(ttl / 2)

	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(ttl)

	AssertEq(false, shouldInvalidate)
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_ExpiredTtl() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	ttl := 10 * time.Second
	t.clock.AdvanceTime(ttl + time.Second)

	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(ttl)

	AssertEq(true, shouldInvalidate)
}

func (t *DirTest) Test_ShouldInvalidateKernelListCache_ZeroTtl() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	ttl := time.Duration(0)

	shouldInvalidate := t.in.ShouldInvalidateKernelListCache(ttl)

	AssertEq(true, shouldInvalidate)
}

func (t *DirTest) Test_InvalidateKernelListCache() {
	d := t.in.(*dirInode)
	d.prevDirListingTimeStamp = d.cacheClock.Now()
	AssertFalse(d.prevDirListingTimeStamp.IsZero())

	t.in.InvalidateKernelListCache()

	AssertTrue(d.prevDirListingTimeStamp.IsZero())
}
