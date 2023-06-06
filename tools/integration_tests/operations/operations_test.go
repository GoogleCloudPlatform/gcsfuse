// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provides integration tests for file and directory operations.
package operations_test

import (
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const MoveFile = "move.txt"
const MoveFileContent = "This is from move file in Test directory.\n"
const SrcCopyDirectory = "srcCopyDir"
const SubSrcCopyDirectory = "subSrcCopyDir"
const SrcCopyFile = "copy.txt"
const SrcCopyFileContent = "This is from copy file in srcCopy directory.\n"
const DestCopyDirectory = "destCopyDir"
const DestNonEmptyCopyDirectory = "destNonEmptyCopyDirectory"
const SubDirInNonEmptyDestCopyDirectory = "subDestCopyDir"
const DestCopyDirectoryNotExist = "notExist"
const NumberOfObjectsInSrcCopyDirectory = 2
const NumberOfObjectsInDestCopyDirectory = 2

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := [][]string{{"--enable-storage-client-library=true", "--implicit-dirs=true"},
		{"--enable-storage-client-library=false"},
		{"--implicit-dirs=true"},
		{"--implicit-dirs=false"}}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() != "" && setup.MountedDirectory() != "" {
		log.Print("Both --testbucket and --mountedDirectory can't be specified at the same time.")
		os.Exit(1)
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTests(flags, m)
	}

	os.Exit(successCode)
}
