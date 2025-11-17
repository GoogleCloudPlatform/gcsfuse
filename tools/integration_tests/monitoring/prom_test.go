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

	"github.com/pkg/xattr"
	promclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/util"
)

const (
	testHNSBucket  = "gcsfuse_monitoring_test_bucket"
	testFlatBucket = "gcsfuse_monitoring_test_bucket_flat"
)

var (
	portNonHNSRun = 9190
	portHNSRun    = 10190
)

var prometheusPort int

func setPrometheusPort(t *testing.T) {
	if isHNSTestRun(t) {
		prometheusPort = portHNSRun
		portHNSRun++
		return
	}
	prometheusPort = portNonHNSRun
	portNonHNSRun++
}

func getBucket(t *testing.T) string {
	if isHNSTestRun(t) {
		return testHNSBucket
	}
	return testFlatBucket
}

func isPortOpen(port int) bool {
	c := exec.Command("lsof", "-t", fmt.Sprintf("-i:%d", port))
	output, _ := c.CombinedOutput()
	return len(output) == 0
}

type PromTestBase struct {
	suite.Suite
	gcsfusePath string
	mountPoint  string
}

func (testSuite *PromTestBase) mountGcsfuse(bucketName string, flags []string) error {
	testSuite.T().Helper()
	if portAvailable := isPortOpen(prometheusPort); !portAvailable {
		require.Failf(testSuite.T(), "prometheus port is not available.", "port: %d", int64(prometheusPort))
	}
	args := append(flags, bucketName, testSuite.mountPoint)

	if err := mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
		return err
	}
	return nil
}

func (testSuite *PromTestBase) SetupSuite() {
	setup.IgnoreTestIfIntegrationTestFlagIsNotSet(testSuite.T())
	_, err := setup.SetUpTestDir()
	require.NoError(testSuite.T(), err, "error while building GCSFuse")
}

func (testSuite *PromTestBase) TearDownTest() {
	if err := util.Unmount(testSuite.mountPoint); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unmount failed: %v\n", err)
	}
	require.True(testSuite.T(), isPortOpen(prometheusPort))

	err := os.Remove(testSuite.mountPoint)
	assert.NoError(testSuite.T(), err)
}

type PromTest struct {
	PromTestBase
}

// isHNSTestRun returns true if the bucket is an HNS bucket.
func isHNSTestRun(t *testing.T) bool {
	storageClient, err := client.CreateStorageClient(context.Background())
	require.NoError(t, err, "error while creating storage client")
	defer storageClient.Close()
	return setup.ResolveIsHierarchicalBucket(context.Background(), setup.TestBucket(), storageClient)
}

func (testSuite *PromTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = setup.BinFile()
	testSuite.mountPoint, err = os.MkdirTemp("", "gcsfuse_monitoring_tests")
	require.NoError(testSuite.T(), err)
	setPrometheusPort(testSuite.T())

	setup.SetLogFile(fmt.Sprintf("%s%s.txt", "/tmp/gcsfuse_monitoring_test_", strings.ReplaceAll(testSuite.T().Name(), "/", "_")))
	err = testSuite.mount(getBucket(testSuite.T()))
	require.NoError(testSuite.T(), err)
}

func (testSuite *PromTest) mount(bucketName string) error {
	testSuite.T().Helper()
	cacheDir, err := os.MkdirTemp("", "gcsfuse-cache")
	require.NoError(testSuite.T(), err)
	testSuite.T().Cleanup(func() { _ = os.RemoveAll(cacheDir) })

	flags := []string{fmt.Sprintf("--prometheus-port=%d", prometheusPort), "--cache-dir", cacheDir}
	return testSuite.mountGcsfuse(bucketName, flags)
}

func parsePromFormat(t *testing.T) (map[string]*promclient.MetricFamily, error) {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", prometheusPort))
	require.NoError(t, err)
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(resp.Body)
}

// assertNonZeroCountMetric asserts that the specified count metric is present and is positive in the Prometheus export
func assertNonZeroCountMetric(t *testing.T, metricName, labelName, labelValue string) {
	t.Helper()
	mf, err := parsePromFormat(t)
	require.NoError(t, err)
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
	assert.Fail(t, fmt.Sprintf("Didn't find the metric with name: %s, labelName: %s and labelValue: %s",
		metricName, labelName, labelValue))
}

