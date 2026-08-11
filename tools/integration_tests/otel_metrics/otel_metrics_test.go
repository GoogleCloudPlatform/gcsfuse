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

package otel_metrics

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

type OTelMetricsTest struct {
	OTelMetricsTestBase
}

func (t *OTelMetricsTest) TestMetricsAreExported() {
	// Perform a simple FS operation to ensure baseline metrics are gathered.
	filePath := path.Join(testEnv.testDirPath, "hello.txt")
	_, err := os.Stat(filePath)
	require.NoError(t.T(), err)

	// Since metrics are exported periodically (every 5 seconds based on our config),
	// we need to poll the mock server to see if metrics arrived.
	assert.Eventually(t.T(), func() bool {
		metricMu.Lock()
		defer metricMu.Unlock()
		return len(metricRecords) > 0
	}, 20*time.Second, 1*time.Second, "Expected to receive OTLP metrics, but got none")

	// Assert that at least one payload is received.
	metricMu.Lock()
	defer metricMu.Unlock()
	assert.Greater(t.T(), len(metricRecords), 0, "Expected to receive OTLP metrics")
}

func TestOTelMetricsTestSuite(t *testing.T) {
	ts := &OTelMetricsTest{}
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		suite.Run(t, ts)
	}
}
