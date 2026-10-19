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
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/kernelparams"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	// vectoredReadMaxRequestSizeKb must match --fuse-max-request-size-kb in the
	// suite's flags (test_config.yaml).
	vectoredReadMaxRequestSizeKb = 16 * 1024
	// vectoredReadChunkSize is larger than the FUSE library's ~1 MiB request
	// buffer, so each read arrives with a nil Dst and is served as a vectored
	// read through op.Data.
	vectoredReadChunkSize = 4 * 1024 * 1024
	vectoredReadFileSize  = 2 * vectoredReadChunkSize
)

// PromVectoredReadTest verifies read/block_sizes for reads larger than the
// FUSE library's request buffer. For those, jacobsa/fuse leaves ReadFileOp.Dst
// nil and the kernel reader serves them as vectored reads. Only a real mount
// can confirm that contract, so this suite runs the whole path: kernel ->
// FUSE library -> GCSFuse -> Prometheus.
type PromVectoredReadTest struct {
	PromTestBase
}

// requireHostAllowsLargeFuseRequests asserts that GCSFuse managed to raise the
// host's FUSE max_pages_limit to what --fuse-max-request-size-kb needs. Without
// that, the kernel caps requests at 1 MiB, no vectored reads happen, and the
// rest of the test would pass vacuously.
func (p *PromVectoredReadTest) requireHostAllowsLargeFuseRequests() {
	limitPath, err := kernelparams.PathForParam(kernelparams.MaxPagesLimit, 0, 0)
	require.NoError(p.T(), err)
	raw, err := os.ReadFile(limitPath)
	require.NoError(p.T(), err, "host kernel does not expose %s; it cannot issue FUSE requests above 1 MiB", limitPath)
	limit, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	require.NoError(p.T(), err, "unexpected content in %s: %q", limitPath, raw)
	required := cfg.MaxPagesForRequestSizeKb(vectoredReadMaxRequestSizeKb)
	require.GreaterOrEqual(p.T(), limit, required,
		"%s is %d pages but --fuse-max-request-size-kb=%d needs %d; GCSFuse could not raise it on this host",
		limitPath, limit, vectoredReadMaxRequestSizeKb, required)
}

func (p *PromVectoredReadTest) TestReadBlockSizesRecordsVectoredReads() {
	p.requireHostAllowsLargeFuseRequests()
	// Arrange: an object created directly in GCS, so nothing is in the page cache.
	testName := strings.ReplaceAll(p.T().Name(), "/", "_")
	gcsDir := path.Join(testDirName, testName)
	fileName := "vectored_read_" + setup.GenerateRandomString(5) + ".txt"
	client.SetupFileInTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir, fileName, vectoredReadFileSize, p.T())
	filePath := path.Join(testEnv.testDirPath, fileName)

	// Act: O_DIRECT bypasses the page cache and readahead, so the kernel forwards
	// each read as one FUSE request of exactly vectoredReadChunkSize. O_DIRECT
	// needs a page-aligned buffer.
	buf, err := cacheutil.GetMemoryAlignedBuffer(vectoredReadChunkSize, int64(os.Getpagesize()))
	require.NoError(p.T(), err)
	f, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(p.T(), err, "open with O_DIRECT")
	defer operations.CloseFileShouldNotThrowError(p.T(), f)
	for offset := int64(0); offset < vectoredReadFileSize; offset += vectoredReadChunkSize {
		n, err := f.ReadAt(buf, offset)
		require.NoError(p.T(), err, "O_DIRECT read at offset %d", offset)
		require.Equal(p.T(), vectoredReadChunkSize, n, "short read at offset %d", offset)
	}

	// Assert.
	hist := waitForHistogramSamples(p.T(), "read_block_sizes", 2, p.prometheusPort)
	// 1. The reads really were larger than the 1 MiB request buffer, i.e. Dst
	//    was nil. Guards against a silent change in the FUSE library contract.
	assert.Less(p.T(), cumulativeBucketCount(p.T(), hist, 1024*1024), hist.GetSampleCount(),
		"no read/block_sizes sample above 1 MiB: the reads were not vectored")
	// 2. No read was recorded as zero-sized.
	assert.Zero(p.T(), cumulativeBucketCount(p.T(), hist, 0),
		"read/block_sizes recorded zero-sized reads; vectored reads must record op.Size, not len(op.Dst)")
	// 3. Every byte read was recorded with its real size.
	assert.GreaterOrEqual(p.T(), hist.GetSampleSum(), float64(vectoredReadFileSize),
		"read/block_sizes sum is smaller than the bytes read")
}

func TestPromVectoredReadSuite(t *testing.T) {
	ts := &PromVectoredReadTest{}
	ts.suiteName = "TestPromVectoredReadSuite"
	flagSets := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagSets {
		ts.flags = flags
		ts.prometheusPort = parsePortFromFlags(flags)
		log.Printf("Running prom vectored read tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
