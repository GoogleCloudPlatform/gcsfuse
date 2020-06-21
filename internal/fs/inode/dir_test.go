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

package inode_test

import (
	"os"
	"path"
	"sort"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
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

	in inode.DirInode
}

var _ SetUpInterface = &DirTest{}
var _ TearDownInterface = &DirTest{}

func init() { RegisterTestSuite(&DirTest{}) }

func (t *DirTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := gcsfake.NewFakeBucket(&t.clock, "some_bucket")
	t.bucket = gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		bucket)
	// Create the inode. No implicit dirs by default.
	t.resetInode(false)
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

func (t *DirTest) resetInode(implicitDirs bool) {
	if t.in != nil {
		t.in.Unlock()
	}

	t.in = inode.NewDirInode(
		dirInodeID,
		inode.NewDirName(inode.NewRootName(), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		implicitDirs,
		typeCacheTTL,
		t.bucket,
		&t.clock,
		&t.clock)

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
				inode.SymlinkMetadataKey: &target,
			},
		})

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
	ExpectFalse(result.Exists())
}

func (t *DirTest) LookUpChild_FileOnly() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var o *gcs.Object
	var err error

	// Create a backing object.
	createObj, err := gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, o.Name)
	ExpectEq(createObj.Generation, o.Generation)
	ExpectEq(createObj.Size, o.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectFalse(result.Exists())
}

func (t *DirTest) LookUpChild_DirOnly() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create a backing object.
	createObj, err := gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectEq(objName, o.Name)
	ExpectEq(createObj.Generation, o.Generation)
	ExpectEq(createObj.Size, o.Size)

	// A conflict marker name shouldn't work.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectFalse(result.Exists())
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Disabled() {
	const name = "qux"
	var err error

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Looking up the name shouldn't work.
	result, err := t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	ExpectFalse(result.Exists())

	// Ditto with a conflict marker.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectFalse(result.Exists())
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Enabled() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Enable implicit dirs.
	t.resetInode(true)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Looking up the name should work.
	result, err := t.in.LookUpChild(t.ctx, name)

	AssertEq(nil, err)
	ExpectEq(nil, result.Object)

	ExpectEq(objName, result.FullName.GcsObjectName())
	ExpectTrue(result.ImplicitDir)

	// A conflict marker should not work.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectFalse(result.Exists())
}

func (t *DirTest) LookUpChild_FileAndDir() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create backing objects.
	fileObj, err := gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	dirObj, err := gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, o.Name)
	ExpectEq(dirObj.Generation, o.Generation)
	ExpectEq(dirObj.Size, o.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, o.Name)
	ExpectEq(fileObj.Generation, o.Generation)
	ExpectEq(fileObj.Size, o.Size)
}

func (t *DirTest) LookUpChild_SymlinkAndDir() {
	const name = "qux"
	linkObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create backing objects.
	linkObj, err := gcsutil.CreateObject(t.ctx, t.bucket, linkObjName, []byte("taco"))
	AssertEq(nil, err)

	err = t.setSymlinkTarget(linkObjName, "blah")
	AssertEq(nil, err)

	dirObj, err := gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, o.Name)
	ExpectEq(dirObj.Generation, o.Generation)
	ExpectEq(dirObj.Size, o.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(linkObjName, result.FullName.GcsObjectName())
	ExpectEq(linkObjName, o.Name)
	ExpectEq(linkObj.Generation, o.Generation)
	ExpectEq(linkObj.Size, o.Size)
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Disabled() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create backing objects.
	fileObj, err := gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	dirObj, err := gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, o.Name)
	ExpectEq(dirObj.Generation, o.Generation)
	ExpectEq(dirObj.Size, o.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, o.Name)
	ExpectEq(fileObj.Generation, o.Generation)
	ExpectEq(fileObj.Size, o.Size)
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Enabled() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Enable implicit dirs.
	t.resetInode(true)

	// Create backing objects.
	fileObj, err := gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	dirObj, err := gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(dirInodeName, name) + "/asdf"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, []byte(""))
	AssertEq(nil, err)

	// Look up with the proper name.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, result.FullName.GcsObjectName())
	ExpectEq(dirObjName, o.Name)
	ExpectEq(dirObj.Generation, o.Generation)
	ExpectEq(dirObj.Size, o.Size)

	// Look up with the conflict marker name.
	result, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, result.FullName.GcsObjectName())
	ExpectEq(fileObjName, o.Name)
	ExpectEq(fileObj.Generation, o.Generation)
	ExpectEq(fileObj.Size, o.Size)
}

