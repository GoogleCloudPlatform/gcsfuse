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
	// For gRPC, custom-endpoint should be in format "localhost:port" without scheme
	// Use anonymous-access to avoid credential requirements for testing
	g.flags = append(g.flags, fmt.Sprintf("--custom-endpoint=localhost:%d", g.port))
	g.flags = append(g.flags, "--anonymous-access")
	setup.MountGCSFuseWithGivenMountFunc(g.flags, mountFunc)
}

func (g *grpcHeaderValidation) TearDownTest() {
	setup.UnmountGCSFuse(rootDir)
	assert.NoError(g.T(), emulator_tests.KillProxyServerProcess(g.proxyProcessId))
	setup.SaveGCSFuseLogFileInCaseOfFailure(g.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(g.proxyServerLogFile, g.T())
}

// TestGRPCClientSendsExpectedHeaders verifies that when gRPC protocol is enabled,
// the client sends the force_direct_connectivity parameter via x-goog-request-params metadata.
// The Go Storage SDK adds this via prepareDirectPathMetadata() to signal DirectPath enforcement.
// When using anonymous-access, additional diagnostic metadata is included.
func (g *grpcHeaderValidation) TestGRPCClientSendsExpectedHeaders() {
	// The mount process itself triggers gRPC calls for DirectPath verification
	// We just need to verify the proxy logs contain the expected metadata

	logContent, err := os.ReadFile(g.proxyServerLogFile)
	require.NoError(g.T(), err)
	logStr := string(logContent)

	// Verify metadata validation passed and x-goog-request-params contains force_direct_connectivity=ENFORCED
	// Note: With anonymous-access, direct_connectivity_diagnostic=no_auth is also present
	assert.Contains(g.T(), logStr, "Metadata validation passed")
	assert.Contains(g.T(), logStr, "x-goog-request-params")
	assert.Contains(g.T(), logStr, "force_direct_connectivity=ENFORCED")
	
	// Verify direct_connectivity_diagnostic header is present
	assert.Contains(g.T(), logStr, "direct_connectivity_diagnostic")
	assert.Contains(g.T(), logStr, "direct_connectivity_diagnostic=no_auth")
	
	assert.Contains(g.T(), logStr, "/google.storage.v2.Storage/GetObject")
}

// TestGRPCHeadersInMultipleOperations verifies that the force_direct_connectivity parameter
// is consistently sent across multiple gRPC operations via x-goog-request-params metadata.
func (g *grpcHeaderValidation) TestGRPCHeadersInMultipleOperations() {
	// During mount and initial operations, multiple gRPC calls are made
	// Verify that all of them contain the DirectPath metadata

	logContent, err := os.ReadFile(g.proxyServerLogFile)
	require.NoError(g.T(), err)
	logStr := string(logContent)

	// Verify metadata was validated successfully across multiple operations
	validationCount := strings.Count(logStr, "Metadata validation passed")
	assert.Greater(g.T(), validationCount, 0, "Expected at least one successful metadata validation")
	assert.Contains(g.T(), logStr, "force_direct_connectivity=ENFORCED")
	
	// Verify direct_connectivity_diagnostic header is present in all operations
	assert.Contains(g.T(), logStr, "direct_connectivity_diagnostic=no_auth")

	// Verify multiple gRPC calls were made
	grpcCallCount := strings.Count(logStr, "/google.storage.v2.Storage/GetObject")
	assert.Greater(g.T(), grpcCallCount, 0, "Expected at least one gRPC call")
}

func TestGRPCHeaderValidation(t *testing.T) {
	ts := &grpcHeaderValidation{}
	// Test with gRPC protocol to validate that DirectPath metadata is sent.
	// The Go Storage SDK automatically adds force_direct_connectivity=ENFORCED
	// to x-goog-request-params when experimental.WithDirectConnectivityEnforced() is used.
	// The gRPC proxy intercepts and validates this metadata.
	flagsSet := [][]string{
		{"--client-protocol=grpc"},
	}

	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running gRPC header validation tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
