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

package implicitdir

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const ExplicitDirectory = "explicitDirectory"
const ExplicitFile = "explicitFile"
const ImplicitDirectory = "implicitDirectory"
const ImplicitSubDirectory = "implicitSubDirectory"
const NumberOfExplicitObjects = 2
const NumberOfTotalObjects = 3
const NumberOfFilesInExplicitDirectory = 2
const NumberOfFilesInImplicitDirectory = 2
const NumberOfFilesInImplicitSubDirectory = 1
const PrefixFileInExplicitDirectory = "fileInExplicitDir"
const FirstFileInExplicitDirectory = "fileInExplicitDir1"
const SecondFileInExplicitDirectory = "fileInExplicitDir2"
const FileInImplicitDirectory = "fileInImplicitDir1"
const FileInImplicitSubDirectory = "fileInImplicitDir2"

func RunTestsForImplicitDir(flags [][]string, m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Clean the bucket.
	setup.RunScriptForTestData("../testdata/delete_objects.sh", setup.TestBucket())

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	os.Exit(successCode)
}

func CreateImplicitDirectory() {
	// Implicit Directory Structure
	// testBucket/implicitDirectory                                                  -- Dir
	// testBucket/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File

	// Clean the bucket.
	setup.RunScriptForTestData("../testdata/delete_objects.sh", setup.TestBucket())

	// Create implicit directory in bucket for testing.
	setup.RunScriptForTestData("../testdata/create_objects.sh", setup.TestBucket())
}

func CreateExplicitDirectory(t *testing.T) {
	// Explicit Directory structure
	// testBucket/explicitDirectory                            -- Dir
	// testBucket/explictFile                                  -- File
	// testBucket/explicitDirectory/fileInExplicitDir1         -- File
	// testBucket/explicitDirectory/fileInExplicitDir2         -- File

	dirPath := path.Join(setup.MntDir(), ExplicitDirectory)
	dir, err := os.Stat(dirPath)
	if err == nil {
		log.Println(dir.Name())
	}
	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirectory, dirPath, PrefixFileInExplicitDirectory, t)
	filePath := path.Join(setup.MntDir(), ExplicitFile)
	_, err = os.Create(filePath)
	if err != nil {
		t.Errorf("Create file at %q: %v", setup.MntDir(), err)
	}
}
