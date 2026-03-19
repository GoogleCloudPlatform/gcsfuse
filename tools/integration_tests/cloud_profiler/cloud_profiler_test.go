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
	"math"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName           = "CloudProfilerTest"
	testVersionPrefix     = "cloud-profiler-test"
	testServiceNamePrefix = "cloud-profiler-test"
	retryFrequency        = 30 * time.Second
	retryDuration         = 30 * time.Minute
)

var (
	storageClient   *storage.Client
	testVersionName string
	testServiceName string
	ctx             context.Context
)

// The alphabet defines the sort order: 0 is smallest, z is largest.
const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
const fixedLength = 13 // math.MaxInt64 in base 36 fits in 13 characters.

// getDecreasingString generates a string that decreases lexicographically as time increases, making newer items appear earlier in sorted results.
func getDecreasingString() string {
	// Calculate the decreasing value
	val := uint64(math.MaxInt64 - time.Now().UnixNano())

	// Map the value to our 36-character alphabet
	res := make([]byte, fixedLength)
	for i := fixedLength - 1; i >= 0; i-- {
		res[i] = alphabet[val%36]
		val /= 36
	}

	return string(res)
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	suffix := getDecreasingString()
	testServiceName = fmt.Sprintf("%s-%s", testServiceNamePrefix, suffix)
	testVersionName = fmt.Sprintf("%s-%s", testVersionPrefix, suffix)
	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.CloudProfiler) == 0 {
		log.Println("No configuration found for cloud profiler tests in config. Using flags instead.")

		// Populate the config manually.
		cfg.CloudProfiler = make([]test_suite.TestConfig, 1)
		cfg.CloudProfiler[0].TestBucket = setup.TestBucket()
		cfg.CloudProfiler[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.CloudProfiler[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.CloudProfiler[0].Configs[0].Flags = []string{
			"--log-severity=TRACE --enable-cloud-profiler --cloud-profiler-cpu",
		}
		testVersionFlag := fmt.Sprintf(" --cloud-profiler-label=%s", testVersionName)
		testServiceNameFlag := fmt.Sprintf(" --cloud-profiler-service-name=%s", testServiceName)
		cfg.CloudProfiler[0].Configs[0].Flags[0] = cfg.CloudProfiler[0].Configs[0].Flags[0] + testVersionFlag + testServiceNameFlag
		cfg.CloudProfiler[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}

	ctx = context.Background()

	bucketType := setup.TestEnvironment(ctx, &cfg.CloudProfiler[0])

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if cfg.CloudProfiler[0].GKEMountedDirectory != "" {
		testVersionName = setup.ExtractServiceVersionFromFlags(cfg.CloudProfiler[0].Configs[0].Flags)
		testServiceName = setup.CloudProfilerServiceNameFromFlags(cfg.CloudProfiler[0].Configs[0].Flags)
		os.Exit(setup.RunTestsForMountedDirectory(cfg.CloudProfiler[0].GKEMountedDirectory, m))
	}

	logger.Infof("Enabling cloud profiler with Service Name: %s and version: %s", testServiceName, testVersionName)

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.CloudProfiler[0], bucketType, "")

	setup.SetUpTestDirForTestBucket(&cfg.CloudProfiler[0])

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.CloudProfiler[0], flags, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
