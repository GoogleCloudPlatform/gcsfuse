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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestReadOnlyTestSuite(t *testing.T) {
	suite.Run(t, new(ReadOnlyTest))
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ReadOnlyTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *ReadOnlyTest) SetupSuite() {
	t.mountCfg.ReadOnly = true
	t.fsTest.SetupSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ReadOnlyTest) TestCreateFile() {
	err := ioutil.WriteFile(path.Join(mntDir, "foo"), []byte{}, 0700)
	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *ReadOnlyTest) TestModifyFile() {
	// Create an object in the bucket.
	_, err := storageutil.CreateObject(ctx, bucket, "foo", []byte("taco"))
	assert.Nil(t.T(), err)

	// Opening it for writing should fail.
	f, err := os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	f.Close()

	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *ReadOnlyTest) TestDeleteFile() {
	// Create an object in the bucket.
	_, err := storageutil.CreateObject(ctx, bucket, "foo", []byte("taco"))
	assert.Nil(t.T(), err)

	// Attempt to delete it via the file system.
	err = os.Remove(path.Join(mntDir, "foo"))
	ExpectThat(err, Error(HasSubstr("read-only")))

	// the bucket should not have been modified.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")

	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))
}
