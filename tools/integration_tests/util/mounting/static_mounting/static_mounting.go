//Copyright 2023 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

// TODO(b/438068132): cleanup deprecated methods after migration is complete.
func MountGcsfuseWithStaticMounting(flags []string) (err error) {
	config := &test_suite.TestConfig{
		TestBucket:              setup.TestBucket(),
		GKEMountedDirectory:     setup.MountedDirectory(),
		GCSFuseMountedDirectory: setup.MntDir(),
		LogFile:                 setup.LogFile(),
	}
	return MountGcsfuseWithStaticMountingWithConfigFile(config, flags)
}

func MountGcsfuseWithStaticMountingWithConfigFile(config *test_suite.TestConfig, flags []string) (err error) {
	var defaultArg []string
	if setup.TestOnTPCEndPoint() {
		defaultArg = append(defaultArg,
			"--key-file=/tmp/sa.key.json")
	}

	defaultArg = append(defaultArg, "--log-severity=trace",
		"--log-file="+config.LogFile,
		config.TestBucket,
		config.GCSFuseMountedDirectory)

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	err = mounting.MountGcsfuse(setup.BinFile(), flags)

	return err
}

func executeTestsForStaticMounting(config *test_suite.TestConfig, flagsSet [][]string, m *testing.M) (successCode int) {
	var err error

	for i := range flagsSet {
		if err = MountGcsfuseWithStaticMountingWithConfigFile(config, flagsSet[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		log.Printf("Running static mounting tests with flags: %s", flagsSet[i])
		successCode = setup.ExecuteTestForFlagsSet(flagsSet[i], m)
		if successCode != 0 {
			return
		}
	}
	return
}

// Deprecated: Use RunTestsWithConfigFile instead.
// TODO(b/438068132): cleanup deprecated methods after migration is complete.
func RunTests(flagsSet [][]string, m *testing.M) (successCode int) {
	config := &test_suite.TestConfig{
		TestBucket:              setup.TestBucket(),
		GKEMountedDirectory:     setup.MountedDirectory(),
		GCSFuseMountedDirectory: setup.MntDir(),
		LogFile:                 setup.LogFile(),
	}
	return RunTestsWithConfigFile(config, flagsSet, m)
}

func RunTestsWithConfigFile(config *test_suite.TestConfig, flagsSet [][]string, m *testing.M) (successCode int) {
	log.Println("Running static mounting tests...")
	successCode = executeTestsForStaticMounting(config, flagsSet, m)
	log.Printf("Test log: %s\n", config.LogFile)
	return successCode
}
