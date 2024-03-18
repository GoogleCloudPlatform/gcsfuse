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

package fs_test

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ReadOnlyTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&ReadOnlyTest{})
}

func (t *ReadOnlyTest) SetUpTestSuite() {
	t.mountCfg.ReadOnly = true
	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ReadOnlyTest) CreateFile() {
	err := ioutil.WriteFile(path.Join(mntDir, "foo"), []byte{}, 0700)
	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *ReadOnlyTest) ModifyFile() {
	// Create an object in the bucket.
	_, err := storageutil.CreateObject(ctx, bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	// Opening it for writing should fail.
	f, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	f.Close()

	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *ReadOnlyTest) DeleteFile() {
	// Create an object in the bucket.
	_, err := storageutil.CreateObject(ctx, bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	// Attempt to delete it via the file system.
	err = os.Remove(path.Join(mntDir, "foo"))
	ExpectThat(err, Error(HasSubstr("read-only")))

	// the bucket should not have been modified.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}
