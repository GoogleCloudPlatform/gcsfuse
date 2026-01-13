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

package monitoring

import (
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PromGrpcMetricsTest is the test suite for gRPC metrics.
type PromGrpcMetricsTest struct {
	PromTestBase
	flags          []string
	prometheusPort int
}

func (p *PromGrpcMetricsTest) SetupSuite() {
	setup.SetUpLogFilePath("TestPromGrpcMetricsSuite", gkeTempDir, "", testEnv.cfg)
	mountGCSFuseAndSetupTestDir(p.flags, testEnv.ctx, testEnv.storageClient)
}

func (p *PromGrpcMetricsTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (p *PromGrpcMetricsTest) SetupTest() {
	// Create a new directory for each test.
	testName := strings.ReplaceAll(p.T().Name(), "/", "_")
	gcsDir := path.Join(testDirName, testName)
	testEnv.testDirPath = path.Join(mountDir, gcsDir)
	operations.CreateDirectory(testEnv.testDirPath, p.T())
	client.SetupFileInTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir, "hello.txt", 10, p.T())
}

func (p *PromGrpcMetricsTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(p.T())
}

func (p *PromGrpcMetricsTest) TestStorageClientGrpcMetrics() {
	_, err := os.ReadFile(path.Join(testEnv.testDirPath, "hello.txt"))
	require.NoError(p.T(), err)

	// Assert that gRPC metrics are present.
	if(testEnv.bucketType=="zonal") {
		assertNonZeroCountMetric(p.T(), "grpc_client_attempt_started", "grpc_method", "google.storage.v2.Storage/BidiReadObject", p.prometheusPort)
	} else {
		assertNonZeroCountMetric(p.T(), "grpc_client_attempt_started", "grpc_method", "google.storage.v2.Storage/ReadObject", p.prometheusPort)
	}
	assertNonZeroCountMetric(p.T(), "grpc_client_attempt_started", "", "", p.prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "grpc_client_attempt_duration_seconds", "", "", p.prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "grpc_client_call_duration_seconds", "", "", p.prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "grpc_client_attempt_rcvd_total_compressed_message_size_bytes", "", "", p.prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "grpc_client_attempt_sent_total_compressed_message_size_bytes", "", "", p.prometheusPort)
}

func TestPromGrpcMetricsSuite(t *testing.T) {
	ts := &PromGrpcMetricsTest{}
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		ts.prometheusPort = parsePortFromFlags(flags)
		log.Printf("Running prom grpc metrics tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
