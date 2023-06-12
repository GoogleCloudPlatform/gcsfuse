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

// Provide test for listing large directory
package list_large_dir_test

import (
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const DirectoryWithTwelveThousandFiles = "directoryWithTwelveThousandFiles"
const PrefixFileInDirectoryWithTwelveThousandFiles = "fileInDirectoryWithTwelveThousandFiles"
const DirectoryWithTwelveThousandFilesAndHundredExplicitDir = "directoryWithTwelveThousandFilesAndHundredExplicitDir"
const DirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = "directoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir"
const PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDir = "fileInDirectoryWithTwelveThousandFilesAndHundredExplicitDir"
const NumberOfFilesInDirectoryWithTwelveThousandFiles = 12000
const NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDir = 12000
const NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = 12000
const NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir = 100
const NumberOfImplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = 100
const PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = "fileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir"
const NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = 100
const ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = "explicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir"
const ImplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = "implicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir"
const ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDir = "explicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDir"

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// --o=ro flag is for stopping the deletion of objects before and after testing.
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/integration_tests/util/setup/setup.go#L172
	flags := [][]string{{"--implicit-dirs", "--o=ro"}}

	if setup.TestBucket() != "" && setup.MountedDirectory() != "" {
		log.Printf("Both --testbucket and --mountedDirectory can't be specified at the same time.")
		os.Exit(1)
	}

	testBucket := setup.TestBucket()

	setup.SetTestBucket("integration-test-data-gcsfuse")

	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	// Setting back the original bucket pass through flag.
	setup.SetTestBucket(testBucket)

	os.Exit(successCode)
}
