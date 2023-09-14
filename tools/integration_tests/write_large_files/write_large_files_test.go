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
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	TmpDir               = "/tmp"
	OneMiB               = 1024 * 1024
	WritePermission_0200 = 0200
)

func compareFileFromGCSBucketAndMntDir(gcsFile, mntDirFile, localFilePathToDownloadGcsFile string) (err error) {
	err = operations.DownloadGcsObject(gcsFile, localFilePathToDownloadGcsFile)
	if err != nil {
		err = fmt.Errorf("Error in downloading object:%w", err)
		return err
	}

	// Remove file after testing.
	defer operations.RemoveFile(localFilePathToDownloadGcsFile)

	// DiffFiles loads the entire files into memory. These are both 500 MiB files, hence would have a 1 GiB
	// requirement just for this step
	diff, err := operations.DiffFiles(mntDirFile, localFilePathToDownloadGcsFile)
	if diff != 0 {
		err = fmt.Errorf("Download of GCS object %s didn't match the Mounted local file (%s): %v", localFilePathToDownloadGcsFile, mntDirFile, err)
		return err
	}
	return err
}

// Write data of chunkSize in file at given offset.
func WriteChunkSizeInFile(file *os.File, chunkSize int, offset int64) (err error) {
	chunk := make([]byte, chunkSize)
	_, err = rand.Read(chunk)
	if err != nil {
		err = fmt.Errorf("error while generating random string: %s", err)
		return err
	}

	// Write data in the file.
	n, err := file.WriteAt(chunk, offset)
	if err != nil {
		err = fmt.Errorf("Error in writing randomly in file:%v", err)
		return err
	}
	if n != chunkSize {
		err = fmt.Errorf("Incorrect number of bytes written in the file actual %d, expected %d", n, chunkSize)
		return err
	}

	err = file.Sync()
	if err != nil {
		err = fmt.Errorf("Error in syncing file:%v", err)
		return err
	}

	return err
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
