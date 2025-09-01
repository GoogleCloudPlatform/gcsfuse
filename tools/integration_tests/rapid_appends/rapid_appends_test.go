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

package rapid_appends

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

const (
	testDirName    = "RapidAppendsTest"
	fileNamePrefix = "rapid-append-file-"
	// Minimum content size to trigger block upload; calculated as (2*blocksize+1) MiB.
	contentSizeForBW = 3
	// Block size for buffered-writes.
	blockSize            = operations.OneMiB
	metadataCacheTTLSecs = 10
	FileOpenModeRplus    = os.O_RDWR
	FileOpenModeA        = os.O_APPEND | os.O_WRONLY
)

var (
	storageClient *storage.Client
	ctx           context.Context
)

// //////////////////////////////////////////////////////////////////////
// Test Configurations
// //////////////////////////////////////////////////////////////////////

// testConfig defines a specific GCSfuse configuration for a test run.
type testConfig struct {
	name string
	// isDualMount indicates whether the test involves two separate GCSFuse mounts.
	isDualMount bool
	// metadataCacheOnRead indicates whether metadata caching is enabled for reads.
	metadataCacheEnabled bool
	// fileCache indicates whether file caching is enabled.
	fileCache bool
	// primaryMountFlags are the GCSFuse flags used for the primary mount.
	primaryMountFlags []string
	// secondaryMountFlags are the GCSFuse flags used for the secondary mount.
	// This is only relevant when isDualMount is true.
	secondaryMountFlags []string
}

// readTestConfigs defines the matrix of configurations for the ReadsTestSuite.
var readTestConfigs = []*testConfig{
	// Single-Mount Scenarios
	{
		name:                 "SingleMount_NoCache",
		isDualMount:          false,
		metadataCacheEnabled: false,
		fileCache:            false,
		primaryMountFlags:    []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                 "SingleMount_MetadataCache",
		isDualMount:          false,
		metadataCacheEnabled: true,
		fileCache:            false,
		primaryMountFlags:    []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                 "SingleMount_FileCache",
		isDualMount:          false,
		metadataCacheEnabled: false,
		fileCache:            true,
		primaryMountFlags:    []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                 "SingleMount_MetadataAndFileCache",
		isDualMount:          false,
		metadataCacheEnabled: true,
		fileCache:            true,
		primaryMountFlags:    []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	// Dual-Mount Scenarios
	{
		name:                 "DualMount_NoCache",
		isDualMount:          true,
		metadataCacheEnabled: false,
		fileCache:            false,
		primaryMountFlags:    []string{"--enable-rapid-appends=true"},
		secondaryMountFlags:  []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                 "DualMount_MetadataCache",
		isDualMount:          true,
		metadataCacheEnabled: true,
		fileCache:            false,
		primaryMountFlags:    []string{"--enable-rapid-appends=true"},
		secondaryMountFlags:  []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                 "DualMount_FileCache",
		isDualMount:          true,
		metadataCacheEnabled: false,
		fileCache:            true,
		primaryMountFlags:    []string{"--enable-rapid-appends=true"},
		secondaryMountFlags:  []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                 "DualMount_MetadataAndFileCache",
		isDualMount:          true,
		metadataCacheEnabled: true,
		fileCache:            true,
		primaryMountFlags:    []string{"--enable-rapid-appends=true"},
		secondaryMountFlags:  []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
}

// appendTestConfigs defines the matrix of configurations for the AppendsTestSuite.
var appendTestConfigs = []*testConfig{
	{
		name:              "SingleMount_BufferedWrite",
		isDualMount:       false,
		primaryMountFlags: []string{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	},
	{
		name:                "DualMount_BufferedWrite",
		isDualMount:         true,
		primaryMountFlags:   []string{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
		secondaryMountFlags: []string{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	},
}

// //////////////////////////////////////////////////////////////////////
// Test Runners
// //////////////////////////////////////////////////////////////////////

// TestReadsSuiteRunner executes all read-after-append tests against the readTestConfigs matrix.
func TestReadsSuiteRunner(t *testing.T) {
	for _, cfg := range readTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			suite.Run(t, &ReadsTestSuite{BaseSuite{cfg: cfg}})
		})
	}
}

// TestAppendsSuiteRunner executes all general append tests against the appendTestConfigs matrix.
func TestAppendsSuiteRunner(t *testing.T) {
	for _, cfg := range appendTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			suite.Run(t, &AppendsTestSuite{BaseSuite{cfg: cfg}})
		})
	}
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if !setup.IsZonalBucketRun() {
		log.Fatalf("This package must be run with a Zonal Bucket.")
	}

	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		log.Fatalf("This package does not support the --mountedDirectory flag.")
	}

	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		if err := closeStorageClient(); err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	log.Println("Running static mounting tests for rapid appends...")
	successCode := m.Run()

	// Clean up the test directory on GCS after all tests have run.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
