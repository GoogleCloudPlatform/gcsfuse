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
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/pkg/xattr"
	promclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testDirName = "monitoring"
	gkeTempDir  = "/gcsfuse-tmp"
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
}

var (
	testEnv   env
	mountFunc func(*test_suite.TestConfig, []string) error
	mountDir  string
	rootDir   string
)

// PromTestBase preserves the base struct as requested.
type PromTestBase struct {
	suite.Suite
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	if testEnv.cfg.GKEMountedDirectory != "" {
		setup.SetMntDir(testEnv.cfg.GKEMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	configFile := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(configFile.Monitoring) == 0 {
		log.Println("No configuration found for monitoring tests in config. Using default flags.")
		configFile.Monitoring = make([]test_suite.TestConfig, 1)
		testEnv.cfg = &configFile.Monitoring[0]
		testEnv.cfg.TestBucket = setup.TestBucket()
		testEnv.cfg.LogFile = setup.LogFile()
		testEnv.cfg.GKEMountedDirectory = setup.MountedDirectory()

		testEnv.cfg.Configs = make([]test_suite.ConfigItem, 5)
		testEnv.cfg.Configs[0].Flags = []string{"--prometheus-port=9190 --cache-dir=/tmp/gcsfuse-cache --log-file=/gcsfuse-tmp/monitoring.log"}
		testEnv.cfg.Configs[0].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
		testEnv.cfg.Configs[0].Run = "TestPromOTELSuite"
		testEnv.cfg.Configs[1].Flags = []string{"--prometheus-port=10190 --cache-dir=/tmp/gcsfuse-cache --log-file=/gcsfuse-tmp/monitoring_hns.log"}
		testEnv.cfg.Configs[1].Compatible = map[string]bool{"flat": false, "hns": true, "zonal": true}
		testEnv.cfg.Configs[1].Run = "TestPromOTELSuite"

		testEnv.cfg.Configs[2].Flags = []string{"--prometheus-port=9191 --enable-buffered-read --read-block-size-mb=4 --read-random-seek-threshold=2 --read-global-max-blocks=5 --read-min-blocks-per-handle=2 --read-start-blocks-per-handle=2 --log-file=/gcsfuse-tmp/prom_buffered_read.log"}
		testEnv.cfg.Configs[2].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
		testEnv.cfg.Configs[2].Run = "TestPromBufferedReadSuite"
		testEnv.cfg.Configs[3].Flags = []string{"--prometheus-port=10191 --enable-buffered-read --read-block-size-mb=4 --read-random-seek-threshold=2 --read-global-max-blocks=5 --read-min-blocks-per-handle=2 --read-start-blocks-per-handle=2 --log-file=/gcsfuse-tmp/prom_buffered_read_hns.log"}
		testEnv.cfg.Configs[3].Compatible = map[string]bool{"flat": false, "hns": true, "zonal": true}
		testEnv.cfg.Configs[3].Run = "TestPromBufferedReadSuite"
		
		testEnv.cfg.Configs[4].Flags = []string{"--client-protocol=grpc --experimental-enable-grpc-metrics=true --prometheus-port=9192 --log-file=/gcsfuse-tmp/prom_grpc_metrics.log",}
		testEnv.cfg.Configs[4].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		testEnv.cfg.Configs[4].Run = "TestPromGrpcMetricsSuite"
	}
	testEnv.cfg = &configFile.Monitoring[0]
	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)

	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	defer testEnv.storageClient.Close()

	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		mountDir = testEnv.cfg.GKEMountedDirectory
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

type PromTest struct {
	PromTestBase
	flags          []string
	prometheusPort int
}

func (p *PromTest) SetupSuite() {
	setup.SetUpLogFilePath("TestPromOTELSuite", gkeTempDir, "", testEnv.cfg)
	mountGCSFuseAndSetupTestDir(p.flags, testEnv.ctx, testEnv.storageClient)
}

func (p *PromTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (p *PromTest) SetupTest() {
	// Create a new directory for each test.
	testName := strings.ReplaceAll(p.T().Name(), "/", "_")
	gcsDir := path.Join(testDirName, testName)
	testEnv.testDirPath = path.Join(mountDir, gcsDir)
	operations.CreateDirectory(testEnv.testDirPath, p.T())
	client.SetupFileInTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir, "hello.txt", 10, p.T())
}

func (p *PromTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(p.T())
}

func parsePromFormat(t *testing.T, prometheusPort int) (map[string]*promclient.MetricFamily, error) {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", prometheusPort))
	require.NoError(t, err)
	parser := expfmt.NewTextParser(model.UTF8Validation)
	return parser.TextToMetricFamilies(resp.Body)
}

// assertNonZeroCountMetric asserts that the specified count metric is present and is positive in the Prometheus export
func assertNonZeroCountMetric(t *testing.T, metricName, labelName, labelValue string, prometheusPort int) {
	t.Helper()
	mf, err := parsePromFormat(t, prometheusPort)
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
func assertNonZeroHistogramMetric(t *testing.T, metricName, labelName, labelValue string, prometheusPort int) {
	t.Helper()
	mf, err := parsePromFormat(t, prometheusPort)
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

func parsePortFromFlags(flags []string) int {
	for _, flagStr := range flags {
		parts := strings.Split(flagStr, " ")
		for _, part := range parts {
			if strings.HasPrefix(part, "--prometheus-port=") {
				portStr := strings.TrimPrefix(part, "--prometheus-port=")
				port, _ := strconv.Atoi(portStr)
				return port
			}
		}
	}
	return 0
}

func TestPromOTELSuite(t *testing.T) {
	ts := &PromTest{}
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		ts.prometheusPort = parsePortFromFlags(flags)
		log.Printf("Running monitoring tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
