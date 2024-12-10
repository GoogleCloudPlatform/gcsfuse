// Copyright 2024 Google LLC
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

package emulator_tests

import (
	"crypto/rand"
	"io"
	"log"
	"os"
	"path"
	"testing"
	"time"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const fileSize = 50 * 1024 * 1024

type chunkTransferTimeoutnInfinity struct {
	flags []string
}

func (s *chunkTransferTimeoutnInfinity) Setup(t *testing.T) {
	emulator_tests.StartProxyServer()
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
}

func (s *chunkTransferTimeoutnInfinity) Teardown(t *testing.T) {
	assert.NoError(t, emulator_tests.KillProxyServerProcess(8020))
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// This test verifies that write operations stall for the expected duration
// when a write stall is induced. It creates a file, writes data to it,
// and then calls Sync() to ensure the data is written to disk. The test
// measures the time taken for the Sync() operation and asserts that it
// is greater than or equal to the configured stall time.
func (s *chunkTransferTimeoutnInfinity) TestWriteStallCausesDelay(t *testing.T) {
	stallTime := 40 * time.Second
	testDirPath := path.Join(setup.MntDir(), testDirName)
	filePath := path.Join(testDirPath, "file.txt")
	// Create a file for writing
	file, err := os.Create(filePath)
	if err != nil {
		assert.NoError(t, err)
	}
	defer file.Close()

	// Generate random data
	data := make([]byte, fileSize)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		assert.NoError(t, err)
	}

	// Write the data to the file
	if _, err := file.Write(data); err != nil {
		assert.NoError(t, err)
	}
	startTime := time.Now()
	err = file.Sync()
	// End timer
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, elapsedTime, stallTime)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestChunkTransferTimeoutInfinity(t *testing.T) {
	ts := &chunkTransferTimeoutnInfinity{}

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--custom-endpoint=http://localhost:8020/storage/v1/b?project=test-project/b?bucket=test-bucket"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
