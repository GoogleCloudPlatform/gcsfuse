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

	"github.com/stretchr/testify/assert"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

func tearDownMountTest(t *testing.T, err error) {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t)
	if err == nil {
		setup.UnmountGCSFuseWithConfig(&testEnv.cfg)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Functions
////////////////////////////////////////////////////////////////////////

func TestMountFails(t *testing.T) {
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		t.Fatalf("This test is not valid for mounted-directory tests.")
	}

	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())

	for _, flags := range flagsSet {
		tcName := strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		t.Run(tcName, func(t *testing.T) {
			// Arrange and Act
			err := mountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer tearDownMountTest(t, err)

			// Assert
			assert.Error(t, err)
		})
	}
}
