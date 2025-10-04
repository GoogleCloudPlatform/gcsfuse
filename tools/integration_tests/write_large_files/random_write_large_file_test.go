// Copyright 2023 Google LLC
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

package write_large_files

import (
	rand2 "math/rand"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	NumberOfRandomWriteCalls = 20
	DirForRandomWrite        = "dirForRandomWrite"
	MaxFileOffset            = 500 * OneMiB
)

func TestWriteLargeFileRandomly(t *testing.T) {
	// Setup Test directory and files to write to.
	randomWriteDir := setup.SetupTestDirectory(DirForRandomWrite)
	mountedDirFilePath := path.Join(randomWriteDir, FiveHundredMBFile)
	localFilePath := path.Join(TmpDir, setup.GenerateRandomString(5))
	t.Cleanup(func() { operations.RemoveFile(localFilePath) })
	// Open local file and mounted directory file for writing.
	filesToWrite := operations.OpenFiles(t, []string{localFilePath, mountedDirFilePath})

	for range NumberOfRandomWriteCalls {
		offset := rand2.Int63n(MaxFileOffset - ChunkSize)
		// Aligning to 4KiB page boundaries is required for O_DIRECT writes to local files.
		offset = offset - offset%(4*util.KiB)
		// Write chunk of random data to both local and mounted directory file.
		err := operations.WriteChunkOfRandomBytesToFiles(filesToWrite, ChunkSize, offset)
		require.NoError(t, err)
	}

	operations.CloseFiles(t, filesToWrite)
	identical, err := operations.AreFilesIdentical(mountedDirFilePath, localFilePath)
	require.NoError(t, err)
	assert.True(t, identical)
}
