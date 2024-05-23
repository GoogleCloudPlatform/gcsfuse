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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func MountGcsfuseWithStaticMounting(flags []string) (err error) {
	var defaultArg []string
	if setup.TestOnTPCEndPoint() {
		defaultArg = append(defaultArg, "--custom-endpoint=storage.apis-tpczero.goog:443",
			"--key-file=/home/tulsishah_google_com/kay.json")
	}

	defaultArg = append(defaultArg, "--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-file="+setup.LogFile(),
		"--log-format=text",
		setup.TestBucket(),
		setup.MntDir())

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	err = mounting.MountGcsfuse(setup.BinFile(), flags)

	return err
}

func executeTestsForStaticMounting(flagsSet [][]string, m *testing.M) (successCode int) {
	var err error

	for i := 0; i < len(flagsSet); i++ {
		if err = MountGcsfuseWithStaticMounting(flagsSet[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		log.Printf("Running static mounting tests with flags: %s", flagsSet[i])
		successCode = setup.ExecuteTestForFlagsSet(flagsSet[i], m)
	}
	return
}

func RunTests(flagsSet [][]string, m *testing.M) (successCode int) {
	log.Println("Running static mounting tests...")

	successCode = executeTestsForStaticMounting(flagsSet, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	return successCode
}
