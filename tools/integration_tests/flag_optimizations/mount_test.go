// Copyright 2025 Google LLC
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

package flag_optimizations

import (
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// Test Functions
////////////////////////////////////////////////////////////////////////

func TestMountSucceeds(t *testing.T) {
	// Nothing to test for mounted directory, as the mount itself must have succeeded
	// to reach this stage.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		return
	}

	flagsSet := [][]string{}
	for _, profile := range supportedAIMLProfiles {
		flagsSet = append(flagsSet, []string{"--profile=" + profile})
	}
	if len(highEndMachines) > 0 {
		for _, machineType := range highEndMachines {
			flagsSet = append(flagsSet, []string{"--machine-type=" + machineType})
		}
		highEndMachine := "a3-highgpuu-8g"
		for _, profile := range supportedAIMLProfiles {
			flagsSet = append(flagsSet, []string{"--profile=" + profile, "--machine-type=" + highEndMachine})
		}
	}

	for _, flags := range flagsSet {
		tcName := strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		t.Run(tcName, func(t *testing.T) {
			err := testEnv.mountFunc(flags)
			defer func() {
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
			}()

			assert.NoError(t, err)
		})
	}
}

func TestMountFails(t *testing.T) {
	// This test is not applicable for mounted directory testing.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		return
	}

	flagsSet := [][]string{{"--profile=unknown-profile"}}

	for _, flags := range flagsSet {
		tcName := strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		t.Run(tcName, func(t *testing.T) {
			err := testEnv.mountFunc(flags)
			defer func() {
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				if err == nil {
					setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
				}
			}()

			assert.Error(t, err)
		})
	}
}
