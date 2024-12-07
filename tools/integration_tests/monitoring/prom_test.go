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
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/util"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// A directory containing outputs created by build_gcsfuse.
var gBuildDir string

const testBucket string = "gcsfuse_monitoring_test_bucket"

type PromTest struct {
	suite.Suite
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	mountPoint string
}

func (testSuite *PromTest) SetupSuite() {
	setup.ParseSetUpFlagsForStretchrTests(testSuite.T())

	var err error

	if setup.TestInstalledPackage() {
		// when testInstalledPackage flag is set, gcsfuse is preinstalled on the
		// machine. Hence, here we are overwriting gBuildDir to /.
		gBuildDir = "/"
		return
	}

	// To test locally built package
	// Set up a directory into which we will build.
	if gBuildDir, err = os.MkdirTemp("", "gcsfuse_integration_tests"); err != nil {
		log.Fatalf("TempDir: %p", err)
		return
	}

	// Build into that directory.
	if err = util.BuildGcsfuse(gBuildDir); err != nil {
		testSuite.T().Fatalf("buildGcsfuse: %p", err)
		return
	}
}

func (testSuite *PromTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
	testSuite.mountPoint, err = os.MkdirTemp("", "gcsfuse_monitoring_tests")
	require.NoError(testSuite.T(), err)

	setup.SetLogFile(fmt.Sprintf("%s%s.txt", "/tmp/gcsfuse_monitoring_", strings.ReplaceAll(testSuite.T().Name(), "/", "_")))
	err = testSuite.mount(testBucket)
	require.NoError(testSuite.T(), err)
}

func (testSuite *PromTest) TearDownTest() {
	if err := util.Unmount(testSuite.mountPoint); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unmount failed: %v\n", err)
	}

	err := os.Remove(testSuite.mountPoint)
	assert.NoError(testSuite.T(), err)
}

func (testSuite *PromTest) TearDownSuite() {
	os.RemoveAll(gBuildDir)
}

func (testSuite *PromTest) mount(bucketName string) error {
	testSuite.T().Helper()
	cacheDir, err := os.MkdirTemp("", "gcsfuse-cache")
	require.NoError(testSuite.T(), err)
	args := []string{"--prometheus-port=9191", "--cache-dir", cacheDir, bucketName, testSuite.mountPoint}

	if err := mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
		return err
	}
	return nil
}

func parsePromFormat(testSuite *PromTest) (map[string]*io_prometheus_client.MetricFamily, error) {
	testSuite.T().Helper()

	resp, err := http.Get("http://localhost:9191/metrics")
	require.NoError(testSuite.T(), err)
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(resp.Body)
}

func assertNonZeroCountMetric(testSuite *PromTest, metricName, labelName, labelValue string) {
	testSuite.T().Helper()
	mf, err := parsePromFormat(testSuite)
	require.NoError(testSuite.T(), err)
	for k, b := range mf {
		if k != metricName || *b.Type != io_prometheus_client.MetricType_COUNTER {
			continue
		}
		for _, m := range b.Metric {
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

func assertNonZeroLatencyMetric(testSuite *PromTest, metricName, labelName, labelValue string) {
	testSuite.T().Helper()
	mf, err := parsePromFormat(testSuite)
	require.NoError(testSuite.T(), err)

	for k, v := range mf {
		if k != metricName || *v.Type != io_prometheus_client.MetricType_HISTOGRAM {
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
	assertNonZeroLatencyMetric(testSuite, "fs_ops_latency", "fs_op", "LookUpInode")
	assertNonZeroCountMetric(testSuite, "gcs_request_count", "gcs_method", "StatObject")
	assertNonZeroLatencyMetric(testSuite, "gcs_request_latencies", "gcs_method", "StatObject")
}

func (testSuite *PromTest) TestFsOpsErrorMetrics() {
	_, err := os.Stat(path.Join(testSuite.mountPoint, "non_existent_path.txt"))
	require.Error(testSuite.T(), err)

	assertNonZeroCountMetric(testSuite, "fs_ops_error_count", "fs_op", "LookUpInode")
	assertNonZeroLatencyMetric(testSuite, "fs_ops_latency", "fs_op", "LookUpInode")
}

func (testSuite *PromTest) TestListMetrics() {
	_, err := os.ReadDir(path.Join(testSuite.mountPoint, "hello"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "ReadDir")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "OpenDir")
	assertNonZeroCountMetric(testSuite, "gcs_request_count", "gcs_method", "ListObjects")
	assertNonZeroLatencyMetric(testSuite, "gcs_request_latencies", "gcs_method", "ListObjects")
}

func (testSuite *PromTest) TestReadMetrics() {
	_, err := os.ReadFile(path.Join(testSuite.mountPoint, "hello/hello.txt"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite, "file_cache_read_count", "cache_hit", "false")
	assertNonZeroCountMetric(testSuite, "file_cache_read_count", "read_type", "Sequential")
	assertNonZeroCountMetric(testSuite, "file_cache_read_bytes_count", "read_type", "Sequential")
	assertNonZeroLatencyMetric(testSuite, "file_cache_read_latencies", "cache_hit", "false")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "OpenFile")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "ReadFile")
	assertNonZeroCountMetric(testSuite, "fs_ops_count", "fs_op", "ReadFile")
	assertNonZeroCountMetric(testSuite, "gcs_request_count", "gcs_method", "NewReader")
	assertNonZeroCountMetric(testSuite, "gcs_reader_count", "io_method", "opened")
	assertNonZeroCountMetric(testSuite, "gcs_reader_count", "io_method", "closed")
	assertNonZeroCountMetric(testSuite, "gcs_read_count", "read_type", "Sequential")
	assertNonZeroCountMetric(testSuite, "gcs_download_bytes_count", "", "")
	assertNonZeroCountMetric(testSuite, "gcs_read_bytes_count", "", "")
	assertNonZeroLatencyMetric(testSuite, "gcs_request_latencies", "gcs_method", "NewReader")

}

func TestPromSuite(t *testing.T) {
	suite.Run(t, new(PromTest))
}
