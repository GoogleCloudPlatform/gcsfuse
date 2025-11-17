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
	"golang.org/x/sync/errgroup"
)

const (
	DirForConcurrentWrite = "dirForConcurrentWrite"
)

func TestWriteMultipleFilesConcurrently(t *testing.T) {
	concurrentWriteDir := setup.SetupTestDirectory(DirForConcurrentWrite)
	var FileOne = "fileOne" + setup.GenerateRandomString(5) + ".txt"
	var FileTwo = "fileTwo" + setup.GenerateRandomString(5) + ".txt"
	var FileThree = "fileThree" + setup.GenerateRandomString(5) + ".txt"
	files := []string{FileOne, FileTwo, FileThree}
	var eG errgroup.Group

	// Concurrently write three different files.
	for i := range files {
		// Copy the current value of i into a local variable to avoid data races.
		fileIndex := i

		// Thread to write the current file.
		eG.Go(func() error {
			mountedDirFilePath := path.Join(concurrentWriteDir, files[fileIndex])
			localFilePath := path.Join(TmpDir, setup.GenerateRandomString(5))
			t.Cleanup(func() { operations.RemoveFile(localFilePath) })

			operations.WriteFilesSequentially(t, []string{localFilePath, mountedDirFilePath}, FiveHundredMB, ChunkSize)

			identical, err := operations.AreFilesIdentical(mountedDirFilePath, localFilePath)
			require.NoError(t, err)
			assert.True(t, identical)
			return nil
		})
	}

	// Wait on threads to end.
	_ = eG.Wait()
}
