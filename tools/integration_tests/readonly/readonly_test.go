// Copyright 2021 Google Inc. All Rights Reserved.
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

// Provides integration tests when implicit_dir flag is set.
package readonly_test

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func executeTest(m *testing.M) (successCode int) {
	// Creating a temporary directory to store files
	// to be used for testing.
	//var err error
	//setup.TmpDir, err = os.MkdirTemp(setup.MntDir, "tmpDir")
	//if err != nil {
	//	setup.LogAndExit(fmt.Sprintf("Mkdir at %q: %v", setup.MntDir, err))
	//}

	successCode = m.Run()

	os.RemoveAll(setup.MntDir)

	return successCode
}

func executeTestForFlags(flags []string, m *testing.M) (successCode int) {
	var err error
	for i := 0; i < len(flags); i++ {
		if err = setup.MountGcsfuse(flags[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}

		successCode = executeTest(m)

		err = setup.UnMount()
		if err != nil {
			setup.LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
		}

		// Print flag on which test fails
		if successCode != 0 {
			log.Print("Test Fails on " + flags[i])
			return
		}

	}
	return
}

func TestMain(m *testing.M) {
	flag.Parse()

	if *setup.TestBucket == "" && *setup.MountedDirectory == "" {
		log.Printf("--testbucket or --mountedDirectory must be specified")
		os.Exit(0)
	} else if *setup.TestBucket != "" && *setup.MountedDirectory != "" {
		log.Printf("Both --testbucket and --mountedDirectory can't be specified at the same time.")
		os.Exit(0)
	}

	if *setup.MountedDirectory != "" {
		setup.MntDir = *setup.MountedDirectory
		successCode := executeTest(m)
		os.Exit(successCode)
	}

	if err := setup.SetUpTestDir(); err != nil {
		log.Printf("setUpTestDir: %v\n", err)
		os.Exit(1)
	}

	flags := []string{"--enable-storage-client-library", "--o=ro", "--implicit-dirs=true"}

	successCode := executeTestForFlags(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile)
	os.Exit(successCode)
}
