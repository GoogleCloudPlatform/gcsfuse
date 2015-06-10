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

	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ReadOnlyTest struct {
	fsTest
}

func init() { ogletest.RegisterTestSuite(&ReadOnlyTest{}) }

func (s *ReadOnlyTest) SetUp(t *ogletest.T) {
	s.mountCfg.ReadOnly = true
	s.fsTest.SetUp(t)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *ReadOnlyTest) CreateFile(t *ogletest.T) {
	err := ioutil.WriteFile(path.Join(s.Dir, "foo"), []byte{}, 0700)
	t.ExpectThat(err, Error(HasSubstr("read-only")))
}

func (s *ReadOnlyTest) ModifyFile(t *ogletest.T) {
	// Create an object in the bucket.
	_, err := gcsutil.CreateObject(t.Ctx, s.bucket, "foo", "taco")
	t.AssertEq(nil, err)

	// Opening it for writing should fail.
	f, err := os.OpenFile(path.Join(s.Dir, "foo"), os.O_RDWR, 0)
	f.Close()

	t.ExpectThat(err, Error(HasSubstr("read-only")))
}

func (s *ReadOnlyTest) DeleteFile(t *ogletest.T) {
	// Create an object in the bucket.
	_, err := gcsutil.CreateObject(t.Ctx, s.bucket, "foo", "taco")
	t.AssertEq(nil, err)

	// Attempt to delete it via the file system.
	err = os.Remove(path.Join(s.Dir, "foo"))
	t.ExpectThat(err, Error(HasSubstr("read-only")))

	// The bucket should not have been modified.
	contents, err := gcsutil.ReadObject(t.Ctx, s.bucket, "foo")

	t.AssertEq(nil, err)
	t.ExpectEq("taco", string(contents))
}
