// Copyright 2023 Google LLC
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
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	TmpDir               = "/tmp"
	OneMiB               = 1024 * 1024
	WritePermission_0200 = 0200
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// write-global-max-blocks=2 is for checking multiple file writes in parallel.
	// concurrent_write_files_test.go- we are writing 3 files in parallel.
	// with this config, we are giving 2 blocks to 2 files and 1 block to other file.
	flags := [][]string{
		{"--enable-streaming-writes=false"},
		{"--enable-streaming-writes=true", "--write-max-blocks-per-file=2", "--write-global-max-blocks=2"}}

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
	os.Exit(successCode)
}
