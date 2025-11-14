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
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PromGrpcMetricsTest is the test suite for gRPC metrics.
type PromGrpcMetricsTest struct {
	PromTestBase
}

func (testSuite *PromGrpcMetricsTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = setup.BinFile()
	testSuite.mountPoint, err = os.MkdirTemp("", "gcsfuse_monitoring_tests")
	require.NoError(testSuite.T(), err)
	setPrometheusPort(testSuite.T())

	setup.SetLogFile(fmt.Sprintf("%s%s.txt", "/tmp/gcsfuse_monitoring_test_", strings.ReplaceAll(testSuite.T().Name(), "/", "_")))
	err = testSuite.mount(getBucket(testSuite.T()))
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
	cacheDir, err := os.MkdirTemp("", "gcsfuse-cache")
	require.NoError(testSuite.T(), err)
	testSuite.T().Cleanup(func() { _ = os.RemoveAll(cacheDir) })

	// Specify client protocol to "grpc" for gRPC metrics to be emitted and captured.
	flags := []string{"--client-protocol=grpc", "--enable-grpc-metrics=true", fmt.Sprintf("--prometheus-port=%d", prometheusPort), "--cache-dir", cacheDir}
	return testSuite.mountGcsfuse(bucketName, flags)
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
