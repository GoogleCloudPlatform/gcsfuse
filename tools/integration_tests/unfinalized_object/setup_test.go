// Copyright 2025 Google LLC
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

package unfinalized_object

import (
	"log"
	"os"
	"testing"

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	testDirName = "UnfinalizedObjectTest"
)

var (
	mountFunc func([]string) error
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as operations tests validates content from the bucket.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()
	os.Exit(successCode)
}
