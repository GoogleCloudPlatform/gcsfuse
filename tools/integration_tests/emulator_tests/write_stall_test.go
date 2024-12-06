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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const fileSize = 50 * 1024 * 1024

type writeStall struct {
	flags []string
}

func (s *writeStall) Setup(t *testing.T) {
	setup.MountGCSFuseWithGivenMountFunc(s.flags, mountFunc)
}

func (s *writeStall) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *writeStall) TestWriteStall(t *testing.T) {
	stallTime := 40 * time.Second
	filePath := path.Join(setup.MntDir(), "file.txt")
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

func TestWriteStall(t *testing.T) {
	ts := &writeStall{}

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
