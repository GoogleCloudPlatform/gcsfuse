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

package static_mounting

import (
	"fmt"
	"log"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func mountGcsfuseWithStaticMounting(flags []string) (err error) {
	defaultArg := []string{"--debug_gcs",
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