func (t *DirTest) LookUpChild_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create a backing object for a file.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up; we should get the file.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)

	// Create a backing object for a directory.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up again. Even though the directory should shadow the file, because
	// we've cached only seeing the file that's what we should get back.
	result, err = t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
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

	err = gcsutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
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
	t.resetInode(true)

	// Set up contents.
	objs := []string{
		dirInodeName + "backed_dir_empty/",
		dirInodeName + "backed_dir_nonempty/",
		dirInodeName + "backed_dir_nonempty/blah",
		dirInodeName + "file",
		dirInodeName + "implicit_dir/blah",
		dirInodeName + "symlink",
	}

	err = gcsutil.CreateEmptyObjects(t.ctx, t.bucket, objs)
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

	var o *gcs.Object
	var err error

	// Create a backing object for a file.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, []byte("taco"))
	AssertEq(nil, err)

	// Read the directory, priming the type cache.
	_, err = t.readAllEntries()
	AssertEq(nil, err)

	// Create a backing object for a directory.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
}

func (t *DirTest) CreateChildFile_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	b, fn, o, err := t.in.CreateChildFile(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(t.bucket.Name(), b.Name())
	ExpectEq(fn.GcsObjectName(), o.Name)
	ExpectEq(objName, o.Name)
	ExpectFalse(inode.IsSymlink(o))

	ExpectEq(1, len(o.Metadata))
	ExpectEq(
		t.clock.Now().UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])
}

func (t *DirTest) CreateChildFile_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create an existing backing object.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	_, _, _, err = t.in.CreateChildFile(t.ctx, name)
	ExpectThat(err, Error(HasSubstr("Precondition")))
	ExpectThat(err, Error(HasSubstr("exists")))
}

func (t *DirTest) CreateChildFile_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create the name.
	_, _, _, err = t.in.CreateChildFile(t.ctx, name)
	AssertEq(nil, err)

	// Create a backing object for a directory.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
}

func (t *DirTest) CloneToChildFile_SourceDoesntExist() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	var err error

	// Create and then delete the source.
	src, err := gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte(""))
	AssertEq(nil, err)

	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{Name: srcName})

	AssertEq(nil, err)

	// Call the inode.
	_, _, _, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *DirTest) CloneToChildFile_DestinationDoesntExist() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	// Create the source.
	src, err := gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	b, fn, o, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(t.bucket.Name(), b.Name())
	ExpectEq(fn.GcsObjectName(), o.Name)
	ExpectEq(dstName, o.Name)
	ExpectFalse(inode.IsSymlink(o))

	// Check resulting contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, dstName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *DirTest) CloneToChildFile_DestinationExists() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	// Create the source.
	src, err := gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte("taco"))
	AssertEq(nil, err)

	// And a destination object that will be overwritten.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dstName, []byte(""))
	AssertEq(nil, err)

	// Call the inode.
	b, fn, o, err := t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(t.bucket.Name(), b.Name())
	ExpectEq(fn.GcsObjectName(), o.Name)
	ExpectEq(dstName, o.Name)
	ExpectFalse(inode.IsSymlink(o))
	ExpectEq(len("taco"), o.Size)

	// Check resulting contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, dstName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *DirTest) CloneToChildFile_TypeCaching() {
	const srcName = "blah/baz"
	dstName := path.Join(dirInodeName, "qux")

	var o *gcs.Object
	var err error

	// Create the source.
	src, err := gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte(""))
	AssertEq(nil, err)

	// Clone to the destination.
	_, _, _, err = t.in.CloneToChildFile(t.ctx, path.Base(dstName), src)
	AssertEq(nil, err)

	// Create a backing object for a directory.
	dirObjName := dstName + "/"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte(""))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the file,
	// because we've cached only seeing the file that's what we should get back.
	result, err := t.in.LookUpChild(t.ctx, path.Base(dstName))
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dstName, o.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, path.Base(dstName))
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
}

