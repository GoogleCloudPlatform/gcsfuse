// Copyright 2024 Google LLC
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
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/util"
	promclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testBucket    = "gcsfuse_monitoring_test_bucket"
	portNonHNSRun = 9191
	portHNSRun    = 9192
)

var prometheusPort = portNonHNSRun

func isPortOpen(port int) bool {
	c := exec.Command("lsof", "-t", fmt.Sprintf("-i:%d", port))
	output, _ := c.CombinedOutput()
	return len(output) == 0
}

type PromTest struct {
	suite.Suite
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	mountPoint string

	enableOTEL bool
}

// isHNSTestRun returns true if the bucket is an HNS bucket.
func isHNSTestRun(t *testing.T) bool {
	storageClient, err := client.CreateStorageClient(context.Background())
	require.NoError(t, err, "error while creating storage client")
	defer storageClient.Close()
	return setup.IsHierarchicalBucket(context.Background(), storageClient)
}

func (testSuite *PromTest) SetupSuite() {
	setup.IgnoreTestIfIntegrationTestFlagNotIsSet(testSuite.T())
	if isHNSTestRun(testSuite.T()) {
		// sets different Prometheus ports for HNS and non-HNS presubmit runs.
		// This ensures that there is no port contention if both HNS and non-HNS test runs are happening simultaneously.
		prometheusPort = portHNSRun
	}

	err := setup.SetUpTestDir()
	require.NoErrorf(testSuite.T(), err, "error while building GCSFuse: %p", err)
}

func (testSuite *PromTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = setup.BinFile()
	testSuite.mountPoint, err = os.MkdirTemp("", "gcsfuse_monitoring_tests")
	require.NoError(testSuite.T(), err)

	setup.SetLogFile(fmt.Sprintf("%s%s.txt", "/tmp/gcsfuse_monitoring_test_", strings.ReplaceAll(testSuite.T().Name(), "/", "_")))
	err = testSuite.mount(testBucket)
	require.NoError(testSuite.T(), err)
}

func (testSuite *PromTest) TearDownTest() {
	if err := util.Unmount(testSuite.mountPoint); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unmount failed: %v\n", err)
	}
	require.True(testSuite.T(), isPortOpen(prometheusPort))

	err := os.Remove(testSuite.mountPoint)
	assert.NoError(testSuite.T(), err)
}

func (testSuite *PromTest) TearDownSuite() {
	os.RemoveAll(setup.TestDir())
}

func (testSuite *PromTest) mount(bucketName string) error {
	testSuite.T().Helper()
	if portAvailable := isPortOpen(prometheusPort); !portAvailable {
		require.Failf(testSuite.T(), "prometheus port is not available.", "port: %d", int64(prometheusPort))
	}
	cacheDir, err := os.MkdirTemp("", "gcsfuse-cache")
	require.NoError(testSuite.T(), err)
	testSuite.T().Cleanup(func() { _ = os.RemoveAll(cacheDir) })

	flags := []string{fmt.Sprintf("--prometheus-port=%d", prometheusPort), "--cache-dir", cacheDir}
	if testSuite.enableOTEL {
		flags = append(flags, "--enable-otel=true")
	} else {
		flags = append(flags, "--enable-otel=false")
	}
	args := append(flags, bucketName, testSuite.mountPoint)

	if err := mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
		return err
	}
	return nil
}

func parsePromFormat(testSuite *PromTest) (map[string]*promclient.MetricFamily, error) {
	testSuite.T().Helper()

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", prometheusPort))
	require.NoError(testSuite.T(), err)
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(resp.Body)
}

// assertNonZeroCountMetric asserts that the specified count metric is present and is positive in the Prometheus export
func assertNonZeroCountMetric(testSuite *PromTest, metricName, labelName, labelValue string) {
	testSuite.T().Helper()
	mf, err := parsePromFormat(testSuite)
	require.NoError(testSuite.T(), err)
	for k, v := range mf {
		if k != metricName || *v.Type != promclient.MetricType_COUNTER {
			continue
		}
		for _, m := range v.Metric {
			if *m.Counter.Value <= 0 {
				continue
			}
			if labelName == "" {
				return
			}
			for _, l := range m.GetLabel() {
				if *l.Name == labelName && *l.Value == labelValue {
					return
				}
			}
		}

	}
	assert.Fail(testSuite.T(), "Didn't find the metric with name: %s, labelName: %s and labelValue: %s", metricName, labelName, labelValue)
}

