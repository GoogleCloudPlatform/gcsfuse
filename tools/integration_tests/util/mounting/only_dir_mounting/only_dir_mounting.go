// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package only_dir_mounting

import (
	"fmt"
	"log"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"golang.org/x/net/context"
)

func MountGcsfuseWithOnlyDir(flags []string) (err error) {
	defaultArg := []string{"--only-dir",
		setup.OnlyDirMounted(),
		"--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-file=" + setup.LogFile(),
		setup.TestBucket(),
		setup.MntDir()}

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	err = mounting.MountGcsfuse(setup.BinFile(), flags)

	return err
}

func mountGcsFuseForFlagsAndExecuteTests(flags [][]string, m *testing.M) (successCode int) {
	for i := 0; i < len(flags); i++ {
		if err := MountGcsfuseWithOnlyDir(flags[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		log.Printf("Running only dir mounting tests with flags: %s", flags[i])
		successCode = setup.ExecuteTestForFlagsSet(flags[i], m)
		if successCode != 0 {
			return
		}
	}
	return
}

func executeTestsForOnlyDirMounting(flags [][]string, dirName string, m *testing.M) (successCode int) {
	ctx := context.Background()
	storageClient, err := client.CreateStorageClient(ctx)
	if err != nil {
		log.Fatalf("Error creating storage client: %v\n", err)
	}

	defer storageClient.Close()

	// Set onlyDirMounted value to the directory being mounted.
	setup.SetOnlyDirMounted(dirName)

	// Clean the bucket.
	// Test scenario when only-dir-mounted directory does not pre-exist in bucket.
	err = client.DeleteAllObjectsWithPrefix(ctx, storageClient, dirName)
	if err != nil {
		log.Println("Error deleting object on GCS: %v", err)
	}
	successCode = mountGcsFuseForFlagsAndExecuteTests(flags, m)
	if successCode != 0 {
		return
	}

	// Test scenario when only-dir-mounted directory pre-exists in bucket.
	client.SetupTestDirectory(ctx, storageClient, dirName)

	successCode = mountGcsFuseForFlagsAndExecuteTests(flags, m)
	err = client.DeleteAllObjectsWithPrefix(ctx, storageClient, dirName)
	if err != nil {
		log.Println("Error deleting object on GCS: %v", err)
	}

	// Reset onlyDirMounted value to empty string after only dir mount tests are done.
	setup.SetOnlyDirMounted("")
	return
}

func RunTests(flags [][]string, dirName string, m *testing.M) (successCode int) {
	log.Println("Running only dir mounting tests...")

	successCode = executeTestsForOnlyDirMounting(flags, dirName, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	return successCode
}
