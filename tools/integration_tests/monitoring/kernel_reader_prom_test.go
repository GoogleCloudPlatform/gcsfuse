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
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/pkg/xattr"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PromKernelReaderTest struct {
	PromTestBase
}

func (p *PromKernelReaderTest) TestKernelReaderMetrics() {
	// Create a larger file to ensure MRD is exercised
	testName := strings.ReplaceAll(p.T().Name(), "/", "_")
	gcsDir := path.Join(testDirName, testName)
	fileName := "mrd_test_file.txt"
	client.SetupFileInTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir, fileName, 10*1024*1024, p.T())

	// Read file to trigger metrics
	_, err := os.ReadFile(path.Join(testEnv.testDirPath, fileName))

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "ReadFile", p.prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_download_bytes_count", "read_type", "Parallel", p.prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_read_bytes_count", "", "", p.prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_read_count", "read_type", "Parallel", p.prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_request_count", "gcs_method", "MultiRangeDownloader::Add", p.prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "gcs_request_latencies", "gcs_method", "MultiRangeDownloader::Add", p.prometheusPort)
}

func (p *PromKernelReaderTest) TestStatMetrics() {
	prometheusPort := p.prometheusPort

	_, err := os.Stat(path.Join(testEnv.testDirPath, "hello.txt"))

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "LookUpInode", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "fs_ops_latency", "fs_op", "LookUpInode", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_request_count", "gcs_method", "StatObject", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "gcs_request_latencies", "gcs_method", "StatObject", prometheusPort)
}

func (p *PromKernelReaderTest) TestFsOpsErrorMetrics() {
	prometheusPort := p.prometheusPort

	_, err := os.Stat(path.Join(testEnv.testDirPath, "non_existent_path.txt"))

	require.Error(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_error_count", "fs_op", "LookUpInode", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "fs_ops_latency", "fs_op", "LookUpInode", prometheusPort)
}

func (p *PromKernelReaderTest) TestListMetrics() {
	prometheusPort := p.prometheusPort

	_, err := os.ReadDir(testEnv.testDirPath)

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "ReadDir", prometheusPort)
	assertNonZeroCountMetric(p.T(), "fs_ops_count", "fs_op", "OpenDir", prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_request_count", "gcs_method", "ListObjects", prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "gcs_request_latencies", "gcs_method", "ListObjects", prometheusPort)
}

func (p *PromKernelReaderTest) TestSetXAttrMetrics() {
	prometheusPort := p.prometheusPort

	err := xattr.Set(path.Join(testEnv.testDirPath, "hello.txt"), "alpha", []byte("beta"))

	require.Error(p.T(), err)
	assertNonZeroCountMetric(p.T(), "fs_ops_error_count", "fs_op", "Others", prometheusPort)
}

func TestPromKernelReaderSuite(t *testing.T) {
	ts := &PromKernelReaderTest{}
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		ts.prometheusPort = parsePortFromFlags(flags)
		log.Printf("Running prom kernel reader tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
