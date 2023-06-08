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
const NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDir = 1200
const NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = 1200
const NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir = 100
const NumberOfImplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = 100
const PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = "fileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir"
const PrefixFileInSubDirectoryWithTwelveThousandFilesAndHundredExplicitDir = "fileInSubDirectoryWithTwelveThousandFilesAndHundredExplicitDir"
const PrefixFileInSubDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = "explicitFileInSubDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir"
const NumberOfObjectsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir = 12100
const NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir = 100

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := [][]string{{"--implicit-dirs"}}

	if setup.TestBucket() != "" && setup.MountedDirectory() != "" {
		log.Printf("Both --testbucket and --mountedDirectory can't be specified at the same time.")
		os.Exit(1)
	}

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	//if successCode == 0 {
	//	successCode = only_dir_mounting.RunTests(flags, m)
	//}

	os.Exit(successCode)
}
