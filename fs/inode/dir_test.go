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
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestDir(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const inodeID = 17
const inodeName = "foo/bar/"
const typeCacheTTL = time.Second

type DirTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.SimulatedClock

	in *inode.DirInode
}

var _ SetUpInterface = &DirTest{}
var _ TearDownInterface = &DirTest{}

func init() { RegisterTestSuite(&DirTest{}) }

func (t *DirTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")

	// Create the inode. No implicit dirs by default.
	t.resetInode(false)
}

func (t *DirTest) TearDown() {
	t.in.Unlock()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *DirTest) resetInode(implicitDirs bool) {
	if t.in != nil {
		t.in.Unlock()
	}

	t.in = inode.NewDirInode(
		inodeID,
		inodeName,
		implicitDirs,
		typeCacheTTL,
		t.bucket,
		&t.clock)

	t.in.Lock()
}

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
			return
		}
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirTest) ID() {
	ExpectEq(inodeID, t.in.ID())
}

func (t *DirTest) Name() {
	ExpectEq(inodeName, t.in.Name())
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
	ExpectEq(os.FileMode(0700)|os.ModeDir, attrs.Mode)
}

func (t *DirTest) LookUpChild_NonExistent() {
	o, err := t.in.LookUpChild(t.ctx, "qux")

	AssertEq(nil, err)
	ExpectEq(nil, o)
}

func (t *DirTest) LookUpChild_FileOnly() {
	const name = "qux"
	objName := path.Join(inodeName, name)

	var o *gcs.Object
	var err error

	// Create a backing object.
	createObj, err := gcsutil.CreateObject(t.ctx, t.bucket, objName, "taco")
	AssertEq(nil, err)

	// Look up with the proper name.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(objName, o.Name)
	ExpectEq(createObj.Generation, o.Generation)
	ExpectEq(createObj.Size, o.Size)

	// A conflict marker name shouldn't work.
	o, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, o)
}

func (t *DirTest) LookUpChild_DirOnly() {
	const name = "qux"
	objName := path.Join(inodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create a backing object.
	createObj, err := gcsutil.CreateObject(t.ctx, t.bucket, objName, "")
	AssertEq(nil, err)

	// Look up with the proper name.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(objName, o.Name)
	ExpectEq(createObj.Generation, o.Generation)
	ExpectEq(createObj.Size, o.Size)

	// A conflict marker name shouldn't work.
	o, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, o)
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Disabled() {
	const name = "qux"

	var o *gcs.Object
	var err error

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(inodeName, name) + "/asdf"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, "")
	AssertEq(nil, err)

	// Looking up the name shouldn't work.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	ExpectEq(nil, o)

	// Ditto with a conflict marker.
	o, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, o)
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Enabled() {
	const name = "qux"
	objName := path.Join(inodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Enable implicit dirs.
	t.resetInode(true)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(objName, "asdf")
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, "")
	AssertEq(nil, err)

	// Looking up the name should work.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(objName, o.Name)
	ExpectEq(0, o.Generation)

	// A conflict marker should not work.
	o, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	ExpectEq(nil, o)
}

func (t *DirTest) LookUpChild_FileAndDir() {
	const name = "qux"
	fileObjName := path.Join(inodeName, name)
	dirObjName := path.Join(inodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create backing objects.
	fileObj, err := gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, "taco")
	AssertEq(nil, err)

	dirObj, err := gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, "")
	AssertEq(nil, err)

	// Look up with the proper name.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
	ExpectEq(dirObj.Generation, o.Generation)
	ExpectEq(dirObj.Size, o.Size)

	// Look up with the conflict marker name.
	o, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)
	ExpectEq(fileObj.Generation, o.Generation)
	ExpectEq(fileObj.Size, o.Size)
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Disabled() {
	const name = "qux"
	fileObjName := path.Join(inodeName, name)
	dirObjName := path.Join(inodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create backing objects.
	fileObj, err := gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, "taco")
	AssertEq(nil, err)

	dirObj, err := gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, "")
	AssertEq(nil, err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(inodeName, name) + "/asdf"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, "")
	AssertEq(nil, err)

	// Look up with the proper name.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
	ExpectEq(dirObj.Generation, o.Generation)
	ExpectEq(dirObj.Size, o.Size)

	// Look up with the conflict marker name.
	o, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)
	ExpectEq(fileObj.Generation, o.Generation)
	ExpectEq(fileObj.Size, o.Size)
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Enabled() {
	const name = "qux"
	fileObjName := path.Join(inodeName, name)
	dirObjName := path.Join(inodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Enable implicit dirs.
	t.resetInode(true)

	// Create backing objects.
	fileObj, err := gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, "taco")
	AssertEq(nil, err)

	dirObj, err := gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, "")
	AssertEq(nil, err)

	// Create an object that implicitly defines the directory.
	otherObjName := path.Join(inodeName, name) + "/asdf"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, otherObjName, "")
	AssertEq(nil, err)

	// Look up with the proper name.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(dirObjName, o.Name)
	ExpectEq(dirObj.Generation, o.Generation)
	ExpectEq(dirObj.Size, o.Size)

	// Look up with the conflict marker name.
	o, err = t.in.LookUpChild(t.ctx, name+inode.ConflictingFileNameSuffix)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)
	ExpectEq(fileObj.Generation, o.Generation)
	ExpectEq(fileObj.Size, o.Size)
}

func (t *DirTest) LookUpChild_TypeCaching() {
	const name = "qux"
	fileObjName := path.Join(inodeName, name)
	dirObjName := path.Join(inodeName, name) + "/"

	var o *gcs.Object
	var err error

	// Create a backing object for a file.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, fileObjName, "taco")
	AssertEq(nil, err)

	// Look up; we should get the file.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)

	// Create a backing object for a directory.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, dirObjName, "taco")
	AssertEq(nil, err)

	// Look up again. Even though the directory should shadow the file, because
	// we've cached only seeing the file that's what we should get back.
	o, err = t.in.LookUpChild(t.ctx, name)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq(fileObjName, o.Name)

	// But after the TTL expires, the behavior should flip.
	t.clock.AdvanceTime(typeCacheTTL + time.Millisecond)

	o, err = t.in.LookUpChild(t.ctx, name)
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
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_NonEmpty_ImplicitDirsEnabled() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_LotsOfEntries() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildFile_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildFile_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildFile_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildDir_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildDir_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildDir_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildFile_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildFile_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildFile_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildDir_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildDir_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildDir_TypeCaching() {
	AssertTrue(false, "TODO")
}
