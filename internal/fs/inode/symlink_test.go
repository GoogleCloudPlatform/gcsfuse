// Copyright 2021 Google Inc. All Rights Reserved.
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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
)

func TestSymlink(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type SymlinkTest struct {
}

var _ SetUpInterface = &CoreTest{}
var _ TearDownInterface = &CoreTest{}

func init() { RegisterTestSuite(&SymlinkTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SymlinkTest) TestIsSymLinkWhenMetadataKeyIsPresent() {
	metadata := map[string]string{
		inode.SymlinkMetadataKey: "target",
	}
	o := gcs.Object{
		Name:     "test",
		Metadata: metadata,
	}

	AssertEq(true, inode.IsSymlink(&o))
}

func (t *SymlinkTest) TestIsSymLinkWhenMetadataKeyIsNotPresent() {
	o := gcs.Object{
		Name: "test",
	}

	AssertEq(false, inode.IsSymlink(&o))
}

func (t *SymlinkTest) TestIsSymLinkForNilObject() {
	AssertEq(false, inode.IsSymlink(nil))
}
