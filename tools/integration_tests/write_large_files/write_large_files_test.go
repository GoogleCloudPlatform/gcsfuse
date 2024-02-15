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

// Provides integration tests for write large files sequentially and randomly.
package write_large_files

import (
	"fmt"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"io"
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	TmpDir               = "/tmp"
	OneMiB               = 1024 * 1024
	WritePermission_0200 = 0200
)

func compareFileFromGCSBucketAndMntDir(gcsFile, mntDirFile, localFilePathToDownloadGcsFile string, fileSize int64, t *testing.T) error {
	var w io.Writer
	err := client.DownloadFile(w, setup.TestBucket(),gcsFile,localFilePathToDownloadGcsFile)
	if err != nil{
		t.Errorf("Error in downloading file: %v",err)
	}

	// Remove file after testing.
	defer operations.RemoveFile(localFilePathToDownloadGcsFile)

	identical, err := operations.AreFilesIdentical(mntDirFile, localFilePathToDownloadGcsFile)
	if !identical {
		return fmt.Errorf("Download of GCS object %s didn't match the Mounted local file (%s): %v", localFilePathToDownloadGcsFile, mntDirFile, err)
	}

	return nil


	//expectedContent, err := client.ReadLargeFileFromGCS(gcsFile, localFilePathToDownloadGcsFile,fileSize,t)
	//if err != nil {
	//	return fmt.Errorf("Error in downloading object: %v", err)
	//}
	//
	//gotContent , err:= operations.ReadFile(mntDirFile)
	//if err != nil{
	//	t.Errorf("Error in reading file: %v", err)
	//}
	//
	//fmt.Println("Size: ",len(gotContent))
	//if bytes.Compare(expectedContent,gotContent) != 0{
	//	t.Fatalf("GCS file %s content mismatch", gcsFile)
	//}

	return nil
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := [][]string{{"--implicit-dirs"}}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	setup.RemoveBinFileCopiedForTesting()

	os.Exit(successCode)
}
