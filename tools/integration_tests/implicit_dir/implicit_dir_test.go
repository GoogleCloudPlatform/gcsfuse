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

// Provide tests when implicit directory present and mounted bucket with --implicit-dir flag.
package implicit_dir_test

import (
	"context"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

const ExplicitDirInImplicitDir = "explicitDirInImplicitDir"
const ExplicitDirInImplicitSubDir = "explicitDirInImplicitSubDir"
const PrefixFileInExplicitDirInImplicitDir = "fileInExplicitDirInImplicitDir"
const PrefixFileInExplicitDirInImplicitSubDir = "fileInExplicitDirInImplicitSubDir"
const NumberOfFilesInExplicitDirInImplicitSubDir = 1
const NumberOfFilesInExplicitDirInImplicitDir = 1
const DirForImplicitDirTests = "dirForImplicitDirTests"

var (
	storageClient *storage.Client
	ctx           context.Context
)

func createMountConfigsAndEquivalentFlags() (flags [][]string) {
	mountConfig4 := config.MountConfig{
		EnableHNS: true,
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}
	filePath4 := setup.YAMLConfigFile(mountConfig4, "config4.yaml")
	flags = append(flags, []string{"--implicit-dirs", "--config-file=" + filePath4})

	return flags
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Create storage client before running tests.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithTimeOut(&ctx, &storageClient, time.Minute*15)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	flagsSet := [][]string{{"--implicit-dirs"}}

	mountConfigFlags := createMountConfigsAndEquivalentFlags()
	flagsSet = append(flagsSet, mountConfigFlags...)

	if !testing.Short() {
		flagsSet = append(flagsSet, []string{"--client-protocol=grpc", "--implicit-dirs=true"})
	}

	successCode := implicit_and_explicit_dir_setup.RunTestsForImplicitDirAndExplicitDir(flagsSet, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
