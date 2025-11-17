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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	client_util "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/util"
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
		name                 string
		bucketName           string
		expectedSuccess      bool
		expectedLogSubstring string
	}{
		{
			name:                 "SingleRegion_Success",
			bucketName:           g.singleRegionBucketForGRPCSuccess,
			expectedLogSubstring: fmt.Sprintf("Successfully connected over gRPC DirectPath for %s", g.singleRegionBucketForGRPCSuccess),
		},
		{
			name:                 "SingleRegion_Failure",
			bucketName:           g.singleRegionBucketForGRPCFailure,
			expectedLogSubstring: fmt.Sprintf("Direct path connectivity unavailable for %s, reason:", g.singleRegionBucketForGRPCFailure),
		},
		{
			name:                 "MultiRegion_Success",
			bucketName:           g.multiRegionBucketForGRPCSuccess,
			expectedLogSubstring: fmt.Sprintf("Successfully connected over gRPC DirectPath for %s", g.multiRegionBucketForGRPCSuccess),
		},
		{
			name:                 "MultiRegion_Failure",
			bucketName:           g.multiRegionBucketForGRPCFailure,
			expectedLogSubstring: fmt.Sprintf("Direct path connectivity unavailable for %s, reason: ", g.multiRegionBucketForGRPCFailure),
		},
	}

	for _, tc := range testCases {
		g.T().Run(tc.name, func(t *testing.T) {
			mountPoint, err := os.MkdirTemp("", "grpc_validation_test")
			assert.NoError(t, err)
			logFile := fmt.Sprintf("/tmp/grpc_%s_%d.txt", tc.name, time.Now().UnixNano())
			args := []string{"--client-protocol=grpc", "--log-severity=TRACE", fmt.Sprintf("--log-file=%s", logFile), tc.bucketName, mountPoint}
			err = mounting.MountGcsfuse(setup.BinFile(), args)
			if err != nil {
				if tc.expectedSuccess {
					t.Errorf("Unexpected mount failure: %v", err)
				}
				return
			}

			defer func() {
				if err := util.Unmount(mountPoint); err != nil {
					t.Logf("Warning: unmount failed: %v", err)
				}
				os.Remove(mountPoint)
				// Only remove the log file if the test succeeded
				if t.Failed() {
					t.Logf("Test failed, log file '%s' will not be deleted for inspection.", logFile)
				} else {
					os.Remove(logFile)
				}
			}()
			success := operations.CheckLogFileForMessage(g.T(), tc.expectedLogSubstring, logFile)
			require.Equal(t, true, success)
		})
	}
}
