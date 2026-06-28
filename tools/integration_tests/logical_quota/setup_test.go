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

package logical_quota

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	sizeQuotaPrefix      = "LogicalQuotaSizeTest"
	fileCountQuotaPrefix = "LogicalQuotaFileCountTest"
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	cfg           *test_suite.TestConfig
	bucketType    string
}

var testEnv env

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.LogicalQuota) == 0 {
		log.Println("No configuration found for logical_quota tests in config. Using flags instead.")
		cfg.LogicalQuota = []test_suite.TestConfig{
			{
				TestBucket:          setup.TestBucket(),
				GKEMountedDirectory: setup.MountedDirectory(),
				LogFile:             setup.LogFile(),
				Configs: []test_suite.ConfigItem{
					{
						Flags:      []string{fmt.Sprintf("--only-dir=%s --experimental-max-size-mb=5", sizeQuotaPrefix)},
						Compatible: map[string]bool{"flat": true, "hns": true, "zonal": true},
						Run:        "TestSizeQuota",
					},
					{
						Flags:      []string{fmt.Sprintf("--only-dir=%s --experimental-max-file-count=2", fileCountQuotaPrefix)},
						Compatible: map[string]bool{"flat": true, "hns": true, "zonal": true},
						Run:        "TestFileCountQuota",
					},
				},
			},
		}
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.LogicalQuota[0])
	testEnv.cfg = &cfg.LogicalQuota[0]

	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer testEnv.storageClient.Close()

	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		log.Println("Skipping logical_quota tests for pre-mounted directory; tests require quota flags at mount time.")
		os.Exit(0)
	}

	setup.SetUpTestDirForTestBucket(testEnv.cfg)

	successCode := m.Run()
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, sizeQuotaPrefix))
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, fileCountQuotaPrefix))
	setup.SaveLogFileInCaseOfFailure(successCode)
	os.Exit(successCode)
}
