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

package requester_pays_bucket

import (
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Test Functions
////////////////////////////////////////////////////////////////////////

func TestMount(t *testing.T) {
	// Nothing to test for mounted directory, as the mount itself must have succeeded
	// to reach this stage.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		return
	}

	flagsSet := [][]string{{"--billing-project=gcs-fuse-test"}}
	for _, flags := range flagsSet {
		tcName := strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		t.Run(tcName, func(t *testing.T) {
			err := testEnv.mountFunc(flags)
			defer func() {
				setup.SaveGCSFuseLogFileInCaseOfFailure(t)
				setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
			}()

			require.NoError(t, err)
		})
	}
}
