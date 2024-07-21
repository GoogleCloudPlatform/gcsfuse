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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/jacobsa/timeutil"
)

func HNSTestDir(testSuite *testing.T) { suite.Run(testSuite, new(HNSDirTest)) }

type HNSDirTest struct {
	suite.Suite
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock

	in DirInode
	tc metadata.TypeCache
}

func (t *DirTest) SetupTest() {
	//t.ctx = ti.Ctx
	//t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	//bucket := fake.NewFakeBucket(&t.clock, "some_bucket")
	t.bucket = gcsx.NewSyncerBucket(
		1,
		".gcsfuse_tmp/",
		new(storage.TestifyMockBucket))
	// Create the inode. No implicit dirs by default.
	t.resetInode(false, false, true)
}

func (t *DirTest) TearDownTest() {
	t.in.Unlock()
}
