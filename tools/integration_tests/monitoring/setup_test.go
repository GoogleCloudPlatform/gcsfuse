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
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	promclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
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
)

// PromTestBase preserves the base struct and common methods.
type PromTestBase struct {
	suite.Suite
	flags          []string
	prometheusPort int
	suiteName      string
}

func (p *PromTestBase) SetupSuite() {
	setup.SetUpLogFilePath(p.T().Name(), p.flags, gkeTempDir, "", testEnv.cfg)
	mountGCSFuseAndSetupTestDir(p.flags, testEnv.ctx, testEnv.storageClient)
}

func (p *PromTestBase) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (p *PromTestBase) SetupTest() {
	testName := strings.ReplaceAll(p.T().Name(), "/", "_")
	gcsDir := path.Join(testDirName, testName)
	// Use the setup helper to prepare the test directory.
	testEnv.testDirPath = client.SetupTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir)
	// Setup a standard hello.txt file for metrics collection.
	client.SetupFileInTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir, "hello.txt", 10, p.T())
}

func (p *PromTestBase) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(p.T())
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	if testEnv.cfg.GKEMountedDirectory != "" {
		setup.SetMntDir(testEnv.cfg.GKEMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
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

func parsePromFormat(t *testing.T, prometheusPort int) (map[string]*promclient.MetricFamily, error) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", prometheusPort))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	parser := expfmt.NewTextParser(model.UTF8Validation)
	return parser.TextToMetricFamilies(resp.Body)
}

func assertNonZeroCountMetric(t *testing.T, metricName, labelName, labelValue string, prometheusPort int) {
	t.Helper()
	mf, err := parsePromFormat(t, prometheusPort)
	require.NoError(t, err)
	for k, v := range mf {
		if k != metricName || *v.Type != promclient.MetricType_COUNTER {
			continue
		}
		for _, m := range v.Metric {
			if labelName != "" {
				for _, l := range m.Label {
					if *l.Name == labelName && *l.Value == labelValue && *m.Counter.Value > 0 {
						return
					}
				}
			} else if *m.Counter.Value > 0 {
				return
			}
		}
	}
	require.Fail(t, fmt.Sprintf("Metric %s with label %s=%s not found or zero", metricName, labelName, labelValue))
}

func assertNonZeroHistogramMetric(t *testing.T, metricName, labelName, labelValue string, prometheusPort int) {
	t.Helper()
	mf, err := parsePromFormat(t, prometheusPort)
	require.NoError(t, err)
	for k, v := range mf {
		if k != metricName || *v.Type != promclient.MetricType_HISTOGRAM {
			continue
		}
		for _, m := range v.Metric {
			if labelName != "" {
				for _, l := range m.Label {
					if *l.Name == labelName && *l.Value == labelValue && *m.Histogram.SampleCount > 0 {
						return
					}
				}
			} else if *m.Histogram.SampleCount > 0 {
				return
			}
		}
	}
	require.Fail(t, fmt.Sprintf("Metric %s with label %s=%s not found or zero", metricName, labelName, labelValue))
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	configFile := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(configFile.Monitoring) == 0 {
		log.Fatal("No configuration found for Monitoring in config file.")
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
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
