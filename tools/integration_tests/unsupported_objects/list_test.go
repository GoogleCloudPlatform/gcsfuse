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

package unsupported_objects

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWithUnsupportedObjects(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForUnsupportedObjectsTests)
	defer setup.CleanUpDir(testDir)

	// Create objects with supported and unsupported names.
	unsupportedObjects := []string{
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects//unsupported_file1.txt",   // Contains "//"
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/./unsupported_file2.txt",  // Contains "/./"
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/../unsupported_file3.txt", // Contains ".."
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/.",                        // Is "."
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/..",                       // Is ".."
		DirForUnsupportedObjectsTests + "//leading_slash.txt",                                 // Starts with "/"
	}
	for _, obj := range unsupportedObjects {
		client.CreateObjectOnGCS(ctx, storageClient, obj, "")
	}
	client.CreateObjectOnGCS(ctx, storageClient, DirForUnsupportedObjectsTests+"/dirWithUnsupportedObjects/supported_file.txt", "content")
	client.CreateObjectOnGCS(ctx, storageClient, DirForUnsupportedObjectsTests+"/dirWithUnsupportedObjects/supported_dir/", "")

	// List the directory containing both supported and unsupported objects.
	entries, err := os.ReadDir(path.Join(testDir, "dirWithUnsupportedObjects"))

	// Verify that listing succeeds and only returns supported objects.
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "supported_dir", entries[0].Name())
	assert.Equal(t, "supported_file.txt", entries[1].Name())
}