func (t *DirTest) CreateChildSymlink_DoesntExist() {
	const name = "qux"
	const target = "taco"
	objName := path.Join(dirInodeName, name)

	// Call the inode.
	b, fn, o, err := t.in.CreateChildSymlink(t.ctx, name, target)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(t.bucket.Name(), b.Name())
	ExpectEq(fn.GcsObjectName(), o.Name)
	ExpectEq(objName, o.Name)
	ExpectEq(target, o.Metadata[inode.SymlinkMetadataKey])
}

func (t *DirTest) CreateChildSymlink_Exists() {
	const name = "qux"
	const target = "taco"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create an existing backing object.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	AssertEq(nil, err)

	// Call the inode.
	_, _, _, err = t.in.CreateChildSymlink(t.ctx, name, target)
	ExpectThat(err, Error(HasSubstr("Precondition")))
	ExpectThat(err, Error(HasSubstr("exists")))
}

func (t *DirTest) CreateChildSymlink_TypeCaching() {
	const name = "qux"
	linkObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create the name.
	_, _, _, err = t.in.CreateChildSymlink(t.ctx, name, "")
	AssertEq(nil, err)

	// Create a backing object for a directory.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	// Look up the name. Even though the directory should shadow the symlink,
	// because we've cached only seeing the symlink that's what we should get
	// back.
	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(linkObjName, o.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	result, err = t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
}

func (t *DirTest) CreateChildDir_DoesntExist() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	// Call the inode.
	b, fn, o, err := t.in.CreateChildDir(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(t.bucket.Name(), b.Name())
	ExpectEq(fn.GcsObjectName(), o.Name)
	ExpectEq(objName, o.Name)
	ExpectFalse(inode.IsSymlink(o))
}

func (t *DirTest) CreateChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create an existing backing object.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	_, _, _, err = t.in.CreateChildDir(t.ctx, name)
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
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode with the wrong generation. No error should be returned.
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation+1, &o.MetaGeneration)
	AssertEq(nil, err)

	// The original generation should still be there.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, objName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *DirTest) DeleteChildFile_WrongMetaGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode with the wrong meta-generation. No error should be
	// returned.
	precond := o.MetaGeneration + 1
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &precond)

	ExpectThat(err, Error(HasSubstr("Precondition")))
	ExpectThat(err, Error(HasSubstr("meta-generation")))

	// The original generation should still be there.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, objName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *DirTest) DeleteChildFile_LatestGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	err = t.in.DeleteChildFile(t.ctx, name, 0, nil)
	AssertEq(nil, err)

	// Check the bucket.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, objName)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *DirTest) DeleteChildFile_ParticularGenerationAndMetaGeneration() {
	const name = "qux"
	objName := path.Join(dirInodeName, name)

	var err error

	// Create a backing object.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	err = t.in.DeleteChildFile(t.ctx, name, o.Generation, &o.MetaGeneration)
	AssertEq(nil, err)

	// Check the bucket.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, objName)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *DirTest) DeleteChildFile_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(dirInodeName, name)
	dirObjName := path.Join(dirInodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create the name, priming the type cache.
	_, _, _, err = t.in.CreateChildFile(t.ctx, name)
	AssertEq(nil, err)

	// Create a backing object for a directory. It should be shadowed by the
	// file.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, []byte("taco"))
	AssertEq(nil, err)

	result, err := t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)
	AssertEq(fileObjName, o.Name)

	// But after deleting the file via the inode, the directory should be
	// revealed.
	err = t.in.DeleteChildFile(t.ctx, name, 0, nil)
	AssertEq(nil, err)

	result, err = t.in.LookUpChild(t.ctx, name)
	o = result.Object

	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
}

func (t *DirTest) DeleteChildDir_DoesntExist() {
	const name = "qux"

	err := t.in.DeleteChildDir(t.ctx, name)
	ExpectEq(nil, err)
}

func (t *DirTest) DeleteChildDir_Exists() {
	const name = "qux"
	objName := path.Join(dirInodeName, name) + "/"

	var err error

	// Create a backing object.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, objName, []byte("taco"))
	AssertEq(nil, err)

	// Call the inode.
	err = t.in.DeleteChildDir(t.ctx, name)
	AssertEq(nil, err)

	// Check the bucket.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, objName)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}
