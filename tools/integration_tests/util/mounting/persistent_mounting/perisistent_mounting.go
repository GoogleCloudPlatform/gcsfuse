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

package persistent_mounting

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

// Change e.g --log_severity=trace to log_severity=trace
func makePersistentMountingArgs(flags []string) (args []string) {
	var s string
	for i := range flags {
		// We are already passing flags with -o flag.
		s = strings.Replace(flags[i], "--o=", "", -1)
		// e.g. Convert --log-severity=trace to __log_severity=trace
		s = strings.Replace(s, "-", "_", -1)
		// e.g. Convert __log_severity=trace to log_severity=trace
		s = strings.Replace(s, "__", "", -1)
		// e.g. Revert _1 to -1
		s = strings.Replace(s, "_1", "-1", -1)
		args = append(args, s)
	}
	return
}

func mountGcsfuseWithPersistentMountingWithConfigFile(config *test_suite.TestConfig, flags []string) (err error) {
	defaultArg := []string{config.TestBucket,
		setup.MntDir(),
		"-o",
		"log_severity=trace",
		"-o",
		"log_file=" + setup.LogFile(),
	}

	persistentMountingArgs := makePersistentMountingArgs(flags)

	for i := 0; i < len(persistentMountingArgs); i++ {
		// e.g. -o flag1, -o flag2, ...
		defaultArg = append(defaultArg, "-o", persistentMountingArgs[i])
	}

	err = mounting.MountGcsfuse(setup.SbinFile(), defaultArg)

	return err
}

func executeTestsForPersistentMountingWithConfigFile(config *test_suite.TestConfig, flagsSet [][]string, m *testing.M) (successCode int) {
	var err error

	for i := 0; i < len(flagsSet); i++ {
		if err = mountGcsfuseWithPersistentMountingWithConfigFile(config, flagsSet[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		log.Printf("Running persistent mounting tests with flags: %s", flagsSet[i])
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
		TestBucket:       setup.TestBucket(),
		MountedDirectory: setup.MountedDirectory(),
		LogFile:          setup.LogFile(),
	}
	return RunTestsWithConfigFile(config, flagsSet, m)
}

func RunTestsWithConfigFile(config *test_suite.TestConfig, flagsSet [][]string, m *testing.M) (successCode int) {
	log.Println("Running persistent mounting tests...")
	successCode = executeTestsForPersistentMountingWithConfigFile(config, flagsSet, m)
	log.Printf("Test log: %s\n", setup.LogFile())
	return successCode
}
