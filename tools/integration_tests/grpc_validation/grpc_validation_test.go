// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpc_validation

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	client_util "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type gRPCValidation struct {
	suite.Suite
	singleRegionBucketForGRPCSuccess string
	singleRegionBucketForGRPCFailure string
	multiRegionBucketForGRPCSuccess  string
	multiRegionBucketForGRPCFailure  string
}

// Setup involves:
// Finding out test regions.
// Creating unique test bucket names.
// Creating the test buckets.
// Test Cases:
// Test Case 1: single region test: success case , when test VM and bucket are colocated
// Test Case 2: single region test: failure case , when test VM and bucket are out-of-region
// Test Case 3: multi region test: success case , when test VM and bucket are colocated
// Test Case 4: multi region test: failure case , when test VM and bucket are out-of-region
func (g *gRPCValidation) SetupSuite() {

	// Based on the test region which is initialized in the TestMain() function,
	// we will pick out the test region for:
	singleRegionForGRPCSuccess := findSingleRegionForGRPCDirectPathSuccessCase(testRegion)
	singleRegionForGRPCFailure := pickFailureRegionFromListOfRegions(singleRegionForGRPCSuccess, singleRegions)
	multiRegionForGRPCSuccess := findMultiRegionForGRPCDirectPathSuccessCase(testRegion)
	multiRegionForGRPCFailure := pickFailureRegionFromListOfRegions(multiRegionForGRPCSuccess, multiRegions)

	// Set the test bucket names with unique suffix
	g.singleRegionBucketForGRPCSuccess = createTestBucketName(singleRegionForGRPCSuccess)
	g.singleRegionBucketForGRPCFailure = createTestBucketName(singleRegionForGRPCFailure)
	g.multiRegionBucketForGRPCSuccess = createTestBucketName(multiRegionForGRPCSuccess)
	g.multiRegionBucketForGRPCFailure = createTestBucketName(multiRegionForGRPCFailure)

	// Create the test buckets
	if err := createTestBucket(singleRegionForGRPCSuccess, g.singleRegionBucketForGRPCSuccess); err != nil {
		g.T().Fatalf("Could not create bucket in the required region, err : %v", err)
	}
	if err := createTestBucket(singleRegionForGRPCFailure, g.singleRegionBucketForGRPCFailure); err != nil {
		g.T().Fatalf("Could not create bucket in the required region, err : %v", err)
	}
	if err := createTestBucket(multiRegionForGRPCSuccess, g.multiRegionBucketForGRPCSuccess); err != nil {
		g.T().Fatalf("Could not create bucket in the required region, err : %v", err)
	}
	if err := createTestBucket(multiRegionForGRPCFailure, g.multiRegionBucketForGRPCFailure); err != nil {
		g.T().Fatalf("Could not create bucket in the required region, err : %v", err)
	}
}

// Delete the test buckets created.
func (g *gRPCValidation) TearDownSuite() {
	bucketsToDelete := []string{
		g.singleRegionBucketForGRPCSuccess,
		g.singleRegionBucketForGRPCFailure,
		g.multiRegionBucketForGRPCSuccess,
		g.multiRegionBucketForGRPCFailure,
	}
	for _, bucket := range bucketsToDelete {
		if err := client_util.DeleteBucket(ctx, client, bucket); err != nil {
			g.T().Logf("Failed to delete bucket %s: %v", bucket, err)
		}
	}
}

