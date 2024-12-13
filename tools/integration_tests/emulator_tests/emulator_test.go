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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const port = 8020

var (
	testDirPath string
	mountFunc   func([]string) error
	// root directory is the directory to be unmounted.
	rootDir string
)

// writeFileAndSync creates a file at the given path, writes random data to it,
// and then syncs the file to GCS. It returns the time taken for the sync operation
// and any error encountered.
//
// This function is useful for testing scenarios where file write and sync operations
// might be subject to delays or timeouts.
//
// Parameters:
//   - filePath: The path where the file should be created.
//   - fileSize: The size of the random data to be written to the file.
//
// Returns:
//   - time.Duration: The elapsed time for the file.Sync() operation.
//   - error: Any error encountered during file creation, writing, or syncing.
func writeFileAndSync(filePath string, fileSize int) (time.Duration, error) {
	// Create a file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Generate random data
	data := make([]byte, fileSize)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return 0, err
	}

	// Write the data to the file
	if _, err := file.Write(data); err != nil {
		return 0, err
	}

	startTime := time.Now()
	err = file.Sync()
	endTime := time.Now()

	if err != nil {
		return 0, err
	}

	return endTime.Sub(startTime), nil
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if setup.MountedDirectory() != "" {
		log.Printf("These tests will not run with mounted directory..")
		return
	}

	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	rootDir = setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()
	os.Exit(successCode)
}
