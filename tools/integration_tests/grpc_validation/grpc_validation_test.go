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
	"context"
	"os"
	"path"
	"bufio"
	"strings"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"cloud.google.com/go/storage"
)


type gRPCValidation struct {
	suite.Suite
	ctx                 context.Context
    client              *storage.Client
	testRegion string
	singleRegionBucketForGRPCSuccess string
	singleRegionBucketForGRPCFailure string
	multiRegionBucketForGRPCSuccess string
	multiRegionBucketForGRPCFailure string
	mountPoint string	
	gcsfusePath string
	logFile string
}


func (g *gRPCValidation) SetupSuite(){
		// Creating a common storage client for the test
		g.ctx = context.Background()
		var err error
		// Find the test environment, if run in cloudtop or issue fetching the test environment, skip all tests.
		if g.testRegion,err = findTestExecutionEnvironment(g.ctx); g.testRegion == cloudtopProd || err!=nil  {
			g.T().Logf("Skipping tests as environment is cloudtop.")
			g.T().Skip()
		}
		
		setup.SetLogFile(fmt.Sprintf("/tmp/grpc_validation_mount_cmds_%d.txt",time.Now().UnixNano()))
		if g.client, err = storage.NewClient(g.ctx); err!=nil {
			g.T().Fatalf("Failed to create storage client...")
		}
		defer g.client.Close()

		// Figure out the test region for single region test: success case
		singleRegionForGRPCSuccess := findSingleRegionForGRPCDirectPathSuccessCase(g.testRegion)
		// Figure out the test region for single region test: failure case
		singleRegionForGRPCFailure := pickFailureRegionFromListOfRegions(singleRegionForGRPCSuccess,single_regions)
		// Figure out the test region for multi region test: success case
		multiRegionForGRPCSuccess := findMultiRegionForGRPCDirectPathSuccessCase(g.testRegion)
		// Figure out the test region for multi region test: failure case
		multiRegionForGRPCFailure := pickFailureRegionFromListOfRegions(multiRegionForGRPCSuccess,multi_regions)

		// Set the test bucket names
		g.singleRegionBucketForGRPCSuccess = createTestBucketName(singleRegionForGRPCSuccess)
		g.singleRegionBucketForGRPCFailure = createTestBucketName(singleRegionForGRPCFailure)
		g.multiRegionBucketForGRPCSuccess = createTestBucketName(multiRegionForGRPCSuccess)
		g.multiRegionBucketForGRPCFailure = createTestBucketName(multiRegionForGRPCFailure)


		// Create the test buckets .
		if err:= createTestBucket(g.ctx,g.client,singleRegionForGRPCSuccess,g.singleRegionBucketForGRPCSuccess); err != nil{
			g.T().Fatalf("Could not create bucket in the required region, err : %v",err)
		}
		if err:= createTestBucket(g.ctx,g.client,singleRegionForGRPCFailure,g.singleRegionBucketForGRPCFailure); err != nil{
			g.T().Fatalf("Could not create bucket in the required region, err : %v",err)
		}
		if err:= createTestBucket(g.ctx,g.client,multiRegionForGRPCSuccess,g.multiRegionBucketForGRPCSuccess); err != nil{
			g.T().Fatalf("Could not create bucket in the required region, err : %v",err)
		}
		if err:= createTestBucket(g.ctx,g.client,multiRegionForGRPCFailure,g.multiRegionBucketForGRPCFailure); err != nil{
			g.T().Fatalf("Could not create bucket in the required region, err : %v",err)
		}

		// Specifying the path to gcsfuse binary for mounting
		g.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
		
		

}

func (g *gRPCValidation) TearDownSuite(){
	if g.client != nil {
		// Delete the test buckets
		if err := DeleteBucket(g.ctx, g.client, g.singleRegionBucketForGRPCSuccess); err != nil {
			g.T().Logf("Failed to delete bucket %s: %v", g.singleRegionBucketForGRPCSuccess, err)
  		}
		if err := DeleteBucket(g.ctx, g.client, g.singleRegionBucketForGRPCFailure); err != nil {
			g.T().Logf("Failed to delete bucket %s: %v", g.singleRegionBucketForGRPCSuccess, err)
  		}
		if err := DeleteBucket(g.ctx, g.client, g.multiRegionBucketForGRPCSuccess); err != nil {
			g.T().Logf("Failed to delete bucket %s: %v", g.singleRegionBucketForGRPCSuccess, err)
  		}
		if err := DeleteBucket(g.ctx, g.client, g.multiRegionBucketForGRPCFailure); err != nil {
			g.T().Logf("Failed to delete bucket %s: %v", g.singleRegionBucketForGRPCSuccess, err)
  		}

		g.client.Close()
	}

}

func TestGRPCValidationSuite(t *testing.T) {
    suite.Run(t, new(gRPCValidation))
}

func (g *gRPCValidation) SetupTest() {
	var err error
	// Set up the temporary directory.
	g.mountPoint, err = os.MkdirTemp("", "grpc_validation_test")
	assert.NoError(g.T(), err)
	// Set up the logfile.
	g.logFile = fmt.Sprintf("/tmp/%s_%d.txt", logFilePrefix, time.Now().UnixNano())
}