func TestGRPCValidationSuite(t *testing.T) {
	suite.Run(t, new(gRPCValidation))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (g *gRPCValidation) TestGRPCDirectPathConnections() {
	testCases := []struct {
		name                  string
		bucketName            string
		grpcPathStrategy      string
		expectedSuccess       bool
		expectedLogSubstrings []string
	}{
		{
			name:                  "SingleRegion_Success_FallbackStrategy",
			bucketName:            g.singleRegionBucketForGRPCSuccess,
			grpcPathStrategy:      "direct-path-with-fallback",
			expectedSuccess:       true,
			expectedLogSubstrings: []string{"DirectPath verification succeeded, continuing with DirectPath."},
		},
		{
			name:             "SingleRegion_Failure_FallbackStrategy",
			bucketName:       g.singleRegionBucketForGRPCSuccess,
			grpcPathStrategy: "direct-path-with-fallback",
			expectedSuccess:  true,
			expectedLogSubstrings: []string{
				"DirectPath verification failed",
				"Grpc dp is not available and falling back to Http.",
			},
		},
		{
			name:                  "SingleRegion_Success_DirectPathOnlyStrategy",
			bucketName:            g.singleRegionBucketForGRPCSuccess,
			grpcPathStrategy:      "direct-path-only",
			expectedSuccess:       true,
			expectedLogSubstrings: []string{"DirectPath verification succeeded, continuing with DirectPath."},
		},
		{
			name:             "SingleRegion_Failure_DirectPathOnlyStrategy",
			bucketName:       g.singleRegionBucketForGRPCSuccess,
			grpcPathStrategy: "direct-path-only",
			expectedSuccess:  false,
			expectedLogSubstrings: []string{
				"DirectPath verification failed",
				"Grpc dp is not available and not falling back to Http as gRPC path strategy is set to DirectPathOnly",
			},
		},
		{
			name:                  "MultiRegion_Success_FallbackStrategy",
			bucketName:            g.multiRegionBucketForGRPCSuccess,
			grpcPathStrategy:      "direct-path-with-fallback",
			expectedSuccess:       true,
			expectedLogSubstrings: []string{"DirectPath verification succeeded, continuing with DirectPath."},
		},
		{
			name:             "MultiRegion_Failure_FallbackStrategy",
			bucketName:       g.multiRegionBucketForGRPCSuccess,
			grpcPathStrategy: "direct-path-with-fallback",
			expectedSuccess:  true,
			expectedLogSubstrings: []string{
				"DirectPath verification failed",
				"Grpc dp is not available and falling back to Http.",
			},
		},
		{
			name:                  "MultiRegion_Success_DirectPathOnlyStrategy",
			bucketName:            g.multiRegionBucketForGRPCSuccess,
			grpcPathStrategy:      "direct-path-only",
			expectedSuccess:       true,
			expectedLogSubstrings: []string{"DirectPath verification succeeded, continuing with DirectPath."},
		},
		{
			name:             "MultiRegion_Failure_DirectPathOnlyStrategy",
			bucketName:       g.multiRegionBucketForGRPCSuccess,
			grpcPathStrategy: "direct-path-only",
			expectedSuccess:  false,
			expectedLogSubstrings: []string{
				"DirectPath verification failed",
				"Grpc dp is not available and not falling back to Http as gRPC path strategy is set to DirectPathOnly",
			},
		},
	}

	for _, tc := range testCases {
		g.T().Run(tc.name, func(t *testing.T) {
			if strings.Contains(tc.name, "Failure") {
				t.Setenv("GOOGLE_CLOUD_DISABLE_DIRECT_PATH", "true")
			} else {
				t.Setenv("GOOGLE_CLOUD_DISABLE_DIRECT_PATH", "false")
			}

			mountPoint, err := os.MkdirTemp("", "grpc_validation_test")
			assert.NoError(t, err)
			logFile := fmt.Sprintf("/tmp/grpc_%s_%d.txt", tc.name, time.Now().UnixNano())

			defer func() {
				_ = util.Unmount(mountPoint)
				_ = os.Remove(mountPoint)
				// Only remove the log file if the test succeeded
				if t.Failed() {
					t.Logf("Test failed, log file '%s' will not be deleted for inspection.", logFile)
				} else {
					_ = os.Remove(logFile)
				}
			}()

			args := []string{
				"--client-protocol=grpc",
				"--grpc-path-strategy=" + tc.grpcPathStrategy,
				"--log-severity=TRACE",
				fmt.Sprintf("--log-file=%s", logFile),
				tc.bucketName,
				mountPoint,
			}
			err = mounting.MountGcsfuse(setup.BinFile(), args)
			if err != nil {
				if tc.expectedSuccess {
					t.Errorf("Unexpected mount failure: %v", err)
				}
			} else {
				if !tc.expectedSuccess {
					t.Errorf("Expected mount failure but mount succeeded")
				}
			}

			for _, logSubstring := range tc.expectedLogSubstrings {
				success := operations.CheckLogFileForMessage(g.T(), logSubstring, logFile)
				require.Equal(t, true, success, "Expected message %q not found in log file %s", logSubstring, logFile)
			}
		})
	}
}
