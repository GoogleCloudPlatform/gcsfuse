// Copyright 2025 Google LLC
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
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

type PromBufferedReadTest struct {
	PromTestBase
}

func (testSuite *PromBufferedReadTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = setup.BinFile()
	testSuite.mountPoint, err = os.MkdirTemp("", "gcsfuse_monitoring_tests")
	require.NoError(testSuite.T(), err)
	setPrometheusPort(testSuite.T())

	setup.SetLogFile(fmt.Sprintf("%s%s.txt", "/tmp/gcsfuse_monitoring_test_", strings.ReplaceAll(testSuite.T().Name(), "/", "_")))
	err = testSuite.mount(getBucket(testSuite.T()))
	require.NoError(testSuite.T(), err)
}

func (testSuite *PromBufferedReadTest) mount(bucketName string) error {
	testSuite.T().Helper()
	flags := []string{
		fmt.Sprintf("--prometheus-port=%d", prometheusPort),
		"--enable-buffered-read",
		"--read-block-size-mb=4",
		"--read-random-seek-threshold=2",
		"--read-global-max-blocks=5",
		"--read-min-blocks-per-handle=2",
		"--read-start-blocks-per-handle=2",
	}
	return testSuite.mountGcsfuse(bucketName, flags)
}

func (testSuite *PromBufferedReadTest) TestBufferedReadMetrics() {
	_, err := operations.ReadFile(path.Join(testSuite.mountPoint, "hello/hello.txt"))

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite.T(), "gcs_read_bytes_count", "reader", "Buffered")
	assertNonZeroCountMetric(testSuite.T(), "gcs_download_bytes_count", "read_type", "Buffered")
	assertNonZeroHistogramMetric(testSuite.T(), "buffered_read/read_latency", "", "")
}

func (testSuite *PromBufferedReadTest) TestRandomReadFallback() {
	const blockSize = 4 * 1024 * 1024
	const fileSize = 4 * blockSize
	const fileName = "random_read_fallback.txt"
	filePath := path.Join(testSuite.mountPoint, fileName)
	operations.CreateFileOfSize(fileSize, filePath, testSuite.T())
	f, err := operations.OpenFileAsReadonly(filePath)
	require.NoError(testSuite.T(), err)
	defer operations.CloseFileShouldNotThrowError(testSuite.T(), f)
	buf := make([]byte, 10)
	// With random-seek-threshold: 2, the 3rd random read should trigger a fallback.
	// First random read.
	_, err = f.ReadAt(buf, 3*blockSize+100)
	require.NoError(testSuite.T(), err, "ReadAt in block 3 failed")
	// Second random read.
	_, err = f.ReadAt(buf, 2*blockSize+100)
	require.NoError(testSuite.T(), err, "ReadAt in block 2 failed")

	// Third random read, which exceeds the threshold and triggers fallback.
	_, err = f.ReadAt(buf, 1*blockSize+100)

	require.NoError(testSuite.T(), err, "ReadAt in block 1 failed")
	assertNonZeroCountMetric(testSuite.T(), "buffered_read_fallback_trigger_count", "reason", "random_read_detected")
}

func (testSuite *PromBufferedReadTest) TestInsufficientMemoryFallback() {
	const blockSize = 4 * 1024 * 1024
	const fileSize = 10 * blockSize // 40 MiB file
	filePath := path.Join(testSuite.mountPoint, "insufficient_mem_test.txt")
	operations.CreateFileOfSize(fileSize, filePath, testSuite.T())
	f1, err := operations.OpenFileAsReadonly(filePath)
	require.NoError(testSuite.T(), err)
	defer operations.CloseFileShouldNotThrowError(testSuite.T(), f1)
	f2, err := operations.OpenFileAsReadonly(filePath)
	require.NoError(testSuite.T(), err)
	defer operations.CloseFileShouldNotThrowError(testSuite.T(), f2)
	// Read the entire file from the first handle. This will trigger prefetching
	// that allocates blocks up to the global limit, exhausting the pool.
	_, err = io.ReadAll(f1)
	require.NoError(testSuite.T(), err)

	// Attempt to read from the second handle. This should fail to create a
	// BufferedReader due to no available blocks, triggering the metric.
	smallBuf := make([]byte, 10)
	_, err = f2.Read(smallBuf)

	require.NoError(testSuite.T(), err)
	assertNonZeroCountMetric(testSuite.T(), "buffered_read_fallback_trigger_count", "reason", "insufficient_memory")
}

func TestPromBufferedReadSuite(t *testing.T) {
	t.SkipNow()
	suite.Run(t, new(PromBufferedReadTest))
}
