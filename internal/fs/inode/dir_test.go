// Copyright 2015 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
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
const typeCacheMaxSizeMbPerDirectory = 16

type DirTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock

	in DirInode
}

var _ SetUpInterface = &DirTest{}
var _ TearDownInterface = &DirTest{}

func init() { RegisterTestSuite(&DirTest{}) }

func (t *DirTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := fake.NewFakeBucket(&t.clock, "some_bucket")
	t.bucket = gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		bucket)
	// Create the inode. No implicit dirs by default.
	t.resetInode(false, false)
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

func (t *DirTest) resetInode(implicitDirs bool, enableNonexistentTypeCache bool) {
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
		enableNonexistentTypeCache,
		typeCacheTTL,
		&t.bucket,
		&t.clock,
		&t.clock,
		typeCacheMaxSizeMbPerDirectory)

	t.in.Lock()
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
	result, err := t.in.LookUpChild(t.ctx, "qux")

	AssertEq(nil, err)
	AssertEq(nil, result)
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
	AssertNe(nil, result.Object)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, result.Object.Name)
	ExpectEq(createObj.Generation, result.Object.Generation)
	ExpectEq(createObj.Size, result.Object.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
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
	AssertNe(nil, result.Object)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, result.Object.Name)
	ExpectEq(createObj.Generation, result.Object.Generation)
	ExpectEq(createObj.Size, result.Object.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
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

	// Ditto with a conflict marker.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Enabled() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Enable implicit dirs.
	t.resetInode(true, false)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Looking up the name should work.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	ExpectEq(nil, result.Object)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(ImplicitDirType, result.Type())

	// A conflict marker should not work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
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
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.Object.Name)
	ExpectEq(dirObj.Generation, result.Object.Generation)
	ExpectEq(dirObj.Size, result.Object.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, result.Object.Name)
	ExpectEq(fileObj.Generation, result.Object.Generation)
	ExpectEq(fileObj.Size, result.Object.Size)
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
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.Object.Name)
	ExpectEq(dirObj.Generation, result.Object.Generation)
	ExpectEq(dirObj.Size, result.Object.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(linkObjName, result.FullName.GcsObjectName())
	ExpectEq(linkObjName, result.Object.Name)
	ExpectEq(linkObj.Generation, result.Object.Generation)
	ExpectEq(linkObj.Size, result.Object.Size)
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
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.Object.Name)
	ExpectEq(dirObj.Generation, result.Object.Generation)
	ExpectEq(dirObj.Size, result.Object.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, result.Object.Name)
	ExpectEq(fileObj.Generation, result.Object.Generation)
	ExpectEq(fileObj.Size, result.Object.Size)
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Enabled() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Enable implicit dirs.
	t.resetInode(true, false)

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
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, result.Object.Name)
	ExpectEq(dirObj.Generation, result.Object.Generation)
	ExpectEq(dirObj.Size, result.Object.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, result.Object.Name)
	ExpectEq(fileObj.Generation, result.Object.Generation)
	ExpectEq(fileObj.Size, result.Object.Size)
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
	AssertNe(nil, result.Object)

	ExpectEq(fileObjName, result.Object.Name)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up again. Even though the directory should shadow the file, because
	// we've cached only seeing the file that's what we should get back.
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(fileObjName, result.Object.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.Object.Name)
}

func (t *DirTest) LookUpChild_NonExistentTypeCache_ImplicitDirsDisabled() {
	// Enable enableNonexistentTypeCache for type cache
	t.resetInode(false, true)

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

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	// Look up again, should return correct object
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, result.Object.Name)
	ExpectEq(createObj.Generation, result.Object.Generation)
	ExpectEq(createObj.Size, result.Object.Size)
}

