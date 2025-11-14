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
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PromGrpcTest is the test suite for gRPC metrics.
type PromGrpcMetricsTest struct {
	suite.Suite
	gcsfusePath string
	mountPoint  string
}

func (testSuite *PromGrpcMetricsTest) SetupSuite() {
	setup.IgnoreTestIfIntegrationTestFlagIsNotSet(testSuite.T())
	_, err := setup.SetUpTestDir()
	require.NoErrorf(testSuite.T(), err, "error while building GCSFuse: %p", err)
}

func (testSuite *PromGrpcMetricsTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = setup.BinFile()
	testSuite.mountPoint, err = os.MkdirTemp("", "gcsfuse_monitoring_tests")
	require.NoError(testSuite.T(), err)
	setPrometheusPort(testSuite.T())

	err = testSuite.mount(testFlatBucket)
	require.NoError(testSuite.T(), err)
}

func (testSuite *PromGrpcMetricsTest) TearDownTest() {
	if err := util.Unmount(testSuite.mountPoint); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unmount failed: %v\n", err)
	}
	os.Remove(testSuite.mountPoint)
}

func (testSuite *PromGrpcMetricsTest) mount(bucketName string) error {
	testSuite.T().Helper()
	if portAvailable := isPortOpen(prometheusPort); !portAvailable {
		require.Failf(testSuite.T(), "prometheus port is not available.", "port: %d", int64(prometheusPort))
	}
	cacheDir, err := os.MkdirTemp("", "gcsfuse-cache")
	require.NoError(testSuite.T(), err)
	testSuite.T().Cleanup(func() { _ = os.RemoveAll(cacheDir) })

	// Specify client protocol to "grpc" for gRPC metrics to be emitted and captured.
	flags := []string{"--client-protocol=grpc", fmt.Sprintf("--prometheus-port=%d", prometheusPort), "--cache-dir", cacheDir}
	args := append(flags, bucketName, testSuite.mountPoint)

	if err := mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
		return err
	}
	return nil
}

func (testSuite *PromGrpcMetricsTest) TestStorageClientGrpcMetrics() {
	_, err := os.ReadFile(path.Join(testSuite.mountPoint, "hello/hello.txt"))
	require.NoError(testSuite.T(), err)

	// Assert that gRPC metrics are present.
	assertNonZeroCountMetric(testSuite.T(), "grpc_client_attempt_started", "", "")
	assertNonZeroCountMetric(testSuite.T(), "grpc_client_attempt_started", "grpc_method", "google.storage.v2.Storage/ReadObject")
	assertNonZeroHistogramMetric(testSuite.T(), "grpc_client_attempt_duration_seconds", "", "")
	assertNonZeroHistogramMetric(testSuite.T(), "grpc_client_call_duration_seconds", "", "")
	assertNonZeroHistogramMetric(testSuite.T(), "grpc_client_attempt_rcvd_total_compressed_message_size_bytes", "", "")
	assertNonZeroHistogramMetric(testSuite.T(), "grpc_client_attempt_sent_total_compressed_message_size_bytes", "", "")
}

func TestPromGrpcMetricsSuite(t *testing.T) {
	suite.Run(t, new(PromGrpcMetricsTest))
}
