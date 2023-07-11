//Copyright 2023 Google Inc. All Rights Reserved.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package persistent_mounting

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// make e.g --enable-storage-client-library in enable_storage_client_library
func makePersistentMountingArgs(flags []string) (args []string, err error) {
	var s string
	for i := range flags {
		// We are already passing flags with -o flag.
		s = strings.Replace(flags[i], "--o=", "", -1)
		// e.g. Convert --enable-storage-client-library to __enable_storage_client_library
		s = strings.Replace(flags[i], "-", "_", -1)
		// e.g. Convert __enable_storage_client_library to enable_storage_client_library
		s = strings.Replace(s, "__", "", -1)
		args = append(args, s)
	}
	return
}

func mountGcsfuseWithStaticMounting(flags []string) (err error) {
	defaultArg := []string{setup.TestBucket(),
		setup.MntDir(),
		"-o",
		"debug_gcs",
		"-o",
		"debug_fs",
		"-o",
		"debug_fuse",
		"-o",
		"log_file=" + setup.LogFile(),
		"-o",
		"log_format=text",
	}

	persistentMountingArgs, err := makePersistentMountingArgs(flags)
	if err != nil {
		setup.LogAndExit("Error in converting flags for persistent mounting.")
	}

	for i := 0; i < len(persistentMountingArgs); i++ {
		// e.g. -o flag1, -o flag2, ...
		defaultArg = append(defaultArg, "-o", persistentMountingArgs[i])
	}

	err = mounting.MountGcsfuse(setup.SbinFile(), defaultArg)

	return err
}

func executeTestsForStatingMounting(flags [][]string, m *testing.M) (successCode int) {
	var err error

	for i := 0; i < len(flags); i++ {
		if err = mountGcsfuseWithStaticMounting(flags[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		setup.ExecuteTestForFlagsSet(flags[i], m)
	}
	return
}

func RunTests(flags [][]string, m *testing.M) (successCode int) {
	successCode = executeTestsForStatingMounting(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	return successCode
}
