// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitoring

import (
	"io"
	"log"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PromBufferedReadTest struct {
	PromTestBase
	flags          []string
	prometheusPort int
}

func (p *PromBufferedReadTest) SetupSuite() {
	setup.SetUpLogFilePath("TestPromBufferedReadSuite", gkeTempDir, "", testEnv.cfg)
	mountGCSFuseAndSetupTestDir(p.flags, testEnv.ctx, testEnv.storageClient)
}

func (p *PromBufferedReadTest) TestBufferedReadMetrics() {
	_, err := operations.ReadFile(path.Join(testEnv.testDirPath, "hello.txt"))

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "gcs_read_bytes_count", "", "", p.prometheusPort)
	assertNonZeroCountMetric(p.T(), "gcs_download_bytes_count", "read_type", "Buffered", p.prometheusPort)
	assertNonZeroHistogramMetric(p.T(), "buffered_read/read_latency", "", "", p.prometheusPort)
}

func (p *PromBufferedReadTest) TestRandomReadFallback() {
	const blockSize = 4 * 1024 * 1024
	const fileSize = 4 * blockSize
	const fileName = "random_read_fallback.txt"
	filePath := path.Join(testEnv.testDirPath, fileName)
	operations.CreateFileOfSize(fileSize, filePath, p.T())
	f, err := operations.OpenFileAsReadonly(filePath)
	require.NoError(p.T(), err)
	defer operations.CloseFileShouldNotThrowError(p.T(), f)
	buf := make([]byte, 10)
	// With random-seek-threshold: 2, the 3rd random read should trigger a fallback.
	// First random read.
	_, err = f.ReadAt(buf, 3*blockSize+100)
	require.NoError(p.T(), err, "ReadAt in block 3 failed")
	// Second random read.
	_, err = f.ReadAt(buf, 2*blockSize+100)
	require.NoError(p.T(), err, "ReadAt in block 2 failed")

	// Third random read, which exceeds the threshold and triggers fallback.
	_, err = f.ReadAt(buf, 1*blockSize+100)

	require.NoError(p.T(), err, "ReadAt in block 1 failed")
	assertNonZeroCountMetric(p.T(), "buffered_read_fallback_trigger_count", "reason", "random_read_detected", p.prometheusPort)
}

func (p *PromBufferedReadTest) TestInsufficientMemoryFallback() {
	const blockSize = 4 * 1024 * 1024
	const fileSize = 10 * blockSize // 40 MiB file
	filePath := path.Join(testEnv.testDirPath, "insufficient_mem_test.txt")
	operations.CreateFileOfSize(fileSize, filePath, p.T())
	f1, err := operations.OpenFileAsReadonly(filePath)
	require.NoError(p.T(), err)
	defer operations.CloseFileShouldNotThrowError(p.T(), f1)
	f2, err := operations.OpenFileAsReadonly(filePath)
	require.NoError(p.T(), err)
	defer operations.CloseFileShouldNotThrowError(p.T(), f2)
	// Read the entire file from the first handle. This will trigger prefetching
	// that allocates blocks up to the global limit, exhausting the pool.
	_, err = io.ReadAll(f1)
	require.NoError(p.T(), err)

	// Attempt to read from the second handle. This should fail to create a
	// BufferedReader due to no available blocks, triggering the metric.
	smallBuf := make([]byte, 10)
	_, err = f2.Read(smallBuf)

	require.NoError(p.T(), err)
	assertNonZeroCountMetric(p.T(), "buffered_read_fallback_trigger_count", "reason", "insufficient_memory", p.prometheusPort)
}

func TestPromBufferedReadSuite(t *testing.T) {
	//t.SkipNow()
	ts := &PromBufferedReadTest{}
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		ts.prometheusPort = parsePortFromFlags(flags)
		log.Printf("Running prom buffered read tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
