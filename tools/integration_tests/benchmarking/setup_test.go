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
//
// Note that the expected latency thresholds for the various operations has
// been set to 4 times the observed latency. Any failure of the benchmark tests
// is a direct indicator of anomaly.

package benchmarking

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName = "benchmarking"
)

var (
	testEnv   env
	mountFunc func(*test_suite.TestConfig, []string) error
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	setup.SetMntDir(testEnv.cfg.GCSFuseMountedDirectory)
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

// createFiles creates the below objects in the bucket.
// benchmarking/a{i}.txt where i is a counter based on the benchtime value.
func createFiles(b *testing.B) {
	for i := range b.N {
		operations.CreateFileOfSize(1, path.Join(testEnv.testDirPath, fmt.Sprintf("a%d.txt", i)), b)
	}
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.Benchmarking) == 0 {
		log.Println("No configuration found for benchmarking tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.Benchmarking = make([]test_suite.TestConfig, 1)
		cfg.Benchmarking[0].TestBucket = setup.TestBucket()
		cfg.Benchmarking[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.Benchmarking[0].LogFile = setup.LogFile()
		// Manually add configs for each benchmark test.
		cfg.Benchmarking[0].Configs = make([]test_suite.ConfigItem, 3)
		cfg.Benchmarking[0].Configs[0].Flags = []string{"--stat-cache-ttl=0", "--stat-cache-ttl=0 --client-protocol=grpc"}
		cfg.Benchmarking[0].Configs[0].Run = "Benchmark_Stat"
		cfg.Benchmarking[0].Configs[1].Flags = []string{"--stat-cache-ttl=0 --enable-atomic-rename-object=true", "--stat-cache-ttl=0 --enable-atomic-rename-object=true --client-protocol=grpc"}
		cfg.Benchmarking[0].Configs[1].Run = "Benchmark_Rename"
		cfg.Benchmarking[0].Configs[2].Flags = []string{"--stat-cache-ttl=0", "--client-protocol=grpc --stat-cache-ttl=0"}
		cfg.Benchmarking[0].Configs[2].Run = "Benchmark_Delete"
	}

	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.Benchmarking[0]
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)

	// 2. Create storage client before running tests.
	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer testEnv.storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucket(testEnv.cfg)

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, testDirName))
	os.Exit(successCode)
}