func (t *DirTest) LookUpChild_NonExistentTypeCache_ImplicitDirsEnabled() {
	// Enable implicitDirs and enableNonexistentTypeCache for type cache
	t.resetInode(true, true)

	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Look up nonexistent object, return nil
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertEq(nil, result)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = storageutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Look up again, should still return nil due to cache
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertEq(nil, result)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	// Look up again, should return correct object
	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	ExpectEq(nil, result.Object)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(ImplicitDirType, result.Type())

	// A conflict marker should not work.
	result, err = t.in.LookUpChild(t.ctx, name+ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, result)
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
	entries, err := t.readAllEntries()

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
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

	// Read entries.
	entries, err := t.readAllEntries()

	AssertEq(nil, err)
	AssertEq(4, len(entries))

	entry = entries[0]
	ExpectEq("backed_dir_empty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)

	entry = entries[1]
	ExpectEq("backed_dir_nonempty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)

	entry = entries[2]
	ExpectEq("file", entry.Name)
	ExpectEq(fuseutil.DT_File, entry.Type)

	entry = entries[3]
	ExpectEq("symlink", entry.Name)
	ExpectEq(fuseutil.DT_Link, entry.Type)
}

func (t *DirTest) ReadEntries_NonEmpty_ImplicitDirsEnabled() {
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
	AssertEq(nil, err)

	// Set up the symlink target.
	err = t.setSymlinkTarget(dirInodeName+"symlink", "blah")
	AssertEq(nil, err)

	// Read entries.
	entries, err := t.readAllEntries()

	AssertEq(nil, err)
	AssertEq(5, len(entries))

	entry = entries[0]
	ExpectEq("backed_dir_empty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)

	entry = entries[1]
	ExpectEq("backed_dir_nonempty", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)

	entry = entries[2]
	ExpectEq("file", entry.Name)
	ExpectEq(fuseutil.DT_File, entry.Type)

	entry = entries[3]
	ExpectEq("implicit_dir", entry.Name)
	ExpectEq(fuseutil.DT_Directory, entry.Type)

	entry = entries[4]
	ExpectEq("symlink", entry.Name)
	ExpectEq(fuseutil.DT_Link, entry.Type)
}

func (t *DirTest) ReadEntries_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object for a file.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	// Read the directory, priming the type cache.
	_, err = t.readAllEntries()
	AssertEq(nil, err)

	// Create a backing object for a directory.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(fileObjName, result.Object.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.Object.Name)
}

func (t *DirTest) CreateChildFile_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	result, err := t.in.CreateChildFile(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.Object)

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.Object.Name)
	ExpectEq(objName, result.Object.Name)
	ExpectFalse(IsSymlink(result.Object))

	ExpectEq(1, len(result.Object.Metadata))
	ExpectEq(
		t.clock.Now().UTC().Format(time.RFC3339Nano),
		result.Object.Metadata["gcsfuse_mtime"])
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
	AssertNe(nil, result.Object)

	ExpectEq(fileObjName, result.Object.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.Object.Name)
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
	_, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *DirTest) CloneToChildFile_DestinationDoesntExist() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	result, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.Object)

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.Object.Name)
	ExpectEq(dstName, result.Object.Name)
	ExpectFalse(IsSymlink(result.Object))

	// Check resulting contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, dstName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
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
	result, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.Object)

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.Object.Name)
	ExpectEq(dstName, result.Object.Name)
	ExpectFalse(IsSymlink(result.Object))
	ExpectEq(len("taco"), result.Object.Size)

	// Check resulting contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, dstName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *DirTest) CloneToChildFile_TypeCaching() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	var err error

	// Create the source.
	src, err := storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte(""))
	AssertEq(nil, err)

	// Clone to the destination.
	_, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	AssertEq(nil, err)

	// Create a backing object for a directory.
	dirObjName := dstName + "/"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, path.Base(dstName))

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(dstName, result.Object.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, path.Base(dstName))

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.Object.Name)
}

func (t *DirTest) CreateChildSymlink_DoesntExist() {
	const name = "qux"
	const target = "taco"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	result, err := t.in.CreateChildSymlink(t.ctx, name, target)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.Object)

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.Object.Name)
	ExpectEq(objName, result.Object.Name)
	ExpectEq(target, result.Object.Metadata[SymlinkMetadataKey])
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
	AssertNe(nil, result.Object)

	ExpectEq(linkObjName, result.Object.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.Object.Name)
}

func (t *DirTest) CreateChildDir_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Call the inode.
	result, err := t.in.CreateChildDir(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, result)
	AssertNe(nil, result.Object)

	ExpectEq(t.bucket.Name(), result.Bucket.Name())
	ExpectEq(result.FullName.GcsObjectName(), result.Object.Name)
	ExpectEq(objName, result.Object.Name)
	ExpectFalse(IsSymlink(result.Object))
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
}

func (t *DirTest) DeleteChildFile_DoesntExist() {
	const name = "qux"

	err := t.in.DeleteChildFile(t.ctx, name, 0, nil)
	ExpectEq(nil, err)
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
	AssertNe(nil, result.Object)
	AssertEq(fileObjName, result.Object.Name)

	// But after deleting the file via the inode, the directory should be
	// revealed.
	err = t.in.DeleteChildFile(t.ctx, name, 0, nil)
	AssertEq(nil, err)

	result, err = t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	AssertNe(nil, result.Object)

	ExpectEq(dirObjName, result.Object.Name)
}

func (t *DirTest) DeleteChildDir_DoesntExist() {
	const name = "qux"

	err := t.in.DeleteChildDir(t.ctx, name, false)
	ExpectEq(nil, err)
}

func (t *DirTest) DeleteChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	err = t.in.DeleteChildDir(t.ctx, name, false)
	AssertEq(nil, err)

	// Check the bucket.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, objName)
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *DirTest) DeleteChildDir_ImplicitDirTrue() {
	const name = "qux"

	err := t.in.DeleteChildDir(t.ctx, name, true)
	ExpectEq(nil, err)
}

func (t *DirTest) CreateLocalChildFile_ShouldnotCreateObjectInGCS() {
	const name = "qux"

	// Create the local file inode.
	core, err := t.in.CreateLocalChildFile(name)

	AssertEq(nil, err)
	AssertEq(true, core.Local)
	AssertEq(nil, core.Object)

	// Object shouldn't get created in GCS.
	result, err := t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertEq(nil, result)
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
	entryNames := []string{entries[0].Name, entries[1].Name}
	sort.Strings(entryNames)
	AssertEq(entryNames[0], "1_localChildInode")
	AssertEq(entryNames[1], "2_localChildInode")
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
	AssertEq(entries[0].Name, "1_localChildInode")
}
