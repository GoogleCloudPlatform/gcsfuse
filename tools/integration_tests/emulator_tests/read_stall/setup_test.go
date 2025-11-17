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

package read_stall

import (
	"log"
	"os"
	"testing"

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

var (
	testDirPath string
	mountFunc   func([]string) error
	// root directory is the directory to be unmounted.
	rootDir string
)

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
