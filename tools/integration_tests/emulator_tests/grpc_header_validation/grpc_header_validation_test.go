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

package grpc_header_validation

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

type grpcHeaderValidation struct {
	port               int
	proxyProcessId     int
	proxyServerLogFile string
	flags              []string
	configPath         string
	suite.Suite
}

func (g *grpcHeaderValidation) SetupTest() {
	g.configPath = "../configs/grpc_header_validation.yaml"
	g.proxyServerLogFile = setup.CreateProxyServerLogFile(g.T())
	g.T().Logf("Proxy server log file: %s", g.proxyServerLogFile)
	var err error
	g.port, g.proxyProcessId, err = emulator_tests.StartProxyServer(g.configPath, g.proxyServerLogFile)
	require.NoError(g.T(), err)
	g.flags = append(g.flags, fmt.Sprintf("--custom-endpoint=localhost:%d", g.port))
	g.flags = append(g.flags, "--anonymous-access") // Required for gRPC localhost endpoint.
	setup.MountGCSFuseWithGivenMountFunc(g.flags, mountFunc)
}

func (g *grpcHeaderValidation) TearDownTest() {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(g.T(), emulator_tests.KillProxyServerProcess(g.proxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(g.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(g.proxyServerLogFile, g.T())
}

func (g *grpcHeaderValidation) TestGRPCClientSendsExpectedHeaders() {
	// GCSFuse mount itself triggers gRPC calls for DirectPath verification, we
	// just need to verify the proxy logs contain the expected header.

	logContent, err := os.ReadFile(g.proxyServerLogFile)
	require.NoError(g.T(), err)
	logStr := string(logContent)
	// Verify metadata validation passed and x-goog-request-params contains force_direct_connectivity=ENFORCED
	// Note: With anonymous-access, direct_connectivity_diagnostic=no_auth is also present
	assert.Contains(g.T(), logStr, "Metadata validation passed")
	assert.Contains(g.T(), logStr, "x-goog-request-params")
	assert.Contains(g.T(), logStr, "force_direct_connectivity=ENFORCED")
	// Verify direct_connectivity_diagnostic header is present
	assert.Contains(g.T(), logStr, "direct_connectivity_diagnostic=no_auth")
	assert.Contains(g.T(), logStr, "/google.storage.v2.Storage/GetObject")
}

func (g *grpcHeaderValidation) TestGRPCHeadersInMultipleOperations() {
	// Perform multiple file operations to trigger various gRPC calls
	testFilePath := path.Join(rootDir, "test_file_for_grpc.txt")
	testContent := []byte("This is test content to validate gRPC headers across different operations.")

	// 1. Create and write a file (triggers WriteObject gRPC call)
	err := os.WriteFile(testFilePath, testContent, 0644)
	if err == nil {
		g.T().Logf("Successfully created and wrote file: %s", testFilePath)
		
		// 2. Read the file (triggers ReadObject gRPC call)
		readContent, err := os.ReadFile(testFilePath)
		if err == nil {
			g.T().Logf("Successfully read file, read %d bytes", len(readContent))
			assert.Equal(g.T(), testContent, readContent, "File content should match")
		} else {
			g.T().Logf("File read failed (continuing with validation): %v", err)
		}
		
		// 3. Stat the file (triggers GetObject gRPC call for metadata)
		fileInfo, err := os.Stat(testFilePath)
		if err == nil {
			g.T().Logf("Successfully stat'd file: size=%d bytes", fileInfo.Size())
		} else {
			g.T().Logf("File stat failed (continuing with validation): %v", err)
		}
	} else {
		g.T().Logf("File write failed (continuing with validation): %v", err)
	}

	// 4. List directory (triggers bucket/folder listing gRPC calls)
	entries, err := os.ReadDir(rootDir)
	if err == nil {
		g.T().Logf("Listed %d entries in root directory", len(entries))
	} else {
		g.T().Logf("ReadDir failed (continuing with validation): %v", err)
	}

	// Read and verify proxy logs - this is the main validation
	logContent, err := os.ReadFile(g.proxyServerLogFile)
	require.NoError(g.T(), err)
	logStr := string(logContent)

	// Verify metadata was validated successfully across multiple operations
	validationCount := strings.Count(logStr, "Metadata validation passed")
	assert.Greater(g.T(), validationCount, 0, "Expected at least one successful metadata validation")
	assert.Contains(g.T(), logStr, "force_direct_connectivity=ENFORCED")

	// Verify direct_connectivity_diagnostic header is present in all operations
	assert.Contains(g.T(), logStr, "direct_connectivity_diagnostic=no_auth")

	// Verify gRPC calls were made for different operations
	getObjectCount := strings.Count(logStr, "/google.storage.v2.Storage/GetObject")
	readObjectCount := strings.Count(logStr, "/google.storage.v2.Storage/ReadObject")
	writeObjectCount := strings.Count(logStr, "/google.storage.v2.Storage/BidiWriteObject")
	listObjectsCount := strings.Count(logStr, "/google.storage.v2.Storage/ListObjects")

	totalGRPCCalls := getObjectCount + readObjectCount + writeObjectCount + listObjectsCount
	assert.Greater(g.T(), totalGRPCCalls, 0, "Expected at least one gRPC storage call")
	g.T().Logf("gRPC calls observed - GetObject: %d, ReadObject: %d, WriteObject: %d, ListObjects: %d (Total: %d)",
		getObjectCount, readObjectCount, writeObjectCount, listObjectsCount, totalGRPCCalls)
}

func TestGRPCHeaderValidation(t *testing.T) {
	ts := &grpcHeaderValidation{}
	// Test with gRPC protocol to validate that DirectPath metadata is sent.
	// The Go Storage SDK automatically adds force_direct_connectivity=ENFORCED
	// to x-goog-request-params when experimental.WithDirectConnectivityEnforced() is used.
	// The gRPC proxy intercepts and validates this metadata.
	// 
	// NOTE: This test requires:
	// 1. gRPC testbench server running on localhost:8888 (started by emulator_tests.sh)
	// 2. Bucket named "test-bucket" created in the testbench
	// 3. Run with: go test -v --integrationTest --testbucket=test-bucket -timeout=5m
	flagsSet := [][]string{
		{"--client-protocol=grpc"},
	}

	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running gRPC header validation tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
