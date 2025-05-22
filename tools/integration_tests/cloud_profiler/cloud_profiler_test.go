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

package cloud_profiler_test

// Command to run the test from gcsfuse root directory:
// go test ./tools/integration_tests/cloud_profiler/... --integrationTest --testbucket <bucket_name> -testInstalledPackage -v -timeout 20m

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	testDirName     = "CloudProfilerTest"
	testServiceName = "gcsfuse"
)

var (
	testServiceVersion string
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	var storageClient *storage.Client
	ctx := context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	testServiceVersion = fmt.Sprintf("ve2e0.0.0-%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:8])

	// Set up flags to run tests on.
	yamlContent := map[string]interface{}{
		"profiling": map[string]interface{}{
			"enabled":        true,
			"cpu":            true,
			"heap":           true,
			"goroutines":     true,
			"mutex":          true,
			"allocated-heap": true,
			"label":          testServiceVersion,
		},
	}
	flags := [][]string{
		{"--config-file=" + setup.YAMLConfigFile(yamlContent, "cloud_profiler_enabled.yaml")},
	}
	logger.Infof("Enabling cloud profiler with version tag: %s", testServiceVersion)
	successCode := static_mounting.RunTests(flags, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