// assertNonZeroHistogramMetric asserts that the specified histogram metric is present and is positive for at least one of the buckets in the Prometheus export.
func assertNonZeroHistogramMetric(testSuite *PromTest, metricName, labelName, labelValue string) {
	testSuite.T().Helper()
	mf, err := parsePromFormat(testSuite)
	require.NoError(testSuite.T(), err)

	for k, v := range mf {
		if k != metricName || *v.Type != promclient.MetricType_HISTOGRAM {
			continue
		}
		for _, m := range v.Metric {
			for _, bkt := range m.GetHistogram().Bucket {
				if bkt.CumulativeCount == nil || *bkt.CumulativeCount == 0 {
					continue
				}
				if labelName == "" {
					return
				}
				for _, l := range m.GetLabel() {
					if *l.Name == labelName && *l.Value == labelValue {
						return
					}
				}
			}
		}
	}
}

func (testSuite *PromTest) TestStatMetrics() {
	_, err := os.Stat(path.Join(testSuite.mountPoint, "hello/hello.txt"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "LookUpInode")
	assertNonZeroHistogramMetric(testSuite, "fs_ops_latency", "fs_op", "LookUpInode")
	assertNonZeroCountMetric(testSuite, "gcs_request_count", "gcs_method", "StatObject")
	assertNonZeroHistogramMetric(testSuite, "gcs_request_latencies", "gcs_method", "StatObject")
}

func (testSuite *PromTest) TestFsOpsErrorMetrics() {
	_, err := os.Stat(path.Join(testSuite.mountPoint, "non_existent_path.txt"))
	require.Error(testSuite.T(), err)

	assertNonZeroCountMetric(testSuite, "fs_ops_error_count", "fs_op", "LookUpInode")
	assertNonZeroHistogramMetric(testSuite, "fs_ops_latency", "fs_op", "LookUpInode")
}

func (testSuite *PromTest) TestListMetrics() {
	_, err := os.ReadDir(path.Join(testSuite.mountPoint, "hello"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "ReadDir")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "OpenDir")
	assertNonZeroCountMetric(testSuite, "gcs_request_count", "gcs_method", "ListObjects")
	assertNonZeroHistogramMetric(testSuite, "gcs_request_latencies", "gcs_method", "ListObjects")
}

func (testSuite *PromTest) TestReadMetrics() {
	_, err := os.ReadFile(path.Join(testSuite.mountPoint, "hello/hello.txt"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite, "file_cache_read_count", "cache_hit", "false")
	assertNonZeroCountMetric(testSuite, "file_cache_read_count", "read_type", "Sequential")
	assertNonZeroCountMetric(testSuite, "file_cache_read_bytes_count", "read_type", "Sequential")
	assertNonZeroHistogramMetric(testSuite, "file_cache_read_latencies", "cache_hit", "false")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "OpenFile")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "ReadFile")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "ReadFile")
	assertNonZeroCountMetric(testSuite, "gcs_request_count", "gcs_method", "NewReader")
	assertNonZeroCountMetric(testSuite, "gcs_reader_count", "io_method", "opened")
	assertNonZeroCountMetric(testSuite, "gcs_reader_count", "io_method", "closed")
	assertNonZeroCountMetric(testSuite, "gcs_read_count", "read_type", "Sequential")
	assertNonZeroCountMetric(testSuite, "gcs_download_bytes_count", "", "")
	assertNonZeroCountMetric(testSuite, "gcs_read_bytes_count", "", "")
	assertNonZeroHistogramMetric(testSuite, "gcs_request_latencies", "gcs_method", "NewReader")
}

func TestPromOCSuite(t *testing.T) {
	suite.Run(t, &PromTest{enableOTEL: false})
}

func TestPromOTELSuite(t *testing.T) {
	suite.Run(t, &PromTest{enableOTEL: true})
}
