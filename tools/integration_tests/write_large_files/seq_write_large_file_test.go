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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	FiveHundredMB  = 500 * OneMiB
	ChunkSize      = 20 * OneMiB
	DirForSeqWrite = "dirForSeqWrite"
)

var FiveHundredMBFile = "fiveHundredMBFile" + setup.GenerateRandomString(5) + ".txt"

func TestWriteLargeFileSequentially(t *testing.T) {
	seqWriteDir := setup.SetupTestDirectory(DirForSeqWrite)
	mountedDirFilePath := path.Join(seqWriteDir, FiveHundredMBFile)
	localFilePath := path.Join(TmpDir, setup.GenerateRandomString(5))
	t.Cleanup(func() { operations.RemoveFile(localFilePath) })

	// Write sequentially to local and mounted directory file.
	operations.WriteFilesSequentially(t, []string{localFilePath, mountedDirFilePath}, FiveHundredMB, ChunkSize)

	identical, err := operations.AreFilesIdentical(mountedDirFilePath, localFilePath)
	require.NoError(t, err)
	assert.True(t, identical)
}
