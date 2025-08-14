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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

const (
	DirForSlowWrite = "dirForSlowWrite"
)

func TestSlowWriteToFile(t *testing.T) {
	slowWriteDir := setup.SetupTestDirectory(DirForSlowWrite)
	slowWriteFileName := "slowWriteFile" + setup.GenerateRandomString(5) + ".txt"
	mountedDirFilePath := path.Join(slowWriteDir, slowWriteFileName)
	fh := operations.OpenFileWithODirect(t, mountedDirFilePath)
	defer operations.CloseFileShouldNotThrowError(t, fh)
	data := setup.GenerateRandomString(32 * operations.MiB)

	// Write 4 blocks of 33MiB to file each with 1 min delay.
	for range 4 {
		n, err := fh.Write([]byte(data))
		operations.SyncFileShouldThrowError(t, fh)

		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		time.Sleep(1 * time.Minute)
	}
}
