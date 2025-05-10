// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stale_handle

import (
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func TestRenamedFileSyncAndCloseThrowsStaleFileHandleError(t *testing.T) {
	testCases := []struct {
		name     string
		flags    []string
		fileSize int64
	}{
		{
			name:  "WithGRPC",
			flags: []string{"--metadata-cache-ttl-secs=0", "--client-protocol=grpc"},
		},
		{
			name:  "WithoutGRPC",
			flags: []string{"--metadata-cache-ttl-secs=0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setup.MountGCSFuseWithGivenMountFunc(tc.flags, mountFunc)
			defer setup.SaveGCSFuseLogFileInCaseOfFailure(t)
			defer setup.UnmountGCSFuse(rootDir)
			testDirPath := setup.SetupTestDirectory(t.Name())
			// Create an empty object on GCS.
			err := CreateObjectOnGCS(ctx, storageClient, path.Join(t.Name(), FileName1), "")
			assert.NoError(t, err)
			fh := operations.OpenFileWithODirect(t, path.Join(testDirPath, FileName1))
			data := setup.GenerateRandomString(5 * util.MiB)
			_, err = fh.WriteString(data)
			assert.NoError(t, err)
			err = operations.RenameFile(fh.Name(), path.Join(setup.MntDir(), t.Name(), FileName2))
			assert.NoError(t, err)
			// Attempt to write to file should not give any error.
			_, err = fh.WriteString(data)
			assert.NoError(t, err)

			err = fh.Sync()

			operations.ValidateESTALEError(t, err)
			err = fh.Close()
			operations.ValidateESTALEError(t, err)
		})
	}
}
