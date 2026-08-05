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

package otel_logs

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

type OTelLogsTest struct {
	OTelLogsTestBase
}

func (t *OTelLogsTest) TestLogsAreExported() {
	// Perform a simple FS operation to trigger some logs.
	filePath := path.Join(testEnv.testDirPath, "hello.txt")
	_, err := os.Stat(filePath)
	require.NoError(t.T(), err)

	// Since logs are exported asynchronously (batch processor), we need to poll
	// the mock server to see if logs arrived.
	assert.Eventually(t.T(), func() bool {
		logMu.Lock()
		defer logMu.Unlock()
		return len(logRecords) > 0
	}, 15*time.Second, 1*time.Second, "Expected to receive OTLP logs, but got none")

	// Print some received records for debugging in case of failure or interest.
	logMu.Lock()
	defer logMu.Unlock()
	// Assert that at least one payload is received.
	assert.Greater(t.T(), len(logRecords), 0, "Expected to receive OTLP logs")
}

func TestOTelLogsTestSuite(t *testing.T) {
	ts := &OTelLogsTest{}
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		suite.Run(t, ts)
	}
}
