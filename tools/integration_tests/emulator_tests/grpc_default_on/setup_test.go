// Copyright 2026 Google LLC
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

package grpc_default_on

import (
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

var (
	// rootDir is the directory to be mounted and unmounted.
	rootDir string
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if setup.MountedDirectory() != "" {
		log.Printf("These tests will not run with mounted directory..")
		return
	}

	cfg := &test_suite.TestConfig{
		GKEMountedDirectory:     setup.MountedDirectory(),
		GCSFuseMountedDirectory: setup.MntDir(),
		TestBucket:              setup.TestBucket(),
		LogFile:                 setup.LogFile(),
	}
	setup.SetUpTestDirForTestBucket(cfg)

	rootDir = cfg.GCSFuseMountedDirectory

	log.Println("Running gRPC default-on tests with static mounting...")
	successCode := m.Run()
	os.Exit(successCode)
}
