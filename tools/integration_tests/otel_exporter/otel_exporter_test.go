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

package otel_exporter

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OTelExporterTest struct {
	OTelExporterTestBase
}

func (t *OTelExporterTest) TestLogsAreExported() {
	// Perform a simple FS operation to trigger some logs.
	filePath := path.Join(testEnv.testDirPath, "hello.txt")
	_, err := os.Stat(filePath)
	require.NoError(t.T(), err)

	// Since logs are exported asynchronously (batch processor), we need to poll
	// the mock server to see if logs arrived.
	assert.Eventually(t.T(), func() bool {
		recordsMu.Lock()
		defer recordsMu.Unlock()
		return len(logRecords) > 0
	}, 15*time.Second, 1*time.Second, "Expected to receive OTLP logs, but got none")

	recordsMu.Lock()
	defer recordsMu.Unlock()
	assert.Greater(t.T(), len(logRecords), 0, "Expected to receive OTLP logs")
}

func (t *OTelExporterTest) TestMetricsAreExported() {
	if testEnv.cfg.GKEMountedDirectory != "" {
		t.T().Skip("Skipping otel_metrics test on GKE")
	}

	// Perform a simple FS operation to ensure baseline metrics are gathered.
	filePath := path.Join(testEnv.testDirPath, "hello.txt")
	_, err := os.Stat(filePath)
	require.NoError(t.T(), err)

	// Since metrics are exported periodically (every 5 seconds based on our config),
	// we need to poll the mock server to see if metrics arrived.
	assert.Eventually(t.T(), func() bool {
		recordsMu.Lock()
		defer recordsMu.Unlock()
		return len(metricRecords) > 0
	}, 20*time.Second, 1*time.Second, "Expected to receive OTLP metrics, but got none")

	recordsMu.Lock()
	defer recordsMu.Unlock()
	assert.Greater(t.T(), len(metricRecords), 0, "Expected to receive OTLP metrics")
}

func TestOTelExporterTestSuite(t *testing.T) {
	ts := &OTelExporterTest{}
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		suite.Run(t, ts)
	}
}
