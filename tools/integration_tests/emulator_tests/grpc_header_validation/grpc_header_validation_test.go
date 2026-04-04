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

	// Assert: read and confirm the required headers.
	logContent, err := os.ReadFile(g.proxyServerLogFile)
	require.NoError(g.T(), err)
	logStr := string(logContent)
	assert.Contains(g.T(), logStr, "Metadata validation passed")
	assert.Contains(g.T(), logStr, "x-goog-request-params")
	assert.Contains(g.T(), logStr, "force_direct_connectivity=ENFORCED")
	assert.Contains(g.T(), logStr, "direct_connectivity_diagnostic=no_auth")
	assert.Contains(g.T(), logStr, "/google.storage.v2.Storage/GetObject")
}

func (g *grpcHeaderValidation) TestGRPCHeadersInMultipleOperations() {
	// Perform multiple file operations to trigger various gRPC calls
	testFilePath := path.Join(rootDir, "test_file_for_grpc.txt")
	testContent := []byte("This is test content to validate gRPC headers across different operations.")

	// Action
	err := os.WriteFile(testFilePath, testContent, 0644) // Trigger (GetObject + BidiWriteObject)
	require.NoError(g.T(), err)
	readContent, err := os.ReadFile(testFilePath) // Triggers (GetObject + ReadObject)
	require.NoError(g.T(), err)
	require.Equal(g.T(), testContent, readContent, "File content should match")
	_, err = os.Stat(testFilePath) // Triggers (GetObject)
	require.NoError(g.T(), err)
	entries, err := os.ReadDir(rootDir) // Triggers ListObjects
	require.NoError(g.T(), err)
	g.T().Logf("Listed %d entries in root directory", len(entries))

	// Assert: read and verify proxy logs.
	logContent, err := os.ReadFile(g.proxyServerLogFile)
	require.NoError(g.T(), err)
	logStr := string(logContent)
	validationCount := strings.Count(logStr, "Metadata validation passed")
	assert.Greater(g.T(), validationCount, 0, "Expected at least one successful metadata validation")
	assert.Contains(g.T(), logStr, "force_direct_connectivity=ENFORCED")
	assert.Contains(g.T(), logStr, "direct_connectivity_diagnostic=no_auth")
	assert.Equal(g.T(), 3, strings.Count(logStr, "/google.storage.v2.Storage/GetObject"), "Expected 3 GetObject calls (1 for each operation)")
	assert.Equal(g.T(), 1, strings.Count(logStr, "/google.storage.v2.Storage/BidiWriteObject"), "Expected 1 BidiWriteObject call for writing the file")
	assert.Equal(g.T(), 1, strings.Count(logStr, "/google.storage.v2.Storage/ReadObject"), "Expected 1 ReadObject call for reading the file")
	assert.Equal(g.T(), 2, strings.Count(logStr, "/google.storage.v2.Storage/ListObjects"), "Expected 2 ListObjects calls - 1 for prefetch and 1 for listing the directory")
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
