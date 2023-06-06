// Copyright 2023 Google Inc. All Rights Reserved.
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
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const DirectoryInTestBucket = "Test"

func mountGcsfuseWithOnlyDir(flags []string, dir string) (err error) {
	defaultArg := []string{"--only-dir",
		dir,
		"--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-file=" + setup.LogFile(),
		"--log-format=text",
		setup.TestBucket(),
		setup.MntDir()}

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	err = mounting.MountGcsfuse(flags)

	return err
}

func mountGcsFuseForFlagsAndExecuteTests(flags [][]string, dir string, m *testing.M) (successCode int) {
	for i := 0; i < len(flags); i++ {
		if err := mountGcsfuseWithOnlyDir(flags[i], dir); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		successCode = setup.ExecuteTestForFlagsSet(flags[i], m)
	}
	return
}

func executeTestsForOnlyDirMounting(flags [][]string, m *testing.M) (successCode int) {
	mountDirInBucket := path.Join(setup.TestBucket(), DirectoryInTestBucket)
	// Clean the bucket.

	setup.RunScriptForTestData("../util/mounting/only_dir_mounting/testdata/delete_objects.sh", setup.TestBucket())

	// "Test" directory not exist in bucket.
	mountGcsFuseForFlagsAndExecuteTests(flags, DirectoryInTestBucket, m)

	// "Test" directory exist in bucket.
	// Clean the bucket.
	setup.RunScriptForTestData("../util/mounting/only_dir_mounting/testdata/delete_objects.sh", setup.TestBucket())

	// Create Test directory in bucket.
	setup.RunScriptForTestData("../util/mounting/only_dir_mounting/testdata/create_objects.sh", mountDirInBucket)

	successCode = mountGcsFuseForFlagsAndExecuteTests(flags, DirectoryInTestBucket, m)

	// Clean the bucket after testing.
	setup.RunScriptForTestData("../util/mounting/only_dir_mounting/testdata/delete_objects.sh", setup.TestBucket())

	return
}

func RunTests(flags [][]string, m *testing.M) (successCode int) {
	successCode = executeTestsForOnlyDirMounting(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	return successCode
}