// assertNonZeroHistogramMetric asserts that the specified histogram metric is present and is positive for at least one of the buckets in the Prometheus export.
func assertNonZeroHistogramMetric(t *testing.T, metricName, labelName, labelValue string) {
	t.Helper()
	mf, err := parsePromFormat(t)
	require.NoError(t, err)

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
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_count", "fs_op", "LookUpInode")
	assertNonZeroHistogramMetric(testSuite.T(), "fs_ops_latency", "fs_op", "LookUpInode")
	assertNonZeroCountMetric(testSuite.T(), "gcs_request_count", "gcs_method", "StatObject")
	assertNonZeroHistogramMetric(testSuite.T(), "gcs_request_latencies", "gcs_method", "StatObject")
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_count", "fs_op", "LookUpInode")
	assertNonZeroHistogramMetric(testSuite.T(), "fs_ops_latency", "fs_op", "LookUpInode")
	assertNonZeroCountMetric(testSuite.T(), "gcs_request_count", "gcs_method", "StatObject")
	assertNonZeroHistogramMetric(testSuite.T(), "gcs_request_latencies", "gcs_method", "StatObject")
}

func (testSuite *PromTest) TestFsOpsErrorMetrics() {
	_, err := os.Stat(path.Join(testSuite.mountPoint, "non_existent_path.txt"))
	require.Error(testSuite.T(), err)

	assertNonZeroCountMetric(testSuite.T(), "fs_ops_error_count", "fs_op", "LookUpInode")
	assertNonZeroHistogramMetric(testSuite.T(), "fs_ops_latency", "fs_op", "LookUpInode")
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_error_count", "fs_op", "LookUpInode")
	assertNonZeroHistogramMetric(testSuite.T(), "fs_ops_latency", "fs_op", "LookUpInode")
}

func (testSuite *PromTest) TestListMetrics() {
	_, err := os.ReadDir(path.Join(testSuite.mountPoint, "hello"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_count", "fs_op", "ReadDir")
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_count", "fs_op", "OpenDir")
	assertNonZeroCountMetric(testSuite.T(), "gcs_request_count", "gcs_method", "ListObjects")
	assertNonZeroHistogramMetric(testSuite.T(), "gcs_request_latencies", "gcs_method", "ListObjects")
}

func (testSuite *PromTest) TestSetXAttrMetrics() {
	err := xattr.Set(path.Join(testSuite.mountPoint, "hello/hello.txt"), "alpha", []byte("beta"))

	require.Error(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_error_count", "fs_op", "Others")
}

func (testSuite *PromTest) TestReadMetrics() {
	_, err := os.ReadFile(path.Join(testSuite.mountPoint, "hello/hello.txt"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite.T(), "file_cache_read_bytes_count", "read_type", "Sequential")
	assertNonZeroCountMetric(testSuite.T(), "file_cache_read_count", "cache_hit", "false")
	assertNonZeroCountMetric(testSuite.T(), "file_cache_read_count", "read_type", "Sequential")
	assertNonZeroHistogramMetric(testSuite.T(), "file_cache_read_latencies", "cache_hit", "false")
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_count", "fs_op", "OpenFile")
	assertNonZeroCountMetric(testSuite.T(), "fs_ops_count", "fs_op", "ReadFile")
	assertNonZeroCountMetric(testSuite.T(), "gcs_request_count", "gcs_method", "NewReader")
	assertNonZeroCountMetric(testSuite.T(), "gcs_reader_count", "io_method", "opened")
	assertNonZeroCountMetric(testSuite.T(), "gcs_reader_count", "io_method", "closed")
	assertNonZeroCountMetric(testSuite.T(), "gcs_download_bytes_count", "", "")
	assertNonZeroHistogramMetric(testSuite.T(), "gcs_request_latencies", "gcs_method", "NewReader")
}

func TestPromOTELSuite(t *testing.T) {
	suite.Run(t, new(PromTest))
}
