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
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		return
	}

	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())

	for _, flags := range flagsSet {
		tcName := strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		t.Run(tcName, func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer func() {
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				setup.UnmountGCSFuseWithConfig(&testEnv.cfg)
			}()
		})
	}
}

func TestMountFails(t *testing.T) {
	// This test is not applicable for mounted directory testing.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		t.Fatalf("This test is not valid for mounted-directory tests.")
	}

	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())

	for _, flags := range flagsSet {
		tcName := strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		t.Run(tcName, func(t *testing.T) {
			err := mayMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer func() {
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				if err == nil {
					setup.UnmountGCSFuseWithConfig(&testEnv.cfg)
				}
			}()

			assert.Error(t, err)
		})
	}
}
