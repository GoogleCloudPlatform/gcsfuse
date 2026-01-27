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

package monitoring

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/pkg/xattr"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PromTest struct {
	PromTestBase
}

func (p *PromTest) TestStatMetrics() {
	prometheusPort := p.prometheusPort
	_, err := os.Stat(path.Join(testEnv.testDirPath, "hello.txt"))

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "LookUpInode", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "fs_ops_latency", "fs_op", "LookUpInode", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_request_count", "gcs_method", "StatObject", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "gcs_request_latencies", "gcs_method", "StatObject", prometheusPort)
}

func (p *PromTest) TestFsOpsErrorMetrics() {
	prometheusPort := p.prometheusPort
	_, err := os.Stat(path.Join(testEnv.testDirPath, "non_existent_path.txt"))
	require.Error(p.T(), err)

	assertNonZeroCountMetric(p.T(), "fs_ops_error_count", "fs_op", "LookUpInode", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "fs_ops_latency", "fs_op", "LookUpInode", prometheusPort)
}

func (p *PromTest) TestListMetrics() {
	prometheusPort := p.prometheusPort
	_, err := os.ReadDir(testEnv.testDirPath)

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "ReadDir", prometheusPort)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "OpenDir", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_request_count", "gcs_method", "ListObjects", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "gcs_request_latencies", "gcs_method", "ListObjects", prometheusPort)
}

func (p *PromTest) TestSetXAttrMetrics() {
	prometheusPort := p.prometheusPort
	err := xattr.Set(path.Join(testEnv.testDirPath, "hello.txt"), "alpha", []byte("beta"))

	require.Error(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_error_count", "fs_op", "Others", prometheusPort)
}

func (p *PromTest) TestReadMetrics() {
	prometheusPort := p.prometheusPort
	_, err := os.ReadFile(path.Join(testEnv.testDirPath, "hello.txt"))

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "file_cache_read_bytes_count", "read_type", "Sequential", prometheusPort)
	assertNonZeroCountMetric(p.T(), "file_cache_read_count", "cache_hit", "false", prometheusPort)
	assertNonZeroCountMetric(p.T(), "file_cache_read_count", "read_type", "Sequential", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "file_cache_read_latencies", "cache_hit", "false", prometheusPort)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "OpenFile", prometheusPort)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "ReadFile", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_request_count", "gcs_method", "NewReader", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_reader_count", "io_method", "opened", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_reader_count", "io_method", "closed", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_download_bytes_count", "", "", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "gcs_request_latencies", "gcs_method", "NewReader", prometheusPort)
}

func TestPromOTELSuite(t *testing.T) {
	ts := &PromTest{}
	ts.suiteName = "TestPromOTELSuite"
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		ts.prometheusPort = parsePortFromFlags(flags)
		log.Printf("Running monitoring tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