func (g *gRPCValidation) TearDownTest() {
	err := os.Remove(g.mountPoint)
	assert.NoError(g.T(), err)
	// Removing the Logfile.
	err = os.Remove(g.logFile)
	assert.NoError(g.T(),err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

// Single Region Bucket tests
func (g *gRPCValidation) TestGRPCDirectPathConnectionIsUnsuccessfulWithSingleRegionBucket(){
	args := []string{"--client-protocol=grpc", "--log-severity=TRACE",fmt.Sprintf("--log-file=%s",g.logFile), g.singleRegionBucketForGRPCFailure, g.mountPoint}
	if err := mounting.MountGcsfuse(g.gcsfusePath, args); err != nil {
		g.T().Fatalf("Failed to mount the GCS Bucket using GCSFuse. Shutting down tests..")
	}
	
	defer func() {
		if err := util.Unmount(g.mountPoint); err != nil {
			g.T().Logf("Warning: unmount failed: %v", err)
		}
	}()
	
	
	file, err := os.Open(g.logFile)
	require.NoError(g.T(), err, "Failed to open log file")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	directPathLogFound := false
	expectedLog := fmt.Sprintf("Direct path connectivity unavailable for %s, reason: ", g.singleRegionBucketForGRPCFailure)

	for scanner.Scan() {
			if strings.Contains(scanner.Text(), expectedLog) {
					directPathLogFound = true
					break
			}
	}
	require.True(g.T(), directPathLogFound, "gRPC DirectPath connection unsuccessful log message not found in log file")
}


func (g *gRPCValidation) TestGRPCDirectPathConnectionIsSuccessfulWithSingleRegionBucket(){
	args := []string{"--client-protocol=grpc", "--log-severity=TRACE",fmt.Sprintf("--log-file=%s",g.logFile), g.singleRegionBucketForGRPCSuccess, g.mountPoint}
	if err := mounting.MountGcsfuse(g.gcsfusePath, args); err != nil {
		g.T().Fatalf("Failed to mount the GCS Bucket using GCSFuse. Shutting down tests..")
	}
	
	defer func() {
		if err := util.Unmount(g.mountPoint); err != nil {
			g.T().Logf("Warning: unmount failed: %v", err)
		}
	}()
	

	file, err := os.Open(g.logFile)
	require.NoError(g.T(), err, "Failed to open log file")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	directPathLogFound := false
	expectedLog := fmt.Sprintf("Successfully connected over gRPC DirectPath for %s", g.singleRegionBucketForGRPCSuccess)

	for scanner.Scan() {
			if strings.Contains(scanner.Text(), expectedLog) {
					directPathLogFound = true
					break
			}
	}
	require.True(g.T(), directPathLogFound, "gRPC DirectPath connection log message not found in log file")
}


// Multi Region Bucket tests
func (g *gRPCValidation) TestGRPCDirectPathConnectionIsUnsuccessfulWithMultiRegionBucket(){
	args := []string{"--client-protocol=grpc", "--log-severity=TRACE",fmt.Sprintf("--log-file=%s",g.logFile), g.multiRegionBucketForGRPCFailure, g.mountPoint}
	if err := mounting.MountGcsfuse(g.gcsfusePath, args); err != nil {
		g.T().Fatalf("Failed to mount the GCS Bucket using GCSFuse. Shutting down tests..")
	}
	
	defer func() {
		if err := util.Unmount(g.mountPoint); err != nil {
			g.T().Logf("Warning: unmount failed: %v", err)
		}
	}()
	
	file, err := os.Open(g.logFile)
	require.NoError(g.T(), err, "Failed to open log file")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	directPathLogFound := false
	expectedLog := fmt.Sprintf("Direct path connectivity unavailable for %s, reason: ", g.multiRegionBucketForGRPCFailure)

	for scanner.Scan() {
			if strings.Contains(scanner.Text(), expectedLog) {
					directPathLogFound = true
					break
			}
	}
	require.True(g.T(), directPathLogFound, "gRPC DirectPath connection unsuccessful log message not found in log file")
}

func (g *gRPCValidation) TestGRPCDirectPathConnectionIsSuccessfulWithMultiRegionBucket(){
	args := []string{"--client-protocol=grpc", "--log-severity=TRACE",fmt.Sprintf("--log-file=%s",g.logFile), g.multiRegionBucketForGRPCSuccess, g.mountPoint}
	if err := mounting.MountGcsfuse(g.gcsfusePath, args); err != nil {
		g.T().Fatalf("Failed to mount the GCS Bucket using GCSFuse. Shutting down tests..")
	}
	
	defer func() {
		if err := util.Unmount(g.mountPoint); err != nil {
			g.T().Logf("Warning: unmount failed: %v", err)
		}
	}()
	

	file, err := os.Open(g.logFile)
	require.NoError(g.T(), err, "Failed to open log file")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	directPathLogFound := false
	expectedLog := fmt.Sprintf("Successfully connected over gRPC DirectPath for %s", g.multiRegionBucketForGRPCSuccess)

	for scanner.Scan() {
			if strings.Contains(scanner.Text(), expectedLog) {
					directPathLogFound = true
					break
			}
	}
	require.True(g.T(), directPathLogFound, "gRPC DirectPath connection log message not found in log file")
}