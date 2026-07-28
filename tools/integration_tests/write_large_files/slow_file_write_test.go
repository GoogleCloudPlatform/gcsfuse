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

package write_large_files

import (
	"path"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

const (
	DirForSlowWrite = "dirForSlowWrite"
)

func (s *WriteLargeFilesTestSuite) TestSlowWriteToFile() {
	slowWriteDir := setup.SetupTestDirectory(DirForSlowWrite)
	slowWriteFileName := "slowWriteFile" + setup.GenerateRandomString(5) + ".txt"
	mountedDirFilePath := path.Join(slowWriteDir, slowWriteFileName)
	fh := operations.OpenFileWithODirect(s.T(), mountedDirFilePath)
	defer operations.CloseFileShouldNotThrowError(s.T(), fh)
	data := setup.GenerateRandomString(33 * operations.MiB)

	// Write 2 blocks of 33MiB to file
	for range 2 {
		n, err := fh.Write([]byte(data))

		assert.NoError(s.T(), err)
		assert.Equal(s.T(), len(data), n)
		operations.SyncFile(fh, s.T())
		// Add a delay of 40 sec between each block to ensure 32 sec
		// ChunkRetryDeadline is completely used while reading the second Chunk.
		time.Sleep(40 * time.Second)
	}
}
